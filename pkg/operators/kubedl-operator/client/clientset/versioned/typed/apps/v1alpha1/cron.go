/*
Copyright 2021 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/kubeflow/arena/pkg/operators/kubedl-operator/apis/apps/v1alpha1"
	scheme "github.com/kubeflow/arena/pkg/operators/kubedl-operator/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CronsGetter has a method to return a CronInterface.
// A group's client should implement this interface.
type CronsGetter interface {
	Crons(namespace string) CronInterface
}

// CronInterface has methods to work with Cron resources.
type CronInterface interface {
	Create(ctx context.Context, cron *v1alpha1.Cron, opts v1.CreateOptions) (*v1alpha1.Cron, error)
	Update(ctx context.Context, cron *v1alpha1.Cron, opts v1.UpdateOptions) (*v1alpha1.Cron, error)
	UpdateStatus(ctx context.Context, cron *v1alpha1.Cron, opts v1.UpdateOptions) (*v1alpha1.Cron, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Cron, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.CronList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Cron, err error)
	CronExpansion
}

// crons implements CronInterface
type crons struct {
	client rest.Interface
	ns     string
}

// newCrons returns a Crons
func newCrons(c *AppsV1alpha1Client, namespace string) *crons {
	return &crons{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the cron, and returns the corresponding cron object, and an error if there is any.
func (c *crons) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Cron, err error) {
	result = &v1alpha1.Cron{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("crons").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Crons that match those selectors.
func (c *crons) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.CronList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CronList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("crons").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested crons.
func (c *crons) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("crons").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a cron and creates it.  Returns the server's representation of the cron, and an error, if there is any.
func (c *crons) Create(ctx context.Context, cron *v1alpha1.Cron, opts v1.CreateOptions) (result *v1alpha1.Cron, err error) {
	result = &v1alpha1.Cron{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("crons").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cron).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a cron and updates it. Returns the server's representation of the cron, and an error, if there is any.
func (c *crons) Update(ctx context.Context, cron *v1alpha1.Cron, opts v1.UpdateOptions) (result *v1alpha1.Cron, err error) {
	result = &v1alpha1.Cron{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("crons").
		Name(cron.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cron).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *crons) UpdateStatus(ctx context.Context, cron *v1alpha1.Cron, opts v1.UpdateOptions) (result *v1alpha1.Cron, err error) {
	result = &v1alpha1.Cron{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("crons").
		Name(cron.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cron).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the cron and deletes it. Returns an error if one occurs.
func (c *crons) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("crons").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *crons) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("crons").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched cron.
func (c *crons) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Cron, err error) {
	result = &v1alpha1.Cron{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("crons").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
