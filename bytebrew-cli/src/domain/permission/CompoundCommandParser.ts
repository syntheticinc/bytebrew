// CompoundCommandParser - splits compound shell commands by &&, ||, ;, & operators
// Respects quoted strings (single and double quotes)

/**
 * Splits compound shell command by &&, ||, ;, & operators.
 * Respects quoted strings (single and double quotes).
 * Returns array of trimmed non-empty subcommands.
 *
 * Handles both bash (&&, ||, ;) and cmd.exe (&) separators.
 *
 * @param command - compound shell command string
 * @returns array of subcommands (trimmed, non-empty)
 *
 * @example
 * splitCompoundCommand('cd foo && go build') → ['cd foo', 'go build']
 * splitCompoundCommand('echo "hello && world"') → ['echo "hello && world"']
 * splitCompoundCommand('a && b || c ; d') → ['a', 'b', 'c', 'd']
 * splitCompoundCommand('cd /d C:\\path & go build') → ['cd /d C:\\path', 'go build']
 */
export function splitCompoundCommand(command: string): string[] {
  const subcommands: string[] = [];
  let current = '';
  let inSingleQuote = false;
  let inDoubleQuote = false;

  for (let i = 0; i < command.length; i++) {
    const char = command[i];
    const next = command[i + 1];

    // Toggle quote state
    if (char === "'" && !inDoubleQuote) {
      inSingleQuote = !inSingleQuote;
      current += char;
      continue;
    }

    if (char === '"' && !inSingleQuote) {
      inDoubleQuote = !inDoubleQuote;
      current += char;
      continue;
    }

    // Check for operators OUTSIDE quotes
    if (!inSingleQuote && !inDoubleQuote) {
      // Check for && or ||
      if ((char === '&' && next === '&') || (char === '|' && next === '|')) {
        // Found operator
        const trimmed = current.trim();
        if (trimmed) {
          subcommands.push(trimmed);
        }
        current = '';
        i++; // Skip next character (second & or |)
        continue;
      }

      // Check for single & (cmd.exe separator / bash background)
      if (char === '&' && next !== '&') {
        const trimmed = current.trim();
        if (trimmed) {
          subcommands.push(trimmed);
        }
        current = '';
        continue;
      }

      // Check for ;
      if (char === ';') {
        const trimmed = current.trim();
        if (trimmed) {
          subcommands.push(trimmed);
        }
        current = '';
        continue;
      }
    }

    // Regular character
    current += char;
  }

  // Add final subcommand
  const trimmed = current.trim();
  if (trimmed) {
    subcommands.push(trimmed);
  }

  return subcommands;
}
