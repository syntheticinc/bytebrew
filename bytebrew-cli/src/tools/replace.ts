// Replace logic adapted from OpenCode (MIT License)
// https://github.com/anthropics/opencode

export type Replacer = (content: string, find: string) => Generator<string, void, unknown>;

/**
 * Exact match — returns the find string as-is.
 */
export const SimpleReplacer: Replacer = function* (_content, find) {
  yield find;
};

/**
 * Matches lines ignoring leading/trailing whitespace on each line.
 * Useful when LLM adds or removes indentation.
 */
export const LineTrimmedReplacer: Replacer = function* (content, find) {
  const originalLines = content.split('\n');
  const searchLines = find.split('\n');

  if (searchLines[searchLines.length - 1] === '') {
    searchLines.pop();
  }

  for (let i = 0; i <= originalLines.length - searchLines.length; i++) {
    let matches = true;

    for (let j = 0; j < searchLines.length; j++) {
      const originalTrimmed = originalLines[i + j].trim();
      const searchTrimmed = searchLines[j].trim();

      if (originalTrimmed !== searchTrimmed) {
        matches = false;
        break;
      }
    }

    if (matches) {
      let matchStartIndex = 0;
      for (let k = 0; k < i; k++) {
        matchStartIndex += originalLines[k].length + 1;
      }

      let matchEndIndex = matchStartIndex;
      for (let k = 0; k < searchLines.length; k++) {
        matchEndIndex += originalLines[i + k].length;
        if (k < searchLines.length - 1) {
          matchEndIndex += 1;
        }
      }

      yield content.substring(matchStartIndex, matchEndIndex);
    }
  }
};

/**
 * Normalizes all whitespace sequences to single space before matching.
 * Handles tabs vs spaces, multiple spaces, etc.
 */
export const WhitespaceNormalizedReplacer: Replacer = function* (content, find) {
  const normalizeWhitespace = (text: string) => text.replace(/\s+/g, ' ').trim();
  const normalizedFind = normalizeWhitespace(find);

  const lines = content.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (normalizeWhitespace(line) === normalizedFind) {
      yield line;
    } else {
      const normalizedLine = normalizeWhitespace(line);
      if (normalizedLine.includes(normalizedFind)) {
        const words = find.trim().split(/\s+/);
        if (words.length > 0) {
          const pattern = words.map((word) => word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('\\s+');
          try {
            const regex = new RegExp(pattern);
            const match = line.match(regex);
            if (match) {
              yield match[0];
            }
          } catch (_e) {
            // Invalid regex, skip
          }
        }
      }
    }
  }

  // Multi-line matches
  const findLines = find.split('\n');
  if (findLines.length > 1) {
    for (let i = 0; i <= lines.length - findLines.length; i++) {
      const block = lines.slice(i, i + findLines.length);
      if (normalizeWhitespace(block.join('\n')) === normalizedFind) {
        yield block.join('\n');
      }
    }
  }
};

/**
 * Strips common indentation before comparing.
 * Handles cases where LLM uses different indent level.
 */
export const IndentationFlexibleReplacer: Replacer = function* (content, find) {
  const removeIndentation = (text: string) => {
    const lines = text.split('\n');
    const nonEmptyLines = lines.filter((line) => line.trim().length > 0);
    if (nonEmptyLines.length === 0) return text;

    const minIndent = Math.min(
      ...nonEmptyLines.map((line) => {
        const match = line.match(/^(\s*)/);
        return match ? match[1].length : 0;
      }),
    );

    return lines.map((line) => (line.trim().length === 0 ? line : line.slice(minIndent))).join('\n');
  };

  const normalizedFind = removeIndentation(find);
  const contentLines = content.split('\n');
  const findLines = find.split('\n');

  for (let i = 0; i <= contentLines.length - findLines.length; i++) {
    const block = contentLines.slice(i, i + findLines.length).join('\n');
    if (removeIndentation(block) === normalizedFind) {
      yield block;
    }
  }
};

/**
 * Yields all exact occurrences for replaceAll support.
 */
export const MultiOccurrenceReplacer: Replacer = function* (content, find) {
  let startIndex = 0;

  while (true) {
    const index = content.indexOf(find, startIndex);
    if (index === -1) break;

    yield find;
    startIndex = index + find.length;
  }
};

/**
 * Finds the closest matching block in content for better error messages.
 * Uses sliding window with trimmed line comparison.
 */
function findClosestMatch(content: string, find: string): string | null {
  const contentLines = content.split('\n');
  const findLines = find.split('\n');

  // Remove trailing empty line from find
  if (findLines.length > 1 && findLines[findLines.length - 1].trim() === '') {
    findLines.pop();
  }

  if (findLines.length === 0 || contentLines.length === 0) return null;

  const windowSize = findLines.length;
  if (windowSize > contentLines.length) return null;

  let bestStartLine = -1;
  let bestMatchCount = 0;

  for (let i = 0; i <= contentLines.length - windowSize; i++) {
    let matchCount = 0;
    for (let j = 0; j < windowSize; j++) {
      if (contentLines[i + j].trim() === findLines[j].trim()) {
        matchCount++;
      }
    }
    if (matchCount > bestMatchCount) {
      bestMatchCount = matchCount;
      bestStartLine = i;
    }
  }

  if (bestStartLine === -1 || bestMatchCount === 0) return null;

  const ratio = bestMatchCount / windowSize;

  // For single-line: only show if there's an exact trimmed match (handled by replacers already)
  if (windowSize === 1) return null;

  // For multi-line: need at least 30% matching lines
  if (ratio < 0.3) return null;

  // Build differing lines list (max 3)
  const diffs: string[] = [];
  for (let j = 0; j < windowSize && diffs.length < 3; j++) {
    const actual = contentLines[bestStartLine + j].trim();
    const expected = findLines[j].trim();
    if (actual !== expected) {
      const lineNum = bestStartLine + j + 1;
      const truncate = (s: string) => (s.length > 80 ? s.substring(0, 80) + '...' : s);
      diffs.push(`  line ${lineNum}: file has "${truncate(actual)}" but old_string has "${truncate(expected)}"`);
    }
  }

  let hint = `Closest match at line ${bestStartLine + 1} (${Math.round(ratio * 100)}% lines match).`;
  if (diffs.length > 0) {
    hint += '\nDiffering lines:\n' + diffs.join('\n');
  }

  return hint;
}

/**
 * Main replace function. Tries replacers in order from strict to fuzzy.
 * For single replacement: requires unique match (exactly one occurrence).
 * For replaceAll: replaces all occurrences found by first successful replacer.
 *
 * Normalizes CRLF → LF before matching and restores original line endings after.
 */
export function replace(content: string, oldString: string, newString: string, replaceAll = false): string {
  if (oldString === newString) {
    throw new Error('oldString and newString must be different');
  }

  // Normalize CRLF to LF for consistent matching (Windows files have \r\n, LLM sends \n)
  const hasCRLF = content.includes('\r\n');
  if (hasCRLF) {
    content = content.replace(/\r\n/g, '\n');
    oldString = oldString.replace(/\r\n/g, '\n');
    newString = newString.replace(/\r\n/g, '\n');
  }

  let notFound = true;

  for (const replacer of [
    SimpleReplacer,
    LineTrimmedReplacer,
    WhitespaceNormalizedReplacer,
    IndentationFlexibleReplacer,
    MultiOccurrenceReplacer,
  ]) {
    for (const search of replacer(content, oldString)) {
      const index = content.indexOf(search);
      if (index === -1) continue;
      notFound = false;
      if (replaceAll) {
        const result = content.replaceAll(search, newString);
        return hasCRLF ? result.replace(/\n/g, '\r\n') : result;
      }
      const lastIndex = content.lastIndexOf(search);
      if (index !== lastIndex) continue;
      const result = content.substring(0, index) + newString + content.substring(index + search.length);
      return hasCRLF ? result.replace(/\n/g, '\r\n') : result;
    }
  }

  if (notFound) {
    const hint = findClosestMatch(content, oldString);
    const msg = hint
      ? `oldString not found in file content. ${hint}`
      : 'oldString not found in file content';
    throw new Error(msg);
  }
  throw new Error(
    'Found multiple matches for oldString. Provide more surrounding lines for context to uniquely identify the match.',
  );
}
