import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from '../config/ByteBrewHome';

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  email: string;
  userId: string;
}

interface AuthJson {
  access_token: string;
  refresh_token: string;
  email: string;
  user_id: string;
}

function isAuthJson(value: unknown): value is AuthJson {
  if (!value || typeof value !== 'object') return false;
  const obj = value as Record<string, unknown>;
  return (
    typeof obj['access_token'] === 'string' &&
    typeof obj['refresh_token'] === 'string' &&
    typeof obj['email'] === 'string' &&
    typeof obj['user_id'] === 'string'
  );
}

export class AuthStorage {
  private filePath: string;

  constructor(filePath?: string) {
    this.filePath = filePath ?? ByteBrewHome.authFile();
  }

  save(tokens: AuthTokens): void {
    const dir = path.dirname(this.filePath);
    fs.mkdirSync(dir, { recursive: true });

    const data: AuthJson = {
      access_token: tokens.accessToken,
      refresh_token: tokens.refreshToken,
      email: tokens.email,
      user_id: tokens.userId,
    };
    fs.writeFileSync(this.filePath, JSON.stringify(data, null, 2), 'utf-8');

    try {
      fs.chmodSync(this.filePath, 0o600);
    } catch {
      // chmod not supported on Windows — skip
    }
  }

  load(): AuthTokens | null {
    try {
      const content = fs.readFileSync(this.filePath, 'utf-8');
      const parsed: unknown = JSON.parse(content);
      if (!isAuthJson(parsed)) return null;
      return {
        accessToken: parsed.access_token,
        refreshToken: parsed.refresh_token,
        email: parsed.email,
        userId: parsed.user_id,
      };
    } catch {
      return null;
    }
  }

  clear(): void {
    try {
      fs.unlinkSync(this.filePath);
    } catch {
      // File does not exist or already removed — ignore
    }
  }
}
