// Line-level diff computation for write_file/edit_file tools
import { diffLines } from 'diff';
import type { DiffLine } from '../domain/message.js';

export type { DiffLine };

const MAX_DIFF_LINES = 20;

/**
 * Compute line-level diff between old and new content.
 * Returns array of DiffLine for display. Truncated to maxLines.
 */
export function computeLineDiff(oldContent: string, newContent: string, maxLines: number = MAX_DIFF_LINES): DiffLine[] {
  if (oldContent === newContent) return [];

  // Skip binary content
  if (oldContent.includes('\0') || newContent.includes('\0')) return [];

  const changes = diffLines(oldContent, newContent);
  const lines: DiffLine[] = [];

  for (const change of changes) {
    const changeLines = change.value.replace(/\n$/, '').split('\n');
    const type: DiffLine['type'] = change.added ? '+' : change.removed ? '-' : ' ';

    // For context lines, only include 2 lines around changes
    if (type === ' ' && changeLines.length > 4) {
      // Show first 2 and last 2 context lines
      lines.push({ type: ' ', content: changeLines[0] });
      lines.push({ type: ' ', content: changeLines[1] });
      lines.push({ type: ' ', content: '...' });
      lines.push({ type: ' ', content: changeLines[changeLines.length - 2] });
      lines.push({ type: ' ', content: changeLines[changeLines.length - 1] });
      continue;
    }

    for (const line of changeLines) {
      lines.push({ type, content: line });
    }
  }

  // Truncate
  if (lines.length > maxLines) {
    const remaining = lines.length - maxLines;
    const truncated = lines.slice(0, maxLines);
    truncated.push({ type: ' ', content: `... ${remaining} more lines` });
    return truncated;
  }

  return lines;
}
