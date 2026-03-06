export interface PairingResult {
  deviceId: string;
  deviceName: string;
}

export interface IPairingWaiter {
  wait(token: string, timeoutMs: number): Promise<PairingResult>;
  resolve(token: string, deviceId: string, deviceName: string): void;
  cancel(token: string): void;
}

interface PendingWaiter {
  resolve: (result: PairingResult) => void;
  reject: (reason: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

/**
 * Promise-based waiter for pairing completion.
 * Compatible with Go PairingWaiter in bytebrew-srv.
 *
 * Flow:
 * 1. CLI calls wait(token, timeout) before showing QR code
 * 2. Mobile device pairs and handler calls resolve(token, deviceId, deviceName)
 * 3. The waiting Promise resolves with the pairing result
 *
 * If timeout expires or cancel() is called, the Promise rejects.
 */
export class PairingWaiter implements IPairingWaiter {
  private readonly waiters = new Map<string, PendingWaiter>();

  wait(token: string, timeoutMs: number): Promise<PairingResult> {
    if (this.waiters.has(token)) {
      return Promise.reject(new Error(`duplicate wait for token: ${token}`));
    }

    return new Promise<PairingResult>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters.delete(token);
        reject(new Error(`pairing timeout for token: ${token}`));
      }, timeoutMs);

      this.waiters.set(token, { resolve, reject, timer });
    });
  }

  resolve(token: string, deviceId: string, deviceName: string): void {
    const waiter = this.waiters.get(token);
    if (!waiter) {
      return;
    }

    clearTimeout(waiter.timer);
    this.waiters.delete(token);
    waiter.resolve({ deviceId, deviceName });
  }

  cancel(token: string): void {
    const waiter = this.waiters.get(token);
    if (!waiter) {
      return;
    }

    clearTimeout(waiter.timer);
    this.waiters.delete(token);
    waiter.reject(new Error(`pairing cancelled for token: ${token}`));
  }
}
