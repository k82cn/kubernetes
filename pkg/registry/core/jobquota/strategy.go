/*
Copyright 2014 The Kubernetes Authors.

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

package jobquota

import (
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/kubernetes/pkg/api"
)

// resourcequotaStrategy implements behavior for ResourceQuota objects
type jobQuotaStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating ResourceQuota
// objects via the REST API.
var Strategy = jobQuotaStrategy{api.Scheme, names.SimpleNameGenerator}

// NamespaceScoped is true for resourcequotas.
func (jobQuotaStrategy) NamespaceScoped() bool {
	return true
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (jobQuotaStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
	batchjob := obj.(*api.JobQuota)
	batchjob.Status = api.JobQuotaStatus{}
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (jobQuotaStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newResourcequota := obj.(*api.JobQuota)
	oldResourcequota := old.(*api.JobQuota)
	newResourcequota.Status = oldResourcequota.Status
}

// Validate validates a new resourcequota.
func (jobQuotaStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	if _, ok := obj.(*api.JobQuota); !ok {
		return field.ErrorList{}
	}

	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (jobQuotaStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for resourcequotas.
func (jobQuotaStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (jobQuotaStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	if _, ok := obj.(*api.JobQuota); !ok {
		return field.ErrorList{}
	}
	return field.ErrorList{}
}

func (jobQuotaStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type resourcequotaStatusStrategy struct {
	jobQuotaStrategy
}

var StatusStrategy = resourcequotaStatusStrategy{Strategy}

func (resourcequotaStatusStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newResourcequota := obj.(*api.JobQuota)
	oldResourcequota := old.(*api.JobQuota)
	newResourcequota.Spec = oldResourcequota.Spec
}

func (resourcequotaStatusStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	jobQuotaObj, ok := obj.(*api.JobQuota)
	if !ok {
		return nil, nil, fmt.Errorf("not a resourcequota")
	}
	return labels.Set(jobQuotaObj.Labels), JobQuotaToSelectableFields(jobQuotaObj), nil
}

// MatchResourceQuota returns a generic matcher for a given label and field selector.
func MatchJobQuota(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// JobQuotaToSelectableFields returns a field set that represents the object
func JobQuotaToSelectableFields(resourcequota *api.JobQuota) fields.Set {
	return generic.ObjectMetaFieldsSet(&resourcequota.ObjectMeta, true)
}
