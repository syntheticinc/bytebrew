// Headless module - runs the app without React/Ink UI
// Re-exports all headless runners and helper functions

import { registerToolDefinitions } from '../tools/definitions/registerTools.js';

// Register tool definitions once on module load
registerToolDefinitions();

// Export base class for extension
export { BaseHeadlessRunner } from './BaseHeadlessRunner.js';
export type { Container, HeadlessRunnerOptions } from './BaseHeadlessRunner.js';

// Export single-question runner
export { HeadlessRunner, runHeadless } from './HeadlessRunner.js';

// Export interactive runner
export { HeadlessInteractiveRunner, runHeadlessInteractive } from './HeadlessInteractiveRunner.js';
