// Glob search using ripgrep --files for fast file name matching
import { spawn } from 'child_process';
import { stat } from 'fs/promises';
import * as path from 'path';
import { getLogger } from '../../lib/logger.js';
import { resolveRgBinary } from './ripgrepResolver.js';

export interface GlobMatch {
  filePath: string; // relative path from project root
  mtime: number; // ms since epoch, for sorting
}

export interface GlobSearchOptions {
  maxResults?: number;
}

const DEFAULT_MAX_RESULTS = 100;

/**
 * Search for files matching glob pattern using ripgrep --files
 * Falls back to git ls-files if ripgrep not available
 * Results sorted by modification time (newest first)
 */
export async function globSearch(
  projectRoot: string,
  pattern: string,
  options: GlobSearchOptions = {}
): Promise<GlobMatch[]> {
  const logger = getLogger();

  if (!pattern) {
    return [];
  }

  const maxResults = options.maxResults ?? DEFAULT_MAX_RESULTS;

  const rgPath = await resolveRgBinary();

  if (rgPath) {
    try {
      const files = await executeRipgrepFiles(rgPath, projectRoot, pattern, maxResults);
      logger.debug('Glob search completed', { pattern, resultsCount: files.length });
      return files;
    } catch (error) {
      logger.debug('Ripgrep files failed, trying git ls-files fallback', { error });
    }
  } else {
    logger.debug('ripgrep not available, using git ls-files fallback', { pattern });
  }

  try {
    const files = await executeGitLsFiles(projectRoot, pattern, maxResults);
    return files;
  } catch (fallbackError) {
    logger.error('Glob search failed', { error: (fallbackError as Error).message });
    return [];
  }
}

async function executeRipgrepFiles(
  rgPath: string,
  projectRoot: string,
  pattern: string,
  maxResults: number
): Promise<GlobMatch[]> {
  return new Promise((resolve, reject) => {
    // Build args: rg --files --glob '<pattern>' --glob '!.git/*'
    const args = [
      '--files',
      '--glob',
      pattern,
      '--glob',
      '!.git/*',
      '--glob',
      '!node_modules/*',
      '--glob',
      '!dist/*',
      '--glob',
      '!.next/*',
      '--glob',
      '!vendor/*',
    ];

    const proc = spawn(rgPath, args, {
      cwd: projectRoot,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (data) => {
      stdout += data;
    });
    proc.stderr.on('data', (data) => {
      stderr += data;
    });

    proc.on('close', async (code) => {
      // exit 0 = found, 1 = no matches, 2 = error
      if (code === 2 && !stdout) {
        reject(new Error(`ripgrep error: ${stderr}`));
        return;
      }

      const lines = stdout.split(/\r?\n/).filter((line) => line.trim());

      // Stat files for mtime and build results
      const results = (await Promise.all(
        lines.slice(0, maxResults + 10).map(async (line): Promise<GlobMatch | null> => {
          const filePath = line.trim();
          if (!filePath) return null;

          const fullPath = path.join(projectRoot, filePath);
          try {
            const stats = await stat(fullPath);
            return {
              filePath: filePath.replace(/\\/g, '/'), // normalize to forward slashes
              mtime: stats.mtimeMs,
            };
          } catch {
            return {
              filePath: filePath.replace(/\\/g, '/'),
              mtime: 0,
            };
          }
        })
      )).filter((r): r is GlobMatch => r !== null);

      // Sort by mtime DESC (newest first)
      results.sort((a, b) => b.mtime - a.mtime);

      resolve(results.slice(0, maxResults));
    });

    proc.on('error', (err) => {
      reject(err);
    });
  });
}

async function executeGitLsFiles(
  projectRoot: string,
  pattern: string,
  maxResults: number
): Promise<GlobMatch[]> {
  return new Promise((resolve, reject) => {
    const proc = spawn('git', ['ls-files', pattern], {
      cwd: projectRoot,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    proc.stdout.on('data', (data) => {
      stdout += data;
    });

    proc.on('close', async (code) => {
      if (code !== 0 && !stdout) {
        reject(new Error('git ls-files failed'));
        return;
      }

      const lines = stdout.split(/\r?\n/).filter((line) => line.trim());

      const results = (await Promise.all(
        lines.slice(0, maxResults + 10).map(async (line): Promise<GlobMatch | null> => {
          const filePath = line.trim();
          if (!filePath) return null;

          const fullPath = path.join(projectRoot, filePath);
          try {
            const stats = await stat(fullPath);
            return {
              filePath: filePath.replace(/\\/g, '/'),
              mtime: stats.mtimeMs,
            };
          } catch {
            return {
              filePath: filePath.replace(/\\/g, '/'),
              mtime: 0,
            };
          }
        })
      )).filter((r): r is GlobMatch => r !== null);
      results.sort((a, b) => b.mtime - a.mtime);

      resolve(results.slice(0, maxResults));
    });

    proc.on('error', (err) => reject(err));
  });
}
