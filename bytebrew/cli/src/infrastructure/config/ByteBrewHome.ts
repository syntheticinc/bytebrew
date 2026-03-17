// Cross-platform path to ~/.bytebrew/ directory
// Linux/macOS: ~/.bytebrew/
// Windows: %USERPROFILE%\.bytebrew\
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

export class ByteBrewHome {
  /** ~/.bytebrew/ directory path */
  static dir(): string {
    const home = process.env.HOME || process.env.USERPROFILE;
    if (!home) throw new Error('Cannot determine home directory');
    return path.join(home, '.bytebrew');
  }

  /** Ensure ~/.bytebrew/ exists */
  static ensureDir(): void {
    fs.mkdirSync(ByteBrewHome.dir(), { recursive: true });
  }

  /** Path to auth.json */
  static authFile(): string {
    return path.join(ByteBrewHome.dir(), 'auth.json');
  }

  /** Path to license.jwt */
  static licenseFile(): string {
    return path.join(ByteBrewHome.dir(), 'license.jwt');
  }

  /** Cross-platform user data directory (for managed installs) */
  static dataDir(): string {
    switch (process.platform) {
      case 'darwin':
        return path.join(os.homedir(), 'Library', 'Application Support');
      case 'win32':
        return process.env.APPDATA || path.join(os.homedir(), 'AppData', 'Roaming');
      default:
        return process.env.XDG_DATA_HOME || path.join(os.homedir(), '.local', 'share');
    }
  }
}
