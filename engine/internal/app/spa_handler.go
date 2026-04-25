package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// mountSPA serves a single-page application from `dir` under `mountPrefix`,
// falling back to `index.html` for client-side routing. Returns false (and
// logs at INFO) if the directory does not exist — admin/web-client SPA
// assets are optional so the engine boots cleanly in slim builds.
func mountSPA(router chi.Router, mountPrefix, dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		slog.InfoContext(context.Background(), "SPA not found (optional)", "prefix", mountPrefix, "path", dir)
		return false
	}
	spaFS := http.Dir(dir)
	fileHandler := func(w http.ResponseWriter, req *http.Request) {
		filePath := strings.TrimPrefix(req.URL.Path, mountPrefix)
		if filePath == "" || filePath == "/" {
			filePath = "/index.html"
		}
		if _, err := os.Stat(filepath.Join(dir, filePath)); os.IsNotExist(err) {
			http.ServeFile(w, req, filepath.Join(dir, "index.html"))
			return
		}
		http.StripPrefix(mountPrefix, http.FileServer(spaFS)).ServeHTTP(w, req)
	}
	redirect := func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, mountPrefix+"/", http.StatusMovedPermanently)
	}
	router.Get(mountPrefix+"/*", fileHandler)
	router.Get(mountPrefix, redirect)
	slog.InfoContext(context.Background(), "SPA served", "prefix", mountPrefix, "path", dir)
	return true
}
