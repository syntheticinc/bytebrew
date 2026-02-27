// Tool execution hook
import { useCallback, useRef } from 'react';
import { ToolExecutor } from '../../tools/executor.js';
import { ToolCallInfo } from '../../domain/message.js';

interface UseToolExecutionOptions {
  projectRoot: string;
  onToolResult: (callId: string, result: string, error?: string) => void;
}

export function useToolExecution({ projectRoot, onToolResult }: UseToolExecutionOptions) {
  const executorRef = useRef<ToolExecutor | null>(null);

  // Lazy initialize executor
  const getExecutor = useCallback(() => {
    if (!executorRef.current) {
      executorRef.current = new ToolExecutor(projectRoot);
    }
    return executorRef.current;
  }, [projectRoot]);

  const executeTool = useCallback(
    async (toolCall: ToolCallInfo): Promise<{ result: string; error?: Error }> => {
      const executor = getExecutor();
      const result = await executor.execute(toolCall.toolName, toolCall.arguments);

      // Notify about the result
      onToolResult(
        toolCall.callId,
        result.result,
        result.error?.message
      );

      return result;
    },
    [getExecutor, onToolResult]
  );

  const hasTool = useCallback(
    (name: string): boolean => {
      return getExecutor().hasTool(name);
    },
    [getExecutor]
  );

  return {
    executeTool,
    hasTool,
  };
}
