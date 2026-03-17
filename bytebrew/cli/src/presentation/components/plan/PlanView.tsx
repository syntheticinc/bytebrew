// PlanView component - displays execution plan with checkboxes
import React from 'react';
import { Box, Text } from 'ink';
import { Plan, PlanStep, getStepStatusMarker, isPlanComplete } from '../../../domain/plan.js';

interface PlanViewProps {
  plan: Plan;
  isExecuting?: boolean;
}

/**
 * Displays a plan with goal and steps as a tree with checkboxes
 *
 * Example:
 * ● [ ] Plan - Analyze project complexity
 *    └ [✔] step 1 - Analyze core files
 *    └ [ ] step 2 - Review dependencies
 */
export const PlanView: React.FC<PlanViewProps> = ({ plan, isExecuting = false }) => {
  const isComplete = isPlanComplete(plan);

  // Main indicator color
  const mainColor = isComplete ? 'green' : isExecuting ? 'yellow' : 'gray';
  const mainMarker = isComplete ? '✓' : ' ';

  return (
    <Box flexDirection="column" marginBottom={1}>
      {/* Plan header with goal */}
      <Box>
        <Text color={mainColor}>●</Text>
        <Text color="white"> [</Text>
        <Text color={isComplete ? 'green' : 'gray'}>{mainMarker}</Text>
        <Text color="white">] </Text>
        <Text color="cyan" bold>Plan</Text>
        <Text color="white"> - </Text>
        <Text color="white">{plan.goal}</Text>
      </Box>

      {/* Steps */}
      {plan.steps.map((step) => (
        <PlanStepView key={step.index} step={step} />
      ))}
    </Box>
  );
};

interface PlanStepViewProps {
  step: PlanStep;
}

const PlanStepView: React.FC<PlanStepViewProps> = ({ step }) => {
  const marker = getStepStatusMarker(step.status);
  const isActive = step.status === 'in_progress';
  const isCompleted = step.status === 'completed';

  // Color based on status
  const statusColor = isCompleted ? 'green' : isActive ? 'yellow' : 'gray';
  const textColor = isCompleted ? 'gray' : 'white';

  return (
    <Box marginLeft={3}>
      <Text color="gray">└ </Text>
      <Text color={statusColor}>[</Text>
      <Text color={statusColor}>{marker}</Text>
      <Text color={statusColor}>]</Text>
      <Text color={textColor}> step {step.index + 1}</Text>
      <Text color="gray"> - </Text>
      <Text color={textColor}>{step.description}</Text>
    </Box>
  );
};
