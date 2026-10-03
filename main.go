package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"
)

const (
	// Tuya devices accept local-protocol connections on this port.
	tcpPort        = "6668"
	attempts       = 3
	attemptTimeout = time.Second
	retryDelay     = 300 * time.Millisecond
)

// ping relies on ctx for the timeout instead of -W, whose unit differs between Linux and macOS.
func ping(ctx context.Context, ip string) bool {
	return exec.CommandContext(ctx, "ping", "-c", "1", ip).Run() == nil
}

func tcpConnect(ctx context.Context, ip string) bool {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, tcpPort))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// probe runs ping and the TCP check concurrently; the host is up if either succeeds.
func probe(ctx context.Context, ip string) bool {
	ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	result := make(chan bool, 2)
	go func() { result <- ping(ctx, ip) }()
	go func() { result <- tcpConnect(ctx, ip) }()

	for range 2 {
		if <-result {
			return true
		}
	}
	return false
}

func isUp(ctx context.Context, ip string) bool {
	for i := range attempts {
		if i > 0 {
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return false
			}
		}
		if probe(ctx, ip) {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, statusCode int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status: status,
	})
}

type HealthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "missing ip query parameter", http.StatusBadRequest)
		return
	}

	if isUp(r.Context(), ip) {
		writeJSON(w, http.StatusOK, "UP")
	} else {
		log.Printf("%s is DOWN after %d attempts", ip, attempts)
		writeJSON(w, http.StatusServiceUnavailable, "DOWN")
	}
}

func main() {
	http.HandleFunc("/health", healthHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
