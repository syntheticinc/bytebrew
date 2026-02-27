//go:build prompt

package prompt_regression

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestSearchBaseline loads snapshots from the most recent session and prints metrics.
// Not a regression test — a measurement tool. Run it after headless queries to collect data.
//
// Usage:
//
//	cd vector-srv
//	go test -tags prompt -v -timeout 120s -run TestSearchBaseline ./tests/prompt_regression/...
func TestSearchBaseline(t *testing.T) {
	logsDir := filepath.Join("..", "..", "logs") // from tests/prompt_regression → bytebrew-srv/logs
	// Also try absolute path if relative doesn't work
	if _, err := os.Stat(logsDir); err != nil {
		logsDir = `C:\Users\busul\GolandProjects\usm-epicsmasher\bytebrew-srv\logs`
	}

	// Find all session directories
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("read logs dir %s: %v", logsDir, err)
	}

	sessionDirs := make([]string, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Skip saved baseline copies
		if e.Name() == "." || e.Name() == ".." {
			continue
		}
		sessionDir := filepath.Join(logsDir, e.Name())
		// Check if it has snapshot files
		pattern := filepath.Join(sessionDir, "supervisor_step_*_context.json")
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			sessionDirs = append(sessionDirs, sessionDir)
		}
	}

	if len(sessionDirs) == 0 {
		t.Skipf("no session directories with snapshots found in %s", logsDir)
	}

	t.Logf("Found %d sessions with snapshots\n", len(sessionDirs))

	for _, sessionDir := range sessionDirs {
		dirName := filepath.Base(sessionDir)
		t.Run(dirName, func(t *testing.T) {
			snapshots, err := LoadSnapshots(sessionDir)
			if err != nil {
				t.Fatalf("load snapshots: %v", err)
			}

			t.Logf("Loaded %d snapshots from %s\n", len(snapshots), dirName)

			metrics := ExtractSearchMetrics(snapshots)
			report := FormatMetricsReport(metrics)

			// Print report
			fmt.Println(report)
			t.Log("\n" + report)

			// Save report to file
			reportPath := filepath.Join(sessionDir, "search_metrics_report.txt")
			if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
				t.Logf("Warning: failed to save report: %v", err)
			}
		})
	}
}
