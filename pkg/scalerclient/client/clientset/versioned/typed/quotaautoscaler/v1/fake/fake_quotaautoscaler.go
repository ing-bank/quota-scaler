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

package fake

import (
	"context"
	quotaautoscalerv1 "github.com/ing-bank/quota-scaler/pkg/scalerclient/apis/quotaautoscaler/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeQuotaAutoscalers implements QuotaAutoscalerInterface
type FakeQuotaAutoscalers struct {
	Fake *FakeIchpV1
	ns   string
}

var quotaautoscalersResource = schema.GroupVersionResource{Group: "ichp.ing.net", Version: "v1", Resource: "quotaautoscalers"}

var quotaautoscalersKind = schema.GroupVersionKind{Group: "ichp.ing.net", Version: "v1", Kind: "QuotaAutoscaler"}

// Get takes name of the quotaAutoscaler, and returns the corresponding quotaAutoscaler object, and an error if there is any.
func (c *FakeQuotaAutoscalers) Get(ctx context.Context, name string, options v1.GetOptions) (result *quotaautoscalerv1.QuotaAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(quotaautoscalersResource, c.ns, name), &quotaautoscalerv1.QuotaAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*quotaautoscalerv1.QuotaAutoscaler), err
}

// List takes label and field selectors, and returns the list of QuotaAutoscalers that match those selectors.
func (c *FakeQuotaAutoscalers) List(ctx context.Context, opts v1.ListOptions) (result *quotaautoscalerv1.QuotaAutoscalerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(quotaautoscalersResource, quotaautoscalersKind, c.ns, opts), &quotaautoscalerv1.QuotaAutoscalerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &quotaautoscalerv1.QuotaAutoscalerList{ListMeta: obj.(*quotaautoscalerv1.QuotaAutoscalerList).ListMeta}
	for _, item := range obj.(*quotaautoscalerv1.QuotaAutoscalerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested quotaAutoscalers.
func (c *FakeQuotaAutoscalers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(quotaautoscalersResource, c.ns, opts))

}

// Create takes the representation of a quotaAutoscaler and creates it.  Returns the server's representation of the quotaAutoscaler, and an error, if there is any.
func (c *FakeQuotaAutoscalers) Create(ctx context.Context, quotaAutoscaler *quotaautoscalerv1.QuotaAutoscaler, opts v1.CreateOptions) (result *quotaautoscalerv1.QuotaAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(quotaautoscalersResource, c.ns, quotaAutoscaler), &quotaautoscalerv1.QuotaAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*quotaautoscalerv1.QuotaAutoscaler), err
}

// Update takes the representation of a quotaAutoscaler and updates it. Returns the server's representation of the quotaAutoscaler, and an error, if there is any.
func (c *FakeQuotaAutoscalers) Update(ctx context.Context, quotaAutoscaler *quotaautoscalerv1.QuotaAutoscaler, opts v1.UpdateOptions) (result *quotaautoscalerv1.QuotaAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(quotaautoscalersResource, c.ns, quotaAutoscaler), &quotaautoscalerv1.QuotaAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*quotaautoscalerv1.QuotaAutoscaler), err
}

// Delete takes name of the quotaAutoscaler and deletes it. Returns an error if one occurs.
func (c *FakeQuotaAutoscalers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(quotaautoscalersResource, c.ns, name), &quotaautoscalerv1.QuotaAutoscaler{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeQuotaAutoscalers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(quotaautoscalersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &quotaautoscalerv1.QuotaAutoscalerList{})
	return err
}

// Patch applies the patch and returns the patched quotaAutoscaler.
func (c *FakeQuotaAutoscalers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *quotaautoscalerv1.QuotaAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(quotaautoscalersResource, c.ns, name, pt, data, subresources...), &quotaautoscalerv1.QuotaAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*quotaautoscalerv1.QuotaAutoscaler), err
}