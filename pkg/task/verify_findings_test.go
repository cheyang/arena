package task

// Verification harness for arena v2 PR #1486 (author vicoooo26), which combines
// former PRs #1481/#1482/#1483 (and carries the full cumulative v2 tree).
// ADDITIVE ONLY — production code is untouched. Each test encodes one review
// finding with an explicit polarity:
//   CONTRACT = asserts intended-correct behavior; FAILS on buggy code, PASSES once fixed.
//   CANARY   = asserts current (observed) behavior; PASSES now, invert once fixed.
// A red result IS the reproduction of the finding.

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// PR#1482-F1 (setter.go, --set parser) — CONTRACT.
// An intermediate array index created from scratch must round-trip. The prior
// hand-rolled parser grew the intermediate slice locally but never wrote the
// grown header back to its parent, silently dropping the value. #1486 rewrote
// setter.go with updateSliceInParent write-back — this asserts the fix holds.
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
// A nested array index must not panic the process. The prior parser reached
// updateSliceInParent and indexed arr[0] on a zero-length slice.
func TestVerify_PR1482_SetNestedArray_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CONFIRMED BUG: --set matrix[0][1]=x panicked the process: %v", r)
		}
	}()
	out, err := ApplySetOverrides([]byte("name: t\n"), []string{"matrix[0][1]=x"})
	if err != nil {
		t.Logf("returned error (acceptable, not a panic): %v", err)
		return
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal merged yaml: %v", err)
	}
	t.Logf("nested-array result: matrix=%#v", m["matrix"])
}

// PR#1478-F2 (override.go tensorboard block) — CONTRACT (LATENT).
// Documents the API-level footgun: an explicit tensorboard=false should not
// materialize a non-nil TensorBoardConfig.
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
// Documents that a negative priority (legal for k8s signed int32 PriorityClass)
// cannot be applied through the override path.
func TestVerify_PR1478_PriorityNegative_Applies(t *testing.T) {
	tk := &Task{Scheduling: Scheduling{Priority: 50}}
	if err := ApplyOverrides(tk, map[string]interface{}{"priority": -100}); err != nil {
		t.Fatalf("ApplyOverrides error: %v", err)
	}
	if tk.Scheduling.Priority != -100 {
		t.Fatalf("CONFIRMED (latent): priority override to -100 dropped by setInt(n>0) guard — stayed %d", tk.Scheduling.Priority)
	}
}
