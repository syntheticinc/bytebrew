// Symbol search - find code symbols by name using the indexed chunks
import { IChunkStore } from '../../domain/store.js';
import { CodeChunk, ChunkType } from '../../domain/chunk.js';
import { getLogger } from '../../lib/logger.js';

export interface SymbolMatch {
  filePath: string;
  startLine: number;
  endLine: number;
  symbolName: string;
  symbolType: string;
  signature?: string;
  content?: string;
}

export interface SymbolSearchOptions {
  symbolTypes?: ChunkType[];
  maxResults?: number;
  exactMatch?: boolean;
}

const DEFAULT_MAX_RESULTS = 20;

/**
 * Search for symbols by name using the indexed code chunks
 * Uses SQLite index on name column for fast exact matching
 * Falls back to semantic search for fuzzy matching
 */
export async function symbolSearch(
  store: IChunkStore,
  symbolName: string,
  options: SymbolSearchOptions = {}
): Promise<SymbolMatch[]> {
  const logger = getLogger();

  if (!symbolName) {
    return [];
  }

  const maxResults = options.maxResults ?? DEFAULT_MAX_RESULTS;
  const exactMatch = options.exactMatch ?? false;

  try {
    let chunks: CodeChunk[];

    if (exactMatch) {
      // Use exact name match (SQLite index)
      chunks = await store.getByName(symbolName);
    } else {
      // Use semantic search for fuzzy matching
      const searchResults = await store.search(symbolName, maxResults * 2);
      chunks = searchResults.map((r) => r.chunk);
    }

    // Filter by symbol types if specified
    if (options.symbolTypes && options.symbolTypes.length > 0) {
      chunks = chunks.filter((chunk) =>
        options.symbolTypes!.includes(chunk.chunkType)
      );
    }

    // Filter by name similarity for fuzzy search
    if (!exactMatch) {
      chunks = chunks.filter((chunk) =>
        isNameMatch(chunk.name, symbolName)
      );
    }

    const matches = chunks.slice(0, maxResults).map(chunkToSymbolMatch);
    logger.debug('Symbol search completed', { symbolName, resultsCount: matches.length });
    return matches;
  } catch (error) {
    logger.error('Symbol search failed', { symbolName, error });
    return [];
  }
}

/**
 * Search for symbols by partial name match
 * Useful for autocomplete-style search
 */
export async function symbolSearchPartial(
  store: IChunkStore,
  partialName: string,
  options: SymbolSearchOptions = {}
): Promise<SymbolMatch[]> {
  const logger = getLogger();

  if (!partialName || partialName.length < 2) {
    return [];
  }

  const maxResults = options.maxResults ?? DEFAULT_MAX_RESULTS;

  try {
    // Use semantic search with the partial name
    const searchResults = await store.search(partialName, maxResults * 3);
    let chunks = searchResults.map((r) => r.chunk);

    // Filter by partial name match
    const lowerPartial = partialName.toLowerCase();
    chunks = chunks.filter((chunk) =>
      chunk.name.toLowerCase().includes(lowerPartial)
    );

    // Filter by symbol types if specified
    if (options.symbolTypes && options.symbolTypes.length > 0) {
      chunks = chunks.filter((chunk) =>
        options.symbolTypes!.includes(chunk.chunkType)
      );
    }

    const matches = chunks.slice(0, maxResults).map(chunkToSymbolMatch);
    logger.debug('Partial symbol search completed', { partialName, resultsCount: matches.length });
    return matches;
  } catch (error) {
    logger.error('Partial symbol search failed', { partialName, error });
    return [];
  }
}

function chunkToSymbolMatch(chunk: CodeChunk): SymbolMatch {
  return {
    filePath: chunk.filePath,
    startLine: chunk.startLine,
    endLine: chunk.endLine,
    symbolName: chunk.name,
    symbolType: chunk.chunkType,
    signature: chunk.signature,
    content: chunk.content,
  };
}

/**
 * Check if chunk name matches the search term
 * Uses case-insensitive matching and supports partial matches
 */
function isNameMatch(chunkName: string, searchTerm: string): boolean {
  const lowerChunkName = chunkName.toLowerCase();
  const lowerSearchTerm = searchTerm.toLowerCase();

  // Exact match
  if (lowerChunkName === lowerSearchTerm) {
    return true;
  }

  // Contains match
  if (lowerChunkName.includes(lowerSearchTerm)) {
    return true;
  }

  // CamelCase/snake_case token match
  const chunkTokens = tokenizeName(lowerChunkName);
  const searchTokens = tokenizeName(lowerSearchTerm);

  return searchTokens.every((searchToken) =>
    chunkTokens.some((chunkToken) => chunkToken.includes(searchToken))
  );
}

/**
 * Tokenize a name by splitting on camelCase and snake_case boundaries
 */
function tokenizeName(name: string): string[] {
  return name
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/_/g, ' ')
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean);
}
