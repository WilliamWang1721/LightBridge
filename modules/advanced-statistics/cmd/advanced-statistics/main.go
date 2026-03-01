package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"lightbridge/internal/advstats"
)

func main() {
	port := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_HTTP_PORT"))
	if port == "" {
		port = "39141"
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("invalid LIGHTBRIDGE_HTTP_PORT: %q", port)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/stats/aggregate", handleAggregate)

	addr := "127.0.0.1:" + port
	log.Printf("advanced-statistics module listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleAggregate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req advstats.AggregateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	result := advstats.Aggregate(req, time.Now().UTC())
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
