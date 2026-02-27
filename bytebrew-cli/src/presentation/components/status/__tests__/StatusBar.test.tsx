import { describe, it, expect, beforeAll } from 'bun:test';
import React from 'react';
import { StatusBar } from '../StatusBar.js';

let render: typeof import('ink-testing-library').render;

beforeAll(async () => {
  const inkTesting = await import('ink-testing-library');
  render = inkTesting.render;
});

describe('StatusBar', () => {
  it('renders providerBadge when provided', () => {
    const instance = render(
      <StatusBar
        connectionStatus="connected"
        providerBadge={{ label: 'proxy 253/300', color: 'green' }}
      />
    );

    const frame = instance.lastFrame();
    expect(frame).toContain('proxy 253/300');
    instance.unmount();
  });

  it('renders tierBadge and providerBadge together', () => {
    const instance = render(
      <StatusBar
        connectionStatus="connected"
        tierBadge={{ label: '[Personal]', color: 'green' }}
        providerBadge={{ label: 'byok', color: 'cyan' }}
      />
    );

    const frame = instance.lastFrame();
    expect(frame).toContain('[Personal]');
    expect(frame).toContain('byok');
    instance.unmount();
  });

  it('does not render providerBadge when not provided', () => {
    const instance = render(
      <StatusBar connectionStatus="connected" />
    );

    const frame = instance.lastFrame();
    // Should not contain any provider label
    expect(frame).not.toContain('proxy');
    expect(frame).not.toContain('byok');
    expect(frame).not.toContain('auto');
    instance.unmount();
  });
});
