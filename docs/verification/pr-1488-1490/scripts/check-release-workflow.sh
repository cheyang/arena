#!/usr/bin/env bash
# F1488-1 / F1488-2 — static + behavioural checks on the release workflow added by
# kubeflow/arena#1488. Runs against the PR head directly (that PR is based on
# `master`, which carries release.yaml; develop-v2 does not), so it takes a ref
# rather than reading the current worktree.
#
# CONTRACT: exits non-zero while the "manual re-release" path is not idempotent.
set -o errexit
set -o nounset
set -o pipefail

REPO_URL="${REPO_URL:-https://github.com/kubeflow/arena.git}"
PR="${PR:-1488}"
REF="${1:-}"

fail=0
note() { printf '  %s\n' "$*"; }

if [ -z "$REF" ]; then
  echo "Fetching current head of PR #${PR}..."
  REF=$(git fetch -q "$REPO_URL" "pull/${PR}/head" && git rev-parse FETCH_HEAD)
fi
echo "=== F1488 checks against ${REF} ==="

WF=$(git show "${REF}:.github/workflows/release.yaml" 2>/dev/null || true)
if [ -z "$WF" ]; then
  echo "SKIP: .github/workflows/release.yaml absent on ${REF}"
  exit 0
fi

# --- F1488-2: workflow_dispatch present but unguarded --------------------------
if printf '%s' "$WF" | grep -q 'workflow_dispatch'; then
  echo "[F1488-2] workflow_dispatch trigger: present"
  if printf '%s' "$WF" | grep -qE "github\.ref *== *'refs/heads/master'|github\.ref_name *== *'master'"; then
    note "OK: a branch guard is present"
  else
    note "FAIL: no branch guard. push is limited to master + paths:VERSION, but a manual"
    note "      dispatch runs on whatever ref it is launched from and the release job uses"
    note "      target_commitish: \${{ github.sha }}, so it would tag/publish from that ref."
    fail=1
  fi
else
  echo "[F1488-2] workflow_dispatch not present — n/a"
fi

# --- F1488-1: push_tag is not idempotent --------------------------------------
if printf '%s' "$WF" | grep -qE '^\s*git tag -a'; then
  echo "[F1488-1] push_tag creates the tag with a bare 'git tag -a'"
  if printf '%s' "$WF" | grep -qE 'git rev-parse .*refs/tags|git ls-remote --tags|--force|\|\| true|if.*tag.*exists'; then
    note "OK: some existence/force guard is present"
  else
    note "FAIL: no existence guard. On a second dispatch 'git tag -a v\$VERSION' aborts with"
    note "      \"fatal: tag already exists\" (exit 128); draft_release needs push_tag so it is"
    note "      skipped, while build-arena-image/release-image have already re-pushed"
    note "      ghcr.io/kubeflow/arena:\$VERSION. Net effect: the image is mutated and no"
    note "      release is produced — exactly the re-release case the trigger was added for."
    fail=1
  fi
fi

# --- Behavioural repro of the failing step ------------------------------------
echo "[F1488-1] reproducing the git behaviour the step relies on:"
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
(
  cd "$T"
  git init -q .
  git config user.email v@example.com
  git config user.name verifier
  git commit -q --allow-empty -m init
  git tag -a v0.0.0 -m "Release v0.0.0"
  note "first  'git tag -a v0.0.0' -> exit 0"
  if git tag -a v0.0.0 -m "Release v0.0.0" 2>/dev/null; then
    note "second 'git tag -a v0.0.0' -> exit 0 (unexpected)"
  else
    note "second 'git tag -a v0.0.0' -> exit $? (fatal: tag already exists)"
  fi
)

# --- Is the tag this PR documents actually published? -------------------------
VER=$(git show "${REF}:VERSION" 2>/dev/null | tr -d '[:space:]' || true)
if [ -n "$VER" ]; then
  echo "[F1488-1] VERSION on PR head = ${VER}"
  if git ls-remote --tags "$REPO_URL" 2>/dev/null | grep -q "refs/tags/v${VER}$"; then
    note "tag v${VER} IS published upstream -> a dispatch would hit the non-idempotent path immediately"
  else
    note "tag v${VER} is NOT published upstream -> the first dispatch succeeds; the defect"
    note "bites on any subsequent one"
  fi
fi

echo
if [ "$fail" -ne 0 ]; then
  echo "RESULT: F1488 findings STILL PRESENT"
  exit 1
fi
echo "RESULT: F1488 findings appear FIXED"
