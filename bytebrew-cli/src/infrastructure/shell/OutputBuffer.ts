const DEFAULT_MAX_SIZE = 1024 * 1024; // 1MB

// Marker format: __BYTEBREW_DONE_<uuid>_<exitcode>__
const MARKER_REGEX = /__BYTEBREW_DONE_([a-f0-9]+)_(-?\d+)__/;

export interface MarkerResult {
  exitCode: number;
  output: string; // output BEFORE the marker (cleaned)
}

export class OutputBuffer {
  private buffer: string = '';
  private maxSize: number;
  private pendingResolve: ((result: MarkerResult) => void) | null = null;
  private pendingMarkerId: string | null = null;
  private pendingTimeout: ReturnType<typeof setTimeout> | null = null;
  private pendingReject: ((err: Error) => void) | null = null;

  constructor(maxSize: number = DEFAULT_MAX_SIZE) {
    this.maxSize = maxSize;
  }

  /**
   * Append data to buffer. If buffer exceeds maxSize, trim from the start.
   * After appending, check if pending marker is now in the buffer.
   */
  append(chunk: string): void {
    this.buffer += chunk;

    // Ring buffer: trim from start if exceeds max
    if (this.buffer.length > this.maxSize) {
      this.buffer = this.buffer.slice(this.buffer.length - this.maxSize);
    }

    // Check for pending marker
    this.checkMarker();
  }

  /**
   * Wait for a specific marker to appear in the buffer.
   * Returns Promise that resolves with MarkerResult when marker found.
   * Rejects with Error on timeout.
   *
   * Only one waitForMarker can be active at a time.
   */
  waitForMarker(markerId: string, timeoutMs: number): Promise<MarkerResult> {
    if (this.pendingResolve) {
      return Promise.reject(new Error('Another waitForMarker is already active'));
    }

    // Check if marker is already in buffer
    const existing = this.findMarker(markerId);
    if (existing) {
      return Promise.resolve(existing);
    }

    return new Promise<MarkerResult>((resolve, reject) => {
      this.pendingResolve = resolve;
      this.pendingReject = reject;
      this.pendingMarkerId = markerId;

      this.pendingTimeout = setTimeout(() => {
        this.pendingResolve = null;
        this.pendingReject = null;
        this.pendingMarkerId = null;
        this.pendingTimeout = null;
        reject(new Error(`Marker timeout after ${timeoutMs}ms`));
      }, timeoutMs);
    });
  }

  /**
   * Get current buffer contents.
   */
  getOutput(): string {
    return this.buffer;
  }

  /**
   * Clear the buffer.
   */
  reset(): void {
    this.buffer = '';
    // Don't clear pending - it's the caller's responsibility to manage that
  }

  /**
   * Cancel any pending waitForMarker.
   * Does NOT throw — silently clears pending state.
   */
  cancelPending(): void {
    if (this.pendingTimeout) {
      clearTimeout(this.pendingTimeout);
    }
    // Clear state without calling reject (silent cancellation)
    this.pendingResolve = null;
    this.pendingReject = null;
    this.pendingMarkerId = null;
    this.pendingTimeout = null;
  }

  /**
   * Check if the current buffer contains the pending marker.
   */
  private checkMarker(): void {
    if (!this.pendingMarkerId || !this.pendingResolve) return;

    const result = this.findMarker(this.pendingMarkerId);
    if (result) {
      const resolve = this.pendingResolve;
      if (this.pendingTimeout) clearTimeout(this.pendingTimeout);
      this.pendingResolve = null;
      this.pendingReject = null;
      this.pendingMarkerId = null;
      this.pendingTimeout = null;
      resolve(result);
    }
  }

  /**
   * Find a specific marker in the buffer and extract output + exit code.
   */
  private findMarker(markerId: string): MarkerResult | null {
    const markerPattern = new RegExp(`__BYTEBREW_DONE_${markerId}_(-?\\d+)__`);
    const match = this.buffer.match(markerPattern);
    if (!match) return null;

    const exitCode = parseInt(match[1], 10);
    // Output is everything before the marker line
    const markerIndex = match.index!;
    let output = this.buffer.slice(0, markerIndex);

    // Remove trailing newline before marker
    output = output.replace(/\n$/, '');

    return { exitCode, output };
  }

  /**
   * Generate a marker command to wrap around a shell command.
   * Returns { markerId, wrappedCommand }.
   */
  static wrapCommand(command: string): { markerId: string; wrappedCommand: string } {
    // Generate a short hex ID (no dashes, 12 chars)
    const markerId = Math.random().toString(16).slice(2, 14).padEnd(12, '0');

    // Wrap: run command, capture exit code, print marker
    const wrappedCommand = `${command} 2>&1; __vexit=$?; printf '\\n__BYTEBREW_DONE_${markerId}_%d__\\n' $__vexit`;

    return { markerId, wrappedCommand };
  }
}
