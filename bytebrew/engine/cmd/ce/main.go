// ByteBrew Engine — Community Edition entry point.
// CE runs without license validation. Full functionality, no restrictions.
package main

import (
	"fmt"
	"os"
)

var (
	version = "dev-ce"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("bytebrew-ce %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// CE reuses the same server logic but without license validation.
	// For now, CE delegates to cmd/server until infrastructure is refactored
	// to support license-free mode. The key difference is:
	// - CE binary name: bytebrew-ce
	// - No license config required
	// - LicenseInfo = nil (all features enabled)
	fmt.Println("ByteBrew Engine CE — use cmd/server for now (license is optional)")
	fmt.Printf("Version: %s, Commit: %s\n", version, commit)
}
