package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	// Tuya devices accept local-protocol connections on this port.
	tcpPort        = "6668"
	attempts       = 3
	attemptTimeout = time.Second
	retryDelay     = 300 * time.Millisecond
)

// ping sends one ICMP echo over an unprivileged datagram socket, so it needs neither root
// nor a ping binary. On Linux this requires the process's group to be in
// net.ipv4.ping_group_range, which Docker allows for all groups by default.
func ping(ctx context.Context, ip string) bool {
	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip4", ip)
	if err != nil || len(addrs) == 0 {
		return false
	}
	dst := addrs[0]

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Printf("ping: cannot open ICMP socket: %v", err)
		return false
	}
	defer conn.Close()
	// Unblocks ReadFrom once ctx expires or the TCP check has already succeeded.
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("ping-monitor")},
	}
	b, err := msg.Marshal(nil)
	if err != nil {
		return false
	}
	if _, err := conn.WriteTo(b, &net.UDPAddr{IP: dst}); err != nil {
		return false
	}

	buf := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return false
		}
		reply, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), buf[:n])
		if err != nil || reply.Type != ipv4.ICMPTypeEchoReply {
			continue
		}
		if udp, ok := peer.(*net.UDPAddr); ok && udp.IP.Equal(dst) {
			return true
		}
	}
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
