import { NavLink } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

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
];

export default function Sidebar() {
  const { logout } = useAuth();

  return (
    <aside className="w-60 bg-brand-dark text-brand-light flex flex-col min-h-screen">
      <div className="px-4 py-5 border-b border-brand-shade3/15">
        <img src="/logo-dark.svg" alt="ByteBrew" className="h-6" />
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
