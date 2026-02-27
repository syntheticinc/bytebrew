// debugLog.ts — temporary file-based debug logging for diagnosing execute_command hang.
// Active only when BYTEBREW_DEBUG_LOG=1. No-op otherwise. Safe to leave in production code.
import { appendFileSync, mkdirSync } from 'fs';
import path from 'path';

let debugFile: string | null = null;

/**
 * Initialize debug logging to file.
 * Call once at app startup. Only active when BYTEBREW_DEBUG_LOG=1.
 */
export function initDebugLog(projectRoot: string): void {
  if (process.env.BYTEBREW_DEBUG_LOG !== '1') return;
  const dir = path.join(projectRoot, '.bytebrew', 'logs');
  try { mkdirSync(dir, { recursive: true }); } catch { /* ignore */ }
  debugFile = path.join(dir, 'debug.log');
  try {
    appendFileSync(debugFile, `\n--- New session ${new Date().toISOString()} ---\n`);
  } catch { /* ignore */ }
}

/**
 * Write a debug line to the log file.
 * No-op if BYTEBREW_DEBUG_LOG is not set.
 */
export function debugLog(tag: string, msg: string, data?: Record<string, unknown>): void {
  if (!debugFile) return;
  const ts = new Date().toISOString();
  const extra = data ? ' ' + JSON.stringify(data) : '';
  try {
    appendFileSync(debugFile, `[${ts}] [${tag}] ${msg}${extra}\n`);
  } catch { /* ignore */ }
}
