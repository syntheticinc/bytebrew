// Command executor using execa for cross-platform shell execution
import { execa, ExecaError } from 'execa';
import { getLogger } from '../../lib/logger.js';

export interface CommandResult {
  stdout: string;
  stderr: string;
  exitCode: number | null;
  timedOut: boolean;
  error?: Error;
}

export interface ExecuteOptions {
  command: string;
  cwd: string;
  timeout: number; // in seconds
}

const MAX_OUTPUT_SIZE = 1024 * 1024; // 1MB max output

// Cached bash availability (checked once on first command execution)
let bashShell: string | true | undefined;

/**
 * Detect bash availability. Returns 'bash' path if found, or `true` for default shell.
 * On Unix, bash is always available. On Windows, checks Git Bash.
 */
async function detectShell(): Promise<string | true> {
  if (process.platform !== 'win32') {
    return 'bash';
  }

  // Windows: try bash (Git for Windows)
  try {
    await execa({ shell: false, reject: true, timeout: 5000 })`bash --version`;
    return 'bash';
  } catch {
    // bash not found — fallback to default shell (cmd.exe)
    return true;
  }
}

/**
 * Get the shell to use for command execution.
 * Caches the result after first detection.
 */
async function getShell(): Promise<string | true> {
  if (bashShell === undefined) {
    bashShell = await detectShell();
    const logger = getLogger();
    logger.debug('Shell detected', {
      shell: bashShell === true ? 'default (cmd.exe)' : bashShell,
    });
  }
  return bashShell;
}

/**
 * Truncate output to MAX_OUTPUT_SIZE
 */
function truncateOutput(output: string): string {
  if (output.length <= MAX_OUTPUT_SIZE) {
    return output;
  }
  return output.slice(0, MAX_OUTPUT_SIZE) + '\n... (output truncated)';
}

/**
 * Execute a shell command and return the result.
 * Uses bash when available (cross-platform), falls back to default shell (cmd.exe on Windows).
 */
export async function executeCommand(options: ExecuteOptions): Promise<CommandResult> {
  const logger = getLogger();
  const { command, cwd, timeout } = options;
  const timeoutMs = timeout * 1000;
  const shell = await getShell();

  logger.debug('Executing command', { command, cwd, timeout, shell: shell === true ? 'default' : shell });

  try {
    const result = await execa({
      shell,
      cwd,
      timeout: timeoutMs,
      reject: false, // Don't throw on non-zero exit code
      env: {
        ...process.env,
        // Ensure consistent output encoding
        LANG: 'en_US.UTF-8',
        LC_ALL: 'en_US.UTF-8',
        PYTHONIOENCODING: 'utf-8',
      },
    })`${command}`;

    logger.debug('Command completed', {
      command,
      exitCode: result.exitCode,
      timedOut: result.timedOut,
      stdoutLen: result.stdout.length,
      stderrLen: result.stderr.length,
    });

    return {
      stdout: truncateOutput(result.stdout),
      stderr: truncateOutput(result.stderr),
      exitCode: result.exitCode ?? null,
      timedOut: result.timedOut ?? false,
    };
  } catch (err) {
    const error = err as ExecaError;

    // Handle timeout
    if (error.timedOut) {
      logger.warn('Command timed out', { command, timeout });
      return {
        stdout: truncateOutput(String(error.stdout ?? '')),
        stderr: truncateOutput(String(error.stderr ?? '')),
        exitCode: null,
        timedOut: true,
      };
    }

    // Handle other errors
    logger.error('Command execution error', { command, error: error.message });
    return {
      stdout: truncateOutput(String(error.stdout ?? '')),
      stderr: truncateOutput(String(error.stderr ?? '')),
      exitCode: error.exitCode ?? null,
      timedOut: false,
      error,
    };
  }
}

/**
 * Format command result for LLM consumption
 */
export function formatCommandResult(result: CommandResult): string {
  const parts: string[] = [];

  // Add stdout
  if (result.stdout.trim()) {
    parts.push(result.stdout.trim());
  }

  // Add stderr if present
  if (result.stderr.trim()) {
    // Don't prefix stderr if it's the only output (common for git, compilers)
    if (parts.length === 0) {
      parts.push(result.stderr.trim());
    } else {
      parts.push(`\n[stderr]\n${result.stderr.trim()}`);
    }
  }

  // Handle empty output
  if (parts.length === 0) {
    if (result.timedOut) {
      return '[Command timed out with no output]';
    }
    if (result.exitCode === 0) {
      return '[Command completed successfully with no output]';
    }
    return `[Command exited with code ${result.exitCode}]`;
  }

  // Add exit code if non-zero
  let output = parts.join('\n');
  if (result.exitCode !== null && result.exitCode !== 0) {
    output += `\n\n[Exit code: ${result.exitCode}]`;
  }
  if (result.timedOut) {
    output += '\n\n[Command timed out]';
  }

  return output;
}
