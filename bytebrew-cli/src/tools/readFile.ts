// read_file tool implementation
import fs from 'fs/promises';
import path from 'path';
import { Tool, ToolResult } from './registry.js';

const MAX_FILE_SIZE = 1024 * 1024; // 1MB

export class ReadFileTool implements Tool {
  name = 'read_file';
  private projectRoot: string;

  constructor(projectRoot: string) {
    this.projectRoot = projectRoot;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const filePath = args.file_path;

    if (!filePath) {
      return { result: '', error: new Error('file_path argument is required') };
    }

    // Resolve path: absolute stays as-is, relative resolves from projectRoot
    const resolvedPath = path.isAbsolute(filePath)
      ? path.normalize(filePath)
      : path.resolve(this.projectRoot, filePath);

    // Security checks (outside project root, etc.) are handled by
    // PermissionService in ToolExecutorAdapter — not duplicated here.

    try {
      // Check if file exists
      const stats = await fs.stat(resolvedPath);

      if (stats.isDirectory()) {
        return {
          result: `[ERROR] Path is a directory, not a file: ${filePath}. This tool only reads files.`,
          error: new Error(`Path is a directory, not a file: ${filePath}`),
        };
      }

      // Check file size
      if (stats.size > MAX_FILE_SIZE) {
        return {
          result: '',
          error: new Error(
            `File too large: ${stats.size} bytes (max ${MAX_FILE_SIZE})`
          ),
        };
      }

      // Read file
      const content = await fs.readFile(resolvedPath, 'utf-8');

      // Handle line range if specified
      const startLine = parseInt(args.start_line || '0', 10);
      const endLine = parseInt(args.end_line || '0', 10);

      const lineCount = content.split('\n').length;
      const fileName = path.basename(filePath);

      if (startLine > 0 || endLine > 0) {
        const lines = content.split('\n');
        const totalLines = lines.length;

        const start = Math.max(1, startLine) - 1;
        const end = endLine > 0 ? Math.min(endLine, totalLines) : totalLines;

        if (start >= end) {
          return {
            result: `[INFO] File: ${filePath}, Total lines: ${totalLines}, Requested range: ${startLine}-${endLine} (empty)`,
            summary: `${fileName} (empty range)`,
          };
        }

        const rangeLines = end - start;
        return {
          result: lines.slice(start, end).join('\n'),
          summary: `${rangeLines} lines (${fileName})`,
        };
      }

      return {
        result: content,
        summary: `${lineCount} lines (${fileName})`,
      };
    } catch (error: any) {
      if (error.code === 'ENOENT') {
        const fileName = path.basename(filePath);
        return {
          result: `[ERROR] File not found: ${filePath}. The file does not exist. Please check the path or use search_code to find relevant files.`,
          summary: `not found: ${fileName}`,
        };
      }
      return { result: '', error };
    }
  }
}
