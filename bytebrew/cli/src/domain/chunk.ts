// CodeChunk domain types for code indexing

export type ChunkType =
  | 'function'
  | 'method'
  | 'class'
  | 'interface'
  | 'struct'
  | 'type'
  | 'variable'
  | 'constant'
  | 'other';

export interface CodeChunk {
  id: string;
  filePath: string;
  content: string;
  startLine: number;
  endLine: number;
  language: string;
  chunkType: ChunkType;
  name: string;
  parentName?: string;
  signature?: string;
}

export interface IndexStatus {
  totalChunks: number;
  filesCount: number;
  languages: string[];
  lastUpdated: Date;
  isStale: boolean;
}

export interface SearchResult {
  chunk: CodeChunk;
  score: number;
}
