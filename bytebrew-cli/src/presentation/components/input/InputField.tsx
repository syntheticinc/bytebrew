// Input field with history support
import React, { useState, useCallback, useEffect } from 'react';
import { Box, Text, useStdin } from 'ink';
import TextInput from 'ink-text-input';

interface InputFieldProps {
  onSubmit: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  history?: string[];
}

const InputFieldComponent: React.FC<InputFieldProps> = ({
  onSubmit,
  disabled = false,
  placeholder = 'Type a message...',
  history = [],
}) => {
  const [value, setValue] = useState('');
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [savedInput, setSavedInput] = useState('');
  const { stdin, setRawMode } = useStdin();

  const handleSubmit = useCallback(
    (inputValue: string) => {
      const trimmed = inputValue.trim();
      if (trimmed && !disabled) {
        onSubmit(trimmed);
        setValue('');
        setHistoryIndex(-1);
        setSavedInput('');
      }
    },
    [disabled, onSubmit]
  );

  // Navigate history
  const navigateHistory = useCallback(
    (direction: 'up' | 'down') => {
      if (disabled || history.length === 0) return;

      if (direction === 'up') {
        if (historyIndex === -1) {
          // Save current input before navigating history
          setSavedInput(value);
        }
        const newIndex = Math.min(historyIndex + 1, history.length - 1);
        setHistoryIndex(newIndex);
        setValue(history[history.length - 1 - newIndex] || '');
      } else {
        if (historyIndex > 0) {
          const newIndex = historyIndex - 1;
          setHistoryIndex(newIndex);
          setValue(history[history.length - 1 - newIndex] || '');
        } else if (historyIndex === 0) {
          // Restore saved input
          setHistoryIndex(-1);
          setValue(savedInput);
        }
      }
    },
    [disabled, history, historyIndex, savedInput, value]
  );

  // Listen for arrow keys via stdin (escape sequences)
  useEffect(() => {
    if (disabled || !stdin) return;

    const handleData = (data: Buffer) => {
      const input = data.toString();

      // Arrow up: ESC [ A
      if (input === '\x1b[A' || input === '\x1bOA') {
        navigateHistory('up');
      }
      // Arrow down: ESC [ B
      else if (input === '\x1b[B' || input === '\x1bOB') {
        navigateHistory('down');
      }
    };

    stdin.on('data', handleData);
    return () => {
      stdin.off('data', handleData);
    };
  }, [stdin, disabled, navigateHistory]);

  return (
    <Box
      borderStyle="single"
      borderColor={disabled ? 'gray' : 'cyan'}
      paddingX={1}
      width="100%"
      flexDirection="row"
    >
      <Box flexGrow={1} flexDirection="row">
        {disabled ? (
          <Text color="gray">{placeholder}</Text>
        ) : (
          <TextInput
            value={value}
            onChange={setValue}
            onSubmit={handleSubmit}
            placeholder={placeholder}
          />
        )}
      </Box>
    </Box>
  );
};

// Memoize to prevent re-renders when parent updates but props haven't changed
export const InputField = React.memo(InputFieldComponent);
