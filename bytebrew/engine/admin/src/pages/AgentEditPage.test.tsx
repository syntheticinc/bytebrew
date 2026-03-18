import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import AgentEditPage from './AgentEditPage';

vi.mock('../api/client', () => ({
  api: {
    getAgent: vi.fn(),
    createAgent: vi.fn(),
    updateAgent: vi.fn(),
    listModels: vi.fn(),
    listMCPServers: vi.fn(),
    listAgents: vi.fn(),
  },
}));

import { api } from '../api/client';
const mockApi = vi.mocked(api);

const auth: AuthContextType = {
  isAuthenticated: true,
  login: vi.fn(),
  logout: vi.fn(),
};

function renderEditPage(agentName: string) {
  const path = agentName === 'new' ? '/agents/new' : `/agents/${agentName}/edit`;
  return render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/agents/new" element={<AgentEditPage />} />
          <Route path="/agents/:name/edit" element={<AgentEditPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('AgentEditPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.listModels.mockResolvedValue([]);
    mockApi.listMCPServers.mockResolvedValue([]);
    mockApi.listAgents.mockResolvedValue([]);
  });

  it('renders create form for new agent', async () => {
    renderEditPage('new');

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Create Agent');
      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('System Prompt')).toBeInTheDocument();
    });
  });

  it('loads agent data for edit', async () => {
    mockApi.getAgent.mockResolvedValue({
      name: 'developer',
      system_prompt: 'You are a developer agent.',
      tools: ['read_file', 'execute_command'],
      can_spawn: [],
      lifecycle: 'persistent',
      tool_execution: 'sequential',
      max_steps: 50,
      max_context_size: 16000,
      confirm_before: [],
      mcp_servers: [],
      tools_count: 2,
      has_knowledge: false,
      kit: 'developer',
    });

    renderEditPage('developer');

    await waitFor(() => {
      expect(screen.getByText('Edit: developer')).toBeInTheDocument();
    });
  });
});
