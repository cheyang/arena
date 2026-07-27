#!/usr/bin/env bash
# F1490-2 — the helm branch of hack/e2e-setup-cluster.sh is unreachable: CHART_PATH
# defaults to charts/kubeflow-trainer, which does not exist at the pinned TRAINER_REF.
# Needs network (shallow-clones the trainer repo at the pinned ref).
#
# CONTRACT: exits non-zero while INSTALL_METHOD=helm cannot possibly work.
set -o errexit
set -o nounset
set -o pipefail

SCRIPT="${1:-hack/e2e-setup-cluster.sh}"
if [ ! -f "$SCRIPT" ]; then
  echo "SKIP: $SCRIPT not present on this ref"
  exit 0
fi

default_of() { sed -n "s/^${1}=\"\${${1}:-\(.*\)}\"$/\1/p" "$SCRIPT" | head -1; }

REPO=$(default_of TRAINER_REPO)
REF=$(default_of TRAINER_REF)
CHART=$(default_of CHART_PATH)
echo "=== F1490-2: helm path reachability ==="
echo "TRAINER_REPO=${REPO}"
echo "TRAINER_REF =${REF}"
echo "CHART_PATH  =${CHART}"

if ! grep -q 'INSTALL_METHOD' "$SCRIPT" || ! grep -q 'helm)' "$SCRIPT"; then
  echo "SKIP: no helm branch in $SCRIPT"
  exit 0
fi

D=$(mktemp -d)
trap 'rm -rf "$D"' EXIT
git -c advice.detachedHead=false clone -q --depth 1 --branch "$REF" "$REPO" "$D"

if [ -d "$D/$CHART" ]; then
  echo "RESULT: chart ${CHART} EXISTS at ${REF} — helm path is reachable (FIXED)"
  exit 0
fi

echo "  charts/ dir at ${REF}: $(ls "$D/charts" 2>/dev/null || echo '<absent>')"
echo "RESULT: chart ${CHART} does NOT exist at ${REF}."
echo "        INSTALL_METHOD=helm always exits 1 with the script's own"
echo "        'may not include helm charts' error, so the branch (and the"
echo "        install_method input advertised by the composite action) is dead."
echo "        Either repoint CHART_PATH / pin a ref that ships charts, or drop the branch."
exit 1
