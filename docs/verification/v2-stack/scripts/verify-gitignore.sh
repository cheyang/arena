#!/usr/bin/env bash
# PR#1477-F1: the unanchored `arena-v2` .gitignore pattern matches the
# cmd/arena-v2/ SOURCE directory (not just the built binary), so a contributor's
# `git add` would silently omit the whole v2 command package.
# Reproduces in an isolated throwaway repo using the PR's actual .gitignore.
set -uo pipefail
root="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cd "$tmp"
git init -q
cp "$root/.gitignore" .gitignore
mkdir -p cmd/arena-v2
: > cmd/arena-v2/main.go

echo "== .gitignore lines mentioning arena-v2 / bin =="
grep -nE 'arena-v2|(^|/)bin/?$' "$root/.gitignore" || true
echo

echo "== git check-ignore against cmd/arena-v2 source =="
if git check-ignore -v cmd/arena-v2/main.go cmd/arena-v2 2>/dev/null; then
  echo "CONFIRMED: the source path cmd/arena-v2/ is ignored by the unanchored pattern."
  echo "    Fix: anchor the binary rule to '/arena-v2' (bin/arena-v2 is already covered by bin/)."
  exit 1
else
  echo "NOT-REPRODUCED: cmd/arena-v2 source is not ignored (pattern may have been anchored)."
  exit 0
fi
