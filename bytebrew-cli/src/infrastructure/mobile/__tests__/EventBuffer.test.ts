import { describe, it, expect } from 'bun:test';
import { EventBuffer } from '../EventBuffer.js';

describe('EventBuffer', () => {
  describe('push + getAfter', () => {
    it('returns events after given index', () => {
      const buf = new EventBuffer<string>();
      buf.push('s1', 'event-0');
      buf.push('s1', 'event-1');
      buf.push('s1', 'event-2');

      // After index 0 means events with index > 0, so event-1 and event-2
      const { events, lastIndex } = buf.getAfter('s1', 0);
      expect(events).toEqual(['event-1', 'event-2']);
      expect(lastIndex).toBe(2);
    });

    it('returns empty for unknown session', () => {
      const buf = new EventBuffer<string>();
      const { events, lastIndex } = buf.getAfter('unknown', 0);
      expect(events).toEqual([]);
      expect(lastIndex).toBe(0);
    });

    it('returns empty when afterIndex >= lastIndex', () => {
      const buf = new EventBuffer<string>();
      buf.push('s1', 'event-0');

      const { events } = buf.getAfter('s1', 0);
      expect(events).toEqual([]);
    });
  });

  describe('getAfter with -1', () => {
    it('returns all events when afterIndex is -1', () => {
      const buf = new EventBuffer<string>();
      buf.push('s1', 'A');
      buf.push('s1', 'B');
      buf.push('s1', 'C');

      const { events, lastIndex } = buf.getAfter('s1', -1);
      expect(events).toEqual(['A', 'B', 'C']);
      expect(lastIndex).toBe(2);
    });
  });

  describe('overflow (ring buffer)', () => {
    it('drops oldest events when exceeding maxEvents', () => {
      const buf = new EventBuffer<number>(5);

      for (let i = 0; i < 8; i++) {
        buf.push('s1', i);
      }

      // Should have events 3,4,5,6,7 (last 5)
      const { events, lastIndex } = buf.getAfter('s1', -1);
      expect(events).toEqual([3, 4, 5, 6, 7]);
      expect(lastIndex).toBe(7);
    });

    it('adjusts startIndex after overflow', () => {
      const buf = new EventBuffer<number>(3);

      // Push 5 items: 0,1,2,3,4 — buffer keeps last 3: [2,3,4]
      for (let i = 0; i < 5; i++) {
        buf.push('s1', i);
      }

      // getAfter(1) should return events with index > 1, i.e., [2,3,4]
      const { events } = buf.getAfter('s1', 1);
      expect(events).toEqual([2, 3, 4]);
    });

    it('default maxEvents is 1000', () => {
      const buf = new EventBuffer<number>();

      for (let i = 0; i < 1005; i++) {
        buf.push('s1', i);
      }

      const { events } = buf.getAfter('s1', -1);
      expect(events).toHaveLength(1000);
      expect(events[0]).toBe(5);
      expect(events[999]).toBe(1004);
    });
  });

  describe('clear', () => {
    it('removes all events for session', () => {
      const buf = new EventBuffer<string>();
      buf.push('s1', 'A');
      buf.push('s1', 'B');

      buf.clear('s1');

      const { events } = buf.getAfter('s1', -1);
      expect(events).toEqual([]);
    });

    it('does not affect other sessions', () => {
      const buf = new EventBuffer<string>();
      buf.push('s1', 'A');
      buf.push('s2', 'B');

      buf.clear('s1');

      const { events } = buf.getAfter('s2', -1);
      expect(events).toEqual(['B']);
    });
  });

  describe('multiple sessions', () => {
    it('isolates buffers per session', () => {
      const buf = new EventBuffer<string>();
      buf.push('s1', 'alpha');
      buf.push('s2', 'beta');
      buf.push('s1', 'gamma');

      const s1 = buf.getAfter('s1', -1);
      expect(s1.events).toEqual(['alpha', 'gamma']);

      const s2 = buf.getAfter('s2', -1);
      expect(s2.events).toEqual(['beta']);
    });

    it('each session has independent indices', () => {
      const buf = new EventBuffer<number>(3);
      buf.push('s1', 10);
      buf.push('s1', 20);

      buf.push('s2', 100);
      buf.push('s2', 200);
      buf.push('s2', 300);
      buf.push('s2', 400); // triggers overflow for s2

      const s1 = buf.getAfter('s1', -1);
      expect(s1.events).toEqual([10, 20]);
      expect(s1.lastIndex).toBe(1);

      const s2 = buf.getAfter('s2', -1);
      expect(s2.events).toEqual([200, 300, 400]);
      expect(s2.lastIndex).toBe(3);
    });
  });
});
