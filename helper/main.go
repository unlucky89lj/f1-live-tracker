// F1 Koppeling: a tiny local bridge so the F1 Live Tracker website may talk to MultiViewer.
// It only listens on this computer (127.0.0.1:10102), only forwards to MultiViewer's API
// (127.0.0.1:10101/api/graphql) and only answers the F1 Live Tracker website.
package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const target = "http://127.0.0.1:10101/api/graphql"

func allowed(origin string) bool {
	return origin == "https://unlucky89lj.github.io" || origin == "null" ||
		strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1")
}

func cors(w http.ResponseWriter, r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o != "" && !allowed(o) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return false
	}
	if o != "" {
		w.Header().Set("Access-Control-Allow-Origin", o)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Private-Network", "true")
	w.Header().Set("Access-Control-Max-Age", "600")
	return true
}

func main() {
	client := &http.Client{Timeout: 10 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if !cors(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"app":"F1 Koppeling","version":"1.0"}`)
	})
	mux.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		if !cors(w, r) || r.Method != http.MethodPost {
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
		go func() { time.Sleep(200 * time.Millisecond); os.Exit(0) }()
	})
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		if !cors(w, r) {
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req, _ := http.NewRequest(http.MethodPost, target, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, `{"errors":[{"message":"MultiViewer is niet open"}]}`)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:10102")
	if err != nil { // already running
		os.Exit(0)
	}
	http.Serve(ln, mux)
}
