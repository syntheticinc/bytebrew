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
    <aside className="w-60 bg-gray-900 text-gray-100 flex flex-col min-h-screen">
      <div className="px-4 py-5 border-b border-gray-700">
        <h1 className="text-lg font-bold tracking-wide">ByteBrew Admin</h1>
      </div>

      <nav className="flex-1 py-4 space-y-1 px-2">
        {navigation.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-gray-700 text-white'
                  : 'text-gray-300 hover:bg-gray-800 hover:text-white'
              }`
            }
          >
            <span className="w-6 h-6 flex items-center justify-center bg-gray-700 rounded text-xs font-bold">
              {item.icon}
            </span>
            {item.label}
          </NavLink>
        ))}
      </nav>

      <div className="px-4 py-4 border-t border-gray-700">
        <button
          onClick={logout}
          className="w-full px-3 py-2 text-sm text-gray-300 hover:text-white hover:bg-gray-800 rounded-md transition-colors text-left"
        >
          Logout
        </button>
      </div>
    </aside>
  );
}
