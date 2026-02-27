// Centralized color palette for UI components.
// Terminal-safe: named colors (Ink built-ins) + hex where needed.

export const colors = {
  // Semantic status
  success: 'green',
  error: 'red',
  warning: 'yellow',
  info: 'blue',

  // UI chrome
  primary: 'cyan',
  muted: 'gray',
  text: 'white',

  // Agent processing (spinner, tool execution label)
  processing: '#D7513E',
} as const;
