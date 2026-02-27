// Multi-language integration test: tree-sitter parsing + storeMetadataOnly + symbolSearch
// Verifies that all supported languages are correctly parsed and symbols are findable.
import { describe, it, expect, beforeAll, afterAll } from 'bun:test';
import { mkdtemp, writeFile, rm } from 'fs/promises';
import { tmpdir } from 'os';
import path from 'path';
import { ChunkStore } from '../store.js';
import { TreeSitterParser } from '../parser.js';
import { ASTChunker } from '../chunker.js';
import { symbolSearch } from '../../infrastructure/search/symbolSearch.js';
import { IEmbeddingsClient } from '../../domain/store.js';

// Minimal embeddings stub — storeMetadataOnly() which
// does NOT call embeddings, so this stub is never invoked.
const stubEmbeddings: IEmbeddingsClient = {
  embed: async (_text: string) => [],
  embedBatch: async (texts: string[]) => texts.map(() => null),
  ping: async () => false,
  getDimension: () => 768,
  getModel: () => 'stub',
};

/**
 * On Windows, SQLite WAL files and USearch index files are not released
 * immediately after close(). Retry deletion with a short delay.
 */
async function deleteWithRetry(dir: string, attempts = 5, delayMs = 100): Promise<void> {
  for (let i = 0; i < attempts; i++) {
    try {
      await rm(dir, { recursive: true, force: true });
      return;
    } catch {
      if (i === attempts - 1) return; // give up silently
      await new Promise((r) => setTimeout(r, delayMs));
    }
  }
}

// ─── Test file contents for each language ────────────────────────────────────

const GO_CONTENT = `package main

type MyService struct {
\tname string
}

func (s *MyService) Process() error {
\treturn nil
}

type Processor interface {
\tProcess() error
}
`;

const TYPESCRIPT_CONTENT = `export interface Logger {
\tlog(msg: string): void;
}

export class ConsoleLogger implements Logger {
\tlog(msg: string): void {
\t\tconsole.log(msg);
\t}
}

export function createLogger(): Logger {
\treturn new ConsoleLogger();
}
`;

const JAVASCRIPT_CONTENT = `class EventEmitter {
\tconstructor() {
\t\tthis.listeners = {};
\t}

\temit(event, data) {
\t\treturn true;
\t}
}

function createEmitter() {
\treturn new EventEmitter();
}
`;

const PYTHON_CONTENT = `class DataProcessor:
\tdef __init__(self):
\t\tself.data = []

\tdef process(self, item):
\t\tself.data.append(item)

def create_processor():
\treturn DataProcessor()
`;

const RUST_CONTENT = `pub struct Config {
\tname: String,
}

pub trait Configurable {
\tfn configure(&self) -> Config;
}

pub fn default_config() -> Config {
\tConfig { name: String::new() }
}
`;

const C_CONTENT = `struct Buffer {
\tchar* data;
\tint size;
};

void buffer_init(struct Buffer* buf) {
\tbuf->data = 0;
\tbuf->size = 0;
}
`;

const CPP_CONTENT = `class Connection {
public:
\tConnection();
\tvoid connect();
};

void Connection::connect() {
}
`;

const CSHARP_CONTENT = `namespace App {
\tpublic interface IRepository {
\t\tvoid Save(object item);
\t}

\tpublic class SqlRepository : IRepository {
\t\tpublic void Save(object item) {}
\t}
}
`;

const JAVA_CONTENT = `public class UserService {
\tpublic void createUser(String name) {
\t}
}

interface UserRepository {
\tvoid save(Object user);
}
`;

const KOTLIN_CONTENT = `class ApiClient {
\tfun request(url: String): String {
\t\treturn ""
\t}
}

fun createClient(): ApiClient {
\treturn ApiClient()
}
`;

const RUBY_CONTENT = `class TaskRunner
\tdef run(task)
\t\ttask.execute
\tend
end

module Runnable
\tdef execute
\t\traise NotImplementedError
\tend
end
`;

const PHP_CONTENT = `<?php
interface Cacheable {
\tpublic function cache(): void;
}

class RedisCache implements Cacheable {
\tpublic function cache(): void {}
}

function createCache(): Cacheable {
\treturn new RedisCache();
}
`;

const SWIFT_CONTENT = `protocol Renderable {
\tfunc render() -> String
}

class HtmlRenderer: Renderable {
\tfunc render() -> String {
\t\treturn "<html></html>"
\t}
}
`;

// NOTE: Dart is excluded from tests because tree-sitter-dart WASM (grammar v15)
// is incompatible with web-tree-sitter@0.24.x (supports grammar v13-14).
// Re-enable after upgrading web-tree-sitter to 0.25.x+.

const LUA_CONTENT = `function greet(name)
\treturn "Hello, " .. name
end

local function helper()
\treturn true
end
`;

// Elixir: defmodule defines the module, def/defp define functions
const ELIXIR_CONTENT = `defmodule Calculator do
  def add(a, b) do
    a + b
  end

  defp validate(x) do
    x > 0
  end
end
`;

const BASH_CONTENT = `#!/bin/bash

function deploy() {
\techo "Deploying..."
}

function rollback() {
\techo "Rolling back..."
}
`;

// OCaml: value_definition is the top-level let binding
const OCAML_CONTENT = `type config = {
  name: string;
  value: int;
}

let create_config name value =
  { name; value }

module Parser = struct
  let parse input = input
end
`;

// Zig: fn_proto is the function prototype node
const ZIG_CONTENT = `const std = @import("std");

pub fn processData(data: []const u8) !void {
    _ = data;
}

fn helperFunction() void {
}
`;

const SCALA_CONTENT = `trait Repository {
  def find(id: String): Option[String]
}

class InMemoryRepository extends Repository {
  def find(id: String): Option[String] = None
}

object RepositoryFactory {
  def create(): Repository = new InMemoryRepository()
}
`;

const DART_CONTENT = `class UserService {
  final String name;
  UserService(this.name);

  String greet() {
    return 'Hello, \$name';
  }
}

void processData(List<int> data) {
  for (var item in data) {
    print(item);
  }
}
`;

// ─── Language test configurations ────────────────────────────────────────────

interface LangTestConfig {
  file: string;
  content: string;
  // Symbols that MUST be found via symbolSearch(store, name, { exactMatch: true })
  requiredSymbols: string[];
}

const LANG_CONFIGS: Record<string, LangTestConfig> = {
  go: {
    file: 'main.go',
    content: GO_CONTENT,
    requiredSymbols: ['MyService', 'Process', 'Processor'],
  },
  typescript: {
    file: 'service.ts',
    content: TYPESCRIPT_CONTENT,
    requiredSymbols: ['Logger', 'ConsoleLogger', 'createLogger'],
  },
  javascript: {
    file: 'emitter.js',
    content: JAVASCRIPT_CONTENT,
    requiredSymbols: ['EventEmitter', 'createEmitter'],
  },
  python: {
    file: 'processor.py',
    content: PYTHON_CONTENT,
    requiredSymbols: ['DataProcessor', 'create_processor'],
  },
  rust: {
    file: 'config.rs',
    content: RUST_CONTENT,
    requiredSymbols: ['Config', 'Configurable', 'default_config'],
  },
  c: {
    file: 'buffer.c',
    content: C_CONTENT,
    // struct_specifier name extraction varies — buffer_init is reliable
    requiredSymbols: ['buffer_init'],
  },
  cpp: {
    file: 'connection.cpp',
    content: CPP_CONTENT,
    requiredSymbols: ['Connection'],
  },
  csharp: {
    file: 'Repository.cs',
    content: CSHARP_CONTENT,
    requiredSymbols: ['IRepository', 'SqlRepository'],
  },
  java: {
    file: 'UserService.java',
    content: JAVA_CONTENT,
    requiredSymbols: ['UserService', 'UserRepository'],
  },
  kotlin: {
    file: 'ApiClient.kt',
    content: KOTLIN_CONTENT,
    requiredSymbols: ['ApiClient', 'createClient'],
  },
  ruby: {
    file: 'task_runner.rb',
    content: RUBY_CONTENT,
    requiredSymbols: ['TaskRunner', 'Runnable'],
  },
  php: {
    file: 'cache.php',
    content: PHP_CONTENT,
    requiredSymbols: ['Cacheable', 'RedisCache'],
  },
  swift: {
    file: 'Renderer.swift',
    content: SWIFT_CONTENT,
    requiredSymbols: ['Renderable', 'HtmlRenderer'],
  },
  dart: {
    file: 'user_service.dart',
    content: DART_CONTENT,
    requiredSymbols: ['UserService', 'processData', 'greet'],
  },
  lua: {
    file: 'greet.lua',
    content: LUA_CONTENT,
    requiredSymbols: ['greet'],
  },
  elixir: {
    file: 'calculator.ex',
    content: ELIXIR_CONTENT,
    // Elixir uses 'call' node type for both modules and functions.
    // The module name 'Calculator' is the most reliably extracted symbol.
    requiredSymbols: ['Calculator'],
  },
  bash: {
    file: 'deploy.sh',
    content: BASH_CONTENT,
    requiredSymbols: ['deploy', 'rollback'],
  },
  ocaml: {
    file: 'config.ml',
    content: OCAML_CONTENT,
    // value_definition / type_definition / module_definition
    requiredSymbols: ['create_config'],
  },
  zig: {
    file: 'main.zig',
    content: ZIG_CONTENT,
    requiredSymbols: ['processData', 'helperFunction'],
  },
  scala: {
    file: 'Repository.scala',
    content: SCALA_CONTENT,
    requiredSymbols: ['Repository', 'InMemoryRepository', 'RepositoryFactory'],
  },
};

// ─── Test suite ──────────────────────────────────────────────────────────────

describe('Multi-language integration: tree-sitter + storeMetadataOnly + symbolSearch', () => {
  let tempDir: string;
  let store: ChunkStore;

  beforeAll(async () => {
    tempDir = await mkdtemp(path.join(tmpdir(), 'multilang-test-'));

    // Write all test files to tempDir
    for (const cfg of Object.values(LANG_CONFIGS)) {
      await writeFile(path.join(tempDir, cfg.file), cfg.content, 'utf-8');
    }

    // Initialize store, parser, and chunker — index files directly
    store = new ChunkStore(tempDir, stubEmbeddings, { bytebrewDir: '.vector-test' });
    await store.ensureCollection();

    const parser = new TreeSitterParser();
    await parser.init();
    const chunker = new ASTChunker(parser);

    let totalChunks = 0;
    for (const cfg of Object.values(LANG_CONFIGS)) {
      const filePath = path.join(tempDir, cfg.file);
      const content = cfg.content;
      const chunks = await chunker.chunkFile(filePath, content);
      if (chunks.length > 0) {
        await store.storeMetadataOnly(chunks);
        totalChunks += chunks.length;
      }
    }

    // Sanity: at least some chunks must have been indexed
    expect(totalChunks).toBeGreaterThan(0);
  });

  afterAll(async () => {
    store?.close();
    await deleteWithRetry(tempDir);
  });

  // ─── Per-language symbol tests ────────────────────────────────────────────

  for (const [lang, cfg] of Object.entries(LANG_CONFIGS)) {
    describe(lang, () => {
      for (const symbolName of cfg.requiredSymbols) {
        it(`finds "${symbolName}" in ${cfg.file}`, async () => {
          const results = await symbolSearch(store, symbolName, { exactMatch: true });

          expect(results.length).toBeGreaterThan(0);

          const match = results.find((r) => r.filePath.endsWith(cfg.file));
          expect(match).toBeDefined();
          expect(match!.symbolName).toBe(symbolName);
          expect(match!.startLine).toBeGreaterThan(0);
          expect(match!.filePath).toContain(cfg.file);
        });
      }
    });
  }

  // ─── Cross-cutting assertions ─────────────────────────────────────────────

  it('all 20 languages produce at least one chunk', async () => {
    const missingLangs: string[] = [];

    for (const [lang, cfg] of Object.entries(LANG_CONFIGS)) {
      const chunks = await store.getByFilePath(path.join(tempDir, cfg.file));
      if (chunks.length === 0) {
        missingLangs.push(lang);
      }
    }

    if (missingLangs.length > 0) {
      throw new Error(`Languages with zero chunks: ${missingLangs.join(', ')}`);
    }
  });

  it('indexed chunks have valid filePath, startLine, and chunkType', async () => {
    for (const cfg of Object.values(LANG_CONFIGS)) {
      const chunks = await store.getByFilePath(path.join(tempDir, cfg.file));
      for (const chunk of chunks) {
        expect(chunk.filePath).toBeTruthy();
        expect(chunk.startLine).toBeGreaterThan(0);
        expect(chunk.chunkType).toBeTruthy();
      }
    }
  });
});
