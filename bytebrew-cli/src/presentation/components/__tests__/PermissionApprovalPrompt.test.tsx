import { describe, it, expect, mock, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { PermissionApprovalPrompt } from '../PermissionApprovalPrompt.js';

const tick = () => new Promise(r => setTimeout(r, 10));

describe('PermissionApprovalPrompt', () => {
  let instance: ReturnType<typeof render> | null = null;

  afterEach(() => {
    instance?.unmount();
    instance = null;
  });

  const bashRequest = { type: 'bash' as const, value: 'ls -la' };
  const readRequest = { type: 'read' as const, value: 'src/config.ts' };
  const editRequest = { type: 'edit' as const, value: 'src/main.ts' };

  // --- Rendering ---

  describe('rendering', () => {
    it('shows permission type label', () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      expect(instance.lastFrame()).toContain('Command Execution');
    });

    it('shows file read label', () => {
      instance = render(
        <PermissionApprovalPrompt request={readRequest} onApprove={() => {}} onReject={() => {}} />
      );
      expect(instance.lastFrame()).toContain('File Read');
    });

    it('shows file edit label', () => {
      instance = render(
        <PermissionApprovalPrompt request={editRequest} onApprove={() => {}} onReject={() => {}} />
      );
      expect(instance.lastFrame()).toContain('File Edit');
    });

    it('shows the command/path value', () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      expect(instance.lastFrame()).toContain('ls -la');
    });

    it('shows all three options', () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      const frame = instance.lastFrame();
      expect(frame).toContain('Allow once');
      expect(frame).toContain('Always allow');
      expect(frame).toContain('Deny');
    });

    it('shows agent ID when provided', () => {
      instance = render(
        <PermissionApprovalPrompt
          request={bashRequest}
          onApprove={() => {}}
          onReject={() => {}}
          agentId="code-agent-1"
        />
      );
      expect(instance.lastFrame()).toContain('code-agent-1');
    });

    it('does not show agent ID for supervisor', () => {
      instance = render(
        <PermissionApprovalPrompt
          request={bashRequest}
          onApprove={() => {}}
          onReject={() => {}}
          agentId="supervisor"
        />
      );
      expect(instance.lastFrame()).not.toContain('supervisor');
    });
  });

  // --- Number key shortcuts ---

  describe('number keys', () => {
    it('[1] approves once (remember=false)', () => {
      const onApprove = mock(() => {});
      const onReject = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={onApprove} onReject={onReject} />
      );
      instance.stdin.write('1');

      expect(onApprove).toHaveBeenCalledTimes(1);
      expect(onApprove).toHaveBeenCalledWith(false);
      expect(onReject).not.toHaveBeenCalled();
    });

    it('[2] approves always (remember=true)', () => {
      const onApprove = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={onApprove} onReject={() => {}} />
      );
      instance.stdin.write('2');

      expect(onApprove).toHaveBeenCalledTimes(1);
      expect(onApprove).toHaveBeenCalledWith(true);
    });

    it('[3] rejects', () => {
      const onReject = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={onReject} />
      );
      instance.stdin.write('3');

      expect(onReject).toHaveBeenCalledTimes(1);
    });
  });

  // --- Arrow navigation + Enter ---

  describe('arrow navigation', () => {
    it('default selection is "Allow once"', () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      // ">" marker indicates selected item
      expect(instance.lastFrame()).toContain('> [1] Allow once');
    });

    it('down arrow moves to "Always allow"', async () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      instance.stdin.write('\x1b[B'); // Down arrow
      await tick();
      expect(instance.lastFrame()).toContain('> [2] Always allow');
    });

    it('two down arrows moves to "Deny"', async () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      expect(instance.lastFrame()).toContain('> [3] Deny');
    });

    it('down arrow stops at "Deny" (does not wrap)', async () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      instance.stdin.write('\x1b[B'); // Down (should stay at Deny)
      await tick();
      expect(instance.lastFrame()).toContain('> [3] Deny');
    });

    it('up arrow from "Always allow" goes to "Allow once"', async () => {
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={() => {}} />
      );
      instance.stdin.write('\x1b[B'); // Down to "Always allow"
      await tick();
      instance.stdin.write('\x1b[A'); // Up to "Allow once"
      await tick();
      expect(instance.lastFrame()).toContain('> [1] Allow once');
    });

    it('Enter confirms current selection (default = once)', () => {
      const onApprove = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={onApprove} onReject={() => {}} />
      );
      instance.stdin.write('\r'); // Enter on default "once"

      expect(onApprove).toHaveBeenCalledWith(false);
    });

    it('Enter on "Always allow" approves with remember=true', async () => {
      const onApprove = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={onApprove} onReject={() => {}} />
      );
      instance.stdin.write('\x1b[B'); // Down to "Always allow"
      await tick();
      instance.stdin.write('\r');     // Enter

      expect(onApprove).toHaveBeenCalledWith(true);
    });

    it('Enter on "Deny" rejects', async () => {
      const onReject = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={onReject} />
      );
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      instance.stdin.write('\x1b[B'); // Down to "Deny"
      await tick();
      instance.stdin.write('\r');     // Enter

      expect(onReject).toHaveBeenCalledTimes(1);
    });
  });

  // --- Escape ---

  describe('escape', () => {
    it('Escape rejects', () => {
      const onReject = mock(() => {});
      instance = render(
        <PermissionApprovalPrompt request={bashRequest} onApprove={() => {}} onReject={onReject} />
      );
      instance.stdin.write('\x1b'); // Escape

      expect(onReject).toHaveBeenCalledTimes(1);
    });
  });
});
