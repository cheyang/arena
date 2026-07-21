#!/usr/bin/env bash
# Static/grep-based verification of findings that are not Go-unit-testable.
# Run from the repo root of a checkout at the stack tip (pr-1483 / verify branch).
# Prints CONFIRMED / NOT-REPRODUCED per finding. Exit 0 always (informational).
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"
fail=0

echo "== PR#1483-F1: e2e suite not gated by //go:build v2e2e =="
ungated=$(grep -rL 'go:build v2e2e' test/e2e/*.go 2>/dev/null)
if [ -n "$ungated" ]; then
  echo "CONFIRMED: these test/e2e files lack the v2e2e build constraint the Makefile passes (-tags v2e2e):"
  echo "$ungated" | sed 's/^/    /'
  echo "    => plain 'go test ./...' compiles & runs the cluster-dependent suite and hard-fails in BeforeSuite."
  fail=1
else
  echo "NOT-REPRODUCED: all test/e2e files carry //go:build v2e2e"
fi
echo

echo "== PR#1480: constants.LabelJobRole defined but unused in production =="
defs=$(grep -rn 'LabelJobRole' --include='*.go' . | grep -v '_test.go')
uses=$(echo "$defs" | grep -v 'pkg/constants/labels.go' || true)
echo "$defs" | sed 's/^/    /'
if [ -z "$uses" ]; then
  echo "CONFIRMED: LabelJobRole (training.kubeflow.org/job-role) is declared but never referenced in non-test production code."
  echo "    The MPI log selector uses replica-type=launcher instead (see pkg/provider/mpi.go)."
  fail=1
else
  echo "NOT-REPRODUCED: LabelJobRole is used in production:"; echo "$uses" | sed 's/^/    /'
fi
echo

echo "== PR#1482-F2: 'delete -f' ignores YAML namespace (asymmetric with 'run') =="
del=$(grep -n 'resolveNS(' pkg/cli/delete.go 2>/dev/null)
run=$(grep -n 'resolveNS(' pkg/cli/run.go 2>/dev/null)
echo "  delete.go: $del"
echo "  run.go:    $run"
if echo "$del" | grep -q 'resolveNS("")' && echo "$run" | grep -q 'resolveNS(t.Namespace)'; then
  echo "CONFIRMED: delete resolves namespace as resolveNS(\"\") (drops the file's namespace),"
  echo "    while run uses resolveNS(t.Namespace). 'delete -f job.yaml' targets the wrong namespace."
  fail=1
else
  echo "NOT-REPRODUCED / shape changed: inspect the two lines above manually."
fi
echo

echo "== PR#1477 & go.mod: unexpected toolchain bump inside a chore PR =="
echo -n "  go directive at tip: "; grep -m1 '^go ' go.mod
echo "  (informational: go.mod pins go 1.26.1; sandbox toolchain auto-downloads it. Flagged in find stage.)"
echo

echo "OVERALL static findings reproduced: $fail (1 = at least one confirmed)"
exit 0
