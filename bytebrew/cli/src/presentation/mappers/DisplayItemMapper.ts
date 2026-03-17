// DisplayItemMapper - maps messages to display items for ChatView
// Follows DIP: depends on IToolRenderingService interface

import { MessageViewModel } from './MessageViewMapper.js';
import { IToolRenderingService } from '../../domain/ports/IToolRenderingService.js';
import { getToolPrefix } from '../components/tools/formatToolDisplay.js';

/**
 * Represents either a single message, a group of tool messages, or a custom tool
 */
export type DisplayItem =
  | { type: 'message'; message: MessageViewModel; key: string }
  | { type: 'toolGroup'; messages: MessageViewModel[]; key: string }
  | { type: 'customTool'; message: MessageViewModel; toolName: string; key: string };

/**
 * Maps messages to display items, grouping standard tools and separating custom tools
 *
 * @param messages - Array of message view models
 * @param renderingService - Service for tool rendering decisions (optional for standard tools only)
 * @returns Array of display items ready for rendering
 */
export function mapMessagesToDisplayItems(
  messages: MessageViewModel[],
  renderingService?: IToolRenderingService
): DisplayItem[] {
  const items: DisplayItem[] = [];
  let currentToolGroup: MessageViewModel[] = [];
  let currentToolPrefix: string | null = null;

  const flushToolGroup = () => {
    if (currentToolGroup.length > 0) {
      items.push({
        type: 'toolGroup',
        messages: [...currentToolGroup],
        key: `toolgroup-${currentToolGroup[0].id}`,
      });
      currentToolGroup = [];
      currentToolPrefix = null;
    }
  };

  for (const msg of messages) {
    if (msg.role === 'tool' && msg.toolCall) {
      const toolName = msg.toolCall.toolName;

      // Check if tool should be rendered separately (custom renderer)
      if (renderingService?.shouldRenderSeparately(toolName)) {
        flushToolGroup();
        items.push({
          type: 'customTool',
          message: msg,
          toolName,
          key: `custom-${msg.id}`,
        });
        continue;
      }

      const prefix = getToolPrefix(toolName);

      if (currentToolPrefix === null || currentToolPrefix === prefix) {
        currentToolPrefix = prefix;
        currentToolGroup.push(msg);
      } else {
        flushToolGroup();
        currentToolPrefix = prefix;
        currentToolGroup.push(msg);
      }
    } else {
      flushToolGroup();
      items.push({
        type: 'message',
        message: msg,
        key: msg.id,
      });
    }
  }

  flushToolGroup();
  return items;
}
