// write_file tool implementation
// Permission checks are handled by ToolExecutorAdapter.
import fs from 'fs/promises';
import path from 'path';
import { Tool, ToolResult } from './registry.js';
import { getLogger } from '../lib/logger.js';
import { computeLineDiff } from './diff.js';

export class WriteFileTool implements Tool {
  readonly name = 'write_file';
  private projectRoot: string;

  constructor(projectRoot: string) {
    this.projectRoot = projectRoot;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const filePath = args.file_path;
    const content = args.content;

    if (!filePath) {
      return {
        result: '[ERROR] file_path argument is required',
        error: new Error('file_path argument is required'),
      };
    }

    if (content === undefined || content === null) {
      return {
        result: '[ERROR] content argument is required',
        error: new Error('content argument is required'),
      };
    }

    const resolvedPath = path.isAbsolute(filePath)
      ? path.normalize(filePath)
      : path.resolve(this.projectRoot, filePath);

    try {
      // Ensure parent directory exists
      const dir = path.dirname(resolvedPath);
      await fs.mkdir(dir, { recursive: true });

      // Before write - try to read existing content for diff
      let oldContent: string | null = null;
      try {
        oldContent = await fs.readFile(resolvedPath, 'utf-8');
      } catch {
        // File doesn't exist (new file) - no diff
      }

      // Write file
      await fs.writeFile(resolvedPath, content, 'utf-8');

      const lineCount = content.split('\n').length;
      const relativePath = path.relative(this.projectRoot, resolvedPath);

      logger.info('File written', { path: relativePath, lines: lineCount });

      const fileName = path.basename(filePath);
      const diffLines = oldContent !== null ? computeLineDiff(oldContent, content) : undefined;
      return {
        result: `File written: ${relativePath} (${lineCount} lines)`,
        summary: `${lineCount} lines (${fileName})`,
        diffLines,
      };
    } catch (error: any) {
      logger.error('WriteFileTool error', { path: filePath, error: error.message });
      return {
        result: `[ERROR] Failed to write file: ${error.message}`,
        error,
      };
    }
  }
}
