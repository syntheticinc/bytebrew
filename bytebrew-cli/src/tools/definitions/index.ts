// Tool definitions - единая точка регистрации всех инструментов
// Здесь определяется что tool делает (execution) и как отображается (rendering)

export { planToolDefinition } from './planTool.js';

// Re-export types
export type { ToolDefinition, ToolRenderer, ToolRendererProps } from '../ToolManager.js';
