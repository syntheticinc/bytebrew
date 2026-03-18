import { useState, useRef, useCallback, type KeyboardEvent } from 'react';

interface ChatInputProps {
  onSend: (text: string) => void;
  onStop: () => void;
  streaming: boolean;
  disabled?: boolean;
}

export function ChatInput({ onSend, onStop, streaming, disabled }: ChatInputProps) {
  const [value, setValue] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSend = useCallback(() => {
    const trimmed = value.trim();
    if (!trimmed) return;
    onSend(trimmed);
    setValue('');
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }
  }, [value, onSend]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        if (!streaming) handleSend();
      }
    },
    [streaming, handleSend],
  );

  const handleInput = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = Math.min(el.scrollHeight, 200) + 'px';
  }, []);

  return (
    <div className="flex items-end gap-3">
      <textarea
        ref={textareaRef}
        className="chat-input flex-1"
        placeholder="Type a message..."
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        onInput={handleInput}
        rows={1}
        disabled={disabled}
      />
      {streaming ? (
        <button
          className="flex-shrink-0 rounded-btn bg-brand-shade3/30 px-4 py-3 text-sm font-medium text-brand-light transition-colors hover:bg-brand-shade3/50"
          onClick={onStop}
        >
          Stop
        </button>
      ) : (
        <button
          className="flex-shrink-0 rounded-btn bg-brand-accent px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-brand-accent-hover disabled:opacity-40 disabled:cursor-not-allowed"
          onClick={handleSend}
          disabled={disabled || !value.trim()}
        >
          Send
        </button>
      )}
    </div>
  );
}
