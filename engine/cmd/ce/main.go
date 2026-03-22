// ByteBrew Engine — Community Edition entry point.
// CE runs without license validation. Full functionality, no restrictions.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/syntheticinc/bytebrew/engine/internal/app"
)

var (
	version = "dev-ce"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	port := flag.Int("port", 0, "Override server port (0 = use config)")
	managed := flag.Bool("managed", false, "Run in managed subprocess mode")
	bridgeFlag := flag.String("bridge", "", "Bridge WebSocket URL")
	flag.Parse()

	if *showVersion {
		fmt.Printf("bytebrew-ce %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	configExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configExplicit = true
		}
	})

	if err := app.Run(app.ServerConfig{
		ConfigPath:     *configPath,
		ConfigExplicit: configExplicit,
		Port:           *port,
		Managed:        *managed,
		BridgeURL:      *bridgeFlag,
		LicenseInfo:    nil, // CE = no license, no restrictions
		Version:        version,
		Commit:         commit,
		Date:           date,
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
