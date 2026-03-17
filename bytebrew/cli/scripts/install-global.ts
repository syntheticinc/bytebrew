#!/usr/bin/env bun
/**
 * Installs bytebrew CLI globally by creating a wrapper script in ~/.bytebrew/bin/.
 * The wrapper invokes `bun <cli-dist>/index.js` so native modules (usearch, grpc) work.
 *
 * Usage: bun run install:global
 * Then add ~/.bytebrew/bin to your PATH (the script prints instructions).
 */
import fs from 'fs';
import path from 'path';
import os from 'os';

const CLI_ROOT = path.resolve(import.meta.dir, '..');
const DIST_ENTRY = path.join(CLI_ROOT, 'dist', 'index.js');
const BIN_DIR = path.join(os.homedir(), '.bytebrew', 'bin');

// Ensure dist/index.js exists
if (!fs.existsSync(DIST_ENTRY)) {
  console.error(`ERROR: ${DIST_ENTRY} not found. Run "bun run build" first.`);
  process.exit(1);
}

// Create bin directory
fs.mkdirSync(BIN_DIR, { recursive: true });

// Windows: create .cmd wrapper
const cmdPath = path.join(BIN_DIR, 'bytebrew.cmd');
const cmdContent = `@echo off\r\nbun "${DIST_ENTRY}" %*\r\n`;
fs.writeFileSync(cmdPath, cmdContent);

// Also create a bash wrapper (for Git Bash / WSL)
const shPath = path.join(BIN_DIR, 'bytebrew');
const shContent = `#!/usr/bin/env bash\nexec bun "${DIST_ENTRY.replace(/\\/g, '/')}" "$@"\n`;
fs.writeFileSync(shPath, shContent, { mode: 0o755 });

console.log(`Installed bytebrew to ${BIN_DIR}`);
console.log('');

// Check if already in PATH
const pathDirs = (process.env.PATH || '').split(path.delimiter);
const inPath = pathDirs.some(d => path.resolve(d) === path.resolve(BIN_DIR));

if (inPath) {
  console.log('PATH already configured. You can run: bytebrew ask "hello"');
} else {
  console.log('Add to PATH (one-time setup):');
  console.log('');
  console.log('  PowerShell (permanent, user-level):');
  console.log(`  [Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";${BIN_DIR}", "User")`);
  console.log('');
  console.log('  Git Bash (~/.bashrc):');
  console.log(`  export PATH="$PATH:${BIN_DIR.replace(/\\/g, '/')}"`);
  console.log('');
  console.log('After adding to PATH, restart your terminal and run:');
  console.log('  bytebrew ask "hello"');
}
