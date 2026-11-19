package server

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// EmbeddedDist holds static assets built into the Go binary.
//
//go:embed all:static
var EmbeddedDist embed.FS

// RegisterStaticHandlers binds static asset routes to the HTTP multiplexer.
func RegisterStaticHandlers(mux *http.ServeMux) {
	// Serve local examples if present
	examplesDir := "./examples"
	if fi, err := os.Stat(examplesDir); err == nil && fi.IsDir() {
		mux.Handle("/examples/", http.StripPrefix("/examples/", http.FileServer(http.Dir(examplesDir))))
	}

	// 1. Try local web/dist folder (development & local runs)
	localDist := "./web/dist"
	if fi, err := os.Stat(localDist); err == nil && fi.IsDir() {
		fileServer := http.FileServer(http.Dir(localDist))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/stream/") {
				return
			}
			path := filepath.Join(localDist, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				// Single-Page Application (SPA) fallback to index.html
				http.ServeFile(w, r, filepath.Join(localDist, "index.html"))
				return
			}
			fileServer.ServeHTTP(w, r)
		})
		return
	}

	// 2. Try embedded static assets
	subFS, err := fs.Sub(EmbeddedDist, "static")
	if err == nil {
		fileServer := http.FileServer(http.FS(subFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/stream/") {
				return
			}
			fileServer.ServeHTTP(w, r)
		})
		return
	}

	// 3. Fallback placeholder
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/stream/") {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>voiceprobe</title></head><body style="background:#0a0c10;color:#e2e8f0;font-family:sans-serif;padding:40px;"><h2>voiceprobe Core Engine Running</h2><p>Web Cockpit assets loading on :7979</p></body></html>`))
	})
}
