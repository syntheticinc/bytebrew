// AST Chunker - extracts code chunks from parsed AST
import { Node } from 'web-tree-sitter';
import crypto from 'crypto';
import { CodeChunk, ChunkType } from '../domain/chunk.js';
import { TreeSitterParser, LanguageConfig } from './parser.js';

export class ASTChunker {
  private parser: TreeSitterParser;

  constructor(parser: TreeSitterParser) {
    this.parser = parser;
  }

  async chunkFile(filePath: string, content: string): Promise<CodeChunk[]> {
    const language = TreeSitterParser.detectLanguage(filePath);
    const config = this.parser.getLanguageConfig(language);

    // For unsupported languages, return file as single chunk
    if (!config || !TreeSitterParser.isSupported(language)) {
      return [this.createWholeFileChunk(filePath, content, language)];
    }

    const tree = await this.parser.parse(content, language);
    if (!tree) {
      return [this.createWholeFileChunk(filePath, content, language)];
    }

    const chunks = this.extractChunks(filePath, content, tree.rootNode, config, language);

    // If no chunks were extracted, return file as single chunk
    if (chunks.length === 0) {
      return [this.createWholeFileChunk(filePath, content, language)];
    }

    return chunks;
  }

  private extractChunks(
    filePath: string,
    content: string,
    rootNode: Node,
    config: LanguageConfig,
    language: string
  ): CodeChunk[] {
    const chunks: CodeChunk[] = [];

    this.walkTree(rootNode, '', (node, parentName) => {
      const chunkType = this.determineChunkType(node.type, config);
      if (!chunkType) return;

      const chunk = this.nodeToChunk(filePath, content, node, chunkType, language, parentName);
      if (chunk) {
        chunks.push(chunk);
      }
    });

    return chunks;
  }

  private walkTree(
    node: Node,
    parentName: string,
    visitor: (node: Node, parentName: string) => void
  ): void {
    visitor(node, parentName);

    // Update parent name if this is a class/struct
    let newParentName = parentName;
    const nodeType = node.type;
    if (
      nodeType === 'class_declaration' ||
      nodeType === 'class_definition' ||
      nodeType === 'type_spec' ||
      nodeType === 'struct_type'
    ) {
      const name = this.extractNodeName(node);
      if (name) {
        newParentName = name;
      }
    }

    // Recurse into children
    for (let i = 0; i < node.childCount; i++) {
      const child = node.child(i);
      if (child) {
        this.walkTree(child, newParentName, visitor);
      }
    }
  }

  private determineChunkType(nodeType: string, config: LanguageConfig): ChunkType | null {
    // Function types
    if (config.functionTypes.includes(nodeType)) {
      return nodeType.includes('method') ? 'method' : 'function';
    }

    // Class types
    if (config.classTypes.includes(nodeType)) {
      return nodeType.includes('type_spec') ? 'struct' : 'class';
    }

    // Interface types
    if (config.interfaceTypes.includes(nodeType)) {
      return 'interface';
    }

    // Language-specific handling
    if (nodeType === 'method_declaration') return 'method';
    if (nodeType === 'type_spec') return 'struct';
    if (nodeType === 'var_declaration' || nodeType === 'variable_declaration' || nodeType === 'lexical_declaration') {
      return 'variable';
    }
    if (nodeType === 'const_declaration') return 'constant';

    return null;
  }

  private nodeToChunk(
    filePath: string,
    content: string,
    node: Node,
    chunkType: ChunkType,
    language: string,
    parentName: string
  ): CodeChunk | null {
    const startByte = node.startIndex;
    const endByte = node.endIndex;

    // Skip very small chunks (likely noise)
    if (endByte - startByte < 10) {
      return null;
    }

    const chunkContent = content.slice(startByte, endByte);

    // Skip empty or whitespace-only content
    if (!chunkContent.trim()) {
      return null;
    }

    const name = this.extractNodeName(node);
    const signature = this.extractSignature(node, content, language);
    const id = this.generateChunkId(filePath, node.startPosition.row + 1, name);

    return {
      id,
      filePath,
      content: chunkContent,
      startLine: node.startPosition.row + 1, // 1-indexed
      endLine: node.endPosition.row + 1,
      language,
      chunkType,
      name,
      parentName: parentName || undefined,
      signature: signature || undefined,
    };
  }

  private extractNodeName(node: Node): string {
    const nodeType = node.type;

    // Try common field names
    const nameNode = node.childForFieldName('name') || node.childForFieldName('identifier');
    if (nameNode) {
      return nameNode.text;
    }

    // Go type_spec: look for type_identifier
    if (nodeType === 'type_spec') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && child.type === 'type_identifier') {
          return child.text;
        }
      }
    }

    // Kotlin: class_declaration has type_identifier, functions have simple_identifier
    if (nodeType === 'class_declaration' || nodeType === 'object_declaration') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && (child.type === 'type_identifier' || child.type === 'simple_identifier')) {
          return child.text;
        }
      }
    }
    if (nodeType === 'function_declaration') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        // Kotlin uses simple_identifier, Zig uses identifier
        if (child && (child.type === 'simple_identifier' || child.type === 'identifier')) {
          return child.text;
        }
      }
    }

    // C/C++: function_definition has function_declarator child which has identifier child
    if (nodeType === 'function_definition') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && child.type === 'function_declarator') {
          for (let j = 0; j < child.childCount; j++) {
            const grandchild = child.child(j);
            if (grandchild && grandchild.type === 'identifier') {
              return grandchild.text;
            }
          }
        }
      }
    }

    // C struct_specifier: look for type_identifier directly
    if (nodeType === 'struct_specifier' || nodeType === 'enum_specifier' || nodeType === 'union_specifier') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && child.type === 'type_identifier') {
          return child.text;
        }
      }
    }

    // Elixir: call node — defmodule/def/defp. Extract from arguments.
    if (nodeType === 'call') {
      const kw = node.child(0);
      if (kw && (kw.text === 'defmodule' || kw.text === 'def' || kw.text === 'defp')) {
        const args = node.childForFieldName('arguments') || node.child(1);
        if (args) {
          // defmodule Calculator do → alias node
          const firstArg = args.child(0);
          if (firstArg) {
            if (firstArg.type === 'alias') return firstArg.text;
            // def add(a, b) do → call node (add is identifier child)
            if (firstArg.type === 'call') {
              const fnName = firstArg.child(0);
              if (fnName && fnName.type === 'identifier') return fnName.text;
            }
            if (firstArg.type === 'identifier') return firstArg.text;
          }
        }
      }
    }

    // OCaml: value_definition → let_binding → value_name
    if (nodeType === 'value_definition') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && child.type === 'let_binding') {
          for (let j = 0; j < child.childCount; j++) {
            const gc = child.child(j);
            if (gc && gc.type === 'value_name') return gc.text;
          }
        }
      }
    }
    // OCaml let_binding directly
    if (nodeType === 'let_binding') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && child.type === 'value_name') return child.text;
      }
    }
    // OCaml type_definition / module_definition
    if (nodeType === 'type_definition' || nodeType === 'module_definition' || nodeType === 'class_definition') {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && (child.type === 'type_constructor' || child.type === 'module_name' || child.type === 'class_name')) {
          return child.text;
        }
      }
    }

    // Python/JS class/function: first identifier child
    if (
      nodeType === 'class_definition' ||
      nodeType === 'function_definition' ||
      nodeType === 'class_declaration'
    ) {
      for (let i = 0; i < node.childCount; i++) {
        const child = node.child(i);
        if (child && child.type === 'identifier') {
          return child.text;
        }
      }
    }

    // Arrow functions often don't have names directly
    if (nodeType === 'arrow_function') {
      // Check parent for variable declaration
      const parent = node.parent;
      if (parent && (parent.type === 'variable_declarator' || parent.type === 'pair')) {
        const nameChild = parent.childForFieldName('name') || parent.childForFieldName('key');
        if (nameChild) {
          return nameChild.text;
        }
      }
    }

    return '';
  }

  private extractSignature(node: Node, content: string, language: string): string {
    const nodeType = node.type;
    const startByte = node.startIndex;
    const endByte = node.endIndex;
    const text = content.slice(startByte, endByte);

    // Go-specific
    if (language === 'go') {
      if (nodeType === 'function_declaration' || nodeType === 'method_declaration') {
        // Find the block and get everything before it
        for (let i = 0; i < node.childCount; i++) {
          const child = node.child(i);
          if (child && child.type === 'block') {
            return content.slice(startByte, child.startIndex).trim();
          }
        }
      }
      if (nodeType === 'type_spec') {
        return text.trim();
      }
    }

    // Python-specific
    if (language === 'python') {
      if (nodeType === 'function_definition' || nodeType === 'class_definition') {
        const colonIdx = text.indexOf(':');
        if (colonIdx !== -1) {
          return text.slice(0, colonIdx + 1).trim();
        }
      }
    }

    // JavaScript/TypeScript-specific
    if (language === 'javascript' || language === 'typescript') {
      if (nodeType === 'function_declaration' || nodeType === 'class_declaration') {
        for (let i = 0; i < node.childCount; i++) {
          const child = node.child(i);
          if (child && (child.type === 'statement_block' || child.type === 'class_body')) {
            return content.slice(startByte, child.startIndex).trim();
          }
        }
      }
    }

    // Default: first line
    const newlineIdx = text.indexOf('\n');
    if (newlineIdx !== -1) {
      return text.slice(0, newlineIdx).trim();
    }
    return text.trim();
  }

  private createWholeFileChunk(filePath: string, content: string, language: string): CodeChunk {
    const lines = content.split('\n').length;
    return {
      id: this.generateChunkId(filePath, 1, 'file'),
      filePath,
      content,
      startLine: 1,
      endLine: lines,
      language,
      chunkType: 'other',
      name: filePath,
    };
  }

  private generateChunkId(filePath: string, startLine: number, name: string): string {
    const data = `${filePath}:${startLine}:${name}`;
    const hash = crypto.createHash('sha256').update(data).digest('hex');
    return hash.slice(0, 16);
  }
}
