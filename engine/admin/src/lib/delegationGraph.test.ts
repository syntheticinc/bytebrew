import { describe, it, expect } from 'vitest';
import { computeEntryAgent, resolveAgentName } from './delegationGraph';

describe('computeEntryAgent', () => {
  it('returns the source-only agent when relations name both endpoints by name', () => {
    const agents = [
      { id: 'researcher', name: 'researcher' },
      { id: 'synthesizer', name: 'synthesizer' },
    ];
    const relations = [
      { sourceAgentId: 'researcher', targetAgentId: 'synthesizer' },
    ];
    expect(computeEntryAgent(agents, relations)).toBe('researcher');
  });

  it('returns the source-only agent when relations use UUIDs rather than names', () => {
    const agents = [
      { id: 'uuid-r', name: 'researcher' },
      { id: 'uuid-s', name: 'synthesizer' },
    ];
    const relations = [
      { sourceAgentId: 'uuid-r', targetAgentId: 'uuid-s' },
    ];
    expect(computeEntryAgent(agents, relations)).toBe('researcher');
  });

  it('handles mixed relation keys (one side by name, one side by UUID)', () => {
    const agents = [
      { id: 'uuid-r', name: 'researcher' },
      { id: 'uuid-s', name: 'synthesizer' },
    ];
    const relations = [
      // Source as name, target as UUID — still should identify researcher
      // as source-only since synthesizer has an incoming edge regardless of
      // which key the relation used.
      { sourceAgentId: 'researcher', targetAgentId: 'uuid-s' },
    ];
    expect(computeEntryAgent(agents, relations)).toBe('researcher');
  });

  it('returns null when the graph has a cycle (no source-only node)', () => {
    const agents = [
      { id: 'a', name: 'a' },
      { id: 'b', name: 'b' },
    ];
    const relations = [
      { sourceAgentId: 'a', targetAgentId: 'b' },
      { sourceAgentId: 'b', targetAgentId: 'a' },
    ];
    expect(computeEntryAgent(agents, relations)).toBeNull();
  });

  it('returns null for an empty agent list', () => {
    expect(computeEntryAgent([], [])).toBeNull();
  });

  it('returns null when there are agents but no relations (no orchestrator yet)', () => {
    const agents = [
      { id: 'solo', name: 'solo' },
    ];
    expect(computeEntryAgent(agents, [])).toBeNull();
  });

  it('picks the real entry when relations include a "dangling" source endpoint', () => {
    const agents = [
      { id: 'entry', name: 'entry' },
      { id: 'middle', name: 'middle' },
      { id: 'leaf', name: 'leaf' },
    ];
    const relations = [
      { sourceAgentId: 'entry', targetAgentId: 'middle' },
      { sourceAgentId: 'middle', targetAgentId: 'leaf' },
    ];
    expect(computeEntryAgent(agents, relations)).toBe('entry');
  });
});

describe('resolveAgentName', () => {
  it('resolves by id when the key matches an agent id', () => {
    const agents = [
      { id: 'uuid-1', name: 'alpha' },
      { id: 'uuid-2', name: 'beta' },
    ];
    const byId = new Map(agents.map((a) => [a.id, a]));
    const byName = new Map(agents.map((a) => [a.name, a]));
    expect(resolveAgentName('uuid-1', byId, byName)).toBe('alpha');
  });

  it('resolves by name when the key matches an agent name', () => {
    const agents = [
      { id: 'uuid-1', name: 'alpha' },
    ];
    const byId = new Map(agents.map((a) => [a.id, a]));
    const byName = new Map(agents.map((a) => [a.name, a]));
    expect(resolveAgentName('alpha', byId, byName)).toBe('alpha');
  });

  it('returns null when the key matches neither', () => {
    const byId = new Map();
    const byName = new Map();
    expect(resolveAgentName('missing', byId, byName)).toBeNull();
  });
});
