// Plan tool definition - manage_plan
import React from 'react';
import { ToolDefinition, ToolRendererProps } from '../ToolManager.js';
import { PlanView } from '../../presentation/components/plan/PlanView.js';
import { parsePlanFromArgs } from '../../domain/plan.js';

/**
 * Renderer for manage_plan tool
 * Displays plan as a tree with checkboxes
 */
function renderPlan({ arguments: args, isExecuting }: ToolRendererProps): React.ReactNode {
  const plan = parsePlanFromArgs(args as Record<string, unknown>);

  if (!plan) {
    return null; // Fallback to standard rendering
  }

  return <PlanView plan={plan} isExecuting={isExecuting} />;
}

/**
 * manage_plan tool definition
 * - No executor (executes on server)
 * - Custom renderer (tree with checkboxes)
 */
export const planToolDefinition: ToolDefinition = {
  name: 'manage_plan',
  displayName: 'Plan',

  // Server-side execution - no client executor
  executor: undefined,

  // Custom renderer
  renderer: renderPlan,

  // Render separately, not grouped with other tools
  renderSeparately: true,
};
