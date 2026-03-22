// PermissionApprovalPrompt - UI component for permission approval dialog
import React, { useState, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import { PermissionRequest } from '../../domain/permission/Permission.js';

export interface PermissionApprovalPromptProps {
  request: PermissionRequest;
  onApprove: (remember: boolean) => void;
  onReject: () => void;
  agentId?: string;
}

type Selection = 'once' | 'always' | 'reject';

const TYPE_LABELS: Record<string, string> = {
  bash: 'Command Execution',
  read: 'File Read',
  edit: 'File Edit',
  list: 'Directory Listing',
};

/**
 * Permission approval prompt shown when a tool action needs user approval.
 * User can:
 * - [1] Allow once (don't remember)
 * - [2] Always allow (remember pattern)
 * - [3] Deny
 */
export const PermissionApprovalPrompt: React.FC<PermissionApprovalPromptProps> = ({
  request,
  onApprove,
  onReject,
  agentId,
}) => {
  const [selection, setSelection] = useState<Selection>('once');

  useInput((input, key) => {
    if (input === '1') {
      onApprove(false);
      return;
    }
    if (input === '2') {
      onApprove(true);
      return;
    }
    if (input === '3') {
      onReject();
      return;
    }

    if (key.upArrow || key.leftArrow) {
      setSelection((prev) => {
        if (prev === 'always') return 'once';
        if (prev === 'reject') return 'always';
        return prev;
      });
    }
    if (key.downArrow || key.rightArrow) {
      setSelection((prev) => {
        if (prev === 'once') return 'always';
        if (prev === 'always') return 'reject';
        return prev;
      });
    }

    if (key.return) {
      if (selection === 'once') {
        onApprove(false);
      } else if (selection === 'always') {
        onApprove(true);
      } else {
        onReject();
      }
    }

    if (key.escape) {
      onReject();
    }
  });

  const renderOption = useCallback(
    (value: Selection, label: string, keyStr: string) => {
      const isSelected = selection === value;
      return (
        <Box>
          <Text color={isSelected ? 'cyan' : 'white'}>
            {isSelected ? '> ' : '  '}
            [{keyStr}] {label}
          </Text>
        </Box>
      );
    },
    [selection]
  );

  const typeLabel = TYPE_LABELS[request.type] || request.type;
  const MAX_DISPLAY_LEN = 200;
  const displayValue = request.value.length > MAX_DISPLAY_LEN
    ? request.value.slice(0, MAX_DISPLAY_LEN) + '...'
    : request.value;

  return (
    <Box flexDirection="column" borderStyle="round" borderColor="yellow" paddingX={1} marginY={1}>
      <Box marginBottom={1}>
        <Text color="yellow" bold>
          Permission Required: {typeLabel}
          {agentId && agentId !== 'supervisor' ? ` (${agentId})` : ''}
        </Text>
      </Box>

      <Box marginBottom={1} paddingX={2}>
        <Text color="cyan" bold>
          {displayValue}
        </Text>
      </Box>

      <Box marginBottom={1}>
        <Text dimColor>
          Choose an action (use arrow keys or number keys):
        </Text>
      </Box>

      {renderOption('once', 'Allow once', '1')}
      {renderOption('always', 'Always allow (remember)', '2')}
      {renderOption('reject', 'Deny', '3')}

      <Box marginTop={1}>
        <Text dimColor>
          Press Enter to confirm, Escape to deny
        </Text>
      </Box>
    </Box>
  );
};
