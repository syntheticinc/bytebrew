import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import MCPPage from './MCPPage';

vi.mock('../api/client', () => ({
  api: {
    listMCPServers: vi.fn(),
    getWellKnownMCP: vi.fn(),
    listCircuitBreakers: vi.fn().mockResolvedValue([]),
    createMCPServer: vi.fn(),
    updateMCPServer: vi.fn(),
    deleteMCPServer: vi.fn(),
  },
}));

import { api } from '../api/client';
const mockApi = vi.mocked(api);

const auth: AuthContextType = {
  isAuthenticated: true,
  login: vi.fn(),
  logout: vi.fn(),
};

function renderPage() {
  return render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter>
        <MCPPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('MCPPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders MCP servers list', async () => {
    mockApi.listMCPServers.mockResolvedValue([
      {
        id: '1',
        name: 'playwright',
        type: 'stdio' as const,
        command: 'npx',
        args: ['@anthropic/playwright-mcp'],
        is_well_known: true,
        agents: ['e2e-test'],
        status: { status: 'connected' as const, tools_count: 12, connected_at: '2026-03-17T10:00:00Z' },
      },
    ]);
    mockApi.getWellKnownMCP.mockResolvedValue([]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('playwright')).toBeInTheDocument();
      expect(screen.getByText('connected')).toBeInTheDocument();
      expect(screen.getByText('12')).toBeInTheDocument();
    });
  });

  it('shows empty state', async () => {
    mockApi.listMCPServers.mockResolvedValue([]);
    mockApi.getWellKnownMCP.mockResolvedValue([]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No MCP servers configured')).toBeInTheDocument();
    });
  });
});
