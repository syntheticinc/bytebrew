// Grep search using ripgrep (rg) for fast pattern matching
import { spawn, exec } from 'child_process';
import { stat } from 'fs/promises';
import { getLogger } from '../../lib/logger.js';
import { resolveRgBinary } from './ripgrepResolver.js';

export interface GrepMatch {
  filePath: string;
  line: number;
  content: string;
  context?: string;
  mtime?: number; // ms since epoch, for sorting
}

export interface GrepSearchOptions {
  maxResults?: number;
  fileTypes?: string[];
  contextLines?: number;
  ignoreCase?: boolean;
}

const DEFAULT_MAX_RESULTS = 50;
const DEFAULT_CONTEXT_LINES = 2;

/**
 * Search for pattern in project files using ripgrep
 * Falls back to grep if ripgrep not available
 */
export async function grepSearch(
  projectRoot: string,
  pattern: string,
  options: GrepSearchOptions = {}
): Promise<GrepMatch[]> {
  const logger = getLogger();

  if (!pattern) {
    return [];
  }

  const maxResults = options.maxResults ?? DEFAULT_MAX_RESULTS;
  const contextLines = options.contextLines ?? DEFAULT_CONTEXT_LINES;

  const rgPath = await resolveRgBinary();

  if (!rgPath) {
    logger.debug('ripgrep not available, using grep fallback', { pattern });
    return executeGrepFallback(projectRoot, pattern, options);
  }

  try {
    const matches = await executeRipgrep(rgPath, projectRoot, pattern, {
      ...options,
      maxResults,
      contextLines,
    });
    logger.debug('Grep search completed', { pattern, resultsCount: matches.length });
    return matches;
  } catch (error) {
    logger.warn('Ripgrep failed, trying grep fallback', { error });
    return executeGrepFallback(projectRoot, pattern, options);
  }
}

async function executeRipgrep(
  rgPath: string,
  projectRoot: string,
  pattern: string,
  options: Required<Pick<GrepSearchOptions, 'maxResults' | 'contextLines'>> & GrepSearchOptions
): Promise<GrepMatch[]> {
  const args = buildRipgrepArgs(pattern, options);

  return new Promise((resolve, reject) => {
    const rg = spawn(rgPath, args, {
      cwd: projectRoot,
    });

    let stdout = '';
    let stderr = '';

    rg.stdout.on('data', (data) => {
      stdout += data.toString();
    });

    rg.stderr.on('data', (data) => {
      stderr += data.toString();
    });

    rg.on('close', async (code) => {
      // If code is 1 with stderr and no stdout, ripgrep is likely not installed
      // (code 1 with no stderr means "no matches found" which is valid)
      if (code === 1 && stderr.length > 0 && stdout.length === 0) {
        reject(new Error(`ripgrep not available: ${stderr}`));
        return;
      }

      // Exit code > 1 means actual error
      if (code !== null && code > 1) {
        reject(new Error(`ripgrep failed with code ${code}: ${stderr}`));
        return;
      }

      const matches = await parseRipgrepOutput(stdout, options.maxResults);
      resolve(matches);
    });

    rg.on('error', (error) => {
      reject(error);
    });
  });
}

function buildRipgrepArgs(
  pattern: string,
  options: Required<Pick<GrepSearchOptions, 'maxResults' | 'contextLines'>> & GrepSearchOptions
): string[] {
  const args: string[] = [
    '--json',
    '--line-number',
    `--max-count=${options.maxResults}`,
  ];

  if (options.contextLines > 0) {
    args.push(`-C${options.contextLines}`);
  }

  if (options.ignoreCase) {
    args.push('-i');
  }

  if (options.fileTypes && options.fileTypes.length > 0) {
    for (const ft of options.fileTypes) {
      // fileTypes may be glob patterns ("*.go") or rg type names ("go").
      // -t expects a type name; glob patterns require --glob instead.
      if (ft.startsWith('*') || ft.includes('.')) {
        args.push('--glob', ft);
      } else {
        args.push('-t', ft);
      }
    }
  }

  // Common ignore patterns
  args.push(
    '--glob=!node_modules',
    '--glob=!.git',
    '--glob=!.bytebrew',
    '--glob=!dist',
    '--glob=!build',
    '--glob=!*.lock',
    '--glob=!*.min.js',
    '--glob=!*.min.css'
  );

  // Explicit '.' path is required: without it, rg reads from stdin
  // when spawned via Node/Bun (stdin is a pipe, not a terminal)
  args.push('--', pattern, '.');
  return args;
}

interface RipgrepMatch {
  type: string;
  data?: {
    path?: { text: string };
    line_number?: number;
    lines?: { text: string };
    submatches?: Array<{ match: { text: string }; start: number; end: number }>;
  };
}

async function parseRipgrepOutput(output: string, maxResults: number): Promise<GrepMatch[]> {
  const lines = output.trim().split('\n').filter(Boolean);
  const matches: GrepMatch[] = [];
  const contextMap = new Map<string, string[]>();

  for (const line of lines) {
    try {
      const json = JSON.parse(line) as RipgrepMatch;

      if (json.type === 'match' && json.data) {
        const filePath = json.data.path?.text ?? '';
        const lineNumber = json.data.line_number ?? 0;
        const content = json.data.lines?.text?.trimEnd() ?? '';

        if (filePath && lineNumber > 0) {
          matches.push({
            filePath,
            line: lineNumber,
            content,
          });
        }
      } else if (json.type === 'context' && json.data) {
        // Collect context lines
        const filePath = json.data.path?.text ?? '';
        const content = json.data.lines?.text?.trimEnd() ?? '';
        if (filePath && content) {
          const existing = contextMap.get(filePath) ?? [];
          existing.push(content);
          contextMap.set(filePath, existing);
        }
      }
    } catch {
      // Skip invalid JSON lines
    }
  }

  // Attach context to matches
  for (const match of matches) {
    const ctx = contextMap.get(match.filePath);
    if (ctx && ctx.length > 0) {
      match.context = ctx.join('\n');
    }
  }

  // Sort by mtime (newest first), then by line number
  await enrichMatchesWithMtime(matches);
  matches.sort((a, b) => {
    // Sort by mtime DESC (newest first)
    if (a.mtime !== b.mtime) {
      return (b.mtime ?? 0) - (a.mtime ?? 0);
    }
    // Then by line ASC
    return a.line - b.line;
  });

  return matches.slice(0, maxResults);
}

async function enrichMatchesWithMtime(matches: GrepMatch[]): Promise<void> {
  // Collect unique file paths
  const uniqueFiles = new Set<string>();
  for (const match of matches) {
    uniqueFiles.add(match.filePath);
  }

  // Stat all files in parallel
  const mtimeMap = new Map<string, number>();
  await Promise.all(
    Array.from(uniqueFiles).map(async (filePath) => {
      try {
        const stats = await stat(filePath);
        mtimeMap.set(filePath, stats.mtimeMs);
      } catch {
        // If stat fails (file inaccessible), use mtime=0
        mtimeMap.set(filePath, 0);
      }
    })
  );

  // Assign mtime to each match
  for (const match of matches) {
    match.mtime = mtimeMap.get(match.filePath) ?? 0;
  }
}

async function executeGrepFallback(
  projectRoot: string,
  pattern: string,
  options: GrepSearchOptions
): Promise<GrepMatch[]> {
  const logger = getLogger();
  const maxResults = options.maxResults ?? DEFAULT_MAX_RESULTS;
  const isWindows = process.platform === 'win32';

  return new Promise((resolve) => {
    let command: string;

    // Escape pattern for shell
    const escapedPattern = pattern.replace(/"/g, '\\"');

    if (isWindows) {
      // Use Git grep on Windows with extended regex (-E) for better pattern support
      const ignoreFlag = options.ignoreCase ? '-i ' : '';
      if (options.fileTypes && options.fileTypes.length > 0) {
        const fileTypeArgs = options.fileTypes.map(ft => `"${ft}"`).join(' ');
        command = `git grep -E -n --no-color ${ignoreFlag}-e "${escapedPattern}" -- ${fileTypeArgs}`;
      } else {
        command = `git grep -E -n --no-color ${ignoreFlag}-e "${escapedPattern}"`;
      }
    } else {
      // Use grep on Unix with extended regex (-E) for better pattern support
      const ignoreFlag = options.ignoreCase ? '-i ' : '';
      if (options.fileTypes && options.fileTypes.length > 0) {
        const includeArgs = options.fileTypes.map(ft => `--include="${ft}"`).join(' ');
        command = `grep -rn -E ${ignoreFlag}${includeArgs} "${escapedPattern}" .`;
      } else {
        command = `grep -rn -E ${ignoreFlag}"${escapedPattern}" .`;
      }
    }

    exec(command, { cwd: projectRoot, maxBuffer: 10 * 1024 * 1024 }, (error, stdout, stderr) => {
      // git grep/grep return exit code 1 when no matches - not an error
      if (error && error.code !== 1) {
        logger.warn('Grep fallback failed', { error: error.message, stderr });
        resolve([]);
        return;
      }

      const matches = parseGrepOutput(stdout, maxResults);
      logger.debug('Grep fallback completed', { resultsCount: matches.length });
      resolve(matches);
    });
  });
}

function parseGrepOutput(output: string, maxResults: number): GrepMatch[] {
  // Normalize line endings (Windows uses CRLF)
  const normalizedOutput = output.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = normalizedOutput.trim().split('\n').filter(Boolean);
  const matches: GrepMatch[] = [];

  for (const line of lines) {
    // Format: ./path/to/file.ts:123:content
    const match = line.match(/^\.?\/?(.*?):(\d+):(.*)$/);
    if (match) {
      matches.push({
        filePath: match[1],
        line: parseInt(match[2], 10),
        content: match[3].trim(),
      });
    }

    if (matches.length >= maxResults) {
      break;
    }
  }

  return matches;
}
