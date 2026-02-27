import { describe, it, expect, afterEach } from 'bun:test';
import { render } from 'ink-testing-library';
import React, { useEffect } from 'react';
import { Text } from 'ink';
import { usePermissionApproval } from '../usePermissionApproval.js';
import { PermissionApproval } from '../../../infrastructure/permission/PermissionApproval.js';

const tick = () => new Promise(r => setTimeout(r, 50));

// Test component that uses the hook
function TestComponent({ onReady }: { onReady?: (approve: (remember: boolean) => void, reject: () => void) => void }) {
  const { pendingPermission, approve, reject } = usePermissionApproval();

  useEffect(() => {
    if (onReady) {
      onReady(approve, reject);
    }
  }, [approve, reject, onReady]);

  return (
    <Text>
      {pendingPermission ? `Pending: ${pendingPermission.request.value}` : 'No pending permission'}
    </Text>
  );
}

describe('usePermissionApproval', () => {
  afterEach(() => {
    // Reset to headless mode after each test
    PermissionApproval.reset();
  });

  it('shows first permission immediately', async () => {
    let approve: ((remember: boolean) => void) | null = null;

    const instance = render(
      <TestComponent
        onReady={(approveFn) => {
          approve = approveFn;
        }}
      />
    );

    await tick(); // Wait for hook to initialize

    // Request permission
    const resultPromise = PermissionApproval.requestApproval({
      type: 'bash',
      value: 'rm -rf /',
    });

    await tick(); // Wait for state update

    // Should show pending permission
    expect(instance.lastFrame()).toContain('Pending: rm -rf /');

    // Approve
    if (approve) {
      (approve as (remember: boolean) => void)(false);
    }

    const result = await resultPromise;
    expect(result.approved).toBe(true);

    await tick(); // Wait for state update

    // Should clear pending
    expect(instance.lastFrame()).toContain('No pending permission');

    instance.unmount();
  });

  it('queues concurrent permissions and processes sequentially', async () => {
    let approve: ((remember: boolean) => void) | null = null;
    let reject: (() => void) | null = null;

    const instance = render(
      <TestComponent
        onReady={(approveFn, rejectFn) => {
          approve = approveFn;
          reject = rejectFn;
        }}
      />
    );

    await tick(); // Wait for hook to initialize

    // Request two permissions concurrently
    const promise1 = PermissionApproval.requestApproval({
      type: 'bash',
      value: 'command1',
    });

    await tick(); // Wait for first to show

    const promise2 = PermissionApproval.requestApproval({
      type: 'bash',
      value: 'command2',
    });

    await tick(); // Wait for second to queue

    // Should show FIRST permission, not second
    expect(instance.lastFrame()).toContain('Pending: command1');

    // Approve first
    if (approve) {
      (approve as (remember: boolean) => void)(false);
    }

    const result1 = await promise1;
    expect(result1.approved).toBe(true);

    await tick(); // Wait for queue to process next

    // Should now show SECOND permission
    expect(instance.lastFrame()).toContain('Pending: command2');

    // Reject second
    if (reject) {
      (reject as () => void)();
    }

    const result2 = await promise2;
    expect(result2.approved).toBe(false);

    await tick(); // Wait for state update

    // Should be cleared
    expect(instance.lastFrame()).toContain('No pending permission');

    instance.unmount();
  });

  it('handles three concurrent permissions in order', async () => {
    let approve: ((remember: boolean) => void) | null = null;

    const instance = render(
      <TestComponent
        onReady={(approveFn) => {
          approve = approveFn;
        }}
      />
    );

    await tick();

    // Request three permissions
    const promise1 = PermissionApproval.requestApproval({ type: 'bash', value: 'cmd1' });
    await tick();

    const promise2 = PermissionApproval.requestApproval({ type: 'bash', value: 'cmd2' });
    await tick();

    const promise3 = PermissionApproval.requestApproval({ type: 'bash', value: 'cmd3' });
    await tick();

    // First should be shown
    expect(instance.lastFrame()).toContain('Pending: cmd1');

    // Approve first
    if (approve) (approve as (remember: boolean) => void)(false);
    await promise1;
    await tick();

    // Second should be shown
    expect(instance.lastFrame()).toContain('Pending: cmd2');

    // Approve second
    if (approve) (approve as (remember: boolean) => void)(false);
    await promise2;
    await tick();

    // Third should be shown
    expect(instance.lastFrame()).toContain('Pending: cmd3');

    // Approve third
    if (approve) (approve as (remember: boolean) => void)(false);
    await promise3;
    await tick();

    // All cleared
    expect(instance.lastFrame()).toContain('No pending permission');

    instance.unmount();
  });

  it('rejects all pending permissions on cleanup', async () => {
    const instance = render(<TestComponent />);
    await tick();

    // Request two permissions
    const promise1 = PermissionApproval.requestApproval({ type: 'bash', value: 'cmd1' });
    await tick();

    const promise2 = PermissionApproval.requestApproval({ type: 'bash', value: 'cmd2' });
    await tick();

    // Unmount (cleanup)
    instance.unmount();
    await tick();

    // Both should be rejected
    const result1 = await promise1;
    const result2 = await promise2;

    expect(result1.approved).toBe(false);
    expect(result2.approved).toBe(false);
  });
});
