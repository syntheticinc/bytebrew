// Path utilities for consistent path handling across the CLI

import path from 'path';

/**
 * Convert absolute path to relative path for consistent output.
 * Handles cross-platform path normalization.
 */
export function toRelativePath(filePath: string, projectRoot?: string): string {
  if (!projectRoot) {
    return filePath.replace(/\\/g, '/').replace(/^\/+/, '');
  }

  let normalizedPath = filePath.replace(/\\/g, '/');
  const normalizedRoot = projectRoot.replace(/\\/g, '/');

  if (normalizedPath.startsWith(normalizedRoot + '/')) {
    normalizedPath = normalizedPath.slice(normalizedRoot.length + 1);
  } else if (!path.isAbsolute(filePath)) {
    // Already relative
  } else {
    normalizedPath = path.relative(projectRoot, filePath).replace(/\\/g, '/');
  }

  return normalizedPath.replace(/^\/+/, '');
}
