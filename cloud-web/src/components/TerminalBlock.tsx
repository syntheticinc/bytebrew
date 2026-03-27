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
      <div className="relative rounded-[2px] p-5 overflow-x-auto border border-border bg-surface-alt">
        {/* Window dots — muted, matching HeroDemo style */}
        <div className="flex gap-1.5 mb-4">
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
        </div>

        <div className="flex items-start gap-2">
          <span className="font-mono text-sm select-none shrink-0 text-text-tertiary">
            {prefix}
          </span>
          <code className="font-mono text-sm break-all text-text-primary">
            {command}
          </code>
        </div>

        {/* Copy button */}
        <button
          onClick={handleCopy}
          className="absolute top-4 right-4 rounded-[2px] border border-border px-3 py-1 text-xs text-text-secondary hover:text-text-primary transition-colors"
        >
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  );
}
