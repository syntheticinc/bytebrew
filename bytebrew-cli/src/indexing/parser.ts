// TreeSitter parser for code analysis
import { Parser, Language, Tree } from 'web-tree-sitter';
import path from 'path';

// Absolute path to the directory containing this module (dist/ after bundling).
// Used to resolve WASM paths independent of the process CWD.
const DIST_DIR = import.meta.dir;

// Embedded WASM files — bun compile embeds these into the binary
// @ts-ignore: bun import attribute
import treeSitterCorePath from '../../node_modules/web-tree-sitter/tree-sitter.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import goWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-go.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import tsWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-typescript.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import jsWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-javascript.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import pyWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-python.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import rustWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-rust.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import cWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-c.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import cppWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-cpp.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import csharpWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-c_sharp.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import javaWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-java.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import kotlinWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-kotlin.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import rubyWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-ruby.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import phpWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-php.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import swiftWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-swift.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import dartWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-dart.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import luaWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-lua.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import elixirWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-elixir.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import bashWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-bash.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import ocamlWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-ocaml.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import zigWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-zig.wasm' with { type: 'file' };
// @ts-ignore: bun import attribute
import scalaWasmPath from '../../node_modules/tree-sitter-wasms/out/tree-sitter-scala.wasm' with { type: 'file' };

// Map language name to embedded WASM path
const EMBEDDED_WASM: Record<string, string> = {
  go: goWasmPath,
  typescript: tsWasmPath,
  javascript: jsWasmPath,
  python: pyWasmPath,
  rust: rustWasmPath,
  c: cWasmPath,
  cpp: cppWasmPath,
  csharp: csharpWasmPath,
  java: javaWasmPath,
  kotlin: kotlinWasmPath,
  ruby: rubyWasmPath,
  php: phpWasmPath,
  swift: swiftWasmPath,
  dart: dartWasmPath,
  lua: luaWasmPath,
  elixir: elixirWasmPath,
  bash: bashWasmPath,
  ocaml: ocamlWasmPath,
  zig: zigWasmPath,
  scala: scalaWasmPath,
};

export interface LanguageConfig {
  name: string;
  extensions: string[];
  functionTypes: string[];
  classTypes: string[];
  interfaceTypes: string[];
}

// Supported language configurations
const LANGUAGE_CONFIGS: Record<string, LanguageConfig> = {
  go: {
    name: 'go',
    extensions: ['.go'],
    functionTypes: ['function_declaration', 'method_declaration'],
    classTypes: ['type_spec'],
    interfaceTypes: ['interface_type'],
  },
  typescript: {
    name: 'typescript',
    extensions: ['.ts', '.tsx'],
    functionTypes: ['function_declaration', 'arrow_function', 'method_definition'],
    classTypes: ['class_declaration'],
    interfaceTypes: ['interface_declaration'],
  },
  javascript: {
    name: 'javascript',
    extensions: ['.js', '.jsx', '.mjs'],
    functionTypes: ['function_declaration', 'arrow_function', 'method_definition'],
    classTypes: ['class_declaration'],
    interfaceTypes: [],
  },
  python: {
    name: 'python',
    extensions: ['.py'],
    functionTypes: ['function_definition'],
    classTypes: ['class_definition'],
    interfaceTypes: [],
  },
  rust: {
    name: 'rust',
    extensions: ['.rs'],
    functionTypes: ['function_item'],
    classTypes: ['struct_item', 'enum_item', 'union_item', 'type_item'],
    interfaceTypes: ['trait_item'],
  },
  c: {
    name: 'c',
    extensions: ['.c', '.h'],
    functionTypes: ['function_definition'],
    classTypes: ['struct_specifier', 'enum_specifier', 'union_specifier', 'type_definition'],
    interfaceTypes: [],
  },
  cpp: {
    name: 'cpp',
    extensions: ['.cpp', '.cc', '.cxx', '.hpp', '.hxx'],
    functionTypes: ['function_definition'],
    classTypes: ['class_specifier', 'struct_specifier', 'enum_specifier'],
    interfaceTypes: [],
  },
  csharp: {
    name: 'csharp',
    extensions: ['.cs'],
    functionTypes: ['method_declaration', 'constructor_declaration'],
    classTypes: ['class_declaration', 'struct_declaration', 'enum_declaration', 'record_declaration'],
    interfaceTypes: ['interface_declaration'],
  },
  java: {
    name: 'java',
    extensions: ['.java'],
    functionTypes: ['method_declaration', 'constructor_declaration'],
    classTypes: ['class_declaration', 'enum_declaration', 'record_declaration'],
    interfaceTypes: ['interface_declaration'],
  },
  kotlin: {
    name: 'kotlin',
    extensions: ['.kt', '.kts'],
    functionTypes: ['function_declaration'],
    classTypes: ['class_declaration', 'object_declaration'],
    interfaceTypes: [],
  },
  ruby: {
    name: 'ruby',
    extensions: ['.rb'],
    functionTypes: ['method', 'singleton_method'],
    classTypes: ['class', 'module'],
    interfaceTypes: [],
  },
  php: {
    name: 'php',
    extensions: ['.php'],
    functionTypes: ['function_definition', 'method_declaration'],
    classTypes: ['class_declaration', 'enum_declaration'],
    interfaceTypes: ['interface_declaration', 'trait_declaration'],
  },
  swift: {
    name: 'swift',
    extensions: ['.swift'],
    functionTypes: ['function_declaration'],
    classTypes: ['class_declaration', 'struct_declaration', 'enum_declaration'],
    interfaceTypes: ['protocol_declaration'],
  },
  dart: {
    name: 'dart',
    extensions: ['.dart'],
    functionTypes: ['function_signature'],
    classTypes: ['class_definition'],
    interfaceTypes: [],
  },
  lua: {
    name: 'lua',
    extensions: ['.lua'],
    functionTypes: ['function_declaration', 'function_definition_statement'],
    classTypes: [],
    interfaceTypes: [],
  },
  elixir: {
    name: 'elixir',
    extensions: ['.ex', '.exs'],
    functionTypes: ['call'],
    classTypes: ['call'],
    interfaceTypes: [],
  },
  bash: {
    name: 'bash',
    extensions: ['.sh', '.bash'],
    functionTypes: ['function_definition'],
    classTypes: [],
    interfaceTypes: [],
  },
  ocaml: {
    name: 'ocaml',
    extensions: ['.ml', '.mli'],
    functionTypes: ['value_definition', 'let_binding'],
    classTypes: ['type_definition', 'class_definition', 'module_definition'],
    interfaceTypes: ['module_type_definition'],
  },
  zig: {
    name: 'zig',
    extensions: ['.zig'],
    functionTypes: ['function_declaration'],
    classTypes: [],
    interfaceTypes: [],
  },
  scala: {
    name: 'scala',
    extensions: ['.scala', '.sc'],
    functionTypes: ['function_definition'],
    classTypes: ['class_definition', 'object_definition'],
    interfaceTypes: ['trait_definition'],
  },
};

// Extension to language mapping
const EXTENSION_TO_LANG: Record<string, string> = {
  '.go': 'go',
  '.ts': 'typescript',
  '.tsx': 'typescript',
  '.js': 'javascript',
  '.jsx': 'javascript',
  '.mjs': 'javascript',
  '.py': 'python',
  '.rs': 'rust',
  '.c': 'c',
  '.h': 'c',
  '.cpp': 'cpp',
  '.cc': 'cpp',
  '.cxx': 'cpp',
  '.hpp': 'cpp',
  '.hxx': 'cpp',
  '.cs': 'csharp',
  '.java': 'java',
  '.kt': 'kotlin',
  '.kts': 'kotlin',
  '.rb': 'ruby',
  '.php': 'php',
  '.swift': 'swift',
  '.dart': 'dart',
  '.lua': 'lua',
  '.ex': 'elixir',
  '.exs': 'elixir',
  '.sh': 'bash',
  '.bash': 'bash',
  '.ml': 'ocaml',
  '.mli': 'ocaml',
  '.zig': 'zig',
  '.scala': 'scala',
  '.sc': 'scala',
  '.html': 'html',
  '.css': 'css',
  '.json': 'json',
  '.yaml': 'yaml',
  '.yml': 'yaml',
  '.md': 'markdown',
  '.sql': 'sql',
};

export class TreeSitterParser {
  private initialized = false;
  private parsers: Map<string, Parser> = new Map();

  async init(): Promise<void> {
    if (this.initialized) return;

    await Parser.init({
      locateFile: () => path.resolve(DIST_DIR, treeSitterCorePath),
    });
    this.initialized = true;
  }

  async getParser(language: string): Promise<Parser | null> {
    const config = LANGUAGE_CONFIGS[language];
    if (!config) return null;

    if (this.parsers.has(language)) {
      return this.parsers.get(language)!;
    }

    const wasmPath = EMBEDDED_WASM[language];
    if (!wasmPath) return null;

    try {
      const parser = new Parser();
      const Lang = await Language.load(path.resolve(DIST_DIR, wasmPath));
      parser.setLanguage(Lang);
      this.parsers.set(language, parser);
      return parser;
    } catch (error) {
      console.error(`Failed to load parser for ${language}:`, error);
      return null;
    }
  }

  async parse(code: string, language: string): Promise<Tree | null> {
    await this.init();
    const parser = await this.getParser(language);
    if (!parser) return null;

    return parser.parse(code);
  }

  getLanguageConfig(language: string): LanguageConfig | null {
    return LANGUAGE_CONFIGS[language] || null;
  }

  static detectLanguage(filePath: string): string {
    const ext = path.extname(filePath).toLowerCase();
    return EXTENSION_TO_LANG[ext] || 'text';
  }

  static isSupported(language: string): boolean {
    return language in LANGUAGE_CONFIGS;
  }

  static getSupportedLanguages(): string[] {
    return Object.keys(LANGUAGE_CONFIGS);
  }

  static getLanguageByExtension(ext: string): string | null {
    const normalizedExt = ext.startsWith('.') ? ext : `.${ext}`;
    return EXTENSION_TO_LANG[normalizedExt.toLowerCase()] || null;
  }
}
