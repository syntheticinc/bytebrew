// Plan domain types for manage_plan tool

export type PlanStepStatus = 'pending' | 'in_progress' | 'completed';

export interface PlanStep {
  index: number;
  description: string;
  reasoning?: string;
  status: PlanStepStatus;
}

export interface Plan {
  goal: string;
  steps: PlanStep[];
}

// Cache last known plan to fill in missing descriptions on updates
// (LLM often sends only status updates without descriptions)
let lastKnownPlan: Plan | null = null;

/**
 * Reset cached plan (useful for new sessions)
 */
export function resetPlanCache(): void {
  lastKnownPlan = null;
}

/**
 * Parse plan from manage_plan tool arguments
 * Handles various input formats: JSON string, object, or mixed
 * Also fills in missing descriptions from cached plan
 */
export function parsePlanFromArgs(args: Record<string, unknown>): Plan | null {
  try {
    // Handle case where entire args might be a JSON string
    let parsedArgs = args;
    if (typeof args === 'string') {
      parsedArgs = JSON.parse(args);
    }

    // Handle _json wrapper (server sends complex args wrapped in _json field)
    if (parsedArgs._json && typeof parsedArgs._json === 'string') {
      parsedArgs = JSON.parse(parsedArgs._json);
    }

    // Get goal
    const goal = parsedArgs.goal;
    if (!goal || typeof goal !== 'string') return null;

    // Get steps - can be JSON string or array
    let steps: PlanStep[];
    const rawSteps = parsedArgs.steps;

    if (typeof rawSteps === 'string') {
      steps = JSON.parse(rawSteps);
    } else if (Array.isArray(rawSteps)) {
      steps = rawSteps as PlanStep[];
    } else {
      return null;
    }

    // Validate steps is an array
    if (!Array.isArray(steps) || steps.length === 0) {
      return null;
    }

    // Validate and normalize steps, filling missing descriptions from cache
    const normalizedSteps: PlanStep[] = steps.map((step, idx) => {
      let description = String(step.description || '');

      // Fill from cache if description is empty
      if (!description && lastKnownPlan && idx < lastKnownPlan.steps.length) {
        description = lastKnownPlan.steps[idx].description;
      }

      return {
        index: typeof step.index === 'number' ? step.index : idx,
        description,
        reasoning: step.reasoning ? String(step.reasoning) : undefined,
        status: normalizeStatus(String(step.status || 'pending')),
      };
    });

    const plan: Plan = {
      goal,
      steps: normalizedSteps,
    };

    // Update cache
    lastKnownPlan = plan;

    return plan;
  } catch {
    return null;
  }
}

function normalizeStatus(status: string): PlanStepStatus {
  switch (status) {
    case 'pending':
    case 'in_progress':
    case 'completed':
      return status;
    default:
      return 'pending';
  }
}

/**
 * Get status marker for display
 */
export function getStepStatusMarker(status: PlanStepStatus): string {
  switch (status) {
    case 'completed':
      return '✓';
    case 'in_progress':
      return '○';
    case 'pending':
    default:
      return ' ';
  }
}

/**
 * Check if plan is complete (all steps done)
 */
export function isPlanComplete(plan: Plan): boolean {
  return plan.steps.every(step => step.status === 'completed');
}

/**
 * Get plan progress
 */
export function getPlanProgress(plan: Plan): { completed: number; total: number } {
  const completed = plan.steps.filter(step => step.status === 'completed').length;
  return { completed, total: plan.steps.length };
}
