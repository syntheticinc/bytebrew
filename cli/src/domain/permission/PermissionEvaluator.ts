// PermissionEvaluator - evaluates permission requests against allow/deny lists
// Logic: deny list blocks dangerous commands, everything else is allowed by default.
// Compatible with Claude Code format: Bash(pattern), Read, Edit, Write
import {
  PermissionRequest,
  PermissionEvalResult,
  PermissionConfig,
} from './Permission.js';
import { matchesPattern } from '../wildcard/WildcardMatcher.js';
import { splitCompoundCommand } from './CompoundCommandParser.js';

/**
 * Evaluate a permission request against the config.
 *
 * Logic:
 * 1. Check deny list (first match → DENY)
 * 2. Everything else → ALLOW (allow by default)
 *
 * For bash type with compound commands (&&, ||, ;), evaluates each subcommand
 * and returns most restrictive: deny > allow
 */
export function evaluatePermission(
  request: PermissionRequest,
  config: PermissionConfig
): PermissionEvalResult {
  const { type, value } = request;

  // For bash — check if compound command
  if (type === 'bash') {
    const subcommands = splitCompoundCommand(value);

    // Single command
    if (subcommands.length === 1) {
      return evaluateCommand(value, config);
    }

    // Compound command — deny if ANY subcommand is denied, otherwise allow
    for (const subcommand of subcommands) {
      const result = evaluateCommand(subcommand, config);
      if (result.action === 'deny') {
        return result;
      }
    }

    return { action: 'allow' };
  }

  // For non-bash types (read, edit, list, write) — check simple type
  return evaluateSimpleType(type, config);
}

/**
 * Evaluate a bash command against deny list.
 * Deny list blocks dangerous commands, everything else is allowed.
 */
function evaluateCommand(
  command: string,
  config: PermissionConfig
): PermissionEvalResult {
  // Check deny list — blocked commands
  for (const rule of config.permissions.deny) {
    const pattern = parseRulePattern(rule, 'bash');
    if (pattern && matchesPattern(command, pattern)) {
      return { action: 'deny', matchedPattern: rule };
    }
  }

  // Allow by default
  return { action: 'allow' };
}

/**
 * Evaluate simple types (read, edit, write, list) against deny list.
 * Deny list blocks, everything else is allowed by default.
 */
function evaluateSimpleType(
  type: string,
  config: PermissionConfig
): PermissionEvalResult {
  const normalizedType = type.toLowerCase();
  const ruleNames = getRuleNamesForType(normalizedType);

  // Check deny list
  for (const rule of config.permissions.deny) {
    if (ruleNames.some(name => rule.toLowerCase() === name.toLowerCase())) {
      return { action: 'deny', matchedPattern: rule };
    }
  }

  // Allow by default
  return { action: 'allow' };
}

/**
 * Parse a rule from allow/deny list to extract pattern.
 * Supports formats:
 * - "Bash(pattern)" → extracts pattern for bash matching
 * - "Read", "Edit", "Write", "List" → type names (handled separately)
 *
 * Returns null if rule doesn't match expected format for given type.
 */
function parseRulePattern(rule: string, type: string): string | null {
  if (type === 'bash') {
    // Match "Bash(pattern)"
    const match = rule.match(/^Bash\((.+)\)$/i);
    if (match) {
      return match[1];
    }
    // Also support bare patterns for backward compatibility
    return null;
  }

  return null;
}

/**
 * Get possible rule names for a given type.
 * e.g., "edit" → ["Edit", "Write"]
 */
function getRuleNamesForType(type: string): string[] {
  switch (type) {
    case 'read':
      return ['Read'];
    case 'edit':
      return ['Edit', 'Write'];
    case 'list':
      return ['List'];
    default:
      return [type];
  }
}
