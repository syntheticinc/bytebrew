#!/usr/bin/env bash
# Wrapper for goreleaser: uses garble if available, falls back to go.
if command -v garble &>/dev/null; then
  exec garble -literals -tiny "$@"
fi
exec go "$@"
