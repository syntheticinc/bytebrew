package shell

import (
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	KillGracePeriod     = 5 * time.Second
	AutoCleanupDuration = 30 * time.Minute
)

// BackgroundProcess holds metadata about a background process.
type BackgroundProcess struct {
	ID        string
	Command   string
	PID       int
	StartTime time.Time
	Status    string // "running" or "exited"
	ExitCode  int
}

// bgEntry is the internal bookkeeping for a background process.
type bgEntry struct {
	info         BackgroundProcess
	cmd          *exec.Cmd
	buffer       *OutputBuffer
	cleanupTimer *time.Timer
}

// BackgroundProcessManager spawns and manages background processes.
type BackgroundProcessManager struct {
	processes map[string]*bgEntry
	nextID    int
	mu        sync.Mutex
}

// NewBackgroundProcessManager creates a new BackgroundProcessManager.
func NewBackgroundProcessManager() *BackgroundProcessManager {
	return &BackgroundProcessManager{
		processes: make(map[string]*bgEntry),
	}
}

// Spawn starts a command in the background and returns its metadata.
func (m *BackgroundProcessManager) Spawn(command, cwd string) (*BackgroundProcess, error) {
	shell := detectShell()
	cmd := exec.Command(shell, "-c", command)
	cmd.Dir = cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start background process: %w", err)
	}

	m.mu.Lock()
	m.nextID++
	id := fmt.Sprintf("bg-%d", m.nextID)

	buf := NewOutputBuffer(DefaultMaxSize)
	entry := &bgEntry{
		info: BackgroundProcess{
			ID:        id,
			Command:   command,
			PID:       cmd.Process.Pid,
			StartTime: time.Now(),
			Status:    "running",
		},
		cmd:    cmd,
		buffer: buf,
	}
	m.processes[id] = entry
	m.mu.Unlock()

	// Pipe output to buffer
	go pipeReader(buf, stdout)
	go pipeReader(buf, stderr)

	// Wait for exit in background
	go m.waitForExit(id, cmd)

	slog.Info("background process spawned",
		"id", id,
		"pid", cmd.Process.Pid,
		"command", command)

	return &entry.info, nil
}

// List returns metadata for all tracked background processes.
func (m *BackgroundProcessManager) List() []*BackgroundProcess {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*BackgroundProcess, 0, len(m.processes))
	for _, e := range m.processes {
		info := e.info // copy
		result = append(result, &info)
	}
	return result
}

// ReadOutput returns the current output buffer for a background process.
func (m *BackgroundProcessManager) ReadOutput(id string) (string, error) {
	m.mu.Lock()
	entry, ok := m.processes[id]
	m.mu.Unlock()

	if !ok {
		return "", fmt.Errorf("background process %s not found", id)
	}

	return entry.buffer.GetOutput(), nil
}

// Kill terminates a background process.
// On Windows: taskkill /T /F. On Unix: SIGTERM then SIGKILL after grace period.
func (m *BackgroundProcessManager) Kill(id string) error {
	m.mu.Lock()
	entry, ok := m.processes[id]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("background process %s not found", id)
	}

	if entry.info.Status == "exited" {
		return nil
	}

	return killProcess(entry.cmd)
}

// DisposeAll kills and removes all background processes.
func (m *BackgroundProcessManager) DisposeAll() {
	m.mu.Lock()
	// Collect entries and reset map under lock, then release before killing
	// to avoid deadlock with waitForExit goroutines.
	entries := m.processes
	m.processes = make(map[string]*bgEntry)
	m.mu.Unlock()

	for _, entry := range entries {
		if entry.cleanupTimer != nil {
			entry.cleanupTimer.Stop()
		}
		if entry.info.Status == "running" {
			_ = killProcess(entry.cmd)
		}
	}
}

// waitForExit waits for the process to exit, updates status, and starts cleanup timer.
func (m *BackgroundProcessManager) waitForExit(id string, cmd *exec.Cmd) {
	err := cmd.Wait()

	m.mu.Lock()
	entry, ok := m.processes[id]
	if !ok {
		m.mu.Unlock()
		return
	}

	entry.info.Status = "exited"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			entry.info.ExitCode = exitErr.ExitCode()
		} else {
			entry.info.ExitCode = -1
		}
	}

	// Auto-cleanup after duration
	entry.cleanupTimer = time.AfterFunc(AutoCleanupDuration, func() {
		m.mu.Lock()
		delete(m.processes, id)
		m.mu.Unlock()
	})

	m.mu.Unlock()

	slog.Info("background process exited",
		"id", id,
		"exit_code", entry.info.ExitCode)
}

// killProcess kills a process tree. Platform-specific implementation.
func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		return killProcessWindows(cmd)
	}
	return killProcessUnix(cmd)
}

// killProcessWindows kills the process tree using taskkill.
// Falls back to Process.Kill() if taskkill fails or times out.
func killProcessWindows(cmd *exec.Cmd) error {
	pid := strconv.Itoa(cmd.Process.Pid)
	kill := exec.Command("taskkill", "/T", "/F", "/PID", pid)

	done := make(chan error, 1)
	go func() {
		done <- kill.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			// taskkill failed (e.g. process tree issue), fall back to direct kill
			return cmd.Process.Kill()
		}
		return nil
	case <-time.After(3 * time.Second):
		// taskkill hung (process likely already dead), fall back
		if kill.Process != nil {
			_ = kill.Process.Kill()
		}
		return cmd.Process.Kill()
	}
}

// killProcessUnix sends SIGTERM, waits for grace period, then SIGKILL.
func killProcessUnix(cmd *exec.Cmd) error {
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may already be dead
		return cmd.Process.Kill()
	}

	// Wait for graceful shutdown
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(KillGracePeriod):
		return cmd.Process.Kill()
	}
}

// pipeReader reads from r and appends to buffer until EOF.
func pipeReader(buf *OutputBuffer, r io.Reader) {
	tmp := make([]byte, 8192)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf.Append(string(tmp[:n]))
		}
		if err != nil {
			return
		}
	}
}
