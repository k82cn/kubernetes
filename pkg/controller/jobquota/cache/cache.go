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

package cache

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/pkg/api/v1"
	jqapi "k8s.io/kubernetes/pkg/controller/jobquota/api"
)

type Cache struct {
	sync.Mutex

	batchjobs map[types.UID]*jqapi.JoQuotabInfo
	nodes     map[string]*jqapi.NodeInfo
}

func NewCache() *Cache {
	return &Cache{
		batchjobs: map[types.UID]*jqapi.JoQuotabInfo{},
		nodes:     map[string]*jqapi.NodeInfo{},
	}
}

func (c *Cache) AddPod(obj interface{}) {
	c.Lock()
	defer c.Unlock()

	pod := jqapi.NewPodInfo(obj.(*v1.Pod))
	batchjob, found := c.batchjobs[pod.Owner]
	if !found {
		return
	}

	switch pod.Status {
	case v1.PodRunning:
		batchjob.RunningPods.Add(pod)
	case v1.PodFailed, v1.PodSucceeded:
		batchjob.RunningPods.Delete(pod)
		batchjob.ReclaimingPods.Delete(pod)
	}
}

func (c *Cache) UpdatePod(old, obj interface{}) {
	c.AddPod(obj)
}

func (c *Cache) DeletePod(obj interface{}) {
	c.Lock()
	defer c.Unlock()

	pod := jqapi.NewPodInfo(obj.(*v1.Pod))
	alloc, found := c.batchjobs[pod.Owner]
	if !found {
		return
	}

	alloc.RunningPods.Delete(pod)
	alloc.ReclaimingPods.Delete(pod)
}

func (c *Cache) AddNode(obj interface{}) {
	c.Lock()
	defer c.Unlock()

	node := jqapi.NewNodeInfo(obj.(*v1.Node))
	c.nodes[node.Name] = node
}

func (c *Cache) UpdateNode(old, obj interface{}) {
	c.Lock()
	defer c.Unlock()

	node := jqapi.NewNodeInfo(obj.(*v1.Node))
	c.nodes[node.Name] = node
}

func (c *Cache) DeleteNode(obj interface{}) {
	c.Lock()
	defer c.Unlock()

	node := jqapi.NewNodeInfo(obj.(*v1.Node))
	delete(c.nodes, node.Name)
}

func (c *Cache) AddJobQuota(obj interface{}) {
	c.Lock()
	defer c.Unlock()

	ri := jqapi.NewJobQuotaInfo(obj.(*v1.JobQuota))
	c.batchjobs[ri.ConsumerId] = ri
}

func (c *Cache) UpdateJobQuota(old, obj interface{}) {
	c.Lock()
	defer c.Unlock()

	ri := jqapi.NewJobQuotaInfo(obj.(*v1.JobQuota))
	c.batchjobs[ri.ConsumerId] = ri
}

func (c *Cache) DeleteJobQuota(obj interface{}) {
	c.Lock()
	defer c.Unlock()

	ri := jqapi.NewJobQuotaInfo(obj.(*v1.JobQuota))
	delete(c.batchjobs, ri.ConsumerId)
}

func (c *Cache) GetOverused() []*jqapi.JoQuotabInfo {
	bjs := []*jqapi.JoQuotabInfo{}
	for _, r := range c.batchjobs {
		if r.Allocated.Less(r.Used) {
			bjs = append(bjs, r.Clone())
		}
	}
	return bjs
}

func (c *Cache) Reclaim(job *jqapi.JoQuotabInfo, pods []*jqapi.PodInfo) {
	c.Lock()
	defer c.Unlock()
	batchjob := c.batchjobs[job.ConsumerId]
	for _, pod := range pods {
		batchjob.ReclaimingPods.Add(pod)
	}
}

func (c *Cache) GetSnapshot() ([]*jqapi.JoQuotabInfo, []*jqapi.NodeInfo) {
	c.Lock()
	defer c.Unlock()

	rs := []*jqapi.JoQuotabInfo{}
	ns := []*jqapi.NodeInfo{}
	for _, r := range c.batchjobs {
		rs = append(rs, r.Clone())
	}

	for _, n := range c.nodes {
		ns = append(ns, n.Clone())
	}

	return rs, ns
}
