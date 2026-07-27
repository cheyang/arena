#!/usr/bin/env bash
# F1490-3 — the BeforeSuite gate added by #1490 hardcodes deploy/training-operator in
# namespace "kubeflow" while hack/e2e-setup-cluster.sh exposes NAMESPACE (and helm
# names the deployment after the release). Static consistency check.
set -o errexit
set -o nounset
set -o pipefail

SUITE="${1:-test/e2e/suite_test.go}"
SETUP="${2:-hack/e2e-setup-cluster.sh}"
[ -f "$SUITE" ] || { echo "SKIP: $SUITE absent"; exit 0; }

echo "=== F1490-3: setup/gate namespace agreement ==="
if ! grep -q 'deploy/training-operator' "$SUITE"; then
  echo "SKIP: no training-operator gate in $SUITE"
  exit 0
fi
grep -n 'deploy/training-operator' "$SUITE" | sed 's/^/  gate: /'

fail=0
if grep -qE '"kubeflow"' "$SUITE"; then
  echo "  gate namespace is the literal \"kubeflow\""
  if [ -f "$SETUP" ] && grep -q 'NAMESPACE=' "$SETUP"; then
    echo "  FAIL: $SETUP makes the namespace configurable (NAMESPACE=\${NAMESPACE:-kubeflow})"
    echo "        but the gate ignores it, so NAMESPACE=<other> provisions a working cluster"
    echo "        that the suite then refuses to run against."
    fail=1
  fi
fi
if [ -f "$SETUP" ] && grep -q 'helm install training-operator' "$SETUP"; then
  if ! grep -qE 'kubectl wait.*deploy' "$SETUP" || \
     ! awk '/helm\)/,/;;/' "$SETUP" | grep -q 'kubectl wait'; then
    echo "  NOTE: the kustomize branch waits for deploy/training-operator; the helm branch"
    echo "        relies on 'helm --wait' only, and the chart may name the Deployment after"
    echo "        the release, so the gate's fixed name is not guaranteed there either."
  fi
fi
[ "$fail" -eq 0 ] && echo "RESULT: F1490-3 not detected" || echo "RESULT: F1490-3 STILL PRESENT"
exit $fail
