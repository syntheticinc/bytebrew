import { describe, it, expect, mock, beforeEach, afterAll, spyOn } from 'bun:test';
import fs from 'fs/promises';
import { LspTool } from '../lspTool.js';
import type { LspService, LspLocation } from '../../infrastructure/lsp/LspService.js';
import type { IChunkStoreFactory, IChunkStore } from '../../domain/store.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeLocation(filePath: string, line: number): LspLocation {
  return {
    uri: `file://${filePath.replace(/\\/g, '/')}`,
    range: {
      start: { line, character: 0 },
      end: { line, character: 10 },
    },
  };
}

function makeSymbolMatch(opts: {
  filePath: string;
  startLine: number;
}) {
  return {
    filePath: opts.filePath,
    startLine: opts.startLine,
    endLine: opts.startLine,
    symbolName: 'AgentEvent',
    symbolType: 'class',
  };
}

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockStore = {
  getByName: mock(() => Promise.resolve([])),
  search: mock(() => Promise.resolve([])),
} as unknown as IChunkStore;

const mockStoreFactory = {
  getStore: mock(() => Promise.resolve(mockStore)),
} as unknown as IChunkStoreFactory;

const mockLspService = {
  definition: mock(() => Promise.resolve([])),
  references: mock(() => Promise.resolve([])),
  implementation: mock(() => Promise.resolve([])),
  getHealthInfo: mock(() => null),
  waitForReady: mock(() => Promise.resolve()),
  hasActiveClients: mock(() => false),
} as unknown as LspService;

// Mock fs.readFile to return controlled file content
const readFileSpy = spyOn(fs, 'readFile');

// ---------------------------------------------------------------------------
// Fixture
// ---------------------------------------------------------------------------

const PROJECT_ROOT = '/home/user/project';

function createTool(): LspTool {
  return new LspTool({
    lspService: mockLspService,
    storeFactory: mockStoreFactory,
    projectRoot: PROJECT_ROOT,
    readyTimeoutMs: 0, // Skip readiness wait in unit tests
  });
}

/**
 * Set up fs.readFile mock to return specified content for any file read.
 * Lines are joined with newlines and padded so startLine points to the right place.
 */
function mockFileContent(startLine: number, lines: string[]) {
  // Build file content with padding so line indices are correct
  const padding = Array(startLine - 1).fill('');
  const fullLines = [...padding, ...lines];
  readFileSpy.mockResolvedValue(fullLines.join('\n'));
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('LspTool', () => {
  beforeEach(() => {
    // Reset all mocks between tests
    (mockStore.getByName as ReturnType<typeof mock>).mockReset();
    (mockStore.search as ReturnType<typeof mock>).mockReset();
    (mockLspService.definition as ReturnType<typeof mock>).mockReset();
    (mockLspService.references as ReturnType<typeof mock>).mockReset();
    (mockLspService.implementation as ReturnType<typeof mock>).mockReset();
    (mockLspService.getHealthInfo as ReturnType<typeof mock>).mockReset();
    (mockLspService.waitForReady as ReturnType<typeof mock>).mockReset();
    (mockLspService.hasActiveClients as ReturnType<typeof mock>).mockReset();
    (mockStoreFactory.getStore as ReturnType<typeof mock>).mockReset();
    readFileSpy.mockReset();

    // Default: store always available
    (mockStoreFactory.getStore as ReturnType<typeof mock>).mockResolvedValue(mockStore);
    // Default: exact search returns nothing, fuzzy returns nothing
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([]);
    (mockStore.search as ReturnType<typeof mock>).mockResolvedValue([]);
    // Default: file read fails (falls back to startLine, char 0)
    readFileSpy.mockRejectedValue(new Error('ENOENT'));
    // Default: no health issues, no active clients
    (mockLspService.getHealthInfo as ReturnType<typeof mock>).mockReturnValue(null);
    (mockLspService.waitForReady as ReturnType<typeof mock>).mockResolvedValue(undefined);
    (mockLspService.hasActiveClients as ReturnType<typeof mock>).mockReturnValue(false);
  });

  afterAll(() => {
    readFileSpy.mockRestore();
  });

  // -------------------------------------------------------------------------
  // Validation
  // -------------------------------------------------------------------------

  it('returns error when symbol_name is missing', async () => {
    const tool = createTool();
    const result = await tool.execute({ operation: 'definition' });

    expect(result.error).toBeDefined();
    expect(result.error!.message).toContain('symbol_name');
  });

  it('returns error when operation is missing', async () => {
    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'Foo' });

    expect(result.error).toBeDefined();
    expect(result.error!.message).toContain('operation');
  });

  it('returns error for invalid operation', async () => {
    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'Foo', operation: 'hover' });

    expect(result.error).toBeDefined();
    expect(result.error!.message).toContain('Invalid operation');
    expect(result.error!.message).toContain('definition, references, implementation');
  });

  // -------------------------------------------------------------------------
  // Symbol not found
  // -------------------------------------------------------------------------

  it('returns helpful message when symbol is not found (both exact and fuzzy)', async () => {
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([]);
    (mockStore.search as ReturnType<typeof mock>).mockResolvedValue([]);

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'Missing', operation: 'definition' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('No symbols found matching "Missing"');
  });

  it('returns "no symbols found" when exact match returns empty (no fuzzy fallback)', async () => {
    // Fuzzy fallback was removed to avoid Ollama dependency.
    // Only exact match (getByName) is used.
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([]);

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'definition' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('No symbols found matching "AgentEvent"');
    expect(result.summary).toBe('symbol not found');
  });

  // -------------------------------------------------------------------------
  // definition
  // -------------------------------------------------------------------------

  it('definition: symbol found, LSP returns one location', async () => {
    const match = makeSymbolMatch({
      filePath: '/home/user/project/internal/domain/agent_event.go',
      startLine: 12,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    mockFileContent(12, ['type AgentEvent struct {']);

    const loc = makeLocation('/home/user/project/internal/domain/agent_event.go', 11);
    (mockLspService.definition as ReturnType<typeof mock>).mockResolvedValue([loc]);

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'definition' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('Definition of "AgentEvent"');
    expect(result.result).toContain('internal/domain/agent_event.go:12');
    expect(result.summary).toBe('1 result');
  });

  // -------------------------------------------------------------------------
  // references
  // -------------------------------------------------------------------------

  it('references: multiple locations', async () => {
    const match = makeSymbolMatch({
      filePath: '/home/user/project/internal/domain/agent_event.go',
      startLine: 12,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    mockFileContent(12, ['type AgentEvent struct {']);

    const locations = [
      makeLocation('/home/user/project/internal/domain/agent_event.go', 11),
      makeLocation('/home/user/project/internal/service/agent/agent_events.go', 24),
      makeLocation('/home/user/project/internal/service/agent/agent_events.go', 47),
    ];
    (mockLspService.references as ReturnType<typeof mock>).mockResolvedValue(locations);

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'references' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('References of "AgentEvent" (3 found');
    expect(result.result).toContain('internal/domain/agent_event.go:12');
    expect(result.result).toContain('internal/service/agent/agent_events.go:25');
    expect(result.result).toContain('internal/service/agent/agent_events.go:48');
    expect(result.summary).toBe('3 results');
  });

  // -------------------------------------------------------------------------
  // implementation
  // -------------------------------------------------------------------------

  it('implementation: returns locations', async () => {
    const match = makeSymbolMatch({
      filePath: '/home/user/project/internal/domain/iface.go',
      startLine: 5,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    mockFileContent(5, ['type AgentEvent interface {']);

    const loc = makeLocation('/home/user/project/internal/infra/impl.go', 20);
    (mockLspService.implementation as ReturnType<typeof mock>).mockResolvedValue([loc]);

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'implementation' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('Implementation of "AgentEvent"');
    expect(result.result).toContain('internal/infra/impl.go:21');
  });

  // -------------------------------------------------------------------------
  // LSP returns empty
  // -------------------------------------------------------------------------

  it('returns helpful message when LSP returns empty array', async () => {
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/foo.go',
      startLine: 10,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    (mockLspService.definition as ReturnType<typeof mock>).mockResolvedValue([]);

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'definition' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('No results found for operation "definition"');
  });

  // -------------------------------------------------------------------------
  // Position resolution from real file
  // -------------------------------------------------------------------------

  it('resolves column by reading the actual file line', async () => {
    const symbolName = 'AgentEvent';
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/event.go',
      startLine: 5,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    // Simulate file with "type AgentEvent struct {" on line 5
    mockFileContent(5, ['type AgentEvent struct {']);

    let capturedCharacter = -1;
    (mockLspService.definition as ReturnType<typeof mock>).mockImplementation(
      (_filePath: string, _line: number, character: number) => {
        capturedCharacter = character;
        return Promise.resolve([makeLocation('/home/user/project/src/event.go', 4)]);
      },
    );

    const tool = createTool();
    await tool.execute({ symbol_name: symbolName, operation: 'definition' });

    // "AgentEvent" starts at column 5 in "type AgentEvent struct {"
    expect(capturedCharacter).toBe(5);
  });

  it('resolves position from nearby line when symbol is below startLine', async () => {
    const symbolName = 'AgentEvent';
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/event.go',
      startLine: 10,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    // Symbol is on line 11 (startLine+1), not on line 10
    mockFileContent(10, ['// doc comment', 'type AgentEvent struct {']);

    let capturedLine = -1;
    let capturedCharacter = -1;
    (mockLspService.definition as ReturnType<typeof mock>).mockImplementation(
      (_filePath: string, line: number, character: number) => {
        capturedLine = line;
        capturedCharacter = character;
        return Promise.resolve([makeLocation('/home/user/project/src/event.go', 10)]);
      },
    );

    const tool = createTool();
    await tool.execute({ symbol_name: symbolName, operation: 'definition' });

    // Line 11 in file (0-based: 10), column 5
    expect(capturedLine).toBe(10);
    expect(capturedCharacter).toBe(5);
  });

  it('skips comment lines when resolving position', async () => {
    const symbolName = 'AgentEvent';
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/event.go',
      startLine: 8,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    // Line 8 is a comment mentioning AgentEvent, line 9 has the real definition
    mockFileContent(8, ['// AgentEvent is a domain event', 'type AgentEvent struct {']);

    let capturedLine = -1;
    let capturedCharacter = -1;
    (mockLspService.definition as ReturnType<typeof mock>).mockImplementation(
      (_filePath: string, line: number, character: number) => {
        capturedLine = line;
        capturedCharacter = character;
        return Promise.resolve([makeLocation('/home/user/project/src/event.go', 8)]);
      },
    );

    const tool = createTool();
    await tool.execute({ symbol_name: symbolName, operation: 'definition' });

    // Skips comment line 8, finds on line 9 (0-based: 8), column 5
    expect(capturedLine).toBe(8);
    expect(capturedCharacter).toBe(5);
  });

  it('does not match partial identifiers (word boundary check)', async () => {
    const symbolName = 'AgentEvent';
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/event.go',
      startLine: 32,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    // Line 32 is comment (skipped), line 33 has AgentEventType (partial match!),
    // line 34 has the real definition
    mockFileContent(32, [
      '// AgentEvent comment',
      '  Type       AgentEventType         `json:"type"`',
      'type AgentEvent struct {',
    ]);

    let capturedLine = -1;
    let capturedCharacter = -1;
    (mockLspService.definition as ReturnType<typeof mock>).mockImplementation(
      (_filePath: string, line: number, character: number) => {
        capturedLine = line;
        capturedCharacter = character;
        return Promise.resolve([makeLocation('/home/user/project/src/event.go', 33)]);
      },
    );

    const tool = createTool();
    await tool.execute({ symbol_name: symbolName, operation: 'definition' });

    // Should skip "AgentEventType" (partial) and find "AgentEvent" on next line
    expect(capturedLine).toBe(33); // 0-based for line 34
    expect(capturedCharacter).toBe(5); // "type AgentEvent struct {"
  });

  it('falls back to startLine char 0 when file read fails', async () => {
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/event.go',
      startLine: 5,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    // File read fails (default mock behavior)

    let capturedLine = -1;
    let capturedCharacter = -1;
    (mockLspService.definition as ReturnType<typeof mock>).mockImplementation(
      (_filePath: string, line: number, character: number) => {
        capturedLine = line;
        capturedCharacter = character;
        return Promise.resolve([makeLocation('/home/user/project/src/event.go', 4)]);
      },
    );

    const tool = createTool();
    await tool.execute({ symbol_name: 'AgentEvent', operation: 'definition' });

    expect(capturedLine).toBe(4); // startLine - 1
    expect(capturedCharacter).toBe(0);
  });

  // -------------------------------------------------------------------------
  // Graceful degradation when LSP fails
  // -------------------------------------------------------------------------

  it('does not throw when LSP throws, returns error message', async () => {
    const match = makeSymbolMatch({
      filePath: '/home/user/project/src/event.go',
      startLine: 5,
    });
    (mockStore.getByName as ReturnType<typeof mock>).mockResolvedValue([match]);
    (mockLspService.definition as ReturnType<typeof mock>).mockRejectedValue(
      new Error('LSP server crashed'),
    );

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'definition' });

    // Should NOT throw, should return a user-friendly message
    expect(result.error).toBeUndefined();
    expect(result.result).toContain('LSP operation failed');
  });

  it('returns error message when store is unavailable', async () => {
    (mockStoreFactory.getStore as ReturnType<typeof mock>).mockRejectedValue(
      new Error('Store not initialized'),
    );

    const tool = createTool();
    const result = await tool.execute({ symbol_name: 'AgentEvent', operation: 'definition' });

    expect(result.error).toBeUndefined();
    expect(result.result).toContain('Symbol search unavailable');
  });
});
