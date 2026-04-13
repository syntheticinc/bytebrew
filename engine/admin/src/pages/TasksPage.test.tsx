import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import TasksPage from './TasksPage';
import type { TaskDetailResponse, TaskResponse } from '../types';

vi.mock('../api/client', () => ({
  api: {
    listAgents: vi.fn(),
    listTasks: vi.fn(),
    listTasksPaginated: vi.fn(),
    getTask: vi.fn(),
    createTask: vi.fn(),
    cancelTask: vi.fn(),
    listSubtasks: vi.fn(),
    approveTask: vi.fn(),
    startTask: vi.fn(),
    completeTask: vi.fn(),
    failTask: vi.fn(),
    setTaskPriority: vi.fn(),
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

// Shared helpers.

function taskRow(overrides: Partial<TaskResponse> = {}): TaskResponse {
  return {
    id: '1',
    title: 'Deploy API',
    agent_name: 'developer',
    status: 'completed',
    source: 'api',
    priority: 0,
    created_at: '2026-03-17T10:00:00Z',
    ...overrides,
  };
}

function taskDetail(overrides: Partial<TaskDetailResponse> = {}): TaskDetailResponse {
  return {
    ...taskRow(),
    mode: 'interactive',
    description: 'A test task',
    acceptance_criteria: ['it works', 'tests pass'],
    blocked_by: [],
    ...overrides,
  };
}

describe('TasksPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // listAgents called on mount.
    mockApi.listAgents.mockResolvedValue([{ name: 'developer', tools_count: 3, has_knowledge: false }]);
    // subtasks is fetched whenever a task is selected — default to empty.
    mockApi.listSubtasks.mockResolvedValue([]);
  });

  afterEach(() => {
    // Prevent the auto-refresh interval from keeping timers alive between tests.
    vi.clearAllTimers();
  });

  it('renders tasks table with priority badges', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [
        taskRow({ id: '1', title: 'Deploy API', status: 'completed', priority: 0 }),
        taskRow({ id: '2', title: 'Run tests', status: 'in_progress', priority: 1, source: 'dashboard' }),
        taskRow({ id: '3', title: 'Fix bug', status: 'failed', priority: 2 }),
      ],
      total: 3, page: 1, per_page: 20, total_pages: 1,
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Deploy API')).toBeInTheDocument();
      expect(screen.getByText('Run tests')).toBeInTheDocument();
      expect(screen.getByText('Fix bug')).toBeInTheDocument();
    });
    // Priority badges visible.
    expect(screen.getByText('Normal')).toBeInTheDocument();
    expect(screen.getByText('High')).toBeInTheDocument();
    expect(screen.getByText('Critical')).toBeInTheDocument();
  });

  it('shows empty state', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({ data: [], total: 0, page: 1, per_page: 20, total_pages: 0 });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No tasks found.')).toBeInTheDocument();
    });
  });

  it('shows Approve/Start/Complete/Fail/Cancel buttons based on status', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: 'draft-1', status: 'draft' })],
      total: 1, page: 1, per_page: 20, total_pages: 1,
    });
    mockApi.getTask.mockResolvedValueOnce(taskDetail({ id: 'draft-1', status: 'draft' }));

    renderPage();

    await screen.findByText('Deploy API');
    fireEvent.click(screen.getByText('Deploy API'));

    // Draft → only Approve + Cancel buttons.
    await screen.findByRole('button', { name: 'Approve' });
    expect(screen.getByRole('button', { name: 'Approve' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Start' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Complete' })).not.toBeInTheDocument();
  });

  it('approves a task and refetches detail', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: 'd1', status: 'draft' })],
      total: 1, page: 1, per_page: 20, total_pages: 1,
    });
    mockApi.getTask
      .mockResolvedValueOnce(taskDetail({ id: 'd1', status: 'draft' }))
      .mockResolvedValue(taskDetail({ id: 'd1', status: 'approved' }));
    mockApi.approveTask.mockResolvedValue(undefined);

    renderPage();
    await screen.findByText('Deploy API');
    fireEvent.click(screen.getByText('Deploy API'));
    const approve = await screen.findByRole('button', { name: 'Approve' });
    fireEvent.click(approve);

    await waitFor(() => {
      expect(mockApi.approveTask).toHaveBeenCalledWith('d1');
    });
  });

  it('surfaces action errors in a banner', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: 'd1', status: 'draft' })],
      total: 1, page: 1, per_page: 20, total_pages: 1,
    });
    mockApi.getTask.mockResolvedValue(taskDetail({ id: 'd1', status: 'draft' }));
    mockApi.approveTask.mockRejectedValueOnce(new Error('server exploded'));

    renderPage();
    await screen.findByText('Deploy API');
    fireEvent.click(screen.getByText('Deploy API'));
    const approve = await screen.findByRole('button', { name: 'Approve' });
    fireEvent.click(approve);

    // Banner with the error message appears.
    await screen.findByText(/server exploded/);
  });

  it('opens confirm dialog before cancelling', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: 'p1', status: 'in_progress' })],
      total: 1, page: 1, per_page: 20, total_pages: 1,
    });
    mockApi.getTask.mockResolvedValue(taskDetail({ id: 'p1', status: 'in_progress' }));
    mockApi.listSubtasks.mockResolvedValue([taskRow({ id: 'c1', status: 'pending' })]);

    renderPage();
    await screen.findByText('Deploy API');
    fireEvent.click(screen.getByText('Deploy API'));

    const cancelBtn = await screen.findByRole('button', { name: 'Cancel' });
    fireEvent.click(cancelBtn);

    // Confirm dialog appears with cascade warning (1 non-terminal child).
    await screen.findByText('Cancel task?');
    await screen.findByText(/1 non-terminal child task/);
    // Confirm proper.
    expect(mockApi.cancelTask).not.toHaveBeenCalled();
  });

  it('opens the Create Task form with agent options', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({ data: [], total: 0, page: 1, per_page: 20, total_pages: 0 });
    renderPage();

    await screen.findByRole('button', { name: '+ New task' });
    fireEvent.click(screen.getByRole('button', { name: '+ New task' }));

    // Modal heading appears.
    await screen.findByRole('heading', { name: 'Create task' });
    // Agent select is pre-filled with the listed agent.
    const user = userEvent.setup();
    const titleInput = screen.getByPlaceholderText('Short descriptive title');
    await user.type(titleInput, 'My new task');

    // Submit.
    mockApi.createTask.mockResolvedValue({ task_id: 'new-1', status: 'pending' });
    fireEvent.click(screen.getByRole('button', { name: 'Create task' }));

    await waitFor(() => {
      expect(mockApi.createTask).toHaveBeenCalledWith(expect.objectContaining({
        title: 'My new task',
        agent_name: 'developer',
      }));
    });
  });

  it('blocks Create submit when title is empty', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({ data: [], total: 0, page: 1, per_page: 20, total_pages: 0 });
    renderPage();

    await screen.findByRole('button', { name: '+ New task' });
    fireEvent.click(screen.getByRole('button', { name: '+ New task' }));

    await screen.findByRole('heading', { name: 'Create task' });
    // Submit without title.
    fireEvent.click(screen.getByRole('button', { name: 'Create task' }));

    await screen.findByText('Title is required.');
    expect(mockApi.createTask).not.toHaveBeenCalled();
  });

  it('renders pagination when totalPages > 1', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: '1' })],
      total: 42, page: 1, per_page: 20, total_pages: 3,
    });
    renderPage();

    await screen.findByText('Deploy API');
    expect(screen.getByText(/Showing 1–20 of 42 tasks/)).toBeInTheDocument();
    // Page buttons 1, 2, 3 + prev/next.
    expect(screen.getByRole('button', { name: '1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '2' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '3' })).toBeInTheDocument();
  });

  it('displays acceptance criteria and blocked_by chips in detail panel', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: 'p1', status: 'pending' })],
      total: 1, page: 1, per_page: 20, total_pages: 1,
    });
    mockApi.getTask.mockResolvedValue(taskDetail({
      id: 'p1',
      status: 'pending',
      acceptance_criteria: ['A', 'B'],
      blocked_by: ['blocker-1'],
    }));

    renderPage();
    await screen.findByText('Deploy API');
    fireEvent.click(screen.getByText('Deploy API'));

    await screen.findByText('Acceptance criteria');
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
    expect(screen.getByText('Blocked by')).toBeInTheDocument();
    // Blocker chip shows truncated id with full id in title attribute.
    expect(screen.getByTitle('blocker-1')).toBeInTheDocument();
  });

  it('renders inline subtasks list when present', async () => {
    mockApi.listTasksPaginated.mockResolvedValue({
      data: [taskRow({ id: 'p1', status: 'pending' })],
      total: 1, page: 1, per_page: 20, total_pages: 1,
    });
    mockApi.getTask.mockResolvedValue(taskDetail({ id: 'p1', status: 'pending' }));
    mockApi.listSubtasks.mockResolvedValue([
      taskRow({ id: 'sub-1', title: 'Sub one', status: 'pending', priority: 2 }),
    ]);

    renderPage();
    await screen.findByText('Deploy API');
    fireEvent.click(screen.getByText('Deploy API'));

    await screen.findByText(/Sub one/);
    // The inline subtask button shows both Critical priority and pending status.
    const subtaskBtn = screen.getByText(/Sub one/).closest('button');
    expect(subtaskBtn).not.toBeNull();
    expect(subtaskBtn!.textContent).toMatch(/Critical/);
    expect(subtaskBtn!.textContent).toMatch(/pending/);
  });
});
