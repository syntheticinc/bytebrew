import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import APIKeysPage from './APIKeysPage';

vi.mock('../api/client', () => ({
  api: {
    listTokens: vi.fn(),
    createToken: vi.fn(),
    deleteToken: vi.fn(),
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
        <APIKeysPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('APIKeysPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders tokens table', async () => {
    mockApi.listTokens.mockResolvedValue([
      {
        id: 1,
        name: 'ci-pipeline',
        scopes_mask: 3,
        created_at: '2026-03-17T10:00:00Z',
        last_used_at: undefined,
      },
    ]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('ci-pipeline')).toBeInTheDocument();
      expect(screen.getByText('Chat, Tasks')).toBeInTheDocument();
    });
  });

  it('shows generate button', async () => {
    mockApi.listTokens.mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Generate New Token')).toBeInTheDocument();
    });
  });

  it('opens create modal on button click', async () => {
    mockApi.listTokens.mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      fireEvent.click(screen.getByText('Generate New Token'));
    });

    expect(screen.getByText('Token Name')).toBeInTheDocument();
    expect(screen.getByText('Scopes')).toBeInTheDocument();
  });
});
