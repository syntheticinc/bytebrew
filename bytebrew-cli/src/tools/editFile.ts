// edit_file tool implementation — str_replace approach
// Replace logic adapted from OpenCode (MIT License)
// Permission checks are handled by ToolExecutorAdapter.
import fs from 'fs/promises';
import path from 'path';
import { Tool, ToolResult } from './registry.js';
import { replace } from './replace.js';
import { getLogger } from '../lib/logger.js';
import { computeLineDiff } from './diff.js';

export class EditFileTool implements Tool {
  readonly name = 'edit_file';
  private projectRoot: string;

  constructor(projectRoot: string) {
    this.projectRoot = projectRoot;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const filePath = args.file_path;
    const oldString = args.old_string;
    const newString = args.new_string;
    const replaceAll = args.replace_all === 'true';

    if (!filePath) {
      return {
        result: '[ERROR] file_path argument is required',
        error: new Error('file_path argument is required'),
      };
    }

    if (!oldString) {
      return {
        result: '[ERROR] old_string argument is required',
        error: new Error('old_string argument is required'),
      };
    }

    if (newString === undefined || newString === null) {
      return {
        result: '[ERROR] new_string argument is required',
        error: new Error('new_string argument is required'),
      };
    }

    if (oldString === newString) {
      return {
        result: '[ERROR] old_string and new_string must be different',
        error: new Error('old_string and new_string must be different'),
      };
    }

    const resolvedPath = path.isAbsolute(filePath)
      ? path.normalize(filePath)
      : path.resolve(this.projectRoot, filePath);

    try {
      // Read current content
      const content = await fs.readFile(resolvedPath, 'utf-8');

      // Apply replacement using fuzzy matching replacers
      const newContent = replace(content, oldString, newString, replaceAll);

      // Write updated content
      await fs.writeFile(resolvedPath, newContent, 'utf-8');

      // Calculate diff stats
      const oldLines = content.split('\n').length;
      const newLines = newContent.split('\n').length;
      const diff = newLines - oldLines;
      const relativePath = path.relative(this.projectRoot, resolvedPath);
      const diffStr = diff > 0 ? `+${diff}` : diff < 0 ? `${diff}` : '±0';

      logger.info('File edited', { path: relativePath, linesDiff: diffStr });

      const fileName = path.basename(filePath);
      const diffLines = computeLineDiff(oldString, newString);
      return {
        result: `Edit applied: ${relativePath} (${diffStr} lines)`,
        summary: `${diffStr} lines (${fileName})`,
        diffLines,
      };
    } catch (error: any) {
      if (error.code === 'ENOENT') {
        return {
          result: `[ERROR] File not found: ${filePath}. Use write_file to create new files.`,
        };
      }

      // Replace errors (not found, multiple matches) — return as result for LLM to see
      if (error.message.includes('oldString') || error.message.includes('multiple matches')) {
        return {
          result: `[ERROR] ${error.message}`,
        };
      }

      logger.error('EditFileTool error', { path: filePath, error: error.message });
      return {
        result: `[ERROR] ${error.message}`,
        error,
      };
    }
  }
}
