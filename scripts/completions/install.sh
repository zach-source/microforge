#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -n "${ZSH_VERSION:-}" ]]; then
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/mforge.zsh"
  return 0 2>/dev/null || exit 0
fi

if [[ -n "${BASH_VERSION:-}" ]]; then
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/mforge.bash"
  return 0 2>/dev/null || exit 0
fi

case "${SHELL:-}" in
  */zsh)
    # shellcheck disable=SC1091
    source "$SCRIPT_DIR/mforge.zsh"
    ;;
  */bash)
    # shellcheck disable=SC1091
    source "$SCRIPT_DIR/mforge.bash"
    ;;
  *)
    echo "Unsupported shell. Source either mforge.bash or mforge.zsh directly." 1>&2
    exit 1
    ;;
esac
