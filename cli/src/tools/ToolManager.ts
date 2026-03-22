// ToolManager - unified tool management (execution + rendering)
import type { DiffLine } from '../domain/message.js';
import {
  IToolRenderingService,
  ToolRenderer,
  ToolRendererProps,
} from '../domain/ports/IToolRenderingService.js';
import { IToolRegistry } from '../domain/ports/IToolRegistry.js';

// Re-export types for convenience
export type { ToolRendererProps, ToolRenderer };

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

/**
 * Unified tool definition combining execution and rendering
 */
export interface ToolDefinition {
  /** Unique tool name */
  name: string;

  /**
   * Executor for client-side execution
   * If null/undefined, tool executes on server side
   */
  executor?: Tool;

  /**
   * Custom renderer for UI display
   * If null/undefined, uses standard tool display
   */
  renderer?: ToolRenderer;

  /**
   * Whether this tool should be rendered separately (not grouped)
   * Default: true if has custom renderer, false otherwise
   */
  renderSeparately?: boolean;

  /**
   * Display name for UI (optional, defaults to formatted tool name)
   */
  displayName?: string;
}

/**
 * ToolManager - single source of truth for tool definitions
 * Handles both execution routing and rendering decisions
 * Implements IToolRenderingService for use by presentation layer
 * Implements IToolRegistry for use by infrastructure layer
 */
class ToolManagerClass implements IToolRenderingService, IToolRegistry {
  private definitions: Map<string, ToolDefinition> = new Map();

  /**
   * Register a tool definition
   */
  register(definition: ToolDefinition): void {
    this.definitions.set(definition.name, definition);
  }

  /**
   * Register multiple tool definitions
   */
  registerAll(definitions: ToolDefinition[]): void {
    for (const def of definitions) {
      this.register(def);
    }
  }

  /**
   * Get tool definition by name
   */
  get(name: string): ToolDefinition | undefined {
    return this.definitions.get(name);
  }

  /**
   * Check if tool exists
   */
  has(name: string): boolean {
    return this.definitions.has(name);
  }

  /**
   * Check if tool executes on client side
   */
  executesOnClient(name: string): boolean {
    const def = this.definitions.get(name);
    return def?.executor !== undefined;
  }

  /**
   * Get executor for a tool (null if server-side)
   */
  getExecutor(name: string): Tool | undefined {
    return this.definitions.get(name)?.executor;
  }

  /**
   * Check if tool has custom renderer
   */
  hasCustomRenderer(name: string): boolean {
    return this.definitions.get(name)?.renderer !== undefined;
  }

  /**
   * Get custom renderer for a tool (null if standard)
   */
  getRenderer(name: string): ToolRenderer | undefined {
    return this.definitions.get(name)?.renderer;
  }

  /**
   * Check if tool should be rendered separately (not grouped)
   */
  shouldRenderSeparately(name: string): boolean {
    const def = this.definitions.get(name);
    if (!def) return false;

    // Explicit setting takes precedence
    if (def.renderSeparately !== undefined) {
      return def.renderSeparately;
    }

    // Default: separate if has custom renderer
    return def.renderer !== undefined;
  }

  /**
   * Get display name for a tool
   */
  getDisplayName(name: string): string {
    const def = this.definitions.get(name);
    if (def?.displayName) {
      return def.displayName;
    }

    // Format tool name: manage_plan -> Manage Plan
    return name
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  }

  /**
   * Execute a tool (returns null if server-side tool)
   */
  async execute(name: string, args: Record<string, string>): Promise<ToolResult | null> {
    const executor = this.getExecutor(name);
    if (!executor) {
      return null; // Server-side tool
    }
    return executor.execute(args);
  }

  /**
   * List all registered tool names
   */
  list(): string[] {
    return Array.from(this.definitions.keys());
  }

  /**
   * List tools that execute on client
   */
  listClientTools(): string[] {
    return Array.from(this.definitions.entries())
      .filter(([_, def]) => def.executor !== undefined)
      .map(([name]) => name);
  }

  /**
   * List tools with custom renderers
   */
  listCustomRenderedTools(): string[] {
    return Array.from(this.definitions.entries())
      .filter(([_, def]) => def.renderer !== undefined)
      .map(([name]) => name);
  }

  /**
   * Clear all registrations (for testing)
   */
  clear(): void {
    this.definitions.clear();
  }
}

// Singleton instance
export const ToolManager = new ToolManagerClass();
