export function ThinkingIndicator() {
  return (
    <div className="flex items-center gap-2 px-4 py-2 text-sm text-brand-shade3 italic">
      <span>Agent is thinking</span>
      <span className="flex gap-0.5">
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-brand-shade3 animate-bounce-dot-1" />
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-brand-shade3 animate-bounce-dot-2" />
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-brand-shade3 animate-bounce-dot-3" />
      </span>
    </div>
  );
}
