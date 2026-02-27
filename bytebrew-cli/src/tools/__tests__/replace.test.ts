import { describe, it, expect } from 'bun:test';
import { replace } from 'C:/Users/busul/GolandProjects/usm-epicsmasher/bytebrew-cli/src/tools/replace.js';

describe('replace()', () => {
  // === SimpleReplacer ===
  describe('exact match (SimpleReplacer)', () => {
    it('replaces exact string', () => {
      const content = 'hello world';
      const result = replace(content, 'world', 'earth');
      expect(result).toBe('hello earth');
    });

    it('replaces multiline exact match', () => {
      const content = 'line1\nline2\nline3';
      const result = replace(content, 'line1\nline2', 'replaced1\nreplaced2');
      expect(result).toBe('replaced1\nreplaced2\nline3');
    });

    it('replaces in the middle of content', () => {
      const content = 'function hello() {\n  return "world";\n}';
      const result = replace(content, 'return "world"', 'return "earth"');
      expect(result).toBe('function hello() {\n  return "earth";\n}');
    });
  });

  // === LineTrimmedReplacer ===
  describe('whitespace-flexible matching (LineTrimmedReplacer)', () => {
    it('matches despite different indentation', () => {
      const content = '    function hello() {\n      return true;\n    }';
      // LLM sends without leading spaces
      const result = replace(content, 'function hello() {\n  return true;\n}', 'function goodbye() {\n  return false;\n}');
      expect(result).toContain('function goodbye()');
      expect(result).toContain('return false');
    });

    it('matches with trailing whitespace difference', () => {
      const content = 'hello   \nworld';
      const result = replace(content, 'hello\nworld', 'hi\nearth');
      expect(result).toBe('hi\nearth');
    });
  });

  // === WhitespaceNormalizedReplacer ===
  describe('whitespace normalized matching', () => {
    it('matches despite multiple spaces', () => {
      const content = 'const  x   =   42;';
      const result = replace(content, 'const x = 42;', 'const x = 100;');
      expect(result).toBe('const x = 100;');
    });

    it('matches tab vs spaces', () => {
      const content = 'const\tx = 42;';
      const result = replace(content, 'const x = 42;', 'const y = 42;');
      expect(result).toBe('const y = 42;');
    });
  });

  // === IndentationFlexibleReplacer ===
  describe('indentation-flexible matching', () => {
    it('matches code at different indent level', () => {
      const content = '        if (true) {\n            doStuff();\n        }';
      const result = replace(
        content,
        '    if (true) {\n        doStuff();\n    }',
        '    if (false) {\n        doNothing();\n    }',
      );
      expect(result).toContain('if (false)');
      expect(result).toContain('doNothing()');
    });
  });

  // === replaceAll ===
  describe('replaceAll', () => {
    it('replaces all occurrences', () => {
      const content = 'foo bar foo baz foo';
      const result = replace(content, 'foo', 'qux', true);
      expect(result).toBe('qux bar qux baz qux');
    });

    it('replaces all multiline occurrences', () => {
      const content = 'let x = 1;\nlet y = 2;\nlet x = 3;';
      const result = replace(content, 'let', 'const', true);
      expect(result).toBe('const x = 1;\nconst y = 2;\nconst x = 3;');
    });
  });

  // === CRLF handling ===
  describe('CRLF normalization', () => {
    it('matches LF old_string against CRLF file content', () => {
      const content = 'line1\r\nline2\r\nline3';
      // LLM sends \n but file has \r\n
      const result = replace(content, 'line1\nline2', 'replaced1\nreplaced2');
      expect(result).toBe('replaced1\r\nreplaced2\r\nline3');
    });

    it('preserves CRLF in untouched parts', () => {
      const content = 'aaa\r\nbbb\r\nccc\r\nddd';
      const result = replace(content, 'bbb', 'xxx');
      expect(result).toBe('aaa\r\nxxx\r\nccc\r\nddd');
    });

    it('handles multiline CRLF replacement', () => {
      const content = 'func main() {\r\n\tfmt.Println("hello")\r\n}';
      const result = replace(
        content,
        'func main() {\n\tfmt.Println("hello")\n}',
        'func main() {\n\tfmt.Println("world")\n}',
      );
      expect(result).toBe('func main() {\r\n\tfmt.Println("world")\r\n}');
    });

    it('handles CRLF with replaceAll', () => {
      const content = 'let x = 1;\r\nlet y = 2;\r\nlet z = 3;';
      const result = replace(content, 'let', 'const', true);
      expect(result).toBe('const x = 1;\r\nconst y = 2;\r\nconst z = 3;');
    });

    it('works when both content and oldString have CRLF', () => {
      const content = 'hello\r\nworld';
      const result = replace(content, 'hello\r\nworld', 'hi\r\nearth');
      expect(result).toBe('hi\r\nearth');
    });
  });

  // === Error cases ===
  describe('errors', () => {
    it('throws when old_string not found', () => {
      expect(() => replace('hello', 'nonexistent', 'replacement')).toThrow('not found');
    });

    it('throws when multiple matches without replaceAll', () => {
      expect(() => replace('foo foo', 'foo', 'bar')).toThrow('multiple matches');
    });

    it('throws when old_string equals new_string', () => {
      expect(() => replace('hello', 'hello', 'hello')).toThrow('must be different');
    });
  });

  // === Error hints (closest match) ===
  describe('error hints', () => {
    it('shows closest match when most lines match', () => {
      const content = 'func main() {\n\tname := os.Args[1]\n\tfmt.Println("Hello", name)\n\tfmt.Println("Done")\n\tos.Exit(0)\n}';
      try {
        // 4/5 lines match, 1 differs (Goodbye vs Done)
        replace(content, 'func main() {\n\tname := os.Args[1]\n\tfmt.Println("Hello", name)\n\tfmt.Println("Goodbye")\n\tos.Exit(0)\n}', 'replaced');
        expect(true).toBe(false); // should have thrown
      } catch (e: any) {
        expect(e.message).toContain('Closest match at line 1');
        expect(e.message).toContain('lines match');
        expect(e.message).toContain('Differing lines');
        expect(e.message).toContain('Done');
        expect(e.message).toContain('Goodbye');
      }
    });

    it('no hint when content is completely different', () => {
      const content = 'aaa\nbbb\nccc';
      try {
        replace(content, 'xxx\nyyy\nzzz', 'replaced');
        expect(true).toBe(false);
      } catch (e: any) {
        expect(e.message).toBe('oldString not found in file content');
      }
    });
  });

  // === Edge cases ===
  describe('edge cases', () => {
    it('handles empty new_string (deletion)', () => {
      const content = 'hello world';
      const result = replace(content, ' world', '');
      expect(result).toBe('hello');
    });

    it('handles special regex characters in search', () => {
      const content = 'price is $100.00';
      const result = replace(content, '$100.00', '$200.00');
      expect(result).toBe('price is $200.00');
    });

    it('handles newline at end of search', () => {
      const content = 'line1\nline2\nline3';
      const result = replace(content, 'line1\n', 'replaced\n');
      expect(result).toBe('replaced\nline2\nline3');
    });
  });
});
