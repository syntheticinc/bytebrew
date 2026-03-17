// ByteBrew Code Server — BFF proxy to Engine with license validation.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	engineURL := flag.String("engine", "http://localhost:8443", "Engine REST API URL")
	port := flag.String("port", "60401", "Port to listen on")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("bytebrew-code-server %s (commit: %s)\n", version, commit)
		os.Exit(0)
	}

	target, err := url.Parse(*engineURL)
	if err != nil {
		slog.Error("invalid engine URL", "url", *engineURL, "error", err)
		os.Exit(1)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: license validation middleware
		proxy.ServeHTTP(w, r)
	})

	slog.Info("Code BFF starting", "port", *port, "engine", *engineURL)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
