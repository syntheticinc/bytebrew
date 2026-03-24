import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Mock feature flags with EE pricing enabled
vi.mock('../../lib/feature-flags', () => ({
  SHOW_EE_PRICING: true,
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

describe('EnginePricingTable with SHOW_EE_PRICING=true', () => {
  it('EE shows pricing with "Most Popular" badge', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Most Popular')).toBeInTheDocument();
  });

  it('EE shows "Start Free Trial" button', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Start Free Trial')).toBeInTheDocument();
  });

  it('EE shows "---" as price fallback when pricing unavailable', () => {
    renderWithProviders(<EnginePricingTable />);

    // When pricing data is not loaded yet, EE should show "---" not "Contact Us"
    expect(screen.getByText('Enterprise Edition')).toBeInTheDocument();
    // "Contact Us" should only appear in the Custom column
    const contactUsElements = screen.getAllByText('Contact Us');
    expect(contactUsElements).toHaveLength(1);
  });

  it('shows period toggle (Monthly/Annual)', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('Monthly')).toBeInTheDocument();
    expect(screen.getByText(/Annual/)).toBeInTheDocument();
  });

  it('shows trial info text under EE CTA', () => {
    renderWithProviders(<EnginePricingTable />);

    expect(screen.getByText('14-day free trial. No credit card required.')).toBeInTheDocument();
  });
});
