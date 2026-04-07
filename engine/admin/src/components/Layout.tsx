import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import BottomPanel from './BottomPanel';
import QuotaBanner from './QuotaBanner';
import { PrototypeProvider, usePrototype } from '../hooks/usePrototype';
import { BottomPanelProvider } from '../hooks/useBottomPanel';

function ModeToggle() {
  const { isPrototype, togglePrototype, prototypeEnabled } = usePrototype();

  if (!prototypeEnabled) return null;

  return (
    <div className="flex items-center gap-2 px-4 py-1.5 border-b border-brand-shade3/10 bg-brand-dark-surface shrink-0 justify-end">
      <span className="text-[11px] text-brand-shade3 font-mono">
        {isPrototype ? 'Prototype' : 'Production'}
      </span>
      <button
        onClick={togglePrototype}
        role="switch"
        aria-checked={isPrototype}
        aria-label="Toggle prototype mode"
        className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
          isPrototype ? 'bg-purple-500' : 'bg-brand-shade3/40'
        }`}
        title={isPrototype ? 'Switch to Production mode' : 'Switch to Prototype mode'}
      >
        <span
          className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
            isPrototype ? 'translate-x-4' : 'translate-x-0.5'
          }`}
        />
      </button>
    </div>
  );
}

function LayoutInner() {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0">
        <ModeToggle />
        <QuotaBanner />
        <main className="flex-1 bg-brand-dark p-6 overflow-auto animate-fade-in">
          <Outlet />
        </main>
        <BottomPanel />
      </div>
    </div>
  );
}

export default function Layout() {
  return (
    <PrototypeProvider>
      <BottomPanelProvider>
        <LayoutInner />
      </BottomPanelProvider>
    </PrototypeProvider>
  );
}
