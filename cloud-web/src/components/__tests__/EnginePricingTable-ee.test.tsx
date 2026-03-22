import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

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

describe('EnginePricingTable with SHOW_EE_PRICING=true', () => {
  it('EE shows pricing with "Most Popular" badge', () => {
    render(<EnginePricingTable />);

    expect(screen.getByText('Most Popular')).toBeInTheDocument();
  });

  it('EE shows "Start Free Trial" button', () => {
    render(<EnginePricingTable />);

    expect(screen.getByText('Start Free Trial')).toBeInTheDocument();
  });

  it('shows period toggle (Monthly/Annual)', () => {
    render(<EnginePricingTable />);

    expect(screen.getByText('Monthly')).toBeInTheDocument();
    expect(screen.getByText(/Annual/)).toBeInTheDocument();
  });
});
