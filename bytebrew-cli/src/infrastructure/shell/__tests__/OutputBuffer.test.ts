import { describe, it, expect } from 'bun:test';
import { OutputBuffer } from '../OutputBuffer.js';

describe('OutputBuffer', () => {
  // Basic append + getOutput
  it('should store appended data', () => {
    const buf = new OutputBuffer();
    buf.append('hello ');
    buf.append('world');
    expect(buf.getOutput()).toBe('hello world');
  });

  // Reset
  it('should clear on reset', () => {
    const buf = new OutputBuffer();
    buf.append('data');
    buf.reset();
    expect(buf.getOutput()).toBe('');
  });

  // Ring buffer overflow
  it('should trim from start when exceeding maxSize', () => {
    const buf = new OutputBuffer(10); // 10 byte max
    buf.append('1234567890'); // exactly 10
    expect(buf.getOutput()).toBe('1234567890');
    buf.append('abc'); // now 13, should trim to last 10
    expect(buf.getOutput()).toBe('4567890abc');
  });

  // waitForMarker - found immediately (marker already in buffer)
  it('waitForMarker resolves immediately if marker already present', async () => {
    const buf = new OutputBuffer();
    buf.append('some output\n__BYTEBREW_DONE_abc123def456_0__\n');
    const result = await buf.waitForMarker('abc123def456', 5000);
    expect(result.exitCode).toBe(0);
    expect(result.output).toBe('some output');
  });

  // waitForMarker - found after delay
  it('waitForMarker resolves when marker arrives later', async () => {
    const buf = new OutputBuffer();
    const promise = buf.waitForMarker('abc123def456', 5000);

    // Simulate delayed output
    setTimeout(() => {
      buf.append('line 1\nline 2\n__BYTEBREW_DONE_abc123def456_0__\n');
    }, 50);

    const result = await promise;
    expect(result.exitCode).toBe(0);
    expect(result.output).toBe('line 1\nline 2');
  });

  // waitForMarker - timeout
  it('waitForMarker rejects on timeout', async () => {
    const buf = new OutputBuffer();
    try {
      await buf.waitForMarker('nonexistent', 100);
      throw new Error('should not reach');
    } catch (err: any) {
      expect(err.message).toContain('timeout');
    }
  });

  // Exit code parsing - non-zero
  it('should parse non-zero exit code', async () => {
    const buf = new OutputBuffer();
    buf.append('error output\n__BYTEBREW_DONE_abc123def456_1__\n');
    const result = await buf.waitForMarker('abc123def456', 5000);
    expect(result.exitCode).toBe(1);
    expect(result.output).toBe('error output');
  });

  // Exit code parsing - negative (killed by signal)
  it('should parse negative exit code', async () => {
    const buf = new OutputBuffer();
    buf.append('killed\n__BYTEBREW_DONE_abc123def456_-9__\n');
    const result = await buf.waitForMarker('abc123def456', 5000);
    expect(result.exitCode).toBe(-9);
  });

  // Concurrent reject
  it('should reject if another waitForMarker is already active', async () => {
    const buf = new OutputBuffer();
    const p1 = buf.waitForMarker('marker1', 5000);

    try {
      await buf.waitForMarker('marker2', 5000);
      throw new Error('should not reach');
    } catch (err: any) {
      expect(err.message).toContain('already active');
    }

    // Clean up p1 - cancelPending now silently clears without rejection
    buf.cancelPending();
    // p1 will hang forever, but test completes because we don't await it
  });

  // cancelPending
  it('cancelPending should silently clear pending state', async () => {
    const buf = new OutputBuffer();
    const promise = buf.waitForMarker('marker1', 5000);

    // cancelPending now silently clears state without rejecting
    buf.cancelPending();

    // Promise remains pending (no marker will arrive, but also no rejection)
    // After cancel, a new waitForMarker should work
    const marker = OutputBuffer.wrapCommand('echo test').markerId;
    buf.append(`output\n__BYTEBREW_DONE_${marker}_0__\n`);
    const result = await buf.waitForMarker(marker, 1000);
    expect(result.exitCode).toBe(0);
  });

  // wrapCommand
  it('wrapCommand should generate valid wrapped command', () => {
    const { markerId, wrappedCommand } = OutputBuffer.wrapCommand('echo hello');
    expect(markerId).toHaveLength(12);
    expect(wrappedCommand).toContain('echo hello');
    expect(wrappedCommand).toContain('2>&1');
    expect(wrappedCommand).toContain('__BYTEBREW_DONE_');
    expect(wrappedCommand).toContain(markerId);
    expect(wrappedCommand).toContain('__vexit=$?');
    expect(wrappedCommand).toContain('printf');
  });

  // Output with mixed content before marker
  it('should handle multi-line output before marker', async () => {
    const buf = new OutputBuffer();
    buf.append('line 1\nline 2\nline 3\n__BYTEBREW_DONE_aaa111bbb222_42__\n');
    const result = await buf.waitForMarker('aaa111bbb222', 5000);
    expect(result.exitCode).toBe(42);
    expect(result.output).toBe('line 1\nline 2\nline 3');
  });

  // Chunked arrival
  it('should handle marker arriving in multiple chunks', async () => {
    const buf = new OutputBuffer();
    const promise = buf.waitForMarker('aaa111bbb222', 5000);

    buf.append('output data\n');
    buf.append('__BYTEBREW_DONE_');
    buf.append('aaa111bbb222_0__\n');

    const result = await promise;
    expect(result.exitCode).toBe(0);
    expect(result.output).toBe('output data');
  });
});
