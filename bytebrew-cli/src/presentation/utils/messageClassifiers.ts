/**
 * Shared utilities for classifying and styling assistant messages.
 */

/**
 * Check if message content is an agent lifecycle event (spawned, completed, failed, etc.)
 * Lifecycle messages start with +⊕✓✗↻ characters.
 */
export function isLifecycleMessage(content: string): boolean {
  return /^[+⊕✓✗↻]/.test(content.trim());
}

/**
 * Get the display color for a lifecycle message based on its prefix character.
 */
export function getLifecycleColor(content: string): string {
  const ch = content.trim()[0];
  if (ch === '✓') return 'green';
  if (ch === '✗') return 'red';
  if (ch === '+' || ch === '⊕') return 'yellow';
  if (ch === '↻') return 'blue';
  return 'gray';
}

/**
 * Check if message content is a separator line (e.g., "─── Code Agent [abc]: Task ───").
 * Separators contain the '───' pattern.
 */
export function isSeparatorMessage(content: string): boolean {
  return content.includes('───');
}
