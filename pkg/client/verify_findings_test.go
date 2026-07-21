package client

// Verification harness — see pkg/task/verify_findings_test.go for polarity legend.
// ADDITIVE ONLY.

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

// PR#1479-F1 (crd.go ResolveMPIVersion) — CONTRACT.
// GetCRDVersions returns the wrapped sentinel ErrCRDNotFound, but ResolveMPIVersion
// re-wraps it as a bare fmt.Errorf without %w, severing the chain exactly at the
// layer callers use to distinguish "operator not installed" (actionable) from a
// transient/RBAC API error. errors.Is must still see ErrCRDNotFound.
func TestVerify_PR1479_ResolveMPIVersion_PreservesSentinel(t *testing.T) {
	// Fake dynamic client with NO mpijobs CRD object -> GetCRDVersions => ErrCRDNotFound.
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}: "CustomResourceDefinitionList",
		})
	c := NewClientForInterface(fakeClient)

	err := c.ResolveMPIVersion(context.Background())
	if err == nil {
		t.Fatalf("expected an error when MPIJob CRD is absent, got nil")
	}
	if !errors.Is(err, ErrCRDNotFound) {
		t.Fatalf("CONFIRMED BUG: ResolveMPIVersion dropped the ErrCRDNotFound sentinel (errors.Is == false); err=%q", err.Error())
	}
}
