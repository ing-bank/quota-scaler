/*
Copyright The Kubernetes Authors.

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

package v1

import (
	"context"
	v1 "github.com/ing-bank/quota-scaler/pkg/scalerclient/apis/quotaautoscaler/v1"
	"github.com/ing-bank/quota-scaler/pkg/scalerclient/client/clientset/versioned/scheme"
	"time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// QuotaAutoscalersGetter has a method to return a QuotaAutoscalerInterface.
// A group's client should implement this interface.
type QuotaAutoscalersGetter interface {
	QuotaAutoscalers(namespace string) QuotaAutoscalerInterface
}

// QuotaAutoscalerInterface has methods to work with QuotaAutoscaler resources.
type QuotaAutoscalerInterface interface {
	Create(ctx context.Context, quotaAutoscaler *v1.QuotaAutoscaler, opts metav1.CreateOptions) (*v1.QuotaAutoscaler, error)
	Update(ctx context.Context, quotaAutoscaler *v1.QuotaAutoscaler, opts metav1.UpdateOptions) (*v1.QuotaAutoscaler, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.QuotaAutoscaler, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.QuotaAutoscalerList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.QuotaAutoscaler, err error)
	QuotaAutoscalerExpansion
}

// quotaAutoscalers implements QuotaAutoscalerInterface
type quotaAutoscalers struct {
	client rest.Interface
	ns     string
}

// newQuotaAutoscalers returns a QuotaAutoscalers
func newQuotaAutoscalers(c *IchpV1Client, namespace string) *quotaAutoscalers {
	return &quotaAutoscalers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the quotaAutoscaler, and returns the corresponding quotaAutoscaler object, and an error if there is any.
func (c *quotaAutoscalers) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.QuotaAutoscaler, err error) {
	result = &v1.QuotaAutoscaler{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of QuotaAutoscalers that match those selectors.
func (c *quotaAutoscalers) List(ctx context.Context, opts metav1.ListOptions) (result *v1.QuotaAutoscalerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.QuotaAutoscalerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested quotaAutoscalers.
func (c *quotaAutoscalers) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a quotaAutoscaler and creates it.  Returns the server's representation of the quotaAutoscaler, and an error, if there is any.
func (c *quotaAutoscalers) Create(ctx context.Context, quotaAutoscaler *v1.QuotaAutoscaler, opts metav1.CreateOptions) (result *v1.QuotaAutoscaler, err error) {
	result = &v1.QuotaAutoscaler{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(quotaAutoscaler).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a quotaAutoscaler and updates it. Returns the server's representation of the quotaAutoscaler, and an error, if there is any.
func (c *quotaAutoscalers) Update(ctx context.Context, quotaAutoscaler *v1.QuotaAutoscaler, opts metav1.UpdateOptions) (result *v1.QuotaAutoscaler, err error) {
	result = &v1.QuotaAutoscaler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		Name(quotaAutoscaler.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(quotaAutoscaler).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the quotaAutoscaler and deletes it. Returns an error if one occurs.
func (c *quotaAutoscalers) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *quotaAutoscalers) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("quotaautoscalers").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched quotaAutoscaler.
func (c *quotaAutoscalers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.QuotaAutoscaler, err error) {
	result = &v1.QuotaAutoscaler{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("quotaautoscalers").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
