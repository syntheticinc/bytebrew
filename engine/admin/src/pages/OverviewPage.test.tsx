import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { PrototypeProvider } from '../hooks/usePrototype';
import OverviewPage from './OverviewPage';

// ── API mock ──────────────────────────────────────────────────────────────────

vi.mock('../api/client', () => ({
  api: {
    listSessions: vi.fn(),
    listTriggers: vi.fn(),
    listSchemas: vi.fn(),
    health: vi.fn(),
  },
}));

import { api } from '../api/client';
const mockApi = vi.mocked(api);

// ── Helpers ───────────────────────────────────────────────────────────────────

function renderPage(prototypeMode = false) {
  if (prototypeMode) {
    localStorage.setItem('bytebrew_prototype_mode', 'true');
  } else {
    localStorage.removeItem('bytebrew_prototype_mode');
  }

  return render(
    <MemoryRouter>
      <PrototypeProvider>
        <OverviewPage />
      </PrototypeProvider>
    </MemoryRouter>,
  );
}

const emptyPaginated = { sessions: [], total: 0, page: 1, per_page: 50 };
const emptyHealth = { status: 'ok', version: '0.1.0', uptime: '1h', agents_count: 2 };

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('OverviewPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.removeItem('bytebrew_prototype_mode');
  });

  describe('prototype mode', () => {
    it('renders heading', () => {
      renderPage(true);
      expect(screen.getByText('Overview')).toBeInTheDocument();
    });

    it('renders stat labels from mock data', () => {
      renderPage(true);
      expect(screen.getByText('Active Sessions')).toBeInTheDocument();
      expect(screen.getByText('Sessions Today')).toBeInTheDocument();
      expect(screen.getByText('Enabled Triggers')).toBeInTheDocument();
      expect(screen.getByText('Success Rate')).toBeInTheDocument();
    });

    it('renders Live Sessions panel', () => {
      renderPage(true);
      expect(screen.getByText('Live Sessions')).toBeInTheDocument();
    });

    it('renders Recent Events panel', () => {
      renderPage(true);
      expect(screen.getByText('Recent Events')).toBeInTheDocument();
    });

    it('renders Schemas quick access', () => {
      renderPage(true);
      expect(screen.getByText('Schemas')).toBeInTheDocument();
    });
  });

  describe('production mode', () => {
    beforeEach(() => {
      mockApi.listSessions.mockResolvedValue(emptyPaginated);
      mockApi.listTriggers.mockResolvedValue([]);
      mockApi.listSchemas.mockResolvedValue([]);
      mockApi.health.mockResolvedValue(emptyHealth);
    });

    it('renders heading', async () => {
      renderPage(false);
      await waitFor(() => expect(screen.getByText('Overview')).toBeInTheDocument());
    });

    it('renders stat labels', async () => {
      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('Active Sessions')).toBeInTheDocument();
        expect(screen.getByText('Sessions Today')).toBeInTheDocument();
        expect(screen.getByText('Enabled Triggers')).toBeInTheDocument();
        expect(screen.getByText('Success Rate')).toBeInTheDocument();
      });
    });

    it('shows — for Sessions Today (no backend source)', async () => {
      renderPage(false);
      await waitFor(() => {
        // Sessions Today card value is always — in production
        expect(screen.getByText('no daily counter in API')).toBeInTheDocument();
      });
    });

    it('shows — for Success Rate when no finished sessions', async () => {
      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('no completed sessions yet')).toBeInTheDocument();
      });
    });

    it('shows empty state for live sessions', async () => {
      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('No live sessions right now.')).toBeInTheDocument();
      });
    });

    it('shows event stream unavailable message', async () => {
      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('Event stream not available yet.')).toBeInTheDocument();
      });
    });

    it('shows system OK badge when health returns ok', async () => {
      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('System: OK')).toBeInTheDocument();
      });
    });

    it('shows active sessions when API returns them', async () => {
      const activeSessions = [
        {
          session_id: 'sess-abc123',
          entry_agent: 'support-agent',
          status: 'running' as const,
          duration_ms: 0,
          total_tokens: 0,
          created_at: new Date().toISOString(),
        },
      ];
      mockApi.listSessions.mockImplementation((params) => {
        if (params?.status?.includes('running')) {
          return Promise.resolve({ sessions: activeSessions, total: 1, page: 1, per_page: 50 });
        }
        return Promise.resolve(emptyPaginated);
      });

      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('support-agent')).toBeInTheDocument();
      });
    });

    it('shows enabled trigger ratio', async () => {
      mockApi.listTriggers.mockResolvedValue([
        { id: '1', title: 'Trigger 1', type: 'cron', enabled: true, created_at: '' },
        { id: '2', title: 'Trigger 2', type: 'webhook', enabled: false, created_at: '' },
      ]);

      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText('1 / 2')).toBeInTheDocument();
      });
    });

    it('handles API error gracefully', async () => {
      mockApi.listSessions.mockRejectedValue(new Error('network error'));

      renderPage(false);
      await waitFor(() => {
        expect(screen.getByText(/Failed to load overview/)).toBeInTheDocument();
      });
    });
  });
});
