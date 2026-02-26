package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("LIGHTBRIDGE_HTTP_PORT")
	if port == "" {
		port = "39111"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{{
				"id":       "sample-module-model",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "sample-module",
			}},
		})
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		model, _ := payload["model"].(string)
		if stream, _ := payload["stream"].(bool); stream {
			w.Header().Set("content-type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-module\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"module-stream\"},\"finish_reason\":null}]}\n\n", time.Now().Unix(), model)
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-module",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "module-ok"},
				"finish_reason": "stop",
			}},
		})
	})

	addr := "127.0.0.1:" + port
	log.Fatal(http.ListenAndServe(addr, mux))
}
