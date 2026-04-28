package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

// ShellResult represents the result of a command execution.
type ShellResult struct {
	Stdout    string
	ExitCode  int  // -1 if timed out
	Completed bool // true if marker was found
}

// ShellSession maintains a persistent bash process for executing commands.
// Environment state (cwd, exports, aliases) persists between Execute calls.
type ShellSession struct {
	process     *exec.Cmd
	stdin       io.WriteCloser
	buffer      *OutputBuffer
	isExecuting atomic.Bool
	shell       string
	cwd         string
}

// NewShellSession creates a new ShellSession that will spawn processes in the given cwd.
func NewShellSession(cwd string, maxOutputSize int) *ShellSession {
	if maxOutputSize <= 0 {
		maxOutputSize = DefaultMaxSize
	}
	return &ShellSession{
		buffer: NewOutputBuffer(maxOutputSize),
		shell:  detectShell(),
		cwd:    cwd,
	}
}

// Execute runs a command in the persistent shell and waits for completion.
// The ctx parameter controls the wait timeout only; the shell process itself
// persists beyond individual calls.
func (s *ShellSession) Execute(ctx context.Context, command string, timeout time.Duration) (*ShellResult, error) {
	if !s.isExecuting.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("session is already executing a command")
	}
	defer s.isExecuting.Store(false)

	if err := s.ensureAlive(ctx); err != nil {
		return nil, fmt.Errorf("ensure shell alive: %w", err)
	}

	markerID, wrapped := WrapCommand(command)
	s.buffer.Reset()

	if _, err := fmt.Fprintln(s.stdin, wrapped); err != nil {
		return nil, fmt.Errorf("write command to shell: %w", err)
	}

	result, err := s.buffer.WaitForMarker(markerID, timeout)
	if err != nil {
		// Timeout: interrupt and return partial output
		slog.WarnContext(ctx, "command timed out, interrupting", "timeout", timeout)
		_ = s.Interrupt()
		return &ShellResult{
			Stdout:    s.buffer.GetOutput(),
			ExitCode:  -1,
			Completed: false,
		}, nil
	}

	return &ShellResult{
		Stdout:    result.Output,
		ExitCode:  result.ExitCode,
		Completed: true,
	}, nil
}

// Interrupt sends Ctrl+C (ETX) to the shell stdin.
func (s *ShellSession) Interrupt() error {
	if s.stdin == nil {
		return nil
	}
	_, err := s.stdin.Write([]byte{0x03}) // Ctrl+C
	return err
}

// IsExecuting returns true if a command is currently running.
func (s *ShellSession) IsExecuting() bool {
	return s.isExecuting.Load()
}

// IsAlive returns true if the shell process is running.
func (s *ShellSession) IsAlive() bool {
	if s.process == nil {
		return false
	}
	// If ProcessState is available, the process has exited
	if s.process.ProcessState != nil {
		return false
	}
	return true
}

// Destroy kills the shell process and cleans up resources.
func (s *ShellSession) Destroy() {
	s.buffer.CancelPending()

	if s.stdin != nil {
		_ = s.stdin.Close()
		s.stdin = nil
	}

	if s.process != nil && s.process.Process != nil {
		_ = s.process.Process.Kill()
		_ = s.process.Wait()
		s.process = nil
	}
}

// ensureAlive spawns a new shell process if one is not already running.
func (s *ShellSession) ensureAlive(ctx context.Context) error {
	if s.IsAlive() {
		return nil
	}

	// Clean up old process
	s.Destroy()

	cmd := exec.Command(s.shell, "--norc", "--noprofile")
	cmd.Dir = s.cwd
	cmd.Env = append(os.Environ(),
		"TERM=dumb",
		"PS1=",
		"PS2=",
		"PROMPT_COMMAND=",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
		"PYTHONIOENCODING=utf-8",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("start shell process: %w", err)
	}

	s.process = cmd
	s.stdin = stdin
	s.buffer = NewOutputBuffer(s.buffer.maxSize)

	// Goroutines to pipe stdout/stderr to buffer
	go s.pipeToBuffer(stdout)
	go s.pipeToBuffer(stderr)

	slog.InfoContext(ctx, "shell session started",
		"shell", s.shell,
		"cwd", s.cwd,
		"pid", cmd.Process.Pid)

	return nil
}

// pipeToBuffer reads from a reader and appends chunks to the output buffer.
func (s *ShellSession) pipeToBuffer(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		s.buffer.Append(scanner.Text() + "\n")
	}
}
