// Structured logging infrastructure
export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LogEntry {
  level: LogLevel;
  message: string;
  timestamp: Date;
  context?: Record<string, unknown>;
}

export interface Logger {
  debug(message: string, context?: Record<string, unknown>): void;
  info(message: string, context?: Record<string, unknown>): void;
  warn(message: string, context?: Record<string, unknown>): void;
  error(message: string, context?: Record<string, unknown>): void;
  child(context: Record<string, unknown>): Logger;
}

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

class ConsoleLogger implements Logger {
  private minLevel: LogLevel;
  private context: Record<string, unknown>;

  constructor(minLevel: LogLevel = 'info', context: Record<string, unknown> = {}) {
    this.minLevel = minLevel;
    this.context = context;
  }

  private shouldLog(level: LogLevel): boolean {
    return LOG_LEVELS[level] >= LOG_LEVELS[this.minLevel];
  }

  private formatMessage(level: LogLevel, message: string, context?: Record<string, unknown>): string {
    const timestamp = new Date().toISOString();
    const mergedContext = { ...this.context, ...context };
    const contextStr = Object.keys(mergedContext).length > 0
      ? ` ${JSON.stringify(mergedContext)}`
      : '';
    return `[${timestamp}] ${level.toUpperCase()}: ${message}${contextStr}`;
  }

  debug(message: string, context?: Record<string, unknown>): void {
    if (this.shouldLog('debug')) {
      console.debug(this.formatMessage('debug', message, context));
    }
  }

  info(message: string, context?: Record<string, unknown>): void {
    if (this.shouldLog('info')) {
      console.info(this.formatMessage('info', message, context));
    }
  }

  warn(message: string, context?: Record<string, unknown>): void {
    if (this.shouldLog('warn')) {
      console.warn(this.formatMessage('warn', message, context));
    }
  }

  error(message: string, context?: Record<string, unknown>): void {
    if (this.shouldLog('error')) {
      console.error(this.formatMessage('error', message, context));
    }
  }

  child(context: Record<string, unknown>): Logger {
    return new ConsoleLogger(this.minLevel, { ...this.context, ...context });
  }
}

// Silent logger for non-debug mode
class SilentLogger implements Logger {
  debug(): void {}
  info(): void {}
  warn(): void {}
  error(message: string, context?: Record<string, unknown>): void {
    // Always log errors even in silent mode
    console.error(`ERROR: ${message}`, context || '');
  }
  child(): Logger {
    return this;
  }
}

let globalLogger: Logger = new SilentLogger();

export function initLogger(debug: boolean = false): Logger {
  globalLogger = debug ? new ConsoleLogger('debug') : new SilentLogger();
  return globalLogger;
}

export function getLogger(): Logger {
  return globalLogger;
}

export function createLogger(context: Record<string, unknown>): Logger {
  return globalLogger.child(context);
}
