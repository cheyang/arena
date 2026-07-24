#!/usr/bin/env bash
# Re-verify PR #1486 findings. Additive tests only; production untouched.
# Resolves nothing from args — the harness lives at the current tree.
set -uo pipefail
export GOTOOLCHAIN=auto
cd "$(git rev-parse --show-toplevel)"

echo "== HEAD: $(git rev-parse --short HEAD) =="
echo
echo "===== unit contract/canary findings ====="
go test ./pkg/task/ ./pkg/client/ ./pkg/cli/ ./pkg/provider/ -run TestVerify_ -v 2>&1 \
  | grep -E "^(=== RUN|--- (PASS|FAIL)|ok|FAIL|CONFIRMED|CANARY)"

echo
echo "===== PR1483-F1: e2e must be build-gated (this SHOULD fail until gated) ====="
go test ./test/e2e/... 2>&1 | grep -E "BeforeSuite|Ran [0-9]+ of|FAIL|ok" | head -5

echo
echo "Legend: CONTRACT red = bug still present; CANARY green = behavior unchanged."
echo "Expected at head 2c0084fe: PR1482-F1 PASS (fixed), PR1480 PASS (canary),"
echo "  PR1481-F1/PR1479-F1/PR1478-F2/PR1478-F5 FAIL (open), e2e probe FAIL (PR1483-F1)."
