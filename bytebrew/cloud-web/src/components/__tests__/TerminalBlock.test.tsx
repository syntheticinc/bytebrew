import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TerminalBlock } from '../TerminalBlock';

// Mock navigator.clipboard
Object.assign(navigator, {
  clipboard: {
    writeText: vi.fn().mockResolvedValue(undefined),
  },
});

describe('TerminalBlock', () => {
  it('renders command text', () => {
    render(<TerminalBlock command="docker compose up -d" />);

    expect(screen.getByText('docker compose up -d')).toBeInTheDocument();
  });

  it('renders default prefix "$"', () => {
    render(<TerminalBlock command="echo hello" />);

    expect(screen.getByText('$')).toBeInTheDocument();
  });

  it('renders custom prefix', () => {
    render(<TerminalBlock command="Get-Process" prefix=">" />);

    expect(screen.getByText('>')).toBeInTheDocument();
  });

  it('renders title when provided', () => {
    render(<TerminalBlock command="npm install" title="Install dependencies" />);

    expect(screen.getByText('Install dependencies')).toBeInTheDocument();
  });

  it('does not render title when not provided', () => {
    const { container } = render(<TerminalBlock command="npm install" />);

    // No <p> with title class should exist before the terminal block
    const paragraphs = container.querySelectorAll('p');
    const titleP = Array.from(paragraphs).find((p) =>
      p.classList.contains('mb-2'),
    );
    expect(titleP).toBeUndefined();
  });

  it('copy button shows "Copy" initially', () => {
    render(<TerminalBlock command="echo test" />);

    expect(screen.getByText('Copy')).toBeInTheDocument();
  });

  it('copies command to clipboard on click', () => {
    render(<TerminalBlock command="docker run nginx" />);

    const copyBtn = screen.getByText('Copy');
    fireEvent.click(copyBtn);

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('docker run nginx');
  });
});
