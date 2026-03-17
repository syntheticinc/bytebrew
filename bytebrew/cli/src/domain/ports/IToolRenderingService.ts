// IToolRenderingService - interface for tool rendering decisions
// Follows DIP: presentation layer depends on this interface, not concrete ToolManager

import React from 'react';

/**
 * Props passed to custom tool renderers
 */
export interface ToolRendererProps {
  toolName: string;
  arguments: Record<string, unknown>;
  result?: string;
  error?: string;
  isExecuting: boolean;
}

/**
 * Custom renderer function type
 */
export type ToolRenderer = (props: ToolRendererProps) => React.ReactNode;

/**
 * Interface for tool rendering decisions
 * Used by presentation layer to determine how to render tools
 */
export interface IToolRenderingService {
  /**
   * Check if tool should be rendered separately (not grouped with others)
   */
  shouldRenderSeparately(toolName: string): boolean;

  /**
   * Get custom renderer for a tool (undefined if standard rendering)
   */
  getRenderer(toolName: string): ToolRenderer | undefined;
}
