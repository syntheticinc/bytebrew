import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Mock feature flags — default: EE pricing hidden
vi.mock('../../lib/feature-flags', () => ({
  SHOW_EE_PRICING: false,
}));

// Mock tanstack router Link
vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, to, ...props }: { children: React.ReactNode; to: string; [key: string]: unknown }) => (
    <a href={to} {...props}>{children}</a>
  ),
}));

import { EnginePricingTable } from '../EnginePricingTable';

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe('EnginePricingTable', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders 3 pricing columns (CE, EE, Custom)', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Community Edition')).toBeInTheDocument();
    expect(screen.getByText('Enterprise Edition')).toBeInTheDocument();
    expect(screen.getByText('Custom')).toBeInTheDocument();
  });

  it('CE column shows "Free" and "Download" button', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Free')).toBeInTheDocument();
    expect(screen.getByText('Download')).toBeInTheDocument();
  });

  it('Custom column shows "Contact Us" and "Talk to Sales" button', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Contact Us')).toBeInTheDocument();
    expect(screen.getByText('Talk to Sales')).toBeInTheDocument();
  });

  it('EE shows "Coming Soon" when SHOW_EE_PRICING is false', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Coming Soon')).toBeInTheDocument();
    expect(screen.getByText('Join Waitlist')).toBeInTheDocument();
  });

  it('CE column lists expected features', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Unlimited agents & spawn')).toBeInTheDocument();
    expect(screen.getByText('MCP servers & declarative tools')).toBeInTheDocument();
    expect(screen.getByText('BYOK (bring your own keys)')).toBeInTheDocument();
  });

  it('CE column lists new features', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Model Registry & recommendations')).toBeInTheDocument();
    expect(screen.getByText('Docker / bare metal / Kubernetes')).toBeInTheDocument();
    expect(screen.getByText('Multi-agent orchestration')).toBeInTheDocument();
  });

  it('shows "Free Forever" badge on CE column', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Free Forever')).toBeInTheDocument();
  });

  it('does not show period toggle when SHOW_EE_PRICING is false', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.queryByText('Monthly')).not.toBeInTheDocument();
    expect(screen.queryByText('Annual')).not.toBeInTheDocument();
  });
});
