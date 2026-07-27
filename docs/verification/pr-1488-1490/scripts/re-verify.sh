#!/usr/bin/env bash
# Re-run the whole harness for kubeflow/arena #1488 / #1489 / #1490.
#
#   bash docs/verification/pr-1488-1490/scripts/re-verify.sh            # all three PRs, current heads
#   bash docs/verification/pr-1488-1490/scripts/re-verify.sh 1490       # one PR
#   bash docs/verification/pr-1488-1490/scripts/re-verify.sh 1490 <sha> # pin a ref
#
# Run it from a checkout of the verification branch. It resolves each PR's current
# head from verify-manifest.json (via the PR URL, so no local remote name is
# needed), grafts the harness onto a throwaway worktree of that head, and runs the
# layers that apply. Exit 0 iff every finding is fixed.
set -o nounset
set -o pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VDIR="$(dirname "$HERE")"
ROOT="$(git -C "$HERE" rev-parse --show-toplevel)"
MANIFEST="${VDIR}/verify-manifest.json"
REPO_URL="https://github.com/kubeflow/arena.git"
RESULTS="${VDIR}/results"
mkdir -p "$RESULTS"

ONLY="${1:-}"
PINNED="${2:-}"
PRS="1488 1489 1490"
[ -n "$ONLY" ] && PRS="$ONLY"

rc=0
WT_BASE="$(mktemp -d /tmp/arena-verify-XXXXXX)"
cleanup() {
  for d in "$WT_BASE"/*; do
    [ -d "$d" ] && git -C "$ROOT" worktree remove --force "$d" >/dev/null 2>&1
  done
  rm -rf "$WT_BASE"
}
trap cleanup EXIT

echo "###############################################################"
echo "# arena v2 review harness — #1488 / #1489 / #1490"
echo "# verification branch: $(git -C "$ROOT" rev-parse --abbrev-ref HEAD)"
echo "###############################################################"

for pr in $PRS; do
  echo
  echo "==============================================================="
  echo "PR #${pr}"
  echo "==============================================================="

  if [ -n "$PINNED" ]; then
    head="$PINNED"
  else
    if ! git -C "$ROOT" fetch -q "$REPO_URL" "pull/${pr}/head"; then
      echo "  ERROR: could not fetch pull/${pr}/head (PR closed? no network?)"
      rc=1
      continue
    fi
    head="$(git -C "$ROOT" rev-parse FETCH_HEAD)"
  fi
  echo "head: ${head}"

  last="$(cat "${VDIR}/.last-reviewed-${pr}" 2>/dev/null || true)"
  if [ -n "$last" ] && [ "$last" != "$head" ]; then
    echo "delta since last reviewed (${last}..${head}):"
    git -C "$ROOT" log --oneline "${last}..${head}" 2>/dev/null | sed 's/^/  /' || echo "  (cannot resolve range)"
    echo "  files:"
    git -C "$ROOT" diff --name-only "${last}..${head}" 2>/dev/null | sed 's/^/    /'
  elif [ "$last" = "$head" ]; then
    echo "unchanged since last reviewed"
  fi

  wt="${WT_BASE}/pr-${pr}"
  if ! git -C "$ROOT" worktree add -q --detach "$wt" "$head" 2>/dev/null; then
    echo "  ERROR: worktree add failed"
    rc=1
    continue
  fi

  log="${RESULTS}/pr-${pr}.log"
  : > "$log"

  # Graft the additive harness onto the PR tree. Production code stays untouched.
  # The Go layer imports pkg/task, which only exists on the v2 tree — #1488 is based
  # on master (v1 layout), so skip grafting there rather than fail on a missing import.
  if [ -d "$wt/pkg/task" ]; then
    mkdir -p "$wt/test/verify"
    cp "$ROOT/test/verify/verify_test.go" "$wt/test/verify/" 2>/dev/null || true
  else
    echo "--- L1 skipped: pkg/task absent on this ref (v1 tree), Go layer not applicable ---" | tee -a "$log"
  fi

  # ---- L1: Go layer (applies to the v2 trees: 1489 / 1490) ----
  if [ -f "$wt/go.mod" ] && [ -f "$wt/test/verify/verify_test.go" ]; then
    echo "--- L1 go test ./test/verify/ ---" | tee -a "$log"
    (cd "$wt" && go test ./test/verify/ -v 2>&1) | tee -a "$log" | \
      grep -E '^(=== RUN|--- (PASS|FAIL|SKIP)|ok|FAIL|PASS)|F14[89][0-9]-' | sed 's/^/  /'
    st=${PIPESTATUS[0]}
    [ "$st" -ne 0 ] && rc=1
  fi

  # ---- L2: shell layer ----
  case "$pr" in
    1488)
      echo "--- L2 release workflow (F1488-1, F1488-2) ---" | tee -a "$log"
      (cd "$wt" && bash "${HERE}/check-release-workflow.sh" "$head" 2>&1) | tee -a "$log" | sed 's/^/  /'
      [ "${PIPESTATUS[0]}" -ne 0 ] && rc=1
      ;;
    1490)
      echo "--- L2 trainer chart path (F1490-2) ---" | tee -a "$log"
      (cd "$wt" && bash "${HERE}/check-trainer-chart-path.sh" 2>&1) | tee -a "$log" | sed 's/^/  /'
      [ "${PIPESTATUS[0]}" -ne 0 ] && rc=1
      echo "--- L2 suite gate namespace (F1490-3) ---" | tee -a "$log"
      (cd "$wt" && bash "${HERE}/check-suite-gate-namespace.sh" 2>&1) | tee -a "$log" | sed 's/^/  /'
      [ "${PIPESTATUS[0]}" -ne 0 ] && rc=1
      echo "--- L2 go test ./... reproduction (F1490-1) ---" | tee -a "$log"
      (cd "$wt" && timeout 600 go test ./... 2>&1 | grep -E 'BeforeSuite|FAILED|^(FAIL|ok)\s+github.com/kubeflow/arena/test/e2e' | head -8) | tee -a "$log" | sed 's/^/  /'
      ;;
  esac

  echo "  full log: ${log#$ROOT/}"
  echo "  advance the marker after reviewing this round:"
  echo "    echo ${head} > docs/verification/pr-1488-1490/.last-reviewed-${pr}"
done

echo
echo "==============================================================="
if [ "$rc" -eq 0 ]; then
  echo "RESULT: all findings FIXED"
else
  echo "RESULT: at least one finding STILL PRESENT (see per-PR output above)"
  echo "NOTE: polarity matters — TestVerify_F1489_R_AllExamplesLoad is a REGRESSION guard"
  echo "      and is expected to PASS. Only the CONTRACT checks turning green means fixed."
fi
exit $rc
