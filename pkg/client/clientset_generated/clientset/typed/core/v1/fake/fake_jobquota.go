/*
Copyright 2017 The Kubernetes Authors.

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

package fake

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1 "k8s.io/kubernetes/pkg/api/v1"
)

// FakeJobQuotas implements JobQuotaInterface
type FakeJobQuotas struct {
	Fake *FakeCoreV1
	ns   string
}

var jobquotasResource = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "jobquotas"}

var jobquotasKind = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "JobQuota"}

func (c *FakeJobQuotas) Create(jobQuota *v1.JobQuota) (result *v1.JobQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(jobquotasResource, c.ns, jobQuota), &v1.JobQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.JobQuota), err
}

func (c *FakeJobQuotas) Update(jobQuota *v1.JobQuota) (result *v1.JobQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(jobquotasResource, c.ns, jobQuota), &v1.JobQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.JobQuota), err
}

func (c *FakeJobQuotas) UpdateStatus(jobQuota *v1.JobQuota) (*v1.JobQuota, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(jobquotasResource, "status", c.ns, jobQuota), &v1.JobQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.JobQuota), err
}

func (c *FakeJobQuotas) Delete(name string, options *meta_v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(jobquotasResource, c.ns, name), &v1.JobQuota{})

	return err
}

func (c *FakeJobQuotas) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(jobquotasResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1.JobQuotaList{})
	return err
}

func (c *FakeJobQuotas) Get(name string, options meta_v1.GetOptions) (result *v1.JobQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(jobquotasResource, c.ns, name), &v1.JobQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.JobQuota), err
}

func (c *FakeJobQuotas) List(opts meta_v1.ListOptions) (result *v1.JobQuotaList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(jobquotasResource, jobquotasKind, c.ns, opts), &v1.JobQuotaList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.JobQuotaList{}
	for _, item := range obj.(*v1.JobQuotaList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested jobQuotas.
func (c *FakeJobQuotas) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(jobquotasResource, c.ns, opts))

}

// Patch applies the patch and returns the patched jobQuota.
func (c *FakeJobQuotas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.JobQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(jobquotasResource, c.ns, name, data, subresources...), &v1.JobQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.JobQuota), err
}
