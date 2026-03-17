import * as fs from 'fs';
import * as path from 'path';

/**
 * OutputWriter - writes output to console and optionally to a file.
 * Intercepts console.log/console.error to capture all output.
 */
export class OutputWriter {
  private fileStream: fs.WriteStream | null = null;
  private originalLog: typeof console.log;
  private originalError: typeof console.error;
  private isIntercepting = false;

  constructor() {
    this.originalLog = console.log.bind(console);
    this.originalError = console.error.bind(console);
  }

  /**
   * Start writing output to a file.
   * All console.log output will be written to both console and file.
   */
  startFileOutput(filePath: string): void {
    // Ensure directory exists
    const dir = path.dirname(filePath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    // Open file for writing (overwrite if exists)
    this.fileStream = fs.createWriteStream(filePath, { flags: 'w', encoding: 'utf8' });

    // Write header
    const timestamp = new Date().toISOString();
    this.fileStream.write(`# Output captured at ${timestamp}\n\n`);

    // Start intercepting console.log
    this.interceptConsole();
  }

  /**
   * Intercept console.log and console.error to capture output
   */
  private interceptConsole(): void {
    if (this.isIntercepting) return;
    this.isIntercepting = true;

    // Override console.log
    console.log = (...args: unknown[]) => {
      // Write to original console
      this.originalLog(...args);

      // Write to file (strip ANSI codes for clean output)
      if (this.fileStream) {
        const text = args.map(arg => this.formatArg(arg)).join(' ');
        const cleanText = this.stripAnsi(text);
        this.fileStream.write(cleanText + '\n');
      }
    };

    // Override console.error for debug output
    console.error = (...args: unknown[]) => {
      // Write to original console.error
      this.originalError(...args);

      // Write to file with [ERROR] prefix
      if (this.fileStream) {
        const text = args.map(arg => this.formatArg(arg)).join(' ');
        const cleanText = this.stripAnsi(text);
        this.fileStream.write(cleanText + '\n');
      }
    };
  }

  /**
   * Format an argument for string output
   */
  private formatArg(arg: unknown): string {
    if (typeof arg === 'string') return arg;
    if (typeof arg === 'number' || typeof arg === 'boolean') return String(arg);
    if (arg === null) return 'null';
    if (arg === undefined) return 'undefined';
    try {
      return JSON.stringify(arg, null, 2);
    } catch {
      return String(arg);
    }
  }

  /**
   * Strip ANSI escape codes from text
   */
  private stripAnsi(text: string): string {
    // eslint-disable-next-line no-control-regex
    return text.replace(/\x1b\[[0-9;]*m/g, '')  // Colors
               .replace(/\x1b\]8;;[^\x07]*\x07([^\x1b]*)\x1b\]8;;\x07/g, '$1');  // OSC 8 hyperlinks
  }

  /**
   * Stop file output and restore console.
   * Waits for the file stream to flush before returning.
   */
  async stop(): Promise<void> {
    if (this.isIntercepting) {
      console.log = this.originalLog;
      console.error = this.originalError;
      this.isIntercepting = false;
    }

    if (this.fileStream) {
      await new Promise<void>((resolve) => {
        this.fileStream!.end(resolve);
      });
      this.fileStream = null;
    }
  }

  /**
   * Write directly to file (bypasses console)
   */
  writeToFile(text: string): void {
    if (this.fileStream) {
      this.fileStream.write(text + '\n');
    }
  }
}

// Global singleton for easy access
let globalOutputWriter: OutputWriter | null = null;

/**
 * Initialize output writer with optional file output
 */
export function initOutputWriter(outputPath?: string): OutputWriter {
  if (globalOutputWriter) {
    globalOutputWriter.stop();
  }

  globalOutputWriter = new OutputWriter();

  if (outputPath) {
    globalOutputWriter.startFileOutput(outputPath);
  }

  return globalOutputWriter;
}

/**
 * Get the global output writer
 */
export function getOutputWriter(): OutputWriter | null {
  return globalOutputWriter;
}

/**
 * Stop and cleanup the global output writer.
 * Waits for file stream to flush before returning.
 */
export async function stopOutputWriter(): Promise<void> {
  if (globalOutputWriter) {
    await globalOutputWriter.stop();
    globalOutputWriter = null;
  }
}
