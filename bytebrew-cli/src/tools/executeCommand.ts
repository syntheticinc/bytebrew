// execute_command tool implementation
// Permission checks are handled by ToolExecutorAdapter.
// This tool is responsible for: input validation, execution routing (foreground/background/bg_action).
import path from 'path';
import { Tool, ToolResult } from './registry.js';
import type { ShellSessionManager } from '../infrastructure/shell/ShellSessionManager.js';
import { executeCommand, formatCommandResult } from '../infrastructure/shell/commandExecutor.js';
import { getLogger } from '../lib/logger.js';
import { debugLog } from '../lib/debugLog.js';

export class ExecuteCommandTool implements Tool {
  readonly name = 'execute_command';
  readonly needsContext = true;
  private projectRoot: string;
  private shellManager?: ShellSessionManager;

  constructor(projectRoot: string, shellManager?: ShellSessionManager) {
    this.projectRoot = projectRoot;
    this.shellManager = shellManager;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();

    // Background process management actions
    const bgAction = args.bg_action;
    if (bgAction) {
      return this.handleBgAction(bgAction, args.bg_id);
    }

    // Background mode: spawn and return immediately
    const background = args.background === 'true';
    if (background) {
      return this.handleBackground(args);
    }

    // Foreground: execute in persistent session if available, otherwise fallback to execa
    if (this.shellManager) {
      return this.handleForeground(args);
    } else {
      return this.handleLegacy(args);
    }
  }

  private async handleForeground(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();

    // Separate agentId (injected by ToolExecutor) from shell arguments
    const { _agent_id: agentId, ...shellArgs } = args;

    const command = shellArgs.command;
    debugLog('CMD', 'handleForeground start', { command: command?.slice(0, 80), timeout: shellArgs.timeout, agentId });

    if (!command) {
      return {
        result: '[ERROR] command argument is required',
        error: new Error('command argument is required'),
      };
    }

    let timeout = parseInt(shellArgs.timeout || '30', 10);
    if (isNaN(timeout) || timeout <= 0) timeout = 30;
    if (timeout > 120) timeout = 120;

    const session = this.shellManager!.getAvailableSession(this.projectRoot, agentId);
    if (!session) {
      return {
        result: '[ERROR] All shell sessions are busy (3/3). Wait for a command to finish or use background=true.',
        error: new Error('All shell sessions busy'),
      };
    }

    logger.info('Executing command (foreground)', { command, timeout });

    try {
      debugLog('CMD', 'session.execute start', { isAlive: session.isAlive(), isExecuting: session.isExecuting() });
      const result = await session.execute(command, timeout * 1000);
      debugLog('CMD', 'session.execute done', { completed: result.completed, exitCode: result.exitCode, outputLen: result.stdout?.length });

      if (!result.completed) {
        // Timeout — interrupt and return partial output with hint
        await session.interrupt();
        debugLog('CMD', 'foreground timeout', { command: command.slice(0, 80), timeout });

        let output = result.stdout.trim();
        const hint = `[Command timed out after ${timeout}s — interrupted]\n[Use background=true for servers, watchers, and long-running processes]`;

        return {
          result: output ? `${output}\n\n${hint}` : hint,
          summary: 'timed out',
        };
      }

      // Completed
      let output = result.stdout.trim();

      if (!output) {
        if (result.exitCode === 0) {
          output = '[Command completed successfully with no output]';
        } else {
          output = `[Command exited with code ${result.exitCode}]`;
        }
      } else if (result.exitCode !== null && result.exitCode !== 0) {
        output += `\n\n[Exit code: ${result.exitCode}]`;
      }

      return {
        result: output,
        summary: `exit ${result.exitCode}`,
      };
    } catch (error: any) {
      logger.error('ExecuteCommandTool foreground error', { command, error: error.message });
      debugLog('CMD', 'handleForeground ERROR', { error: error.message });
      return {
        result: `[ERROR] ${error.message}`,
        error,
      };
    }
  }

  private handleBackground(args: Record<string, string>): ToolResult {
    const logger = getLogger();
    const command = args.command;

    if (!command) {
      return {
        result: '[ERROR] command argument is required for background execution',
        error: new Error('command argument is required'),
      };
    }

    if (!this.shellManager) {
      return {
        result: '[ERROR] Background execution not supported — ShellSessionManager not available',
        error: new Error('ShellSessionManager not available'),
      };
    }

    const bgManager = this.shellManager.getBackgroundManager();
    const info = bgManager.spawn(command, this.projectRoot);

    logger.info('Started background process', { id: info.id, pid: info.pid, command });

    return {
      result: `Started background process ${info.id} (PID: ${info.pid})\nCommand: ${command}\nTo check output: call execute_command with {"bg_action":"read","bg_id":"${info.id}"}\nTo stop: call execute_command with {"bg_action":"kill","bg_id":"${info.id}"}`,
      summary: `started ${info.id}`,
    };
  }

  private handleBgAction(action: string, bgId?: string): ToolResult {
    if (!this.shellManager) {
      return {
        result: '[ERROR] Background process management not supported — ShellSessionManager not available',
        error: new Error('ShellSessionManager not available'),
      };
    }

    const bgManager = this.shellManager.getBackgroundManager();

    switch (action) {
      case 'list': {
        const processes = bgManager.list();
        if (processes.length === 0) {
          return { result: 'No background processes running.' };
        }
        const lines = processes.map(p => {
          const duration = formatDuration(Date.now() - p.startTime.getTime());
          if (p.status === 'running') {
            return `  ${p.id}: ${p.command} (running, ${duration}, PID: ${p.pid})`;
          }
          return `  ${p.id}: ${p.command} (exited, code ${p.exitCode ?? '?'}, ${duration} ago)`;
        });
        return {
          result: `Background processes:\n${lines.join('\n')}`,
          summary: `${processes.length} process${processes.length !== 1 ? 'es' : ''}`,
        };
      }

      case 'read': {
        if (!bgId) {
          return {
            result: '[ERROR] bg_id is required for read action. Use bg_action="list" to see available processes.',
          };
        }
        const output = bgManager.readOutput(bgId);
        if (output === null) {
          return {
            result: `Process "${bgId}" not found. Background processes don't persist across CLI restarts. Use bg_action="list" to see current processes, or start a new one with background=true.`,
          };
        }
        const proc = bgManager.get(bgId);
        const statusLine = proc?.status === 'running'
          ? '[Process still running]'
          : `[Process exited with code ${proc?.exitCode ?? '?'}]`;

        const content = output.trim() || '(no output yet)';
        return {
          result: `[${bgId} output]:\n${content}\n${statusLine}`,
          summary: proc?.status === 'running' ? 'running' : `exit ${proc?.exitCode}`,
        };
      }

      case 'kill': {
        if (!bgId) {
          return {
            result: '[ERROR] bg_id is required for kill action. Use bg_action="list" to see available processes.',
          };
        }
        const proc = bgManager.get(bgId);
        if (!proc) {
          return {
            result: `Process "${bgId}" not found. Background processes don't persist across CLI restarts. Use bg_action="list" to see current processes.`,
          };
        }
        // Kill async, but return immediately
        void bgManager.kill(bgId);
        return {
          result: `Process ${bgId} killed (PID: ${proc.pid})`,
          summary: `killed ${bgId}`,
        };
      }

      default:
        return {
          result: `[ERROR] Unknown bg_action: "${action}". Valid actions: list, read, kill`,
          error: new Error(`Unknown bg_action: ${action}`),
        };
    }
  }

  /**
   * Legacy execa-based execution for backwards compatibility when ShellSessionManager is not available.
   */
  private async handleLegacy(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const command = args.command;

    if (!command) {
      return {
        result: '[ERROR] command argument is required',
        error: new Error('command argument is required'),
      };
    }

    const cwd = args.cwd || undefined;
    let timeout = parseInt(args.timeout || '30', 10);

    if (isNaN(timeout) || timeout <= 0) {
      timeout = 30;
    }
    if (timeout > 300) {
      timeout = 300;
    }

    logger.debug('ExecuteCommandTool called (legacy mode)', { command, cwd, timeout });

    try {
      // Resolve working directory
      const workingDir = this.resolveWorkingDir(cwd);

      // Execute the command
      logger.info('Executing command (legacy execa)', { command, cwd: workingDir, timeout });

      const result = await executeCommand({
        command,
        cwd: workingDir,
        timeout,
      });

      if (result.error) {
        logger.error('Command execution error', { command, error: result.error.message });
        return {
          result: `[ERROR] Failed to execute command: ${result.error.message}`,
          error: result.error,
        };
      }

      if (result.timedOut) {
        logger.warn('Command timed out', { command, timeout });
        let timeoutResult = formatCommandResult(result);
        if (!timeoutResult.includes('[Command timed out]')) {
          timeoutResult += '\n\n[Command timed out]';
        }
        return {
          result: timeoutResult,
          summary: 'timed out',
        };
      }

      const output = formatCommandResult(result);
      logger.debug('Command completed', { command, exitCode: result.exitCode, outputLen: output.length });

      return {
        result: output,
        summary: `exit ${result.exitCode}`,
      };
    } catch (error: any) {
      logger.error('ExecuteCommandTool error', { command, error: error.message });
      return {
        result: `[ERROR] ${error.message}`,
        error,
      };
    }
  }

  private resolveWorkingDir(cwd: string | undefined): string {
    if (!cwd) {
      return this.projectRoot;
    }
    return path.resolve(this.projectRoot, cwd);
  }
}

function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
}
