// get_project_tree tool implementation
import fs from 'fs/promises';
import path from 'path';
import { Tool, ToolResult } from './registry.js';
import { FileIgnore } from '../domain/file-ignore/FileIgnore.js';

export class ProjectTreeTool implements Tool {
  name = 'get_project_tree';
  private projectRoot: string;
  private fileIgnore: FileIgnore;

  constructor(projectRoot: string, fileIgnore: FileIgnore) {
    this.projectRoot = projectRoot;
    this.fileIgnore = fileIgnore;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const root = this.projectRoot || process.cwd();
    const maxDepth = parseInt(args.max_depth || '3', 10);
    const subPath = args.path || '';

    // Navigate to the requested subdirectory
    const scanRoot = subPath ? path.join(root, subPath) : root;

    try {
      const stats = await fs.stat(scanRoot);
      if (!stats.isDirectory()) {
        return { result: `[ERROR] Path is a file, not a directory: ${subPath}` };
      }

      const lines: string[] = [];
      await this.buildTree(root, scanRoot, 0, maxDepth, lines);

      const itemCount = lines.length;
      return {
        result: lines.join('\n'),
        summary: `${itemCount} items`,
      };
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
        return { result: `[ERROR] Path not found: ${subPath}` };
      }
      return { result: '', error: error as Error };
    }
  }

  private async buildTree(
    root: string,
    currentPath: string,
    depth: number,
    maxDepth: number,
    lines: string[],
  ): Promise<void> {
    if (depth > maxDepth) return;

    try {
      const entries = await fs.readdir(currentPath, { withFileTypes: true });

      const filtered = entries
        .filter((entry) => {
          const relativePath = path.relative(root, path.join(currentPath, entry.name));
          return !this.fileIgnore.shouldIgnore(relativePath, entry.isDirectory());
        })
        .sort((a, b) => {
          // Directories first
          if (a.isDirectory() !== b.isDirectory()) {
            return a.isDirectory() ? -1 : 1;
          }
          return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
        });

      const indent = '  '.repeat(depth);
      for (const entry of filtered) {
        const isDir = entry.isDirectory();
        // Directories end with "/" — clear visual distinction
        lines.push(`${indent}${entry.name}${isDir ? '/' : ''}`);

        if (isDir) {
          await this.buildTree(root, path.join(currentPath, entry.name), depth + 1, maxDepth, lines);
        }
      }
    } catch {
      // Skip unreadable directories
    }
  }
}
