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

package api

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

type Resource struct {
	MilliCPU float64
	Memory   float64
}

type PodInfo struct {
	Owner        types.UID
	Name         string
	Namespace    string
	ConsumerName string
	Status       v1.PodPhase
	Hostname     string
	Resource     *Resource
}

func NewPodInfo(pod *v1.Pod) *PodInfo {
	if len(pod.OwnerReferences) != 1 {
		return nil
	}

	return &PodInfo{
		Owner:     pod.OwnerReferences[0].UID,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Status:    pod.Status.Phase,
		Hostname:  pod.Spec.NodeName,
		Resource:  GetResourceRequest(&pod.Spec),
	}
}

func (p PodInfo) String() string {
	return fmt.Sprintf("%v/%v", p.Namespace, p.Name)
}

func PodInfoKeyFunc(obj interface{}) (string, error) {
	if pod, ok := obj.(*PodInfo); ok {
		return fmt.Sprintf("%s/%s", pod.Namespace, pod.Name), nil
	}

	return "", fmt.Errorf("failed to convert <%v> to *util.PodInfo", obj)
}

func NodeInfoKeyFunc(obj interface{}) (string, error) {
	if node, ok := obj.(*NodeInfo); ok {
		return fmt.Sprintf("%s", node.Name), nil
	}

	return "", fmt.Errorf("failed to convert <%v> to *util.NodeInfo", obj)
}

type NodeInfo struct {
	Name        string
	Allocatable *Resource
	Allocated   *Resource
	Capacity    *Resource
}

func NewNodeInfo(node *v1.Node) *NodeInfo {
	return &NodeInfo{
		Name:        node.Name,
		Allocatable: NewResource(node.Status.Allocatable),
		Capacity:    NewResource(node.Status.Capacity),
		Allocated:   EmptyResource(),
	}
}

func (ni *NodeInfo) Clone() *NodeInfo {
	return &NodeInfo{
		Name:        ni.Name,
		Allocatable: ni.Allocated.Clone(),
		Allocated:   ni.Allocated.Clone(),
		Capacity:    ni.Capacity.Clone(),
	}
}

func (ni *NodeInfo) AddPod(pi *PodInfo) {
	if ni == nil || pi == nil {
		return
	}
	ni.Allocated.Add(pi.Resource)
}

func (ni *NodeInfo) DeletePod(pi *PodInfo) {
	if ni == nil || pi == nil {
		return
	}
	ni.Allocated.Sub(pi.Resource)
}

func EmptyResource() *Resource {
	return &Resource{
		MilliCPU: 0,
		Memory:   0,
	}
}

func NewResource(rl v1.ResourceList) *Resource {
	return &Resource{
		MilliCPU: float64(rl.Cpu().MilliValue()),
		Memory:   float64(rl.Memory().Value()),
	}
}

func (r Resource) ResourceList() v1.ResourceList {
	res := v1.ResourceList{}
	res[v1.ResourceCPU] = *resource.NewMilliQuantity(int64(r.MilliCPU), resource.DecimalExponent)
	res[v1.ResourceMemory] = *resource.NewQuantity(int64(r.Memory), resource.BinarySI)
	return res
}

func (r *Resource) Clone() *Resource {
	return &Resource{
		MilliCPU: r.MilliCPU,
		Memory:   r.Memory,
	}
}

var minMilliCPU float64 = 10
var minMemory float64 = 10 * 1024 * 1024 // 10M

func (r Resource) IsEmpty() bool {
	return r.MilliCPU < minMilliCPU && r.Memory < minMemory
}

func (r *Resource) Add(rr *Resource) *Resource {
	r.MilliCPU += rr.MilliCPU
	r.Memory += rr.Memory
	return r
}

func (r *Resource) Sub(rr *Resource) *Resource {
	r.MilliCPU -= rr.MilliCPU
	r.Memory -= rr.Memory
	return r
}

func (r *Resource) Multi(rep int32) *Resource {
	r.MilliCPU *= float64(rep)
	r.Memory *= float64(rep)
	return r
}

func (r *Resource) Less(rr *Resource) bool {
	return r.MilliCPU < rr.MilliCPU && r.Memory < rr.Memory
}

func (r *Resource) LessEqual(rr *Resource) bool {
	return (r.MilliCPU < rr.MilliCPU || math.Abs(r.MilliCPU-rr.MilliCPU) < 0.01) &&
		(r.Memory < rr.Memory || math.Abs(r.Memory-rr.Memory) < 1)
}

func (r Resource) String() string {
	return fmt.Sprintf("cpu %f, mem %f", r.MilliCPU, r.Memory)
}

func GetResourceRequest(podSpec *v1.PodSpec) *Resource {
	result := Resource{}
	for _, container := range podSpec.Containers {
		for rName, rQuantity := range container.Resources.Requests {
			switch rName {
			case v1.ResourceMemory:
				result.Memory += float64(rQuantity.Value())
			case v1.ResourceCPU:
				result.MilliCPU += float64(rQuantity.MilliValue())
			default:
				continue
			}
		}
	}

	// Ignore init Container in scheduling

	return &result
}

type JoQuotabInfo struct {
	JobQuota *v1.JobQuota

	ConsumerId types.UID
	Namespace  string

	Replicas    int32
	RequestUnit *Resource

	// The resources that allocated to this job.
	Allocated *Resource
	// The resources that used by consumer.
	Used *Resource

	RunningPods    *cache.FIFO
	ReclaimingPods *cache.FIFO
}

func BatchJobInfoKeyFunc(obj interface{}) (string, error) {
	if bj, ok := obj.(*JoQuotabInfo); !ok {
		return "", fmt.Errorf("failed to convert %v to *BatchJobInfo", obj)
	} else {
		return string(bj.ConsumerId), nil
	}
}

func NewJobQuotaInfo(rr *v1.JobQuota) *JoQuotabInfo {
	bj := &JoQuotabInfo{
		JobQuota:    rr,
		Namespace:   rr.Namespace,
		Replicas:    rr.Spec.Replicas,
		RequestUnit: NewResource(rr.Spec.RequestUnit),

		Allocated: NewResource(rr.Status.Used),
		Used:      NewResource(rr.Status.Allocated),

		RunningPods:    cache.NewFIFO(PodInfoKeyFunc),
		ReclaimingPods: cache.NewFIFO(PodInfoKeyFunc),
	}

	if len(rr.OwnerReferences) == 1 {
		bj.ConsumerId = rr.OwnerReferences[0].UID
	}

	return bj
}

func (ri *JoQuotabInfo) Clone() *JoQuotabInfo {
	bj := &JoQuotabInfo{
		JobQuota:    ri.JobQuota,
		ConsumerId:  ri.ConsumerId,
		Namespace:   ri.Namespace,
		Replicas:    ri.Replicas,
		RequestUnit: ri.RequestUnit.Clone(),

		Allocated: ri.Allocated.Clone(),
		Used:      ri.Used.Clone(),

		ReclaimingPods: cache.NewFIFO(PodInfoKeyFunc),
		RunningPods:    cache.NewFIFO(PodInfoKeyFunc),
	}

	for _, pod := range ri.RunningPods.List() {
		bj.RunningPods.Add(pod)
	}

	for _, pod := range ri.ReclaimingPods.List() {
		bj.ReclaimingPods.Add(pod)
	}

	return bj
}

type AllocationInfo struct {
	GroupId string
	// The resources that allocated to this job.
	Allocated *Resource
	// The resources that used by consumer.
	Used *Resource
}

func (ai *AllocationInfo) Clone() *AllocationInfo {
	return &AllocationInfo{
		GroupId:   ai.GroupId,
		Allocated: ai.Allocated.Clone(),
		Used:      ai.Used.Clone(),
	}
}
