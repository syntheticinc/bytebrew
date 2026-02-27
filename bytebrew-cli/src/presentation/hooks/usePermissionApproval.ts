// usePermissionApproval hook - manages permission approval state in interactive mode
import { useState, useEffect, useCallback, useRef } from 'react';
import { PermissionApproval, ApprovalResult } from '../../infrastructure/permission/PermissionApproval.js';
import { PermissionRequest } from '../../domain/permission/Permission.js';

export interface PendingPermission {
  request: PermissionRequest;
  resolve: (result: ApprovalResult) => void;
}

export interface UsePermissionApprovalResult {
  pendingPermission: PendingPermission | null;
  approve: (remember: boolean) => void;
  reject: () => void;
}

/**
 * Hook that connects PermissionApproval to React state.
 * When a permission check results in "ask", it sets pendingPermission state
 * which triggers UI to show the approval dialog.
 *
 * Handles concurrent permission requests via queue - if there's a pending
 * permission, new requests are queued and processed sequentially.
 */
export function usePermissionApproval(): UsePermissionApprovalResult {
  // Queue for pending permissions when one is already being shown
  const queueRef = useRef<PendingPermission[]>([]);
  // Currently displayed permission
  const currentRef = useRef<PendingPermission | null>(null);
  // React state for rendering (kept as single value to preserve external interface)
  const [pendingPermission, setPendingPermission] = useState<PendingPermission | null>(null);

  useEffect(() => {
    const approvalCallback = async (request: PermissionRequest): Promise<ApprovalResult> => {
      return new Promise<ApprovalResult>((resolve) => {
        const pending: PendingPermission = { request, resolve };

        if (!currentRef.current) {
          // No current permission - show immediately
          currentRef.current = pending;
          setPendingPermission(pending);
        } else {
          // There's a current permission - queue this one
          queueRef.current.push(pending);
        }
      });
    };

    PermissionApproval.setInteractiveMode(approvalCallback);

    return () => {
      PermissionApproval.setHeadlessMode();

      // Reject current permission if any
      if (currentRef.current) {
        currentRef.current.resolve({ approved: false, remember: false });
        currentRef.current = null;
      }

      // Reject all queued permissions
      queueRef.current.forEach(p => p.resolve({ approved: false, remember: false }));
      queueRef.current = [];

      setPendingPermission(null);
    };
  }, []);

  const approve = useCallback((remember: boolean) => {
    if (currentRef.current) {
      // Resolve current permission
      currentRef.current.resolve({ approved: true, remember });

      // Move to next permission in queue
      const next = queueRef.current.shift() || null;
      currentRef.current = next;
      setPendingPermission(next);
    }
  }, []);

  const reject = useCallback(() => {
    if (currentRef.current) {
      // Resolve current permission
      currentRef.current.resolve({ approved: false, remember: false });

      // Move to next permission in queue
      const next = queueRef.current.shift() || null;
      currentRef.current = next;
      setPendingPermission(next);
    }
  }, []);

  return {
    pendingPermission,
    approve,
    reject,
  };
}
