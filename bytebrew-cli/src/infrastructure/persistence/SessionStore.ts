// Session persistence - stores last_session_id in project directory
import * as fs from 'fs';
import * as path from 'path';

/**
 * Stores last session ID in project's .bytebrew directory.
 * Each project has its own last_session, allowing multiple projects
 * to maintain separate session history.
 */
export class SessionStore {
  private filePath: string;

  constructor(projectRoot: string) {
    this.filePath = path.join(projectRoot, '.bytebrew', 'last_session');
  }

  /**
   * Get last session ID from file, or null if not found
   */
  getLastSessionId(): string | null {
    try {
      const content = fs.readFileSync(this.filePath, 'utf-8');
      return content.trim() || null;
    } catch {
      // File doesn't exist or read error
      return null;
    }
  }

  /**
   * Save session ID to file, creating directory if needed
   */
  saveSessionId(id: string): void {
    const dir = path.dirname(this.filePath);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(this.filePath, id, 'utf-8');
  }
}
