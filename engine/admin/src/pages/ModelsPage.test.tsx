import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext, type AuthContextType } from '../hooks/useAuth';
import ModelsPage from './ModelsPage';

// jsdom doesn't support HTMLDialogElement.showModal/close
beforeEach(() => {
  HTMLDialogElement.prototype.showModal = vi.fn();
  HTMLDialogElement.prototype.close = vi.fn();
});

vi.mock('../api/client', () => ({
  api: {
    listModels: vi.fn(),
    createModel: vi.fn(),
    updateModel: vi.fn(),
    deleteModel: vi.fn(),
    getModelRegistry: vi.fn(),
    getRegistryProviders: vi.fn(),
  },
}));

import { api } from '../api/client';
const mockApi = vi.mocked(api);

const auth: AuthContextType = {
  isAuthenticated: true,
  login: vi.fn(),
  logout: vi.fn(),
};

function renderModelsPage() {
  return render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter>
        <ModelsPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

const MOCK_REGISTRY = [
  {
    id: 'openai/gpt-4o',
    display_name: 'GPT-4o',
    provider: 'openrouter',
    tier: 1,
    context_window: 128000,
    supports_tools: true,
    pricing_input: 2.5,
    pricing_output: 10,
    description: 'OpenAI flagship model',
    recommended_for: ['orchestrator', 'coding'],
  },
  {
    id: 'anthropic/claude-3.5-sonnet',
    display_name: 'Claude 3.5 Sonnet',
    provider: 'openrouter',
    tier: 1,
    context_window: 200000,
    supports_tools: true,
    pricing_input: 3,
    pricing_output: 15,
    description: 'Anthropic flagship model',
    recommended_for: ['orchestrator', 'analysis'],
  },
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
    recommended_for: ['sub-agent', 'coding'],
  },
];

const MOCK_MODELS = [
  {
    id: '1',
    name: 'main-model',
    type: 'openrouter',
    base_url: 'https://openrouter.ai/api/v1',
    model_name: 'openai/gpt-4o',
    has_api_key: true,
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: '2',
    name: 'custom-model',
    type: 'ollama',
    base_url: 'http://localhost:11434',
    model_name: 'my-custom-llama',
    has_api_key: false,
    created_at: '2026-01-02T00:00:00Z',
  },
];

describe('ModelsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.listModels.mockResolvedValue(MOCK_MODELS);
    mockApi.getModelRegistry.mockResolvedValue(MOCK_REGISTRY);
  });

  it('renders models list with tier badges', async () => {
    renderModelsPage();

    await waitFor(() => {
      expect(screen.getByText('main-model')).toBeInTheDocument();
      expect(screen.getByText('custom-model')).toBeInTheDocument();
    });

    // Tier 1 badge for openai/gpt-4o
    expect(screen.getByText('Tier 1 - Orchestrator')).toBeInTheDocument();

    // Custom badge for unknown model
    expect(screen.getByText('Custom')).toBeInTheDocument();
  });

  it('shows empty state when no models', async () => {
    mockApi.listModels.mockResolvedValue([]);
    renderModelsPage();

    await waitFor(() => {
      expect(screen.getByText('No models configured')).toBeInTheDocument();
    });
  });

  it('handles registry API failure gracefully', async () => {
    mockApi.getModelRegistry.mockRejectedValue(new Error('Network error'));
    renderModelsPage();

    // Should still render models without badges
    await waitFor(() => {
      expect(screen.getByText('main-model')).toBeInTheDocument();
      expect(screen.getByText('custom-model')).toBeInTheDocument();
    });
  });

  it('shows detail panel with tier info on row click', async () => {
    renderModelsPage();
    const user = userEvent.setup();

    await waitFor(() => {
      expect(screen.getByText('main-model')).toBeInTheDocument();
    });

    await user.click(screen.getByText('main-model'));

    await waitFor(() => {
      // Detail panel should show tier badge
      const tierBadges = screen.getAllByText('Tier 1 - Orchestrator');
      // One in table, one in detail panel
      expect(tierBadges.length).toBeGreaterThanOrEqual(2);
    });
  });

  it('opens form modal with provider options including openrouter', async () => {
    renderModelsPage();
    const user = userEvent.setup();

    await waitFor(() => {
      expect(screen.getByText('Add Model')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Add Model'));

    await waitFor(() => {
      expect(screen.getByText('OpenRouter')).toBeInTheDocument();
      expect(screen.getByText('Azure OpenAI')).toBeInTheDocument();
      expect(screen.getByText('Google (Gemini)')).toBeInTheDocument();
    });
  });
});
