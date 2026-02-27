// File scanner with FileIgnore support
import fs from 'fs/promises';
import path from 'path';
import { TreeSitterParser } from './parser.js';
import { FileIgnore } from '../domain/file-ignore/FileIgnore.js';

export interface ScanResult {
  filePath: string;
  language: string;
  relativePath: string;
}

export interface ScanOptions {
  extensions?: string[];
  maxFileSize?: number;
}

const DEFAULT_EXTENSIONS = [
  '.go', '.ts', '.tsx', '.js', '.jsx', '.mjs',
  '.py', '.java', '.c', '.cpp', '.h', '.hpp', '.cc', '.cxx', '.hxx',
  '.rs', '.rb', '.php', '.swift', '.kt', '.kts', '.cs',
  '.sh', '.bash', '.sql',
  // Additional tree-sitter-supported languages
  '.dart', '.lua', '.ex', '.exs', '.ml', '.mli', '.zig', '.scala', '.sc',
];

const MAX_FILE_SIZE = 1024 * 1024; // 1MB

export class FileScanner {
  private rootPath: string;
  private fileIgnore: FileIgnore;
  private options: Required<ScanOptions>;

  constructor(rootPath: string, fileIgnore: FileIgnore, options: ScanOptions = {}) {
    this.rootPath = path.resolve(rootPath);
    this.fileIgnore = fileIgnore;
    this.options = {
      extensions: options.extensions || DEFAULT_EXTENSIONS,
      maxFileSize: options.maxFileSize || MAX_FILE_SIZE,
    };
  }

  async scan(): Promise<ScanResult[]> {
    const results: ScanResult[] = [];
    await this.scanDirectory(this.rootPath, results);
    return results;
  }

  private async scanDirectory(dirPath: string, results: ScanResult[]): Promise<void> {
    let entries;
    try {
      entries = await fs.readdir(dirPath, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      const fullPath = path.join(dirPath, entry.name);
      const relativePath = path.relative(this.rootPath, fullPath);

      // Check FileIgnore (handles gitignore + built-in patterns)
      if (this.fileIgnore.shouldIgnore(relativePath, entry.isDirectory())) {
        continue;
      }

      if (entry.isDirectory()) {
        await this.scanDirectory(fullPath, results);
      } else if (entry.isFile()) {
        // Check extension
        const ext = path.extname(entry.name).toLowerCase();
        if (!this.options.extensions.includes(ext)) {
          continue;
        }

        // Check file size
        try {
          const stats = await fs.stat(fullPath);
          if (stats.size > this.options.maxFileSize) {
            continue;
          }
        } catch {
          continue;
        }

        const language = TreeSitterParser.detectLanguage(fullPath);
        results.push({
          filePath: fullPath,
          language,
          relativePath,
        });
      }
    }
  }

  async readFile(filePath: string): Promise<string> {
    return fs.readFile(filePath, 'utf-8');
  }

  async getFileStats(filePath: string): Promise<{ size: number; mtime: Date } | null> {
    try {
      const stats = await fs.stat(filePath);
      return {
        size: stats.size,
        mtime: stats.mtime,
      };
    } catch {
      return null;
    }
  }

  getRootPath(): string {
    return this.rootPath;
  }
}
