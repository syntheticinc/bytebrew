import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import Sidebar from './Sidebar';

vi.mock('../api/client', () => ({
  api: {
    health: vi.fn(),
    isAuthenticated: vi.fn(() => true),
  },
}));

import { api } from '../api/client';
const mockApi = vi.mocked(api);

const auth: AuthContextType = {
  isAuthenticated: true,
  login: vi.fn(),
  logout: vi.fn(),
};

function renderSidebar() {
  return render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('Sidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders navigation links', async () => {
    mockApi.health.mockResolvedValue({
      status: 'ok',
      version: '1.0.0',
      uptime: '1h',
      agents_count: 0,
    });

    renderSidebar();

    expect(screen.getByText('Health')).toBeInTheDocument();
    expect(screen.getByText('Canvas')).toBeInTheDocument();
    expect(screen.getByText('MCP Servers')).toBeInTheDocument();
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('shows update banner when update is available', async () => {
    mockApi.health.mockResolvedValue({
      status: 'ok',
      version: '1.0.0',
      uptime: '1h',
      agents_count: 0,
      update_available: '1.0.1',
    });

    renderSidebar();

    await waitFor(() => {
      expect(screen.getByText('v1.0.1 available')).toBeInTheDocument();
    });
  });

  it('does not show update banner when no update available', async () => {
    mockApi.health.mockResolvedValue({
      status: 'ok',
      version: '1.0.0',
      uptime: '1h',
      agents_count: 0,
    });

    renderSidebar();

    await waitFor(() => {
      expect(mockApi.health).toHaveBeenCalled();
    });

    expect(screen.queryByText(/available/)).not.toBeInTheDocument();
  });
});
