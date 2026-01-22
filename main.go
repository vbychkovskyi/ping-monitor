package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"time"
)

func ping(ip string) bool {
	var cmd *exec.Cmd

	cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)

	return cmd.Run() == nil
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

	result := make(chan bool, 1)

	go func() {
		result <- ping(ip)
	}()

	select {
	case ok := <-result:
		if ok {
			writeJSON(w, http.StatusOK, "UP")
		} else {
			writeJSON(w, http.StatusServiceUnavailable, "DOWN")
		}
	case <-time.After(2 * time.Second):
		writeJSON(w, http.StatusServiceUnavailable, "DOWN")
	}
}

func main() {
	http.HandleFunc("/health", healthHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
