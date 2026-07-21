package task

// Verification harness for arena v2 stacked PRs (author vicoooo26).
// ADDITIVE ONLY — production code is untouched. Each test encodes one review
// finding with an explicit polarity:
//   CONTRACT = asserts intended-correct behavior; FAILS on current (buggy) code, PASSES once fixed.
//   CANARY   = asserts current (observed) behavior; PASSES now, must be inverted once fixed.
// A red result IS the reproduction of the finding.

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// PR#1482-F1 (setter.go, --set parser) — CONTRACT.
// An intermediate array index must round-trip. helm.sh/helm/v3/pkg/strvals (removed
// in PR#1482) handled this; the hand-rolled parser grows the intermediate slice
// locally but never writes the grown header back to its parent, so the value is
// silently dropped. arena's schema uses lists-of-objects (storages, envs), so
// `--set storages[0].name=data` is a realistic invocation.
func TestVerify_PR1482_SetIntermediateArray_RoundTrips(t *testing.T) {
	out, err := ApplySetOverrides([]byte("name: t\n"), []string{"storages[0].name=data"})
	if err != nil {
		t.Fatalf("CONFIRMED BUG (error path): ApplySetOverrides(storages[0].name=data) returned error: %v", err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal merged yaml: %v", err)
	}
	arr, ok := m["storages"].([]interface{})
	if !ok || len(arr) != 1 {
		t.Fatalf("CONFIRMED BUG: intermediate array silently dropped — storages=%#v (want 1-elem array)", m["storages"])
	}
	obj, _ := arr[0].(map[string]interface{})
	if obj["name"] != "data" {
		t.Fatalf("CONFIRMED BUG: storages[0].name lost — got %#v", arr[0])
	}
}

// PR#1482-F1 (setter.go, --set parser) — CONTRACT.
// A nested array index must not panic the process. `matrix[0][1]=x` reaches
// updateSliceInParent which indexes arr[0] on a zero-length slice.
func TestVerify_PR1482_SetNestedArray_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CONFIRMED BUG: --set matrix[0][1]=x panicked the process: %v", r)
		}
	}()
	if _, err := ApplySetOverrides([]byte("name: t\n"), []string{"matrix[0][1]=x"}); err != nil {
		// A clean error would be acceptable behavior; a panic is not. Record it.
		t.Logf("returned error (acceptable, not a panic): %v", err)
	}
}

// PR#1478-F2 (override.go tensorboard block) — CONTRACT (LATENT).
// NOTE: not reachable via the current CLI — buildSubmitFlags() only inserts the
// "tensorboard" key when the flag is true (submit.go:372). This documents the
// API-level footgun: an explicit tensorboard=false materializes a non-nil
// TensorBoardConfig, so any caller using `Logging.TensorBoard != nil` as
// "requested" would wrongly enable it.
func TestVerify_PR1478_TensorboardFalse_StaysNil(t *testing.T) {
	tk := &Task{}
	if err := ApplyOverrides(tk, map[string]interface{}{"tensorboard": false}); err != nil {
		t.Fatalf("ApplyOverrides error: %v", err)
	}
	if tk.Logging.TensorBoard != nil {
		t.Fatalf("CONFIRMED (latent/API-level): tensorboard=false produced non-nil config %#v", tk.Logging.TensorBoard)
	}
}

// PR#1478-F5 (override.go setInt guard n>0) — CONTRACT (LATENT).
// NOTE: also CLI-gated (buildSubmitFlags only emits "priority" when >0). Documents
// that setInt cannot lower priority to 0 or set a negative priority, though k8s
// PriorityClass values are signed int32 and negatives are legal. Also affects the
// file-vs-flag precedence contract (a file value can't be overridden down to 0).
func TestVerify_PR1478_PriorityNegative_Applies(t *testing.T) {
	tk := &Task{Scheduling: Scheduling{Priority: 50}}
	if err := ApplyOverrides(tk, map[string]interface{}{"priority": -100}); err != nil {
		t.Fatalf("ApplyOverrides error: %v", err)
	}
	if tk.Scheduling.Priority != -100 {
		t.Fatalf("CONFIRMED (latent): priority override to -100 dropped by setInt(n>0) guard — stayed %d", tk.Scheduling.Priority)
	}
}
