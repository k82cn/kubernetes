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

package jobquota

import (
	glog "github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	jqapi "k8s.io/kubernetes/pkg/controller/jobquota/api"
	jqcache "k8s.io/kubernetes/pkg/controller/jobquota/cache"
	jqpolicy "k8s.io/kubernetes/pkg/controller/jobquota/policy"

	_ "k8s.io/kubernetes/pkg/controller/jobquota/policy/drf"
)

type JobQuotaController struct {
	Informers informers.SharedInformerFactory
	Config    *JobQuotaControllerConfig
	Cache     *jqcache.Cache
	Allocator jqpolicy.Allocator
}

type JobQuotaControllerConfig struct {
	KubeClient     kubernetes.Interface
	StopEverything <-chan struct{}
}

func NewJobQuotaController(config *JobQuotaControllerConfig) *JobQuotaController {
	c := jqcache.NewCache()

	informerFactory := informers.NewSharedInformerFactory(config.KubeClient, 0)

	nodeInformer := informerFactory.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc:    c.AddNode,
		DeleteFunc: c.DeleteNode,
		UpdateFunc: c.UpdateNode,
	})

	podInformer := informerFactory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc:    c.AddPod,
		DeleteFunc: c.DeletePod,
		UpdateFunc: c.UpdatePod,
	})

	jobQuotaInformer := informerFactory.Core().V1().JobQuotas().Informer()
	jobQuotaInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.AddJobQuota,
		DeleteFunc: c.DeleteJobQuota,
		UpdateFunc: c.UpdateJobQuota,
	})

	return &JobQuotaController{
		Cache:     c,
		Config:    config,
		Informers: informerFactory,
		Allocator: jqpolicy.NewPolicy("drf"),
	}
}

func (ac *JobQuotaController) Run() {
	// Start pod/node/jobquotas informers.
	ac.Informers.Start(ac.Config.StopEverything)
	ac.Informers.WaitForCacheSync(ac.Config.StopEverything)

	glog.Info("Starting JobQuotaController.")
	go wait.Until(ac.allocate, 3, ac.Config.StopEverything)
	go wait.Until(ac.reclaim, 5, ac.Config.StopEverything)

	<-ac.Config.StopEverything
}

func (ac *JobQuotaController) allocate() {
	// Get the snapshot of current cluster.
	jobs_, nodes := ac.Cache.GetSnapshot()

	glog.V(4).Infof("Get snapshot of cache: jobs %+v, nodes %+v", jobs_, nodes)

	// Groups jobs according to policy.
	jobs := ac.Allocator.Group(jobs_)

	glog.V(4).Infof("Group jobs: %+v", jobs)

	// Allocates resources to each group.
	allocs := ac.Allocator.Allocate(jobs, nodes)

	glog.V(4).Infof("Get allocations: %+v", jobs)

	// Assign allocated resources to each job.
	for group, alloc := range allocs {
		glog.V(4).Infof("Assign allocations %+v to jobs %+v", alloc, jobs[group])
		ac.Allocator.Assign(jobs[group], alloc)
	}

	// Update BatchJob's status for BatchJobQuota Admission & cli.
	ac.update(jobs)
}

func (ac *JobQuotaController) reclaim() {
	// Get the oversed jobs in cluster.
	overused := ac.Cache.GetOverused()

	for _, job := range overused {
		res := job.Used.Clone().Sub(job.Allocated)

		// Get pods which will be evicted to release resources.
		pods := ac.Allocator.Reclaim(job, res)
		ac.Cache.Reclaim(job, pods)

		for _, pod := range pods {
			ac.evict(pod.Namespace, pod.Name)
		}
	}
}

func (ac *JobQuotaController) evict(ns, name string) {
	go func() {
		ac.Config.KubeClient.CoreV1().Pods(ns).Delete(name, &metav1.DeleteOptions{})
	}()
}

func (ac *JobQuotaController) update(groups map[string][]*jqapi.JoQuotabInfo) {
	for _, jobs := range groups {
		for _, job := range jobs {
			jobQuota := *job.JobQuota
			jobQuota.Status.Allocated = job.Allocated.ResourceList()
			jobQuota.Status.Used = job.Used.ResourceList()
			glog.V(4).Infof("job quota %v/%v is updated to allocated %v, used %v",
				job.Namespace, job.ConsumerId, jobQuota.Status.Allocated, jobQuota.Status.Used)
			ac.Config.KubeClient.CoreV1().JobQuotas(job.Namespace).UpdateStatus(&jobQuota)
		}
	}
}
