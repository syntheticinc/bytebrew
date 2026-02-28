package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ExecuteCommandArgs represents arguments for execute_command tool
type ExecuteCommandArgs struct {
	Command    string `json:"command,omitempty"`
	Cwd        string `json:"cwd,omitempty"`
	Timeout    int32  `json:"timeout,omitempty"`
	Background bool   `json:"background,omitempty"`
	BgAction   string `json:"bg_action,omitempty"`
	BgID       string `json:"bg_id,omitempty"`
}

// ExecuteCommandTool implements Eino InvokableTool for executing shell commands via gRPC
type ExecuteCommandTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewExecuteCommandTool creates a new execute_command tool
func NewExecuteCommandTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &ExecuteCommandTool{
		proxy:     proxy,
		sessionID: sessionID,
	}
}

// Info returns tool information for LLM
func (t *ExecuteCommandTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "execute_command",
		Desc: `Run shell commands in a persistent bash session. State (cwd, env vars) persists between calls.

Two modes:
1. FOREGROUND (default): blocking, waits for completion, returns output. Timeout 30s (max 120s).
   Use for: build, test, git, install, any command that finishes.
2. BACKGROUND (background=true): non-blocking, returns immediately with process ID.
   MUST use for: servers, watchers, "go run ./cmd/server", "npm start", "npm run dev", any process that runs indefinitely.

Background process management:
- Start: execute_command(command="go run ./cmd/server", background=true) → returns "bg-1"
- Read:  execute_command(bg_action="read", bg_id="bg-1") → shows output
- List:  execute_command(bg_action="list") → all processes
- Kill:  execute_command(bg_action="kill", bg_id="bg-1") → stop

NEVER for file operations — use dedicated tools:
- read_file, write_file, edit_file (NOT cat/echo/sed/awk)
- smart_search (NOT grep/rg/findstr)
- get_project_tree (NOT ls/dir/find/tree)

Rules:
- Non-interactive only (no prompts, no stdin). Add -y or --yes flags.
- Check ENVIRONMENT context for OS before using platform-specific syntax.
- If command fails, read the error and fix — don't retry the same command.
- State persists: cd, export, aliases survive between calls.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "The shell command to execute (e.g., 'npm install', 'go build ./...')",
				Required: false,
			},
			"cwd": {
				Type:     schema.String,
				Desc:     "Working directory relative to project root (optional, defaults to project root)",
				Required: false,
			},
			"timeout": {
				Type:     schema.Integer,
				Desc:     "Timeout in seconds (default 30, max 120). If exceeded, command is interrupted.",
				Required: false,
			},
			"background": {
				Type:     schema.Boolean,
				Desc:     "Run command in background (for servers, watchers). Returns immediately with process ID.",
				Required: false,
			},
			"bg_action": {
				Type:     schema.String,
				Desc:     "Manage background processes (use instead of 'command'): 'list', 'read', 'kill'. Example: {\"bg_action\":\"read\",\"bg_id\":\"bg-1\"}",
				Required: false,
			},
			"bg_id": {
				Type:     schema.String,
				Desc:     "Background process ID for read/kill bg_action (e.g. 'bg-1'). Returned when starting with background=true.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *ExecuteCommandTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ExecuteCommandArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "ExecuteCommandTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for execute_command: %v. Please provide command parameter.", err), nil
	}

	// Validate arguments based on mode
	if args.BgAction != "" {
		// Background action mode - command is optional (bg_action=list doesn't need command)
		if (args.BgAction == "read" || args.BgAction == "kill") && args.BgID == "" {
			slog.WarnContext(ctx, "ExecuteCommandTool: bg_id required for bg_action read/kill", "bg_action", args.BgAction)
			return fmt.Sprintf("[ERROR] bg_id is required for bg_action '%s'. Please provide the background process ID.", args.BgAction), nil
		}
	} else {
		// Regular command execution mode - command is required
		if args.Command == "" {
			slog.WarnContext(ctx, "ExecuteCommandTool: command is required but was empty", "raw_args", argumentsInJSON)
			return "[ERROR] command is required. Please specify the command you want to execute, e.g. {\"command\": \"npm install\"}", nil
		}

		// Detect source code being passed as command (only for foreground commands)
		if !args.Background && looksLikeSourceCode(args.Command) {
			slog.WarnContext(ctx, "ExecuteCommandTool: command contains source code, should use write_file",
				"command_length", len(args.Command),
				"command_preview", truncateForLog(args.Command, 100))
			return "[ERROR] The command parameter contains source code. Do NOT use execute_command to create files. " +
				"Use write_file tool instead to create or overwrite files with source code content.", nil
		}

		// Check for dangerous commands (data exfiltration, destructive operations)
		if dangerous, reason := isDangerousCommand(args.Command); dangerous {
			slog.WarnContext(ctx, "ExecuteCommandTool: dangerous command blocked",
				"command", args.Command, "reason", reason)
			return fmt.Sprintf("[SECURITY] Command blocked: %s. Reason: %s.", args.Command, reason), nil
		}
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	// Default timeout is 30 seconds, max 120.
	// Commands exceeding timeout are auto-promoted to background on the client.
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 120 {
		timeout = 120
	}

	slog.InfoContext(ctx, "ExecuteCommandTool: executing command",
		"command", args.Command,
		"cwd", args.Cwd,
		"timeout", timeout,
		"background", args.Background,
		"bg_action", args.BgAction,
		"bg_id", args.BgID)

	// Build arguments map for proxy
	arguments := map[string]string{
		"timeout": fmt.Sprintf("%d", timeout),
	}
	if args.Command != "" {
		arguments["command"] = args.Command
	}
	if args.Cwd != "" {
		arguments["cwd"] = args.Cwd
	}
	if args.Background {
		arguments["background"] = "true"
	}
	if args.BgAction != "" {
		arguments["bg_action"] = args.BgAction
	}
	if args.BgID != "" {
		arguments["bg_id"] = args.BgID
	}

	// Call gRPC client proxy to execute command
	result, err := t.proxy.ExecuteCommandFull(ctx, t.sessionID, arguments)
	if err != nil {
		errCode := errors.GetCode(err)
		errMsg := err.Error()
		slog.WarnContext(ctx, "ExecuteCommandTool: error executing command",
			"command", args.Command,
			"error", err,
			"code", errCode)

		// Security block - return as soft error
		if errCode == errors.CodePermissionDenied {
			return fmt.Sprintf("[SECURITY] Command blocked: %s. This command is not allowed for security reasons.", args.Command), nil
		}

		// Timeout
		if errCode == errors.CodeTimeout {
			return fmt.Sprintf("[ERROR] Command timed out after %d seconds: %s", timeout, args.Command), nil
		}

		// User rejected command
		if errCode == errors.CodeCancelled {
			return fmt.Sprintf("[CANCELLED] Command was rejected by user: %s", args.Command), nil
		}

		// Generic error
		return fmt.Sprintf("[ERROR] Failed to execute command '%s': %v", args.Command, errMsg), nil
	}

	slog.InfoContext(ctx, "ExecuteCommandTool: command executed successfully",
		"command", args.Command,
		"result_length", len(result))

	return result, nil
}

// looksLikeSourceCode detects when a command contains source code or non-command content.
// This prevents LLM from using execute_command to create files via heredoc/echo
// or passing JSON/structured data as a command.
func looksLikeSourceCode(command string) bool {
	// Short commands are never source code
	if len(command) < 200 {
		return false
	}

	// Very long single-line strings are suspicious — real shell commands are rarely > 500 chars
	if len(command) > 500 && strings.Count(command, "\n") < 3 {
		return true
	}

	// JSON arrays/objects are not shell commands
	trimmed := strings.TrimSpace(command)
	if (strings.HasPrefix(trimmed, "[{") && strings.HasSuffix(trimmed, "}]")) ||
		(strings.HasPrefix(trimmed, "{\"") && len(trimmed) > 500) {
		return true
	}

	// Multi-line check: count actual newlines + escaped \n sequences
	lineCount := strings.Count(command, "\n") + strings.Count(command, `\n`)
	if lineCount < 5 {
		return false
	}

	// Check for common source code patterns
	codePatterns := []string{
		"package ", "func ", "import (", "type ", "struct {",    // Go
		"class ", "interface ", "extends ", "implements ",         // Java/TS/C#
		"def ", "from ", "import ",                                // Python
		"const ", "let ", "var ", "function ", "export ",          // JS/TS
		"#include", "namespace ", "using ",                        // C/C++/C#
	}

	patternCount := 0
	for _, pattern := range codePatterns {
		if strings.Contains(command, pattern) {
			patternCount++
		}
	}

	// 3+ code patterns in a long command = almost certainly source code
	return patternCount >= 3
}

// isDangerousCommand detects potentially dangerous commands that could be used
// for data exfiltration or destructive operations.
// Returns (isDangerous bool, reason string).
func isDangerousCommand(command string) (bool, string) {
	cmdLower := strings.ToLower(command)

	// Data exfiltration: piping to network tools (case-insensitive)
	exfilPatterns := []struct {
		pattern string
		reason  string
	}{
		{"| curl", "piping output to curl (potential data exfiltration)"},
		{"| wget", "piping output to wget (potential data exfiltration)"},
		{"| nc ", "piping output to netcat (potential data exfiltration)"},
		{"| nc\n", "piping output to netcat (potential data exfiltration)"},
		{"|curl", "piping output to curl (potential data exfiltration)"},
		{"|wget", "piping output to wget (potential data exfiltration)"},
		{"curl -d ", "sending data with curl (potential data exfiltration)"},
		{"curl --data", "sending data with curl (potential data exfiltration)"},
		{"curl --form", "uploading file with curl (potential data exfiltration)"},
		{"curl --upload", "uploading with curl (potential data exfiltration)"},
	}

	for _, p := range exfilPatterns {
		if strings.Contains(cmdLower, p.pattern) {
			return true, p.reason
		}
	}

	// Case-sensitive exfiltration patterns (curl -F is --form, -f is --fail)
	caseSensitivePatterns := []struct {
		pattern string
		reason  string
	}{
		{"curl -F ", "uploading file with curl (potential data exfiltration)"},
	}

	for _, p := range caseSensitivePatterns {
		if strings.Contains(command, p.pattern) {
			return true, p.reason
		}
	}

	// Destructive patterns — check with boundary awareness
	// "rm -rf /" must not match "rm -rf /tmp/build" (legitimate path)
	if isDestructiveRm(cmdLower) {
		return true, "recursive deletion of critical directory"
	}

	simpleDestructivePatterns := []struct {
		pattern string
		reason  string
	}{
		{"mkfs.", "filesystem formatting"},
		{"mkfs ", "filesystem formatting"},
		{"dd if=/dev/zero", "disk overwrite with zeros"},
		{":(){ :|:& };:", "fork bomb"},
	}

	for _, p := range simpleDestructivePatterns {
		if strings.Contains(cmdLower, p.pattern) {
			return true, p.reason
		}
	}

	return false, ""
}

// isDestructiveRm checks for dangerous rm -rf patterns while avoiding false positives.
// "rm -rf /" is dangerous, but "rm -rf /tmp/build" is legitimate.
func isDestructiveRm(cmdLower string) bool {
	dangerousTargets := []string{"rm -rf /", "rm -rf ~", "rm -rf $home"}
	for _, target := range dangerousTargets {
		idx := strings.Index(cmdLower, target)
		if idx == -1 {
			continue
		}
		endIdx := idx + len(target)
		// Pattern must be at end of string or followed by space/semicolon/pipe/newline
		// NOT followed by a path character (letter, digit, dot, underscore, dash)
		if endIdx >= len(cmdLower) {
			return true
		}
		next := cmdLower[endIdx]
		if next == ' ' || next == '\n' || next == '\t' || next == ';' || next == '&' || next == '|' {
			return true
		}
	}
	return false
}

// truncateForLog truncates a string for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
