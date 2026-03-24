// ByteBrew Engine — Community Edition entry point.
// CE runs without license validation by default. If a license file exists
// at ~/.bytebrew/license.jwt (or is specified via --license), EE features
// are enabled with live-reloading via a background watcher.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/app"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/license"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
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
	licenseFlag := flag.String("license", "", "Path to license.jwt file")
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

	sc := app.ServerConfig{
		ConfigPath:     *configPath,
		ConfigExplicit: configExplicit,
		Port:           *port,
		Managed:        *managed,
		BridgeURL:      *bridgeFlag,
		LicenseInfo:    nil, // CE = no license, no restrictions
		Version:        version,
		Commit:         commit,
		Date:           date,
	}

	// Try to set up license watcher if a license file is available.
	watcher := initLicenseWatcher(*configPath, *licenseFlag)
	if watcher != nil {
		sc.LicenseProvider = watcher
		sc.LicenseInfo = watcher.Current()
		watcher.Start()
	}

	if err := app.Run(sc); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// initLicenseWatcher creates a LicenseWatcher if a license file and public key
// are available. Returns nil when running in pure CE mode (no license file or
// no public key configured).
func initLicenseWatcher(configPath, licenseOverride string) *license.LicenseWatcher {
	// Load license config to get public key and default path.
	licenseCfg := loadLicenseConfig(configPath)
	if licenseCfg.PublicKeyHex == "" {
		return nil
	}

	licensePath := resolveLicensePath(licenseCfg.LicensePath, licenseOverride)

	// Only create watcher if file exists or explicit flag was provided.
	if licenseOverride == "" {
		if _, err := os.Stat(licensePath); os.IsNotExist(err) {
			return nil
		}
	}

	validator, err := license.New(licenseCfg.PublicKeyHex)
	if err != nil {
		log.Printf("Warning: invalid license public key, running in CE mode: %v", err)
		return nil
	}

	return license.NewLicenseWatcher(validator, licensePath, 5*time.Minute)
}

// loadLicenseConfig attempts to load the license section from the config file.
func loadLicenseConfig(configPath string) config.LicenseConfig {
	cfg, err := config.Load(configPath)
	if err != nil {
		return config.LicenseConfig{}
	}
	return cfg.License
}

// resolveLicensePath returns the license file path, preferring the explicit
// flag, then config, then the default ~/.bytebrew/license.jwt.
func resolveLicensePath(cfgPath, flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	if cfgPath != "" {
		return cfgPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bytebrew", "license.jwt")
}
