#!/usr/bin/env bash
# ctx-lint.sh — enforces the ctx-doctrine: every Resolve*, GetBy*, FindBy*, LookupBy*
# method that accesses per-tenant data must take ctx context.Context as its first param.
#
# Usage:
#   bash scripts/ctx-lint.sh           # run from engine root
#   bash scripts/ctx-lint.sh internal/ # override search path
#
# Exit codes:
#   0 — no violations found
#   1 — violations found (human-readable message printed)
#
# Cross-platform: works on macOS, Linux, and Windows (Git Bash / WSL).

set -euo pipefail

SEARCH_PATH="${1:-internal/}"

# Pattern: function signature for Resolve*/GetBy*/FindBy*/LookupBy* methods
# that do NOT start with ctx context.Context as the first param.
#
# Matches:  func (r *Foo) ResolveBar(name string)
# Matches:  func (r *Foo) GetByName(id string, other int)
# Does NOT match: func (r *Foo) ResolveBar(ctx context.Context, name string)
# Does NOT match: func (r *Foo) ResolveBar(_ context.Context, name string)
VIOLATIONS=$(grep -rn \
  'func ([^)]\+) \(Resolve[A-Z]\|GetBy[A-Z]\|FindBy[A-Z]\|LookupBy[A-Z]\)[^(]*([ ]*[a-zA-Z_][a-zA-Z0-9_]* [^c]' \
  --include="*.go" \
  "${SEARCH_PATH}" \
  | grep -v "_test\.go" \
  | grep -v "ctx context\.Context\|_ context\.Context" \
  || true)

if [ -z "${VIOLATIONS}" ]; then
  echo "ctx-lint: OK — all Resolve*/GetBy*/FindBy*/LookupBy* methods have ctx as first param."
  exit 0
fi

echo ""
echo "ctx-lint: FAIL — the following methods access per-tenant data but are missing"
echo "          'ctx context.Context' as their first parameter."
echo ""
echo "  Doctrine: every method that resolves tenant-scoped data must accept ctx so"
echo "  multi-tenant registries can dispatch to the caller's tenant_id."
echo "  See docs/architecture/ctx-doctrine.md for full rules and exceptions."
echo ""
echo "Violations:"
echo "${VIOLATIONS}" | while IFS= read -r line; do
  echo "  ${line}"
done
echo ""
echo "Fix: add 'ctx context.Context' (or '_ context.Context' if truly tenant-global)"
echo "     as the first parameter to each listed method."
echo ""
exit 1
