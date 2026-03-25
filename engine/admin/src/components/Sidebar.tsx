import { useEffect, useRef, useState } from 'react';
import { NavLink } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { api } from '../api/client';

interface NavItem {
  to: string;
  label: string;
  icon: string;
}

const navigation: NavItem[] = [
  { to: '/health', label: 'Health', icon: 'H' },
  { to: '/agents', label: 'Agents', icon: 'A' },
  { to: '/mcp', label: 'MCP Servers', icon: 'M' },
  { to: '/models', label: 'Models', icon: 'L' },
  { to: '/triggers', label: 'Triggers', icon: 'T' },
  { to: '/tasks', label: 'Tasks', icon: 'K' },
  { to: '/api-keys', label: 'API Keys', icon: 'K' },
  { to: '/settings', label: 'Settings', icon: 'S' },
  { to: '/config', label: 'Config', icon: 'C' },
  { to: '/audit', label: 'Audit Log', icon: 'L' },
];

export default function Sidebar() {
  const { logout } = useAuth();
  const [updateAvailable, setUpdateAvailable] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  useEffect(() => {
    const checkUpdate = () => {
      api.health()
        .then((h) => setUpdateAvailable(h.update_available ?? null))
        .catch(() => { /* ignore — health page handles errors */ });
    };

    checkUpdate();
    intervalRef.current = setInterval(checkUpdate, 60000);
    return () => clearInterval(intervalRef.current);
  }, []);

  return (
    <aside className="w-60 bg-brand-dark text-brand-light flex flex-col min-h-screen">
      <div className="px-4 py-5 border-b border-brand-shade3/15">
        <img src={import.meta.env.BASE_URL + 'logo-dark.svg'} alt="ByteBrew" className="h-6" />
        <span className="text-xs text-brand-shade3 mt-1 block">Admin Dashboard</span>
      </div>

      <nav className="flex-1 py-4 space-y-1 px-2">
        {navigation.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-btn text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-brand-dark-alt text-white border-l-2 border-brand-accent'
                  : 'text-brand-shade2 hover:bg-brand-dark-alt hover:text-white'
              }`
            }
          >
            <span className="w-6 h-6 flex items-center justify-center bg-brand-dark-alt rounded text-xs font-bold">
              {item.icon}
            </span>
            {item.label}
            {item.to === '/health' && updateAvailable && (
              <span
                className="ml-auto w-2 h-2 rounded-full bg-amber-400"
                title={`Update available: v${updateAvailable}`}
              />
            )}
          </NavLink>
        ))}
      </nav>

      <div className="px-4 py-4 border-t border-brand-shade3/15">
        <button
          onClick={logout}
          className="w-full px-3 py-2 text-sm text-brand-shade2 hover:text-white hover:bg-brand-dark-alt rounded-btn transition-colors text-left"
        >
          Logout
        </button>
      </div>
    </aside>
  );
}
