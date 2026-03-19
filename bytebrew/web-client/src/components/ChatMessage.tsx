import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { ChatMessage as ChatMessageType } from '../types';
import { ThinkingIndicator } from './ThinkingIndicator';
import { ToolCallCard } from './ToolCallCard';

interface ChatMessageProps {
  message: ChatMessageType;
}

export function ChatMessage({ message }: ChatMessageProps) {
  switch (message.role) {
    case 'user':
      return (
        <div className="flex justify-end animate-fade-in">
          <div className="max-w-[80%] rounded-2xl rounded-br-sm bg-brand-accent px-4 py-2 text-sm text-white">
            <p className="whitespace-pre-wrap break-words">{message.content}</p>
          </div>
        </div>
      );

    case 'assistant':
      return (
        <div className="flex justify-start animate-fade-in">
          <div className="max-w-[90%] min-w-0 rounded-2xl rounded-bl-sm bg-brand-dark-alt px-4 py-2 text-sm text-brand-light">
            <div className="markdown-body">
              <Markdown
                remarkPlugins={[remarkGfm]}
                components={{
                  table: ({ children }) => (
                    <div className="overflow-x-auto mb-2">
                      <table>{children}</table>
                    </div>
                  ),
                }}
              >
                {message.content}
              </Markdown>
            </div>
          </div>
        </div>
      );

    case 'thinking':
      return (
        <div className="flex justify-start animate-fade-in">
          <ThinkingIndicator />
        </div>
      );

    case 'tool_call':
      return (
        <div className="max-w-[90%] animate-fade-in">
          <ToolCallCard
            toolName={message.toolName ?? 'unknown'}
            content={message.content}
            variant="call"
          />
        </div>
      );

    case 'tool_result':
      return (
        <div className="max-w-[90%] animate-fade-in">
          <ToolCallCard
            toolName={message.toolName ?? 'unknown'}
            content={message.content}
            variant="result"
          />
        </div>
      );

    case 'error':
      return (
        <div className="animate-fade-in">
          <div className="rounded-xl border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-400 break-words">
            <span className="font-medium">Error:</span> {message.content}
          </div>
        </div>
      );
  }
}
