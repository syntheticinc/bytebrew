import { useState, useMemo } from 'react';

interface ConfirmationCardProps {
  toolName: string;
  args: Record<string, unknown>;
  prompt: string;
  onRespond: (answer: 'yes' | 'no') => void;
}

export function ConfirmationCard({ toolName, args, prompt, onRespond }: ConfirmationCardProps) {
  const [responded, setResponded] = useState<'yes' | 'no' | null>(null);

  const formattedArgs = useMemo(() => {
    try {
      return JSON.stringify(args, null, 2);
    } catch {
      return String(args);
    }
  }, [args]);

  const handleRespond = (answer: 'yes' | 'no') => {
    if (responded) return;
    setResponded(answer);
    onRespond(answer);
  };

  return (
    <div className="rounded-xl border border-yellow-500/30 bg-yellow-500/5 px-4 py-3 text-sm">
      <div className="flex items-center gap-2 mb-2">
        <span className="text-xs">&#x26A0;&#xFE0F;</span>
        <span className="font-medium text-yellow-400">Confirmation Required</span>
        <span className="text-brand-accent font-medium">{toolName}</span>
      </div>

      {prompt && (
        <p className="text-brand-light mb-3">{prompt}</p>
      )}

      {Object.keys(args).length > 0 && (
        <pre className="mb-3 rounded-lg border border-brand-shade3/10 bg-brand-dark p-3 text-xs text-brand-shade2 max-h-40 overflow-auto whitespace-pre-wrap break-words">
          {formattedArgs}
        </pre>
      )}

      <div className="flex gap-2">
        <button
          onClick={() => handleRespond('yes')}
          disabled={responded !== null}
          className={`rounded-btn px-4 py-1.5 text-xs font-medium transition-all ${
            responded === 'yes'
              ? 'bg-green-600/30 text-green-400 border border-green-500/30 cursor-default'
              : responded !== null
                ? 'bg-brand-shade3/10 text-brand-shade3 cursor-not-allowed opacity-40'
                : 'bg-green-600/20 text-green-400 border border-green-500/30 hover:bg-green-600/30'
          }`}
        >
          {responded === 'yes' ? 'Approved' : 'Approve'}
        </button>
        <button
          onClick={() => handleRespond('no')}
          disabled={responded !== null}
          className={`rounded-btn px-4 py-1.5 text-xs font-medium transition-all ${
            responded === 'no'
              ? 'bg-brand-shade3/20 text-brand-shade3 border border-brand-shade3/30 cursor-default'
              : responded !== null
                ? 'bg-brand-shade3/10 text-brand-shade3 cursor-not-allowed opacity-40'
                : 'bg-brand-shade3/10 text-brand-shade3 border border-brand-shade3/20 hover:bg-brand-shade3/20 hover:text-brand-light'
          }`}
        >
          {responded === 'no' ? 'Rejected' : 'Reject'}
        </button>
      </div>
    </div>
  );
}
