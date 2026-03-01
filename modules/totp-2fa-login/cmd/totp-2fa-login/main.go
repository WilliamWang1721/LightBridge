package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	port := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_HTTP_PORT"))
	if port == "" {
		port = "39113"
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("invalid LIGHTBRIDGE_HTTP_PORT: %q", port)
	}

	dataDir := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_DATA_DIR"))
	if dataDir == "" {
		dataDir = "."
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("mkdir data dir: %v", err)
	}

	s := &server{
		dataDir:      dataDir,
		storePath:    filepath.Join(dataDir, "totp_devices.json"),
		enrollStates: map[string]pendingEnroll{},
		stateTTL:     5 * time.Minute,
		stateMaxSkew: 10 * time.Second,
	}
	if err := s.loadStore(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("store: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/totp/enroll/begin", s.handleEnrollBegin)
	mux.HandleFunc("/totp/enroll/confirm", s.handleEnrollConfirm)
	mux.HandleFunc("/totp/verify", s.handleVerify)
	mux.HandleFunc("/totp/devices", s.handleDevices)
	mux.HandleFunc("/totp/devices/delete", s.handleDeviceDelete)

	addr := "127.0.0.1:" + port
	log.Printf("totp-2fa-login module listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

type server struct {
	dataDir   string
	storePath string

	storeMu sync.Mutex
	store   totpStore

	stateMu      sync.Mutex
	enrollStates map[string]pendingEnroll
	stateTTL     time.Duration
	stateMaxSkew time.Duration
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": strings.TrimSpace(msg)})
}
