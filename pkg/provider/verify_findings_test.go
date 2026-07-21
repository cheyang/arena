package provider

// Verification harness — see pkg/task/verify_findings_test.go for polarity legend.
// ADDITIVE ONLY.

import (
	"strings"
	"testing"

	"github.com/kubeflow/arena/pkg/constants"
)

// PR#1480 (mpi.go GetLogPodSelector) — CANARY.
// Documents & locks the CURRENT behavior: the MPI launcher log selector keys on
// training.kubeflow.org/replica-type=launcher. The standalone mpi-operator
// (kubeflow.org/v2beta1) labels launcher/worker pods with
// training.kubeflow.org/job-role instead — and this PR *defines*
// constants.LabelJobRole but never uses it in production code (see
// scripts/verify-static.sh), strong evidence the selector was meant to use job-role.
// If the operator contract is confirmed, invert this canary to a CONTRACT test
// asserting LabelJobRole and the selector fix.
func TestVerify_PR1480_MPILogSelector_CurrentlyUsesReplicaType(t *testing.T) {
	p := &MPIProvider{}
	sel := p.GetLogPodSelector("myjob")

	usesReplicaType := strings.Contains(sel, constants.LabelReplicaType+"="+constants.ReplicaRoleLauncher)
	usesJobRole := strings.Contains(sel, constants.LabelJobRole)

	if !usesReplicaType || usesJobRole {
		// Canary flipped: behavior changed (likely fixed). Re-evaluate & invert.
		t.Fatalf("CANARY FLIPPED (re-evaluate): MPI selector no longer uses replica-type=launcher exclusively: %q", sel)
	}
	t.Logf("CONFIRMED current behavior (canary green): MPI log selector = %q; constants.LabelJobRole (%q) is unused in production",
		sel, constants.LabelJobRole)
}
