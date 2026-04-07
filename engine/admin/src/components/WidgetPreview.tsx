interface WidgetPreviewProps {
  primaryColor: string;
  position: 'bottom-right' | 'bottom-left';
  welcomeMessage: string;
  placeholderText: string;
  size: 'compact' | 'standard' | 'full';
  avatarUrl?: string;
  name: string;
}

const SIZE_CLASSES = {
  compact: { w: 'w-[260px]', h: 'h-[320px]' },
  standard: { w: 'w-[320px]', h: 'h-[400px]' },
  full: { w: 'w-[380px]', h: 'h-[480px]' },
};

export default function WidgetPreview({
  primaryColor,
  position,
  welcomeMessage,
  placeholderText,
  size,
  avatarUrl,
  name,
}: WidgetPreviewProps) {
  const sizeClass = SIZE_CLASSES[size];
  const align = position === 'bottom-left' ? 'items-start' : 'items-end';

  return (
    <div className={`flex flex-col ${align} gap-3`}>
      <span className="text-[10px] text-brand-shade3 uppercase tracking-wider font-mono">Live Preview</span>

      {/* Chat window preview */}
      <div
        className={`${sizeClass.w} ${sizeClass.h} rounded-xl overflow-hidden border border-brand-shade3/20 flex flex-col bg-[#1a1a2e] shadow-2xl`}
      >
        {/* Header */}
        <div
          className="flex items-center gap-2.5 px-4 py-3 shrink-0"
          style={{ backgroundColor: primaryColor }}
        >
          {avatarUrl ? (
            <img
              src={avatarUrl}
              alt=""
              className="w-8 h-8 rounded-full object-cover border-2 border-white/20"
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = 'none';
              }}
            />
          ) : (
            <div className="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.5">
                <rect x="4" y="4" width="16" height="16" rx="2" />
                <rect x="9" y="9" width="6" height="6" rx="1" />
              </svg>
            </div>
          )}
          <div className="flex-1 min-w-0">
            <span className="text-sm font-semibold text-white block truncate">{name}</span>
            <span className="text-[10px] text-white/60">Online</span>
          </div>
          <button className="text-white/60 hover:text-white p-1">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Messages area */}
        <div className="flex-1 p-3 overflow-hidden">
          {/* Welcome message bubble */}
          <div className="flex items-start gap-2 mb-3">
            <div
              className="w-6 h-6 rounded-full shrink-0 flex items-center justify-center"
              style={{ backgroundColor: primaryColor + '30' }}
            >
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke={primaryColor} strokeWidth="2">
                <rect x="4" y="4" width="16" height="16" rx="2" />
                <rect x="9" y="9" width="6" height="6" rx="1" />
              </svg>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-lg rounded-tl-none px-3 py-2 max-w-[85%]">
              <p className="text-xs text-white/80 leading-relaxed">{welcomeMessage}</p>
            </div>
          </div>
        </div>

        {/* Input area */}
        <div className="px-3 pb-3 shrink-0">
          <div className="flex items-center gap-2 bg-white/5 border border-white/10 rounded-lg px-3 py-2">
            <span className="text-xs text-white/30 flex-1 truncate">{placeholderText}</span>
            <div
              className="w-6 h-6 rounded-full flex items-center justify-center shrink-0"
              style={{ backgroundColor: primaryColor }}
            >
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2">
                <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {/* Bubble button preview */}
      <div className={`flex ${position === 'bottom-left' ? '' : 'justify-end'} w-full`}>
        <div
          className="w-14 h-14 rounded-full shadow-lg flex items-center justify-center cursor-pointer hover:scale-105 transition-transform"
          style={{ backgroundColor: primaryColor }}
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.5">
            <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
          </svg>
        </div>
      </div>
    </div>
  );
}
