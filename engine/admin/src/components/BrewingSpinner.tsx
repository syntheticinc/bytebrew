const BREW_PHRASES = ['Grinding beans...', 'Brewing...', 'Pulling a shot...', 'Steaming...', 'Almost ready...'];
let brewCounter = 0;

export default function BrewingSpinner() {
  const phrase = BREW_PHRASES[brewCounter++ % BREW_PHRASES.length];
  return (
    <div className="flex items-center gap-2 py-1">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="w-3.5 h-3.5 text-brand-shade3">
        <path d="M17 8h1a4 4 0 010 8h-1" strokeLinecap="round" />
        <path d="M3 8h14v9a4 4 0 01-4 4H7a4 4 0 01-4-4V8z" />
        <path d="M7 2v3" strokeLinecap="round" style={{ animation: 'brewSteam 1.2s ease-in-out infinite' }} />
        <path d="M10 1v3" strokeLinecap="round" style={{ animation: 'brewSteam 1.2s ease-in-out infinite 0.3s' }} />
        <path d="M13 2v3" strokeLinecap="round" style={{ animation: 'brewSteam 1.2s ease-in-out infinite 0.6s' }} />
      </svg>
      <span className="text-xs font-mono text-brand-shade3 animate-pulse">{phrase}</span>
      <style>{`@keyframes brewSteam{0%{opacity:.3;transform:translateY(0)}50%{opacity:1;transform:translateY(-3px)}100%{opacity:.3;transform:translateY(0)}}`}</style>
    </div>
  );
}
