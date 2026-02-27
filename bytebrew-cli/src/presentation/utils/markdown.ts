// Shared markdown rendering for terminal output.
// Configured ONCE to avoid conflicting marked.use() calls.
import { marked } from 'marked';
import { markedTerminal } from 'marked-terminal';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
marked.use(markedTerminal({
  width: 100,
  reflowText: true,
  showSectionPrefix: false,
}) as any);

/**
 * Parse markdown string into terminal-formatted text (with ANSI codes).
 * Returns trimmed result. Falls back to raw text on error.
 */
export function renderMarkdown(text: string): string {
  if (!text) return '';
  try {
    return (marked.parse(text) as string).trimEnd();
  } catch {
    return text;
  }
}
