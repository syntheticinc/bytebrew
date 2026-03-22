// Test that all proxied tools return summary field
import { describe, it, expect, beforeAll, afterAll } from 'bun:test';
import path from 'path';
import fs from 'fs/promises';
import { ReadFileTool } from '../src/tools/readFile.js';
import { WriteFileTool } from '../src/tools/writeFile.js';
import { EditFileTool } from '../src/tools/editFile.js';
import { ExecuteCommandTool } from '../src/tools/executeCommand.js';
import { ProjectTreeTool } from '../src/tools/projectTree.js';
import { GrepSearchTool } from '../src/tools/grepSearch.js';
import { AskUserTool } from '../src/tools/askUser.js';
import { FileIgnore } from '../src/domain/file-ignore/FileIgnore.js';

const testRoot = path.resolve(import.meta.dir, '../test-project');
const testFile = path.join(testRoot, 'test-summary.txt');

beforeAll(async () => {
  // Create test file
  await fs.writeFile(testFile, 'hello world\nline 2\nline 3', 'utf-8');
});

afterAll(async () => {
  // Cleanup
  try {
    await fs.unlink(testFile);
  } catch {}
});

describe('Tool Summary', () => {
  describe('ReadFileTool', () => {
    it('returns summary with line count', async () => {
      const tool = new ReadFileTool(testRoot);
      const result = await tool.execute({ file_path: testFile });

      expect(result.summary).toBeDefined();
      expect(result.summary).toMatch(/^\d+ lines \(.+\)$/);
      expect(result.summary).toContain('test-summary.txt');
    });

    it('returns summary for not found file', async () => {
      const tool = new ReadFileTool(testRoot);
      const result = await tool.execute({ file_path: 'nonexistent.txt' });

      expect(result.summary).toBeDefined();
      expect(result.summary).toContain('not found');
    });
  });

  describe('WriteFileTool', () => {
    it('returns summary with line count', async () => {
      const tool = new WriteFileTool(testRoot);
      const testFile = path.join(testRoot, 'test-write.txt');
      const result = await tool.execute({
        file_path: testFile,
        content: 'line1\nline2\nline3',
      });

      expect(result.summary).toBeDefined();
      expect(result.summary).toMatch(/^\d+ lines \(.+\)$/);
    });
  });

  describe('EditFileTool', () => {
    it('returns summary with line diff', async () => {
      const tool = new EditFileTool(testRoot);
      const result = await tool.execute({
        file_path: testFile,
        old_string: 'line 2',
        new_string: 'line 2\nextra line',
      });

      expect(result.summary).toBeDefined();
      expect(result.summary).toMatch(/^[+\-±]\d+ lines \(.+\)$/);
    });
  });

  describe('ExecuteCommandTool', () => {
    it('returns summary with exit code', async () => {
      const tool = new ExecuteCommandTool(testRoot);
      const result = await tool.execute({ command: 'echo test' });

      expect(result.summary).toBeDefined();
      expect(result.summary).toMatch(/^exit \d+$/);
    });
  });

  describe('ProjectTreeTool', () => {
    it('returns summary with item count', async () => {
      const fileIgnore = new FileIgnore(testRoot);
      const tool = new ProjectTreeTool(testRoot, fileIgnore);
      const result = await tool.execute({});

      expect(result.summary).toBeDefined();
      expect(result.summary).toMatch(/^\d+ items$/);
    });
  });

  describe('GrepSearchTool', () => {
    it('returns summary with result count', async () => {
      const tool = new GrepSearchTool({ projectRoot: testRoot });
      const result = await tool.execute({ pattern: 'func', limit: '5' });

      expect(result.summary).toBeDefined();
      // Could be "X results" or "no results" depending on what's in test-project
      expect(result.summary).toMatch(/^(\d+ results?|no results)$/);
    });

    it('returns "no results" when nothing found', async () => {
      const tool = new GrepSearchTool({ projectRoot: testRoot });
      const result = await tool.execute({ pattern: 'NONEXISTENT_PATTERN_XYZ' });

      expect(result.summary).toBeDefined();
      expect(result.summary).toBe('no results');
    });
  });

  describe('AskUserTool', () => {
    it('returns summary with truncated answer', async () => {
      const tool = new AskUserTool(true); // headless mode
      const result = await tool.execute({
        question: 'Test question?',
        default_answer: 'Short answer',
      });

      expect(result.summary).toBeDefined();
      expect(result.summary).toBe('Short answer');
    });

    it('truncates long answers', async () => {
      const tool = new AskUserTool(true);
      const longAnswer = 'A'.repeat(50);
      const result = await tool.execute({
        question: 'Test?',
        default_answer: longAnswer,
      });

      expect(result.summary).toBeDefined();
      expect(result.summary?.length).toBeLessThanOrEqual(30);
      expect(result.summary).toContain('…');
    });
  });
});
