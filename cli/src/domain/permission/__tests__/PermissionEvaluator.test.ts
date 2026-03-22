import { describe, it, expect } from 'bun:test';
import { evaluatePermission } from '../PermissionEvaluator.js';
import { PermissionConfig } from '../Permission.js';

function makeConfig(overrides?: Partial<PermissionConfig>): PermissionConfig {
  return {
    permissions: {
      allow: [
        'Read',
        'Edit',
        'Bash(npm *)',
        'Bash(git status)',
        'Bash(cd *)',
      ],
      deny: [
        'Bash(sudo *)',
        'Bash(git push *)',
        'Bash(rm -rf /)',
      ],
    },
    ...overrides,
  };
}

describe('PermissionEvaluator', () => {
  describe('simple permission types (read, edit, list)', () => {
    const config = makeConfig();

    it('should allow read by default', () => {
      const result = evaluatePermission({ type: 'read', value: 'any/file.ts' }, config);
      expect(result.action).toBe('allow');
    });

    it('should allow edit by default', () => {
      const result = evaluatePermission({ type: 'edit', value: 'any/file.ts' }, config);
      expect(result.action).toBe('allow');
    });

    it('should allow list by default (not in deny list)', () => {
      const result = evaluatePermission({ type: 'list', value: '' }, config);
      expect(result.action).toBe('allow');
    });

    it('should deny read when "Read" in deny list', () => {
      const denyConfig = makeConfig({
        permissions: {
          allow: [],
          deny: ['Read'],
        },
      });

      const result = evaluatePermission({ type: 'read', value: 'file.ts' }, denyConfig);
      expect(result.action).toBe('deny');
      expect(result.matchedPattern).toBe('Read');
    });

    it('should deny edit when "Write" in deny list', () => {
      const denyConfig = makeConfig({
        permissions: {
          allow: [],
          deny: ['Write'],
        },
      });

      const result = evaluatePermission({ type: 'edit', value: 'file.ts' }, denyConfig);
      expect(result.action).toBe('deny');
      expect(result.matchedPattern).toBe('Write');
    });
  });

  describe('bash permission type with Bash(pattern) format', () => {
    const config = makeConfig();

    it('should allow commands not in deny list', () => {
      const result = evaluatePermission({ type: 'bash', value: 'npm install' }, config);
      expect(result.action).toBe('allow');
    });

    it('should allow safe git commands', () => {
      const result = evaluatePermission({ type: 'bash', value: 'git status' }, config);
      expect(result.action).toBe('allow');
    });

    it('should deny matched deny rules', () => {
      const result = evaluatePermission({ type: 'bash', value: 'git push origin main' }, config);
      expect(result.action).toBe('deny');
      expect(result.matchedPattern).toBe('Bash(git push *)');
    });

    it('should deny sudo', () => {
      const result = evaluatePermission({ type: 'bash', value: 'sudo apt install vim' }, config);
      expect(result.action).toBe('deny');
      expect(result.matchedPattern).toBe('Bash(sudo *)');
    });

    it('should allow unmatched commands by default', () => {
      const result = evaluatePermission({ type: 'bash', value: 'unknown-tool --flag' }, config);
      expect(result.action).toBe('allow');
    });
  });

  describe('first-match-wins semantics', () => {
    it('deny first-match should win over allow', () => {
      const config = makeConfig({
        permissions: {
          allow: ['Bash(rm *)'],
          deny: ['Bash(rm -rf /)'],
        },
      });

      // 'rm file.txt' matches only allow rule → allow
      const result1 = evaluatePermission({ type: 'bash', value: 'rm file.txt' }, config);
      expect(result1.action).toBe('allow');

      // 'rm -rf /' matches deny FIRST → deny (first-match-wins)
      const result2 = evaluatePermission({ type: 'bash', value: 'rm -rf /' }, config);
      expect(result2.action).toBe('deny');
      expect(result2.matchedPattern).toBe('Bash(rm -rf /)');
    });

    it('should allow when not in deny list regardless of allow list', () => {
      const config = makeConfig({
        permissions: {
          allow: [],
          deny: [],
        },
      });

      const result = evaluatePermission({ type: 'bash', value: 'git status' }, config);
      expect(result.action).toBe('allow');
    });

    it('deny order matters (first deny wins)', () => {
      const config = makeConfig({
        permissions: {
          allow: [],
          deny: [
            'Bash(git push *)',   // More specific
            'Bash(git *)',        // Less specific
          ],
        },
      });

      const result = evaluatePermission({ type: 'bash', value: 'git push origin' }, config);
      expect(result.action).toBe('deny');
      expect(result.matchedPattern).toBe('Bash(git push *)'); // First match
    });

    it('should allow when no rules match (allow by default)', () => {
      const config = makeConfig({
        permissions: {
          allow: [],
          deny: [],
        },
      });

      const result = evaluatePermission({ type: 'bash', value: 'anything' }, config);
      expect(result.action).toBe('allow');
    });
  });

  describe('compound commands', () => {
    it('should evaluate each subcommand and return most restrictive (allow + allow = allow)', () => {
      const config = makeConfig({
        permissions: {
          allow: [
            'Bash(cd *)',
            'Bash(go *)',
          ],
          deny: [],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'cd /path && go build' },
        config
      );
      expect(result.action).toBe('allow');
    });

    it('should return deny if any subcommand is denied (allow + deny = deny)', () => {
      const config = makeConfig({
        permissions: {
          allow: ['Bash(cd *)'],
          deny: ['Bash(rm *)'],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'cd /path && rm -rf /' },
        config
      );
      expect(result.action).toBe('deny');
      expect(result.matchedPattern).toBe('Bash(rm *)');
    });

    it('should allow when all subcommands are allowed (allow + allow = allow)', () => {
      const config = makeConfig({
        permissions: {
          allow: ['Bash(cd *)'],
          deny: [],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'cd /path && unknown_cmd' },
        config
      );
      expect(result.action).toBe('allow'); // unknown_cmd allowed by default
    });

    it('should handle three subcommands with mixed actions (deny > ask > allow)', () => {
      const config = makeConfig({
        permissions: {
          allow: [
            'Bash(npm *)',
            'Bash(git status)',
          ],
          deny: ['Bash(sudo *)'],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'npm install && git status && sudo apt update' },
        config
      );
      expect(result.action).toBe('deny');
    });

    it('should handle || operator', () => {
      const config = makeConfig({
        permissions: {
          allow: [
            'Bash(npm *)',
            'Bash(echo *)',
          ],
          deny: [],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'npm test || echo "failed"' },
        config
      );
      expect(result.action).toBe('allow');
    });

    it('should handle semicolon separator', () => {
      const config = makeConfig({
        permissions: {
          allow: ['Bash(cd *)'],
          deny: [],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'cd src ; ls' },
        config
      );
      expect(result.action).toBe('allow'); // ls allowed by default
    });

    it('should NOT split operators inside quotes', () => {
      const config = makeConfig({
        permissions: {
          allow: ['Bash(echo *)'],
          deny: [],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'echo "hello && world"' },
        config
      );
      expect(result.action).toBe('allow');
    });

    it('should handle real-world build command', () => {
      const config = makeConfig({
        permissions: {
          allow: [
            'Bash(cd *)',
            'Bash(go *)',
          ],
          deny: [],
        },
      });

      const result = evaluatePermission(
        { type: 'bash', value: 'cd "/path/to/project" && go build ./...' },
        config
      );
      expect(result.action).toBe('allow');
    });
  });

  describe('case-insensitive matching', () => {
    it('should match Bash() case-insensitively', () => {
      const config = makeConfig({
        permissions: {
          allow: ['bash(npm *)'],  // lowercase "bash"
          deny: [],
        },
      });

      const result = evaluatePermission({ type: 'bash', value: 'npm install' }, config);
      expect(result.action).toBe('allow');
    });

    it('should match type names case-insensitively', () => {
      const config = makeConfig({
        permissions: {
          allow: ['read'],  // lowercase
          deny: [],
        },
      });

      const result = evaluatePermission({ type: 'read', value: 'file.ts' }, config);
      expect(result.action).toBe('allow');
    });
  });
});
