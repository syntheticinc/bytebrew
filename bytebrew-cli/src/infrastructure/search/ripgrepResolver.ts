// Resolve the path to the `rg` (ripgrep) binary.
// Checks system PATH only — auto-install removed (was via lsp/install/).

let cachedRgPath: string | null = null;

/**
 * Resolve the path to the `rg` binary.
 * Returns the absolute path to `rg`, or null if unavailable.
 */
export async function resolveRgBinary(): Promise<string | null> {
  if (cachedRgPath !== null) return cachedRgPath;

  // Check system PATH
  const systemRg = Bun.which('rg');
  if (systemRg) {
    cachedRgPath = systemRg;
    return cachedRgPath;
  }

  return null;
}
