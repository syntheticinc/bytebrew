#!/usr/bin/env bash
# slog-lint.sh — enforces the observability doctrine: every slog call in
# production code must pass a context.Context as its first argument so
# tenant_id / user_sub / trace_id are automatically included in log lines.
#
# Usage:
#   bash scripts/slog-lint.sh           # run from engine root
#   bash scripts/slog-lint.sh internal/ # override search path
#
# Exit codes:
#   0 — no violations found
#   1 — violations found (human-readable message printed)
#
# Cross-platform: works on macOS, Linux, and Windows (Git Bash / WSL).

set -euo pipefail

SEARCH_PATH="${1:-internal/}"

# Pattern: bare slog.Info/Warn/Error/Debug calls that do NOT use the Context
# variant. Matches:
#   slog.Info("msg", ...)
#   slog.Warn("msg", ...)
#   slog.Error("msg", ...)
#   slog.Debug("msg", ...)
# Does NOT match:
#   slog.InfoContext(...)
#   slog.WarnContext(...)
#   slog.ErrorContext(...)
#   slog.DebugContext(...)
VIOLATIONS=$(grep -rn \
  'slog\.\(Info\|Warn\|Error\|Debug\)("' \
  --include="*.go" \
  "${SEARCH_PATH}" \
  | grep -v "_test\.go" \
  || true)

if [ -z "${VIOLATIONS}" ]; then
  echo "slog-lint: OK — all slog calls use the Context variant."
  exit 0
fi

echo ""
echo "slog-lint: FAIL — the following slog calls are missing a context.Context argument."
echo ""
echo "  Doctrine: every slog call in production code must use slog.InfoContext,"
echo "  slog.WarnContext, slog.ErrorContext, or slog.DebugContext so that"
echo "  tenant_id / user_sub / trace_id are included in structured log lines."
echo "  Use context.Background() when no request context is available."
echo ""
echo "Violations:"
echo "${VIOLATIONS}" | while IFS= read -r line; do
  echo "  ${line}"
done
echo ""
echo "Fix: replace slog.Info(\"...\") with slog.InfoContext(ctx, \"...\")"
echo "     (use context.Background() if no ctx is in scope)"
echo ""
exit 1
