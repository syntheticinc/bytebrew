import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import HealthPage from './HealthPage';

vi.mock('../api/client', () => ({
  api: {
    health: vi.fn(),
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
        <HealthPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('HealthPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders health information', async () => {
    mockApi.health.mockResolvedValue({
      status: 'ok',
      version: '0.1.0',
      uptime: '2h30m',
      agents_count: 3,
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('ok')).toBeInTheDocument();
      expect(screen.getByText('0.1.0')).toBeInTheDocument();
      expect(screen.getByText('2h30m')).toBeInTheDocument();
      expect(screen.getByText('3')).toBeInTheDocument();
    });
  });

  it('shows error state', async () => {
    mockApi.health.mockRejectedValue(new Error('connection refused'));
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Error: connection refused')).toBeInTheDocument();
    });
  });
});
