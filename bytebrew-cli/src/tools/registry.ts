// Tool registry for managing available tools
import type { DiffLine } from '../domain/message.js';

export interface ToolResult {
  result: string;
  error?: Error;
  summary?: string;
  diffLines?: DiffLine[];
}

export interface Tool {
  name: string;
  needsContext?: boolean;
  execute(args: Record<string, string>): Promise<ToolResult>;
}

export class ToolRegistry {
  private tools: Map<string, Tool> = new Map();

  register(tool: Tool): void {
    this.tools.set(tool.name, tool);
  }

  get(name: string): Tool | undefined {
    return this.tools.get(name);
  }

  has(name: string): boolean {
    return this.tools.has(name);
  }

  list(): string[] {
    return Array.from(this.tools.keys());
  }
}
