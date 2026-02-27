// FileIgnoreFactory - reads .gitignore from disk and creates FileIgnore instance
import fs from 'fs/promises';
import path from 'path';
import { FileIgnore } from '../../domain/file-ignore/FileIgnore.js';

export class FileIgnoreFactory {
  /**
   * Create a FileIgnore instance for the given project root.
   * Loads .gitignore if present.
   */
  static async create(projectRoot: string): Promise<FileIgnore> {
    const gitignorePatterns = await FileIgnoreFactory.loadGitignore(projectRoot);
    return new FileIgnore(gitignorePatterns);
  }

  private static async loadGitignore(projectRoot: string): Promise<string[] | undefined> {
    const gitignorePath = path.join(projectRoot, '.gitignore');
    try {
      const content = await fs.readFile(gitignorePath, 'utf-8');
      // Split into lines, filter empty/comments
      return content
        .split('\n')
        .map(line => line.trim())
        .filter(line => line && !line.startsWith('#'));
    } catch {
      return undefined;
    }
  }
}
