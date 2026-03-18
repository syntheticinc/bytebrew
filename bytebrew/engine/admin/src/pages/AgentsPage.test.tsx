import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import AgentsPage from './AgentsPage';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../api/client', () => ({
  api: {
    listAgents: vi.fn(),
    deleteAgent: vi.fn(),
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
        <AgentsPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('AgentsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state', () => {
    mockApi.listAgents.mockReturnValue(new Promise(() => {})); // never resolves
    renderPage();
    expect(screen.getByText('Loading agents...')).toBeInTheDocument();
  });

  it('renders agents table', async () => {
    mockApi.listAgents.mockResolvedValue([
      { name: 'developer', tools_count: 5, kit: 'developer', has_knowledge: false },
      { name: 'sales', tools_count: 2, kit: '', has_knowledge: true },
    ]);

    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText('developer').length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText('sales')).toBeInTheDocument();
    });
  });

  it('navigates to create page', async () => {
    mockApi.listAgents.mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Create Agent')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Agent'));
    expect(mockNavigate).toHaveBeenCalledWith('/agents/new');
  });

  it('shows error state', async () => {
    mockApi.listAgents.mockRejectedValue(new Error('network error'));
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Error: network error')).toBeInTheDocument();
    });
  });
});
