import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TierBadge } from '../TierBadge';

describe('TierBadge', () => {
  it('renders "Community" for tier "ce"', () => {
    render(<TierBadge tier="ce" />);

    expect(screen.getByText('Community')).toBeInTheDocument();
  });

  it('renders "Enterprise" for tier "ee"', () => {
    render(<TierBadge tier="ee" />);

    expect(screen.getByText('Enterprise')).toBeInTheDocument();
  });

  it('renders "Trial" for tier "trial" (backward compat)', () => {
    render(<TierBadge tier="trial" />);

    expect(screen.getByText('Trial')).toBeInTheDocument();
  });

  it('renders "Personal" for tier "personal" (backward compat)', () => {
    render(<TierBadge tier="personal" />);

    expect(screen.getByText('Personal')).toBeInTheDocument();
  });

  it('renders "Teams" for tier "teams"', () => {
    render(<TierBadge tier="teams" />);

    expect(screen.getByText('Teams')).toBeInTheDocument();
  });

  it('renders raw tier value for unknown tiers', () => {
    render(<TierBadge tier="custom-tier" />);

    expect(screen.getByText('custom-tier')).toBeInTheDocument();
  });

  it('applies emerald styling for "ce" tier', () => {
    render(<TierBadge tier="ce" />);

    const badge = screen.getByText('Community');
    expect(badge.className).toContain('emerald');
  });

  it('applies purple styling for "ee" tier', () => {
    render(<TierBadge tier="ee" />);

    const badge = screen.getByText('Enterprise');
    expect(badge.className).toContain('purple');
  });
});
