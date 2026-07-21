#!/usr/bin/env bash
# Re-run the verification harness after a fix and print a per-finding verdict.
#
# Usage:
#   bash scripts/re-verify.sh            # run harness in place on the current checkout
#   bash scripts/re-verify.sh <ref>      # graft the harness onto <ref> (e.g. a fixed PR head) and run there
#   bash scripts/re-verify.sh pr         # fetch the tip PR head (pull/1483/head) and run there
#
# Polarity legend (from verify-manifest.json):
#   contract test -> PASS means FIXED, FAIL means STILL-BROKEN
#   canary   test -> FAIL means FIXED (behavior changed; invert it), PASS means STILL-CURRENT
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"    # docs/verification/v2-stack
REPO="$(git -C "$HERE" rev-parse --show-toplevel)"
export GOTOOLCHAIN=auto
MANIFEST="$HERE/verify-manifest.json"

ARG="${1:-}"
WORKDIR="$REPO"
CLEANUP=""
if [ -n "$ARG" ]; then
  if [ "$ARG" = "pr" ]; then
    echo ">> fetching tip PR head (pull/1483/head) from upstream ..."
    git -C "$REPO" fetch -q https://github.com/kubeflow/arena.git pull/1483/head
    REF=FETCH_HEAD
  else
    REF="$ARG"
  fi
  LAST="$(cat "$HERE/.last-reviewed" 2>/dev/null || git -C "$REPO" merge-base "$REF" HEAD)"
  echo ">> incremental delta since .last-reviewed: ${LAST}..${REF}"
  git -C "$REPO" --no-pager log --oneline "${LAST}..${REF}" 2>/dev/null | sed 's/^/     /' || true
  WORKDIR="$(mktemp -d)"
  echo ">> grafting harness onto $REF in $WORKDIR"
  git -C "$REPO" worktree add -q --detach "$WORKDIR" "$REF"
  CLEANUP="$WORKDIR"
  for f in pkg/task/verify_findings_test.go pkg/cli/verify_findings_test.go \
           pkg/client/verify_findings_test.go pkg/provider/verify_findings_test.go; do
    mkdir -p "$WORKDIR/$(dirname "$f")"
    git -C "$REPO" show HEAD:"$f" > "$WORKDIR/$f"
  done
  mkdir -p "$WORKDIR/docs/verification/v2-stack/scripts"
  for s in verify-static.sh verify-gitignore.sh; do
    git -C "$REPO" show HEAD:"docs/verification/v2-stack/scripts/$s" > "$WORKDIR/docs/verification/v2-stack/scripts/$s"
  done
  chmod +x "$WORKDIR/docs/verification/v2-stack/scripts/"*.sh
fi

cd "$WORKDIR"
OUT="$(mktemp)"
echo "== running Go verify tests (unit layer) =="
for pkg in task cli client provider; do
  go test "./pkg/$pkg/" -run Verify -v 2>&1 | tee -a "$OUT" | grep -E '^--- (PASS|FAIL)' || true
done

echo
echo "== per-finding verdict (polarity-aware) =="
python3 - "$MANIFEST" "$OUT" <<'PY'
import json,sys,re
manifest,outfile=sys.argv[1],sys.argv[2]
m=json.load(open(manifest))
txt=open(outfile).read()
def status(testname):
    if re.search(r'^--- PASS: %s' % re.escape(testname), txt, re.M): return "PASS"
    if re.search(r'^--- FAIL: %s' % re.escape(testname), txt, re.M): return "FAIL"
    return "N/A"
allfixed=True
for f in m["findings"]:
    goc=[t for t in f["tests"] if t.startswith("pkg/")]
    if not goc:
        print(f"  [manual] {f['id']:32s} ({f['polarity']}) -> run scripts/verify-static.sh / verify-gitignore.sh")
        continue
    verds=[]
    for t in goc:
        name=t.split("-run ",1)[1] if "-run " in t else t
        st=status(name)
        if f["polarity"]=="contract":
            v="FIXED" if st=="PASS" else ("STILL-BROKEN" if st=="FAIL" else "HARNESS-UPDATE")
        else:  # canary
            v="FIXED(flip-invert)" if st=="FAIL" else ("STILL-CURRENT" if st=="PASS" else "HARNESS-UPDATE")
        verds.append(f"{name.split('_',2)[-1] if '_' in name else name}:{st}->{v}")
        if v not in ("FIXED","FIXED(flip-invert)"): allfixed=False
    print(f"  {f['id']:32s} ({f['polarity']:8s}) {'; '.join(verds)}")
print()
print("ALL findings resolved." if allfixed else "Some findings NOT yet resolved (see above).")
sys.exit(0 if allfixed else 1)
PY
rc=$?

echo
echo "== static / script layer =="
bash "$HERE/scripts/verify-static.sh"    2>/dev/null | grep -E 'CONFIRMED|NOT-REPRODUCED' | sed 's/^/  /'
bash "$HERE/scripts/verify-gitignore.sh" 2>/dev/null | grep -E 'CONFIRMED|NOT-REPRODUCED' | sed 's/^/  /'

echo
echo ">> to advance the review point after a clean round:"
echo "     git rev-parse ${ARG:-HEAD} > $HERE/.last-reviewed && git add -A && git commit -m 'verify: advance .last-reviewed'"

[ -n "$CLEANUP" ] && git -C "$REPO" worktree remove --force "$CLEANUP" >/dev/null 2>&1
exit $rc
