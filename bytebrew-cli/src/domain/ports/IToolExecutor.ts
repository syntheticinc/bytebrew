// IToolExecutor port - interface for tool execution
import { ToolCallInfo } from '../entities/Message.js';
import type { DiffLine } from '../message.js';

export interface ToolExecutionResult {
  result: string;
  error?: Error;
  summary?: string;
  diffLines?: DiffLine[];
}

/**
 * Interface for executing tools.
 * Abstracts the tool execution implementation from the application layer.
 */
export interface IToolExecutor {
  /**
   * Execute a tool with the given arguments
   */
  execute(toolCall: ToolCallInfo): Promise<ToolExecutionResult>;

  /**
   * Check if a tool is available
   */
  hasTool(name: string): boolean;

  /**
   * List available tools
   */
  listTools(): string[];
}
