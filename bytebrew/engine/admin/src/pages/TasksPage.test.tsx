import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import TasksPage from './TasksPage';

vi.mock('../api/client', () => ({
  api: {
    listTasks: vi.fn(),
    listTasksPaginated: vi.fn(),
    getTask: vi.fn(),
    cancelTask: vi.fn(),
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
        <TasksPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('TasksPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders tasks table', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [
        { id: 1, title: 'Deploy API', agent_name: 'developer', status: 'completed', source: 'api', created_at: '2026-03-17T10:00:00Z' },
        { id: 2, title: 'Run tests', agent_name: 'developer', status: 'in_progress', source: 'dashboard', created_at: '2026-03-17T11:00:00Z' },
      ],
      total: 2, page: 1, per_page: 20, total_pages: 1,
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Deploy API')).toBeInTheDocument();
      expect(screen.getByText('Run tests')).toBeInTheDocument();
      expect(screen.getAllByText('completed').length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText('in progress').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('shows empty state', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({ data: [], total: 0, page: 1, per_page: 20, total_pages: 0 });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No tasks found.')).toBeInTheDocument();
    });
  });
});
