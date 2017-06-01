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

package policy

import (
	jqapi "k8s.io/kubernetes/pkg/controller/jobquota/api"
)

// Allocator is the interface
type Allocator interface {
	// The unique name of allocator.
	Name() string

	// Initialize initializes the allocator plugins.
	Initialize()

	// Group grouping the job into different bucket, and allocate those resources based on those groups.
	Group(job []*jqapi.JoQuotabInfo) map[string][]*jqapi.JoQuotabInfo

	// Allocate allocates the cluster's resources into each group.
	Allocate(jobs map[string][]*jqapi.JoQuotabInfo, nodes []*jqapi.NodeInfo) map[string]*jqapi.AllocationInfo

	// Assign allocates resources of group into each jobs.
	Assign(jobs []*jqapi.JoQuotabInfo, alloc *jqapi.AllocationInfo) *jqapi.Resource

	// Reclaim returns the Pods that should be evict to release resources.
	Reclaim(job *jqapi.JoQuotabInfo, res *jqapi.Resource) []*jqapi.PodInfo

	// UnIntialize un-initializes the allocator plugins.
	UnInitialize()
}
