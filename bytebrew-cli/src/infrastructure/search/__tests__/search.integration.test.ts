/**
 * Integration tests for search functions
 *
 * Run with: npx tsx src/infrastructure/search/__tests__/search.integration.test.ts
 *
 * Prerequisites:
 * - ripgrep (rg) installed and in PATH
 * - Indexed codebase (run `vector index` first)
 */
import path from 'path';
import { fileURLToPath } from 'url';
import { grepSearch, GrepMatch } from '../grepSearch.js';
import { symbolSearch, symbolSearchPartial, SymbolMatch } from '../symbolSearch.js';
import { getStoreFactory } from '../../../indexing/storeFactory.js';
import { IChunkStore } from '../../../domain/store.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Project root for testing (usm-epicsmasher)
const PROJECT_ROOT = path.resolve(__dirname, '../../../../../');
const BYTEBREW_CLI_ROOT = path.resolve(__dirname, '../../../../');

interface TestResult {
  name: string;
  passed: boolean;
  message: string;
  data?: unknown;
}

const results: TestResult[] = [];

function test(name: string, fn: () => Promise<void>) {
  return async () => {
    try {
      await fn();
      results.push({ name, passed: true, message: 'OK' });
      console.log(`  ✓ ${name}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      results.push({ name, passed: false, message });
      console.log(`  ✗ ${name}: ${message}`);
    }
  };
}

function assert(condition: boolean, message: string) {
  if (!condition) {
    throw new Error(message);
  }
}

async function assertArrayNotEmpty<T>(arr: T[], message: string): Promise<void> {
  if (!arr || arr.length === 0) {
    throw new Error(message);
  }
}

// ============================================================================
// GREP SEARCH TESTS
// ============================================================================

async function testGrepFindsPattern() {
  const matches = await grepSearch(BYTEBREW_CLI_ROOT, 'export class');
  await assertArrayNotEmpty(matches, 'Grep should find "export class" pattern');
  assert(
    matches.every((m) => m.content.includes('export') || m.content.includes('class')),
    'All matches should contain pattern components'
  );
}

async function testGrepFindsFunctionCalls() {
  const matches = await grepSearch(BYTEBREW_CLI_ROOT, 'getLogger\\(');
  await assertArrayNotEmpty(matches, 'Grep should find getLogger() calls');
  assert(
    matches.every((m) => m.content.includes('getLogger')),
    'All matches should contain getLogger'
  );
}

async function testGrepRespectsMaxResults() {
  const matches = await grepSearch(BYTEBREW_CLI_ROOT, 'import', { maxResults: 5 });
  assert(matches.length <= 5, `Should return max 5 results, got ${matches.length}`);
}

async function testGrepCaseInsensitive() {
  const matches = await grepSearch(BYTEBREW_CLI_ROOT, 'LOGGER', { ignoreCase: true });
  // Even with case insensitive, we might not find it depending on code
  // Just verify no error thrown
  assert(Array.isArray(matches), 'Should return array');
}

async function testGrepFileTypeFilter() {
  const matches = await grepSearch(BYTEBREW_CLI_ROOT, 'interface', { fileTypes: ['ts'] });
  if (matches.length > 0) {
    // rg -t ts matches all TypeScript files: .ts, .tsx, .cts, .mts
    const tsExtensions = ['.ts', '.tsx', '.cts', '.mts'];
    assert(
      matches.every((m) => tsExtensions.some((ext) => m.filePath.endsWith(ext))),
      'All matches should be from TypeScript files (.ts, .tsx, .cts, .mts)'
    );
  }
}

async function testGrepReturnsMatchStructure() {
  const matches = await grepSearch(BYTEBREW_CLI_ROOT, 'export');
  if (matches.length > 0) {
    const match = matches[0];
    assert(typeof match.filePath === 'string', 'Should have filePath');
    assert(typeof match.line === 'number', 'Should have line number');
    assert(typeof match.content === 'string', 'Should have content');
    assert(match.line > 0, 'Line number should be positive');
  }
}

// ============================================================================
// SYMBOL SEARCH TESTS
// ============================================================================

let store: IChunkStore | null = null;

async function initStore(): Promise<IChunkStore> {
  if (store) return store;

  const factory = getStoreFactory(BYTEBREW_CLI_ROOT);
  store = await factory.getStore();
  return store;
}

async function testSymbolFindsFunction() {
  const s = await initStore();
  const matches = await symbolSearch(s, 'getLogger');
  // Might not find if not indexed, but should not error
  assert(Array.isArray(matches), 'Should return array');
  if (matches.length > 0) {
    assert(
      matches.some((m) => m.symbolName.toLowerCase().includes('getlogger') || m.symbolName.toLowerCase().includes('logger')),
      'Should find logger-related symbol'
    );
  }
}

async function testSymbolFindsClass() {
  const s = await initStore();
  const matches = await symbolSearch(s, 'ChunkStore');
  assert(Array.isArray(matches), 'Should return array');
  if (matches.length > 0) {
    const classMatch = matches.find((m) => m.symbolType === 'class');
    if (classMatch) {
      assert(classMatch.symbolName.includes('ChunkStore'), 'Should find ChunkStore class');
    }
  }
}

async function testSymbolFiltersByType() {
  const s = await initStore();
  const matches = await symbolSearch(s, 'Store', { symbolTypes: ['class', 'interface'] });
  assert(Array.isArray(matches), 'Should return array');
  if (matches.length > 0) {
    assert(
      matches.every((m) => m.symbolType === 'class' || m.symbolType === 'interface'),
      'All matches should be classes or interfaces'
    );
  }
}

async function testSymbolReturnsMatchStructure() {
  const s = await initStore();
  const matches = await symbolSearch(s, 'search');
  if (matches.length > 0) {
    const match = matches[0];
    assert(typeof match.filePath === 'string', 'Should have filePath');
    assert(typeof match.startLine === 'number', 'Should have startLine');
    assert(typeof match.endLine === 'number', 'Should have endLine');
    assert(typeof match.symbolName === 'string', 'Should have symbolName');
    assert(typeof match.symbolType === 'string', 'Should have symbolType');
  }
}

async function testSymbolPartialSearch() {
  const s = await initStore();
  const matches = await symbolSearchPartial(s, 'Store');
  assert(Array.isArray(matches), 'Should return array');
  if (matches.length > 0) {
    assert(
      matches.every((m) => m.symbolName.toLowerCase().includes('store')),
      'All matches should contain "store" in name'
    );
  }
}

// ============================================================================
// COMPLEMENTARY SEARCH TEST
// ============================================================================

async function testAllThreeComplementary() {
  const s = await initStore();
  const query = 'search';

  // 1. Grep search - finds literal text
  const grepMatches = await grepSearch(BYTEBREW_CLI_ROOT, 'search');

  // 2. Symbol search - finds named symbols
  const symbolMatches = await symbolSearch(s, 'search');

  console.log(`    Grep found: ${grepMatches.length} matches`);
  console.log(`    Symbol found: ${symbolMatches.length} matches`);

  // All three should return arrays (even if empty due to indexing)
  assert(Array.isArray(grepMatches), 'Grep should return array');
  assert(Array.isArray(symbolMatches), 'Symbol should return array');

  // At least grep should find something
  await assertArrayNotEmpty(grepMatches, 'Grep should find "search" in codebase');
}

// ============================================================================
// TEST RUNNER
// ============================================================================

async function runTests() {
  console.log('\n=== Search Integration Tests ===\n');

  console.log('Grep Search Tests:');
  await test('grep_finds_pattern', testGrepFindsPattern)();
  await test('grep_finds_function_calls', testGrepFindsFunctionCalls)();
  await test('grep_respects_max_results', testGrepRespectsMaxResults)();
  await test('grep_case_insensitive', testGrepCaseInsensitive)();
  await test('grep_file_type_filter', testGrepFileTypeFilter)();
  await test('grep_returns_match_structure', testGrepReturnsMatchStructure)();

  console.log('\nSymbol Search Tests:');
  try {
    await test('symbol_finds_function', testSymbolFindsFunction)();
    await test('symbol_finds_class', testSymbolFindsClass)();
    await test('symbol_filters_by_type', testSymbolFiltersByType)();
    await test('symbol_returns_match_structure', testSymbolReturnsMatchStructure)();
    await test('symbol_partial_search', testSymbolPartialSearch)();
  } catch (error) {
    console.log('  ⚠ Symbol tests skipped (codebase may not be indexed)');
  }

  console.log('\nComplementary Search Test:');
  await test('all_three_complementary', testAllThreeComplementary)();

  // Summary
  console.log('\n=== Summary ===');
  const passed = results.filter((r) => r.passed).length;
  const failed = results.filter((r) => !r.passed).length;
  console.log(`Passed: ${passed}, Failed: ${failed}`);

  if (failed > 0) {
    console.log('\nFailed tests:');
    results.filter((r) => !r.passed).forEach((r) => {
      console.log(`  - ${r.name}: ${r.message}`);
    });
  }

  // Cleanup
  if (store) {
    store.close();
  }

  console.log('');
  process.exit(failed > 0 ? 1 : 0);
}

// Run tests
runTests().catch((error) => {
  console.error('Test runner error:', error);
  process.exit(1);
});
