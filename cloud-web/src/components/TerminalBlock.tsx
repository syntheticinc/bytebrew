import { useState, type ReactNode } from 'react';

interface TerminalBlockProps {
  command: string;
  prefix?: string;
  title?: string;
  /** Additional output lines rendered below the command */
  children?: ReactNode;
}

export function TerminalBlock({ command, prefix = '$', title, children }: TerminalBlockProps) {
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
      <div className="relative rounded-[2px] p-5 overflow-x-auto border" style={{ backgroundColor: '#252525', borderColor: 'rgba(135,134,127,0.15)' }}>
        {/* Window dots — muted, matching HeroDemo style */}
        <div className="flex gap-1.5 mb-4">
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
        </div>

        <div className="flex items-start gap-2">
          <span className="font-mono text-sm select-none shrink-0" style={{ color: '#87867F' }}>
            {prefix}
          </span>
          <code className="font-mono text-sm break-all" style={{ color: '#4ade80' }}>
            {command}
          </code>
        </div>

        {children && (
          <div className="mt-3 font-mono text-sm leading-relaxed" style={{ color: '#DFD8D0' }}>
            {children}
          </div>
        )}

        {/* Copy button */}
        <button
          onClick={handleCopy}
          className="absolute top-4 right-4 rounded-[2px] px-3 py-1 text-xs transition-colors"
          style={{ color: '#DFD8D0', borderWidth: 1, borderColor: 'rgba(135,134,127,0.2)' }}
        >
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  );
}
