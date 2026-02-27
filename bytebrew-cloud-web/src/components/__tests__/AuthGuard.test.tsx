import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';

// Mock useAuth to control authentication state
const mockUseAuth = vi.fn();
vi.mock('../../lib/auth', () => ({
  useAuth: () => mockUseAuth(),
}));

// Mock useNavigate from tanstack router
const mockNavigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mockNavigate,
}));

// Import after mocks are set up
import { AuthGuard } from '../AuthGuard';

describe('AuthGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading spinner while auth is loading', () => {
    mockUseAuth.mockReturnValue({ isAuthenticated: false, isLoading: true });

    render(
      <AuthGuard>
        <div data-testid="protected">Secret content</div>
      </AuthGuard>,
    );

    expect(screen.getByText('Loading...')).toBeInTheDocument();
    expect(screen.queryByTestId('protected')).not.toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('redirects to /login when not authenticated', () => {
    mockUseAuth.mockReturnValue({ isAuthenticated: false, isLoading: false });

    render(
      <AuthGuard>
        <div data-testid="protected">Secret content</div>
      </AuthGuard>,
    );

    expect(screen.queryByTestId('protected')).not.toBeInTheDocument();
    expect(mockNavigate).toHaveBeenCalledWith({ to: '/login' });
  });

  it('renders children when authenticated', () => {
    mockUseAuth.mockReturnValue({ isAuthenticated: true, isLoading: false });

    render(
      <AuthGuard>
        <div data-testid="protected">Secret content</div>
      </AuthGuard>,
    );

    expect(screen.getByTestId('protected')).toBeInTheDocument();
    expect(screen.getByText('Secret content')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });
});
