// IToolRegistry - interface for tool registration checks
// Used by infrastructure layer to determine if tool should be executed locally

/**
 * Interface for checking tool registration and execution location
 */
export interface IToolRegistry {
  /**
   * Check if tool is registered
   */
  has(toolName: string): boolean;

  /**
   * Check if tool executes on client side
   * Returns false for server-side tools
   */
  executesOnClient(toolName: string): boolean;

  /**
   * List all registered tool names
   */
  list(): string[];
}
