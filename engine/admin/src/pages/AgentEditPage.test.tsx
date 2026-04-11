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
    listToolMetadata: vi.fn(),
    getModelRegistry: vi.fn(),
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
    mockApi.listToolMetadata.mockResolvedValue([
      { name: 'ask_user', description: 'Ask user', security_zone: 'safe' },
      { name: 'read_file', description: 'Read file', security_zone: 'dangerous', risk_warning: 'Filesystem access' },
    ]);
    mockApi.getModelRegistry.mockResolvedValue([]);
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
      max_turn_duration: 120,
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

  it('shows warning when orchestrator agent uses low-tier model', async () => {
    mockApi.listModels.mockResolvedValue([
      {
        id: '1',
        name: 'sub-model',
        type: 'openrouter',
        base_url: 'https://openrouter.ai/api/v1',
        model_name: 'qwen/qwen3-coder',
        has_api_key: true,
        created_at: '2026-01-01T00:00:00Z',
      },
    ]);
    mockApi.getModelRegistry.mockResolvedValue([
      {
        id: 'qwen/qwen3-coder',
        display_name: 'Qwen3 Coder',
        provider: 'openrouter',
        tier: 2,
        context_window: 32000,
        supports_tools: true,
        pricing_input: 0.5,
        pricing_output: 1,
        description: 'Good for sub-agent tasks',
        recommended_for: ['sub-agent'],
      },
    ]);

    mockApi.getAgent.mockResolvedValue({
      name: 'orchestrator',
      model_id: '1',
      system_prompt: 'You are an orchestrator.',
      tools: [],
      can_spawn: ['worker-agent'],
      lifecycle: 'persistent',
      tool_execution: 'sequential',
      max_steps: 50,
      max_context_size: 16000,
      max_turn_duration: 120,
      confirm_before: [],
      mcp_servers: [],
      tools_count: 0,
      has_knowledge: false,
    });
    mockApi.listAgents.mockResolvedValue([
      { name: 'orchestrator', tools_count: 0, has_knowledge: false },
      { name: 'worker-agent', tools_count: 3, has_knowledge: false },
    ]);

    renderEditPage('orchestrator');

    await waitFor(() => {
      expect(screen.getByText(/may not reliably handle complex multi-step tool calling/)).toBeInTheDocument();
    });
  });

  it('shows info message when model is not in registry', async () => {
    mockApi.listModels.mockResolvedValue([
      {
        id: '1',
        name: 'custom-local',
        type: 'ollama',
        base_url: 'http://localhost:11434',
        model_name: 'my-custom-model',
        has_api_key: false,
        created_at: '2026-01-01T00:00:00Z',
      },
    ]);
    mockApi.getModelRegistry.mockResolvedValue([]);

    mockApi.getAgent.mockResolvedValue({
      name: 'test-agent',
      model_id: '1',
      system_prompt: 'Test agent.',
      tools: [],
      can_spawn: [],
      lifecycle: 'persistent',
      tool_execution: 'sequential',
      max_steps: 50,
      max_context_size: 16000,
      max_turn_duration: 120,
      confirm_before: [],
      mcp_servers: [],
      tools_count: 0,
      has_knowledge: false,
    });

    renderEditPage('test-agent');

    await waitFor(() => {
      expect(screen.getByText(/Model not in registry/)).toBeInTheDocument();
    });
  });
});
