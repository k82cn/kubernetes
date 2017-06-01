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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
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

	// TODO: found a general way for all API Objects
	rcInformer := informerFactory.Core().V1().ReplicationControllers().Informer()
	rcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			rc := obj.(*v1.ReplicationController)
			rps := int32(0)
			if rc.Spec.Replicas != nil {
				rps = *rc.Spec.Replicas
			}

			config.KubeClient.CoreV1().JobQuotas(rc.Namespace).Create(&v1.JobQuota{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{UID: rc.UID},
					},
					Name:      string(rc.UID),
					Namespace: rc.Namespace,
				},
				Spec: v1.JobQuotaSpec{
					Replicas:    rps,
					RequestUnit: jqapi.GetResourceRequest(&rc.Spec.Template.Spec).ResourceList(),
				},
			})
		},
		DeleteFunc: func(obj interface{}) {
			rc := obj.(*v1.ReplicationController)
			config.KubeClient.CoreV1().JobQuotas(rc.Namespace).Delete(string(rc.UID), &metav1.DeleteOptions{})
		},
		UpdateFunc: func(old, obj interface{}) {
			rc := obj.(*v1.ReplicationController)
			jobQuota, err := config.KubeClient.CoreV1().JobQuotas(rc.Namespace).Get(string(rc.UID), metav1.GetOptions{})
			if err != nil {
				return
			}

			if rc.Spec.Replicas == nil {
				jobQuota.Spec.Replicas = int32(0)
			} else {
				jobQuota.Spec.Replicas = *rc.Spec.Replicas
			}

			jobQuota.Spec.RequestUnit = jqapi.GetResourceRequest(&rc.Spec.Template.Spec).ResourceList()
			config.KubeClient.CoreV1().JobQuotas(rc.Namespace).Update(jobQuota)
		},
	})

	return &JobQuotaController{
		Cache:     c,
		Config:    config,
		Informers: informerFactory,
		Allocator: jqpolicy.NewPolicy("drf"),
	}
}

func (ac *JobQuotaController) Run() {
	// Start pod/node/batchjob informers.
	ac.Informers.Start(ac.Config.StopEverything)
	ac.Informers.WaitForCacheSync(ac.Config.StopEverything)

	go wait.Until(ac.allocate, 3, ac.Config.StopEverything)
	go wait.Until(ac.reclaim, 5, ac.Config.StopEverything)

	<-ac.Config.StopEverything
}

func (ac *JobQuotaController) allocate() {
	// Get the snapshot of current cluster.
	jobs_, nodes := ac.Cache.GetSnapshot()

	// Groups jobs according to policy.
	jobs := ac.Allocator.Group(jobs_)

	// Allocates resources to each group.
	allocs := ac.Allocator.Allocate(jobs, nodes)

	// Assign allocated resources to each job.
	for group, alloc := range allocs {
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
			job.JobQuota.Status.Allocated = job.Allocated.ResourceList()
			job.JobQuota.Status.Used = job.Used.ResourceList()
			ac.Config.KubeClient.CoreV1().JobQuotas(job.Namespace).Update(job.JobQuota)
		}
	}
}
