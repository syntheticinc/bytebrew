// Permission system types (Claude Code compatible format)

/** Actions that can be taken on a permission request */
export type PermissionAction = 'allow' | 'deny' | 'ask';

/** Known permission types */
export type PermissionType = 'bash' | 'read' | 'edit' | 'write' | 'list';

/** A permission request to be evaluated */
export interface PermissionRequest {
  /** Type of operation: bash (execute_command), read (read_file), edit (write_file, edit_file), list (get_project_tree) */
  type: PermissionType;
  /** The value to match against rules (command string, file path, etc.) */
  value: string;
  /** Agent ID that triggered this request (for auto-approval logic) */
  agentId?: string;
}

/** Result of evaluating a permission request */
export interface PermissionEvalResult {
  action: PermissionAction;
  /** Matched pattern from allow/deny lists (if any) */
  matchedPattern?: string;
}

/** Permission configuration (Claude Code compatible) */
export interface PermissionConfig {
  permissions: {
    /** Allow list - rules in "Bash(pattern)" or "Read" format */
    allow: string[];
    /** Deny list - rules in "Bash(pattern)" or "Read" format */
    deny: string[];
  };
}

/** Result of a permission check including prompt result */
export type PermissionCheckResult =
  | { allowed: true }
  | { allowed: false; reason: string };
