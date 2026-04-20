//go:build integration

// Package integration contains CE-side HTTP integration tests for the engine.
// Tests run against a real CE server wired to a real PostgreSQL container with
// Liquibase-applied schema. Each file uses //go:build integration so the suite
// is opt-in (go test -tags integration).
//
// Run with:
//
//	go test -tags integration ./tests/integration/... -v -timeout 180s
//
// Requires a running Docker daemon. When Docker is unavailable the suite
// auto-skips so the build stays green.
package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	ceserver "github.com/syntheticinc/bytebrew/engine/pkg/server"
)

const (
	// jwtSecret must match writeBootstrapConfig's security.jwt_secret value —
	// the HMACVerifier is built from exactly this string.
	jwtSecret = "integration-test-hmac-secret"
	// ceTenantID is the single tenant row CE seeds for LoginEnabled=true
	// deployments. All CE rows default to this uuid via GORM defaults.
	ceTenantID = "00000000-0000-0000-0000-000000000001"
)

var (
	baseURL    string
	adminToken string
	testDB     *gorm.DB

	// suiteSkipReason — non-empty means setup bailed (no Docker, etc.) and
	// each test file's requireSuite(t) will call t.Skip instead of fail.
	suiteSkipReason atomic.Value // string
)

func skipReason() string {
	v := suiteSkipReason.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

// TestMain is the suite entry point. It runs once per process.
//
// Existing non-CE tests (production_harness_test.go, streaming_api_test.go,
// ws_api_test.go, v2_test.go) live in this same package and do NOT call
// requireSuite — they must keep running even when Docker is missing. So we
// never hard-fail here: setup errors flip suiteSkipReason and only CE suite
// tests skip.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cleanup, err := setupSuite(ctx)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		suiteSkipReason.Store(fmt.Sprintf("ce integration suite setup failed: %v", err))
		os.Exit(m.Run())
	}
	os.Exit(m.Run())
}

func setupSuite(ctx context.Context) (func(), error) {
	cleanups := &cleanupStack{}
	cleanup := func() { cleanups.run() }

	pg, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("bytebrew_ce_test"),
		tcpostgres.WithUsername("bytebrew"),
		tcpostgres.WithPassword("bytebrew_ce_test_pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return cleanup, fmt.Errorf("start postgres: %w", err)
	}
	cleanups.push(func() { _ = pg.Terminate(context.Background()) })

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return cleanup, fmt.Errorf("postgres connection string: %w", err)
	}

	// Liquibase migrations live at ../../migrations relative to this file
	// (engine/tests/integration → engine/migrations).
	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		return cleanup, fmt.Errorf("resolve migrations dir: %w", err)
	}
	if _, statErr := os.Stat(migrationsDir); statErr != nil {
		return cleanup, fmt.Errorf("migrations dir not found: %w", statErr)
	}
	if err := applyLiquibaseMigrations(ctx, pg, migrationsDir); err != nil {
		return cleanup, fmt.Errorf("apply liquibase migrations: %w", err)
	}

	httpPort, err := pickFreePort()
	if err != nil {
		return cleanup, fmt.Errorf("pick free port: %w", err)
	}

	dataDir, err := os.MkdirTemp("", "bytebrew-ce-it-")
	if err != nil {
		return cleanup, fmt.Errorf("mkdir data: %w", err)
	}
	cleanups.push(func() { _ = os.RemoveAll(dataDir) })

	configPath := filepath.Join(dataDir, "config.yaml")
	if err := writeBootstrapConfig(configPath, connStr, httpPort); err != nil {
		return cleanup, fmt.Errorf("write bootstrap config: %w", err)
	}

	// Isolate the engine's portfile / logs inside dataDir so we don't collide
	// with a developer's running engine under their real profile dir.
	restoreEnv := setEnvIsolated(dataDir)
	cleanups.push(restoreEnv)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	cleanups.push(serverCancel)

	go func() {
		_ = ceserver.Run(ceserver.Config{
			ConfigPath:     configPath,
			ConfigExplicit: true,
			LoginEnabled:   true,
			RequireTenant:  false,
			Version:        "ce-integration-test",
			Commit:         "none",
			Date:           "none",
		})
		_ = serverCtx
	}()

	baseURL = fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	if err := waitForHealthy(ctx, baseURL, 60*time.Second); err != nil {
		return cleanup, fmt.Errorf("wait for engine healthy: %w", err)
	}

	// Open a direct GORM connection for test-side seeding + assertions.
	// This is intentionally separate from the engine's pool — truncation
	// must work regardless of engine state.
	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	if err != nil {
		return cleanup, fmt.Errorf("open test gorm: %w", err)
	}
	testDB = db

	// Seed an admin user so TC-SEC-06 and any login-path test has credentials.
	if err := seedAdminUser(ctx, db); err != nil {
		return cleanup, fmt.Errorf("seed admin user: %w", err)
	}

	// Cache an always-admin token for test helpers that need one without
	// going through /auth/login. jwtSecret matches the server's verifier.
	adminToken = tokenFor("admin-test")

	return cleanup, nil
}

// seedAdminUser inserts the canonical admin/admin123 user in the CE default
// tenant. ON CONFLICT makes re-runs idempotent if someone wires a persistent
// container layer.
func seedAdminUser(ctx context.Context, db *gorm.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
	if err != nil {
		return fmt.Errorf("bcrypt admin password: %w", err)
	}
	return db.WithContext(ctx).Exec(`
		INSERT INTO users (id, tenant_id, username, password_hash, role, disabled)
		VALUES (gen_random_uuid(), ?::uuid, 'admin', ?, 'admin', false)
		ON CONFLICT (username) DO NOTHING
	`, ceTenantID, string(hash)).Error
}

// cleanupStack is a tiny LIFO teardown stack. Panics in one cleanup don't
// abort later ones.
type cleanupStack struct{ fns []func() }

func (c *cleanupStack) push(f func()) { c.fns = append(c.fns, f) }
func (c *cleanupStack) run() {
	for i := len(c.fns) - 1; i >= 0; i-- {
		func() {
			defer func() { _ = recover() }()
			c.fns[i]()
		}()
	}
}

// applyLiquibaseMigrations runs the official liquibase image against the
// testcontainers postgres instance using its docker-network IP (not the
// host-mapped port, which the liquibase container can't reach).
func applyLiquibaseMigrations(ctx context.Context, pg *tcpostgres.PostgresContainer, migrationsDir string) error {
	pgHost, err := pg.ContainerIP(ctx)
	if err != nil {
		return fmt.Errorf("postgres container ip: %w", err)
	}
	jdbcURL := fmt.Sprintf("jdbc:postgresql://%s:5432/bytebrew_ce_test", pgHost)

	req := testcontainers.ContainerRequest{
		Image: "liquibase/liquibase:4.30",
		Mounts: testcontainers.ContainerMounts{
			{
				Source: testcontainers.GenericBindMountSource{HostPath: migrationsDir},
				Target: "/liquibase/changelog",
			},
		},
		Cmd: []string{
			"--url=" + jdbcURL,
			"--username=bytebrew",
			"--password=bytebrew_ce_test_pass",
			"--changeLogFile=db.changelog-master.yaml",
			"--searchPath=/liquibase/changelog",
			"update",
		},
		WaitingFor: wait.ForExit().WithExitTimeout(120 * time.Second),
	}

	liq, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("start liquibase container: %w", err)
	}
	defer func() { _ = liq.Terminate(context.Background()) }()

	state, err := liq.State(ctx)
	if err != nil {
		return fmt.Errorf("liquibase state: %w", err)
	}
	if state.ExitCode != 0 {
		logs, _ := liq.Logs(ctx)
		body := ""
		if logs != nil {
			buf := make([]byte, 8192)
			n, _ := logs.Read(buf)
			body = string(buf[:n])
		}
		return fmt.Errorf("liquibase exited %d: %s", state.ExitCode, body)
	}
	return nil
}

// writeBootstrapConfig emits a YAML config that passes both config.Load
// (legacy validation) and config.LoadBootstrap (bootstrap path). Also writes
// prompts.yaml next to config.yaml — config.Load fails without it.
func writeBootstrapConfig(path, dbURL string, port int) error {
	content := fmt.Sprintf(`engine:
  host: "127.0.0.1"
  port: %d
database:
  url: %q
  host: "localhost"
security:
  jwt_secret: %q
logging:
  level: "warn"
llm:
  default_provider: "ollama"
`, port, dbURL, jwtSecret)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	promptsPath := filepath.Join(filepath.Dir(path), "prompts.yaml")
	promptsContent := "prompts:\n  system_prompt: \"integration-test assistant\"\n"
	return os.WriteFile(promptsPath, []byte(promptsContent), 0644)
}

// pickFreePort grabs a free TCP port on 127.0.0.1 and closes the listener.
// Tiny TOCTOU window between close and server bind — acceptable for a
// one-shot harness.
func pickFreePort() (int, error) {
	lst, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = lst.Close() }()
	addr, ok := lst.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type")
	}
	return addr.Port, nil
}

// setEnvIsolated points platform-specific data-dir env vars inside dataDir so
// the engine's portfile/logs land in our temp dir, not the user's profile.
func setEnvIsolated(dataDir string) func() {
	type kv struct {
		key string
		old string
		had bool
	}
	keys := []string{"APPDATA", "XDG_DATA_HOME", "HOME"}
	saved := make([]kv, 0, len(keys))
	for _, k := range keys {
		old, had := os.LookupEnv(k)
		saved = append(saved, kv{key: k, old: old, had: had})
	}
	_ = os.Setenv("APPDATA", dataDir)
	_ = os.Setenv("XDG_DATA_HOME", dataDir)
	_ = os.Setenv("HOME", dataDir)
	return func() {
		for _, s := range saved {
			if s.had {
				_ = os.Setenv(s.key, s.old)
			} else {
				_ = os.Unsetenv(s.key)
			}
		}
	}
}

// waitForHealthy polls GET /api/v1/health until 200 or timeout.
func waitForHealthy(ctx context.Context, base string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := client.Get(base + "/api/v1/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("engine did not become healthy within %s: %w", timeout, lastErr)
}

// Keep the jwt import reachable from suite_test.go so build tag parsing
// doesn't complain when the only other user lives in helpers_test.go.
var _ = jwt.SigningMethodHS256
