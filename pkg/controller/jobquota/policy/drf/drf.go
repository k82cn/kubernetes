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

package drf

import (
	"math"

	"github.com/golang/glog"

	"k8s.io/client-go/tools/cache"
	jqapi "k8s.io/kubernetes/pkg/controller/jobquota/api"
	"k8s.io/kubernetes/pkg/controller/jobquota/policy"
	jqutil "k8s.io/kubernetes/pkg/controller/jobquota/util"
)

func init() {
	policy.Register(&drf{})
}

type drf struct {
	total     *jqapi.Resource
	available *jqapi.Resource
	consumers []*consumer
}

type consumer struct {
	Name     string
	Share    float64
	Deserved *jqapi.Resource
	Request  *jqapi.Resource
	Jobs     *cache.FIFO
}

func (c *consumer) Priority() float64 {
	return c.Share
}

func (d *drf) Name() string {
	return "drf"
}

func (d *drf) Initialize() {
	glog.V(4).Info("DRF initialize.")
}

func (d *drf) UnInitialize() {
	glog.V(4).Info("DRF UnInitialize.")
}

// Group grouping the job into different bucket by BatchJob's namespace.
func (d *drf) Group(jobs []*jqapi.JoQuotabInfo) map[string][]*jqapi.JoQuotabInfo {
	res := map[string][]*jqapi.JoQuotabInfo{}
	for _, job := range jobs {
		ns := job.Namespace
		res[ns] = append(res[ns], job)
	}

	return res
}

func (d *drf) Assign(jobs []*jqapi.JoQuotabInfo, alloc_ *jqapi.AllocationInfo) *jqapi.Resource {
	alloc := alloc_.Clone()

	glog.V(4).Infof("Allocate <%v> to %v jobs.", alloc.Allocated, len(jobs))

	for _, job := range jobs {
		req := job.RequestUnit.Clone().Multi(job.Replicas)
		if alloc.Allocated.Less(req) {
			break
		}

		glog.V(4).Infof("Allocate <%v> to job %v/%v.", req, job.Namespace, job.ConsumerId)
		alloc.Allocated.Sub(req)
		job.Allocated = req
	}

	return alloc.Allocated
}

func (d *drf) Reclaim(job *jqapi.JoQuotabInfo, res *jqapi.Resource) []*jqapi.PodInfo {
	pods := []*jqapi.PodInfo{}
	for _, pod := range job.RunningPods.List() {
		pods = append(pods, pod.(*jqapi.PodInfo))
	}

	return pods
}

func (d *drf) Allocate(jobs map[string][]*jqapi.JoQuotabInfo, nodes []*jqapi.NodeInfo) map[string]*jqapi.AllocationInfo {
	allocation := map[string]*jqapi.AllocationInfo{}
	if len(nodes) == 0 || len(jobs) == 0 {
		return allocation
	}

	d.total = jqapi.EmptyResource()
	d.available = jqapi.EmptyResource()

	// Got allocatable resources in the cluster.
	for _, node := range nodes {
		d.total.Add(node.Allocatable)
		d.available.Add(node.Allocatable)
	}

	d.consumers = d.buildConsumers(jobs)

	// Allocate resources.
	for {
		pq := d.sortConsumer()

		allocatedOnce := false

		glog.V(4).Infof("total <%v>, available <%v>", d.total, d.available)

		for {
			if d.available.IsEmpty() || pq.Len() == 0 {
				break
			}

			consumer := pq.Pop().(*consumer)
			// If no tasks, continue to handle next consumer.
			if consumer.Jobs.Len() == 0 {
				continue
			}

			job := cache.Pop(consumer.Jobs).(*jqapi.JoQuotabInfo)
			req := job.RequestUnit.Clone().Multi(job.Replicas)

			// If available resource does not have enough resource for the pod, skip it.
			if !req.LessEqual(d.available) {
				continue
			}

			d.allocate(consumer, req)
			consumer.Share = d.calculateShare(consumer)

			pq.Push(consumer)

			allocatedOnce = true

			glog.V(4).Infof("<%s> priority is <%f> (total: <%v>, available: <%v>)",
				consumer.Name, consumer.Share, d.total, d.available)
		}

		if !allocatedOnce {
			break
		}
	}

	for _, consumer := range d.consumers {
		allocation[consumer.Name] = &jqapi.AllocationInfo{
			Allocated: consumer.Deserved,
			// TODO: filled by jobquota controller.
			Used: jqapi.EmptyResource(),
		}
		glog.V(4).Infof("Allocated <%v> to %v", consumer.Deserved, consumer.Name)
	}

	return allocation
}

// sortJobs sorts job's tasks to request resources; it's FCFS for now.
func (d *drf) buildConsumers(jobs_ map[string][]*jqapi.JoQuotabInfo) []*consumer {
	consumers := []*consumer{}

	for ns, jobs := range jobs_ {
		consumer := &consumer{
			Name:     ns,
			Jobs:     cache.NewFIFO(jqapi.BatchJobInfoKeyFunc),
			Request:  jqapi.EmptyResource(),
			Deserved: jqapi.EmptyResource(),
		}

		for _, job := range jobs {
			consumer.Jobs.Add(job)
			consumer.Request.Add(job.RequestUnit.Clone().Multi(job.Replicas))
		}

		consumers = append(consumers, consumer)
	}

	return consumers
}

func (d *drf) sortConsumer() *jqutil.PriorityQueue {
	pq := jqutil.NewPriorityQueue()

	for _, consumer := range d.consumers {
		pq.Push(consumer)
	}

	return pq
}

func (d *drf) allocate(consumer *consumer, request *jqapi.Resource) {
	glog.V(4).Infof("allocate <%v> to <%v>", request, consumer)

	consumer.Deserved.Add(request)
	d.available.Sub(request)
}

func (d *drf) calculateShare(consumer *consumer) float64 {
	cpuShare := consumer.Request.MilliCPU / d.total.MilliCPU
	memShare := consumer.Request.Memory / d.total.Memory

	// if dominate resource is CPU, return its share
	if cpuShare > memShare {
		return consumer.Deserved.MilliCPU / d.total.MilliCPU
	}

	// if dominate resource is memory, return its share
	if cpuShare < memShare {
		return consumer.Deserved.Memory / d.total.Memory
	}

	return math.Max(consumer.Deserved.MilliCPU/d.total.MilliCPU,
		consumer.Deserved.Memory/d.total.Memory)
}
