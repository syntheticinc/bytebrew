// Shell detection and process management utilities.
// Adapted from OpenCode (MIT): https://github.com/anthropics/opencode
import path from 'path';
import { spawn, type ChildProcess } from 'child_process';

const SIGKILL_TIMEOUT_MS = 200;

/** Shells that are not POSIX-compatible and need fallback */
const BLACKLIST = new Set(['fish', 'nu']);

export namespace Shell {
  /**
   * Kill a process and all its children.
   * - Windows: taskkill /f /t (force + tree)
   * - Unix: SIGTERM to process group, SIGKILL after 200ms if still alive
   */
  export async function killTree(proc: ChildProcess, opts?: { exited?: () => boolean }): Promise<void> {
    const pid = proc.pid;
    if (!pid || opts?.exited?.()) return;

    if (process.platform === 'win32') {
      await new Promise<void>((resolve) => {
        const killer = spawn('taskkill', ['/pid', String(pid), '/f', '/t'], { stdio: 'ignore' });
        killer.once('exit', () => resolve());
        killer.once('error', () => resolve());
      });
      return;
    }

    // Unix: kill process group
    try {
      process.kill(-pid, 'SIGTERM');
      await Bun.sleep(SIGKILL_TIMEOUT_MS);
      if (!opts?.exited?.()) {
        process.kill(-pid, 'SIGKILL');
      }
    } catch {
      // No process group — kill directly
      proc.kill('SIGTERM');
      await Bun.sleep(SIGKILL_TIMEOUT_MS);
      if (!opts?.exited?.()) {
        proc.kill('SIGKILL');
      }
    }
  }

  /**
   * Fallback shell detection when $SHELL is unavailable or blacklisted.
   */
  function fallback(): string {
    if (process.platform === 'win32') {
      // Env override for non-standard Git installations
      const override = process.env.BYTEBREW_GIT_BASH_PATH;
      if (override) return override;

      // Derive bash.exe from git.exe location
      const git = Bun.which('git');
      if (git) {
        // git.exe: C:\Program Files\Git\cmd\git.exe
        // bash.exe: C:\Program Files\Git\bin\bash.exe
        const bash = path.join(git, '..', '..', 'bin', 'bash.exe');
        try {
          if (Bun.file(bash).size) return bash;
        } catch { /* not found at this path */ }
      }

      // cmd.exe fallback (limited compatibility with POSIX commands)
      return process.env.COMSPEC || 'cmd.exe';
    }

    if (process.platform === 'darwin') return '/bin/zsh';

    const bash = Bun.which('bash');
    if (bash) return bash;
    return '/bin/sh';
  }

  /** Cached shell path */
  let _acceptable: string | null = null;

  /**
   * Returns a POSIX-compatible shell suitable for command execution.
   * Uses $SHELL if set and not blacklisted, otherwise falls back
   * to platform-specific detection.
   *
   * On Windows: finds Git Bash via git.exe location (avoids WSL's bash.exe).
   * Override with BYTEBREW_GIT_BASH_PATH env var if needed.
   *
   * Result is cached after first call.
   */
  export function acceptable(): string {
    if (_acceptable !== null) return _acceptable;

    const s = process.env.SHELL;
    if (s && !BLACKLIST.has(
      process.platform === 'win32' ? path.win32.basename(s) : path.basename(s)
    )) {
      _acceptable = s;
      return _acceptable;
    }

    _acceptable = fallback();
    return _acceptable;
  }
}
