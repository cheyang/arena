package cli

// Verification harness — see pkg/task/verify_findings_test.go for polarity legend.
// ADDITIVE ONLY.

import (
	"testing"

	"github.com/kubeflow/arena/pkg/task"
)

// PR#1481-F1 (submit.go buildSubmitTask + override.go ensureWorker) — CONTRACT.
// The common single-node PyTorch case that requests GPUs must submit.
// buildSubmitTask sets master-only (Worker=nil) for --workers<=1; then
// ApplyOverrides.ensureWorker() resurrects Worker{Replicas:0} for --gpus,
// and Validate rejects it with "worker.replicas must be > 0, got 0".
// Drives the full submit -> override -> defaults -> validate pipeline.
func TestVerify_PR1481_PytorchSingleNodeWithGPU_Validates(t *testing.T) {
	oName, oImage, oWorkers, oGPUs := submitName, submitImage, submitWorkers, submitGPUs
	defer func() { submitName, submitImage, submitWorkers, submitGPUs = oName, oImage, oWorkers, oGPUs }()

	submitName = "n"
	submitImage = "busybox"
	submitWorkers = 1 // single-node (the default)
	submitGPUs = 1    // request a GPU

	tk := buildSubmitTask("pytorch", []string{"python", "train.py"})
	if err := task.ApplyOverrides(tk, buildSubmitFlags()); err != nil {
		t.Fatalf("ApplyOverrides error: %v", err)
	}
	tk.SetDefaults()
	if err := task.Validate(tk); err != nil {
		t.Fatalf("CONFIRMED BUG: single-node pytorch + --gpus fails validation end-to-end: %v", err)
	}
}
