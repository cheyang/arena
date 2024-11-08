// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RayServiceStatusesApplyConfiguration represents an declarative configuration of the RayServiceStatuses type for use
// with apply.
type RayServiceStatusesApplyConfiguration struct {
	LastUpdateTime       *v1.Time                            `json:"lastUpdateTime,omitempty"`
	ServiceStatus        *rayv1.ServiceStatus                `json:"serviceStatus,omitempty"`
	ActiveServiceStatus  *RayServiceStatusApplyConfiguration `json:"activeServiceStatus,omitempty"`
	PendingServiceStatus *RayServiceStatusApplyConfiguration `json:"pendingServiceStatus,omitempty"`
	NumServeEndpoints    *int32                              `json:"numServeEndpoints,omitempty"`
	ObservedGeneration   *int64                              `json:"observedGeneration,omitempty"`
}

// RayServiceStatusesApplyConfiguration constructs an declarative configuration of the RayServiceStatuses type for use with
// apply.
func RayServiceStatuses() *RayServiceStatusesApplyConfiguration {
	return &RayServiceStatusesApplyConfiguration{}
}

// WithLastUpdateTime sets the LastUpdateTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastUpdateTime field is set to the value of the last call.
func (b *RayServiceStatusesApplyConfiguration) WithLastUpdateTime(value v1.Time) *RayServiceStatusesApplyConfiguration {
	b.LastUpdateTime = &value
	return b
}

// WithServiceStatus sets the ServiceStatus field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ServiceStatus field is set to the value of the last call.
func (b *RayServiceStatusesApplyConfiguration) WithServiceStatus(value rayv1.ServiceStatus) *RayServiceStatusesApplyConfiguration {
	b.ServiceStatus = &value
	return b
}

// WithActiveServiceStatus sets the ActiveServiceStatus field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ActiveServiceStatus field is set to the value of the last call.
func (b *RayServiceStatusesApplyConfiguration) WithActiveServiceStatus(value *RayServiceStatusApplyConfiguration) *RayServiceStatusesApplyConfiguration {
	b.ActiveServiceStatus = value
	return b
}

// WithPendingServiceStatus sets the PendingServiceStatus field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PendingServiceStatus field is set to the value of the last call.
func (b *RayServiceStatusesApplyConfiguration) WithPendingServiceStatus(value *RayServiceStatusApplyConfiguration) *RayServiceStatusesApplyConfiguration {
	b.PendingServiceStatus = value
	return b
}

// WithNumServeEndpoints sets the NumServeEndpoints field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NumServeEndpoints field is set to the value of the last call.
func (b *RayServiceStatusesApplyConfiguration) WithNumServeEndpoints(value int32) *RayServiceStatusesApplyConfiguration {
	b.NumServeEndpoints = &value
	return b
}

// WithObservedGeneration sets the ObservedGeneration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ObservedGeneration field is set to the value of the last call.
func (b *RayServiceStatusesApplyConfiguration) WithObservedGeneration(value int64) *RayServiceStatusesApplyConfiguration {
	b.ObservedGeneration = &value
	return b
}
