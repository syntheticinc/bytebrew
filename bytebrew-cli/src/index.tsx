#!/usr/bin/env node
// Entry point for Vector CLI

// Global crash handlers — catch silent deaths
process.on('uncaughtException', (err) => {
  process.stderr.write(`\n[FATAL] Uncaught exception: ${err.stack || err.message}\n`);
  process.exit(1);
});
process.on('unhandledRejection', (reason) => {
  const msg = reason instanceof Error ? reason.stack || reason.message : String(reason);
  process.stderr.write(`\n[FATAL] Unhandled rejection: ${msg}\n`);
  process.exit(1);
});

import { program } from './cli.js';

// Run CLI
program.parse(process.argv);
