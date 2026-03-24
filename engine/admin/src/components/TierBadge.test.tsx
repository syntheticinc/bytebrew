import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import TierBadge, { CustomModelBadge } from './TierBadge';

describe('TierBadge', () => {
  it('renders Tier 1 badge with green styling', () => {
    render(<TierBadge tier={1} />);
    const badge = screen.getByText('Tier 1 - Orchestrator');
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain('bg-green-500/15');
    expect(badge.className).toContain('text-green-400');
  });

  it('renders Tier 2 badge with blue styling', () => {
    render(<TierBadge tier={2} />);
    const badge = screen.getByText('Tier 2 - Sub-agent');
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain('bg-blue-500/15');
    expect(badge.className).toContain('text-blue-400');
  });

  it('renders Tier 3 badge with gray styling', () => {
    render(<TierBadge tier={3} />);
    const badge = screen.getByText('Tier 3 - Utility');
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain('bg-brand-shade3/15');
  });

  it('renders nothing for unknown tier', () => {
    const { container } = render(<TierBadge tier={99} />);
    expect(container.firstChild).toBeNull();
  });

  it('applies custom className', () => {
    render(<TierBadge tier={1} className="ml-2" />);
    const badge = screen.getByText('Tier 1 - Orchestrator');
    expect(badge.className).toContain('ml-2');
  });
});

describe('CustomModelBadge', () => {
  it('renders Custom badge with yellow styling', () => {
    render(<CustomModelBadge />);
    const badge = screen.getByText('Custom');
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain('bg-yellow-500/15');
    expect(badge.className).toContain('text-yellow-400');
  });
});
