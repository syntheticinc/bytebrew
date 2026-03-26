import { useState } from 'react';

interface TerminalBlockProps {
  command: string;
  prefix?: string;
  title?: string;
}

export function TerminalBlock({ command, prefix = '$', title }: TerminalBlockProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div>
      {title && (
        <p className="text-sm text-text-secondary mb-2 font-medium">{title}</p>
      )}
      <div className="relative bg-surface border border-border rounded-[2px] p-5 overflow-x-auto">
        {/* Window dots */}
        <div className="flex gap-1.5 mb-4">
          <span className="w-3 h-3 rounded-full bg-red-500/80" />
          <span className="w-3 h-3 rounded-full bg-yellow-500/80" />
          <span className="w-3 h-3 rounded-full bg-green-500/80" />
        </div>

        <div className="flex items-start gap-2">
          <span className="text-text-tertiary font-mono text-sm select-none shrink-0">
            {prefix}
          </span>
          <code className="font-mono text-sm text-green-400 break-all">
            {command}
          </code>
        </div>

        {/* Copy button */}
        <button
          onClick={handleCopy}
          className="absolute top-4 right-4 rounded-[2px] border border-border px-3 py-1 text-xs text-text-secondary hover:text-text-primary hover:border-border-hover transition-colors"
        >
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  );
}
