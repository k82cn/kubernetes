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

package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	scheme "k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/pkg/api/v1"
	rest "k8s.io/client-go/rest"
)

// JobQuotasGetter has a method to return a JobQuotaInterface.
// A group's client should implement this interface.
type JobQuotasGetter interface {
	JobQuotas(namespace string) JobQuotaInterface
}

// JobQuotaInterface has methods to work with JobQuota resources.
type JobQuotaInterface interface {
	Create(*v1.JobQuota) (*v1.JobQuota, error)
	Update(*v1.JobQuota) (*v1.JobQuota, error)
	UpdateStatus(*v1.JobQuota) (*v1.JobQuota, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.JobQuota, error)
	List(opts meta_v1.ListOptions) (*v1.JobQuotaList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.JobQuota, err error)
	JobQuotaExpansion
}

// jobQuotas implements JobQuotaInterface
type jobQuotas struct {
	client rest.Interface
	ns     string
}

// newJobQuotas returns a JobQuotas
func newJobQuotas(c *CoreV1Client, namespace string) *jobQuotas {
	return &jobQuotas{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a jobQuota and creates it.  Returns the server's representation of the jobQuota, and an error, if there is any.
func (c *jobQuotas) Create(jobQuota *v1.JobQuota) (result *v1.JobQuota, err error) {
	result = &v1.JobQuota{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("jobquotas").
		Body(jobQuota).
		Do().
		Into(result)
	return
}

// Update takes the representation of a jobQuota and updates it. Returns the server's representation of the jobQuota, and an error, if there is any.
func (c *jobQuotas) Update(jobQuota *v1.JobQuota) (result *v1.JobQuota, err error) {
	result = &v1.JobQuota{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("jobquotas").
		Name(jobQuota.Name).
		Body(jobQuota).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *jobQuotas) UpdateStatus(jobQuota *v1.JobQuota) (result *v1.JobQuota, err error) {
	result = &v1.JobQuota{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("jobquotas").
		Name(jobQuota.Name).
		SubResource("status").
		Body(jobQuota).
		Do().
		Into(result)
	return
}

// Delete takes name of the jobQuota and deletes it. Returns an error if one occurs.
func (c *jobQuotas) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("jobquotas").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *jobQuotas) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("jobquotas").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the jobQuota, and returns the corresponding jobQuota object, and an error if there is any.
func (c *jobQuotas) Get(name string, options meta_v1.GetOptions) (result *v1.JobQuota, err error) {
	result = &v1.JobQuota{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("jobquotas").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of JobQuotas that match those selectors.
func (c *jobQuotas) List(opts meta_v1.ListOptions) (result *v1.JobQuotaList, err error) {
	result = &v1.JobQuotaList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("jobquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested jobQuotas.
func (c *jobQuotas) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("jobquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched jobQuota.
func (c *jobQuotas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.JobQuota, err error) {
	result = &v1.JobQuota{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("jobquotas").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
