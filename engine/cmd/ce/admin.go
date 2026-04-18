// Admin user management CLI subcommand.
//
// Usage: ce admin <create|list|reset-password> [flags]
//
// Talks directly to the database (DATABASE_URL env var) — the engine does
// NOT need to be running. No fallback on config/env admin credentials: the
// `users` table is the single source of truth for admin/system users.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

func runAdminCommand(args []string) int {
	if len(args) == 0 {
		printAdminUsage()
		return 2
	}
	switch args[0] {
	case "create":
		return adminCreate(args[1:])
	case "list":
		return adminList(args[1:])
	case "reset-password":
		return adminResetPassword(args[1:])
	case "-h", "--help", "help":
		printAdminUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown admin subcommand: %s\n\n", args[0])
		printAdminUsage()
		return 2
	}
}

func printAdminUsage() {
	fmt.Fprintln(os.Stderr, "Usage: ce admin <subcommand> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  create          Create a new admin/system user")
	fmt.Fprintln(os.Stderr, "  list            List all users")
	fmt.Fprintln(os.Stderr, "  reset-password  Reset a user's password")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run 'ce admin <subcommand> -h' for subcommand flags.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Requires DATABASE_URL environment variable (same connection string as the engine).")
}

func openDBForAdmin() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL env var is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// readPasswordFromTerminal prompts on stderr and reads a password without echo.
// Uses os.Stdin's file descriptor so it works across Windows, macOS, and Linux.
func readPasswordFromTerminal(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimSpace(string(pw)), nil
}

func adminCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	username := fs.String("username", "", "username (required)")
	password := fs.String("password", "", "password (if empty, read from stdin)")
	role := fs.String("role", "admin", "role: admin | system")
	tenantID := fs.String("tenant-id", "00000000-0000-0000-0000-000000000001", "tenant UUID (single-tenant default)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *username == "" {
		fmt.Fprintln(os.Stderr, "--username is required")
		fs.Usage()
		return 2
	}
	if *role != "admin" && *role != "system" {
		fmt.Fprintln(os.Stderr, "--role must be 'admin' or 'system'")
		return 2
	}

	if *password == "" {
		pw, err := readPasswordFromTerminal("Password: ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		*password = pw
	}
	if *password == "" {
		fmt.Fprintln(os.Stderr, "password cannot be empty")
		return 2
	}

	db, err := openDBForAdmin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bcrypt: %v\n", err)
		return 1
	}

	user := models.UserModel{
		TenantID:     *tenantID,
		Username:     *username,
		PasswordHash: string(hash),
		Role:         *role,
	}
	if err := db.WithContext(context.Background()).Create(&user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "create user: %v\n", err)
		return 1
	}
	fmt.Printf("Created user: id=%s username=%s role=%s tenant_id=%s\n",
		user.ID, user.Username, user.Role, user.TenantID)
	return 0
}

func adminList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	db, err := openDBForAdmin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	var users []models.UserModel
	if err := db.WithContext(context.Background()).
		Order("created_at").
		Find(&users).Error; err != nil {
		fmt.Fprintf(os.Stderr, "list users: %v\n", err)
		return 1
	}

	if len(users) == 0 {
		fmt.Println("(no users)")
		return 0
	}

	fmt.Printf("%-36s  %-30s  %-8s  %-8s  %s\n", "ID", "USERNAME", "ROLE", "DISABLED", "TENANT_ID")
	for _, u := range users {
		fmt.Printf("%-36s  %-30s  %-8s  %-8v  %s\n",
			u.ID, u.Username, u.Role, u.Disabled, u.TenantID)
	}
	return 0
}

func adminResetPassword(args []string) int {
	fs := flag.NewFlagSet("reset-password", flag.ContinueOnError)
	username := fs.String("username", "", "username (required)")
	password := fs.String("password", "", "new password (if empty, read from stdin)")
	tenantID := fs.String("tenant-id", "00000000-0000-0000-0000-000000000001", "tenant UUID")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *username == "" {
		fmt.Fprintln(os.Stderr, "--username is required")
		return 2
	}
	if *password == "" {
		pw, err := readPasswordFromTerminal("New password: ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		*password = pw
	}
	if *password == "" {
		fmt.Fprintln(os.Stderr, "password cannot be empty")
		return 2
	}

	db, err := openDBForAdmin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bcrypt: %v\n", err)
		return 1
	}

	result := db.WithContext(context.Background()).
		Model(&models.UserModel{}).
		Where("tenant_id = ? AND username = ?", *tenantID, *username).
		Update("password_hash", string(hash))
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "update: %v\n", result.Error)
		return 1
	}
	if result.RowsAffected == 0 {
		fmt.Fprintf(os.Stderr, "no user found with username %q in tenant %s\n", *username, *tenantID)
		return 1
	}
	fmt.Printf("Password reset for user %s\n", *username)
	return 0
}
