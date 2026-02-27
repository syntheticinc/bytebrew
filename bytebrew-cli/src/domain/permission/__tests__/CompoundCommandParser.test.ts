import { describe, it, expect } from 'bun:test';
import { splitCompoundCommand } from '../CompoundCommandParser.js';

describe('CompoundCommandParser', () => {
  describe('single commands', () => {
    it('should return single command unchanged', () => {
      expect(splitCompoundCommand('go test ./...')).toEqual(['go test ./...']);
    });

    it('should trim single command', () => {
      expect(splitCompoundCommand('  npm install  ')).toEqual(['npm install']);
    });

    it('should return empty array for empty string', () => {
      expect(splitCompoundCommand('')).toEqual([]);
    });

    it('should return empty array for whitespace only', () => {
      expect(splitCompoundCommand('   ')).toEqual([]);
    });
  });

  describe('compound commands with &&', () => {
    it('should split by &&', () => {
      expect(splitCompoundCommand('cd foo && go build')).toEqual([
        'cd foo',
        'go build',
      ]);
    });

    it('should handle multiple &&', () => {
      expect(splitCompoundCommand('a && b && c')).toEqual(['a', 'b', 'c']);
    });

    it('should trim each subcommand', () => {
      expect(splitCompoundCommand('  cd foo  &&  go build  ')).toEqual([
        'cd foo',
        'go build',
      ]);
    });
  });

  describe('compound commands with ||', () => {
    it('should split by ||', () => {
      expect(splitCompoundCommand('npm test || echo failed')).toEqual([
        'npm test',
        'echo failed',
      ]);
    });

    it('should handle multiple ||', () => {
      expect(splitCompoundCommand('a || b || c')).toEqual(['a', 'b', 'c']);
    });
  });

  describe('compound commands with ;', () => {
    it('should split by semicolon', () => {
      expect(splitCompoundCommand('cd src ; ls')).toEqual(['cd src', 'ls']);
    });

    it('should handle multiple semicolons', () => {
      expect(splitCompoundCommand('a ; b ; c')).toEqual(['a', 'b', 'c']);
    });

    it('should filter empty subcommands from trailing semicolons', () => {
      expect(splitCompoundCommand('echo hello ;')).toEqual(['echo hello']);
    });

    it('should filter empty subcommands from consecutive semicolons', () => {
      expect(splitCompoundCommand('a ;; b')).toEqual(['a', 'b']);
    });
  });

  describe('mixed operators', () => {
    it('should handle && and || together', () => {
      expect(splitCompoundCommand('a && b || c')).toEqual(['a', 'b', 'c']);
    });

    it('should handle all three operators', () => {
      expect(splitCompoundCommand('a && b || c ; d')).toEqual([
        'a',
        'b',
        'c',
        'd',
      ]);
    });
  });

  describe('quoted strings', () => {
    it('should NOT split inside double quotes', () => {
      expect(splitCompoundCommand('echo "hello && world"')).toEqual([
        'echo "hello && world"',
      ]);
    });

    it('should NOT split inside single quotes', () => {
      expect(splitCompoundCommand("echo 'a && b'")).toEqual(["echo 'a && b'"]);
    });

    it('should handle || inside quotes', () => {
      expect(splitCompoundCommand('echo "a || b"')).toEqual(['echo "a || b"']);
    });

    it('should handle semicolon inside quotes', () => {
      expect(splitCompoundCommand('echo "a ; b"')).toEqual(['echo "a ; b"']);
    });

    it('should split outside quotes but preserve inside', () => {
      expect(
        splitCompoundCommand('echo "hello && world" && echo "foo || bar"')
      ).toEqual(['echo "hello && world"', 'echo "foo || bar"']);
    });

    it('should handle single quotes protecting double quotes', () => {
      expect(splitCompoundCommand('echo \'say "hi && bye"\'')).toEqual([
        'echo \'say "hi && bye"\'',
      ]);
    });

    it('should handle double quotes protecting single quotes', () => {
      expect(splitCompoundCommand('echo "don\'t split"')).toEqual([
        'echo "don\'t split"',
      ]);
    });
  });

  describe('edge cases', () => {
    it('should handle only operators', () => {
      expect(splitCompoundCommand('&&')).toEqual([]);
      expect(splitCompoundCommand('  &&  ')).toEqual([]);
      expect(splitCompoundCommand('||')).toEqual([]);
      expect(splitCompoundCommand(';')).toEqual([]);
    });

    it('should handle mixed operators without commands', () => {
      expect(splitCompoundCommand('&& || ;')).toEqual([]);
    });

    it('should handle empty subcommands between operators', () => {
      expect(splitCompoundCommand('a &&  && b')).toEqual(['a', 'b']);
    });

    it('should split on single & (cmd.exe separator)', () => {
      expect(splitCompoundCommand('echo a & b')).toEqual(['echo a', 'b']);
      expect(splitCompoundCommand('cd /d C:\\path & go build')).toEqual(['cd /d C:\\path', 'go build']);
    });

    it('should preserve single | (pipe, not operator)', () => {
      // Single | is a pipe — not a command separator
      expect(splitCompoundCommand('echo a | b')).toEqual(['echo a | b']);
    });
  });

  describe('real-world examples', () => {
    it('should handle typical build command', () => {
      expect(splitCompoundCommand('cd /path/to/project && go build ./...')).toEqual([
        'cd /path/to/project',
        'go build ./...',
      ]);
    });

    it('should handle test with fallback', () => {
      expect(
        splitCompoundCommand('npm test || echo "Tests failed but continuing"')
      ).toEqual(['npm test', 'echo "Tests failed but continuing"']);
    });

    it('should handle multi-step deployment', () => {
      expect(
        splitCompoundCommand('npm run build && npm run test && npm run deploy')
      ).toEqual(['npm run build', 'npm run test', 'npm run deploy']);
    });
  });
});
