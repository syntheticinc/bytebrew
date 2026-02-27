// WildcardMatcher - reusable wildcard pattern matching
// `*` matches any characters (including \ / spaces etc)
// Case-insensitive matching

// Cache compiled regexes for performance
const patternCache = new Map<string, RegExp>();

/**
 * Convert a wildcard pattern to a RegExp.
 * `*` matches any characters (including \ / spaces etc).
 * All other characters are escaped for literal matching.
 * Matching is case-insensitive.
 */
export function patternToRegex(pattern: string): RegExp {
  const cached = patternCache.get(pattern);
  if (cached) {
    return cached;
  }

  // Escape all regex special characters except *, then replace * with .*
  const escaped = pattern.replace(/[.+?^${}()|[\]\\]/g, '\\$&');
  const regexStr = '^' + escaped.replace(/\*/g, '.*') + '$';
  const regex = new RegExp(regexStr, 'i');

  patternCache.set(pattern, regex);
  return regex;
}

/**
 * Check if a string matches a wildcard pattern.
 * `*` matches any sequence of characters (unlike file-glob, matches \ and / too).
 * Case-insensitive.
 */
export function matchesPattern(value: string, pattern: string): boolean {
  const normalizedValue = value.trim();
  const normalizedPattern = pattern.trim();

  if (normalizedValue === normalizedPattern) {
    return true;
  }

  return patternToRegex(normalizedPattern).test(normalizedValue);
}

/**
 * Generate a wildcard pattern from a command string.
 * Used when user approves a command and wants to remember it.
 *
 * Examples:
 * - "make build" -> "make *"
 * - "docker compose up" -> "docker compose *"
 */
export function generatePattern(command: string): string {
  const parts = command.trim().split(/\s+/);

  if (parts.length === 1) {
    return `${parts[0]} *`;
  }

  if (parts.length === 2) {
    return `${parts[0]} *`;
  }

  return `${parts[0]} ${parts[1]} *`;
}
