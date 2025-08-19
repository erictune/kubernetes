/*
Copyright 2025 The Kubernetes Authors.

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

package podset

// This scheduler plugin partially implements support for podset scheduling.
// Other parts of the implementation are in core.
// It is intended to move podset support out of this plugin and into the core of the scheduler
// in the future.

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/names"
	psutil "k8s.io/kubernetes/pkg/scheduler/util/podset"

	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

// Name of the plugin used in the plugin registry and configurations.
const Name = names.PodSet

// PodSet checks if a Pod is part of a pod group that requires podset-scheduling
// behavior. If so, it checks if the specified  number of pods for that group have been seen by
// the scheduler's informers, if each pod is podset compatible, and if the pods are equivalent to
// each other.
type PodSet struct {
	context   context.Context
	handle    framework.Handle
	podLister listerscorev1.PodLister
}

var _ framework.PreEnqueuePlugin = &Gang{}

func (pl *Gang) Name() string {
	return Name
}

// PreEnqueue checks if the pod belongs to a gang and if the gang is ready to be scheduled.
//
// We will not begin scheduling any pods of a gang if we have not observed
// at least the minimum number of such pods. This should keep incomplete gangs
// out of the ready queue.

// A more precise approach would be to hold back all pods of a gang where any pod does not
// pass PreEnqueue - that requires core changes.

// Idea - record the first pod of the gang as the leader pod, and make all the other pods point to it's uid.
// Then that pods's lifecycle is the gang's lifecycle.
// Or use the lifecycle of the PodGroup.
func (pl *Gang) PreEnqueue(ctx context.Context, p *v1.Pod) *fwk.Status {
	pgFullName := psutil.PodSetFullName(p)
	if pgFullName == "" {
		return nil
	}
	pgSize, err := psutil.PodSetSize(p)
	if err != nil {
		klog.ErrorS(err, "Failed to get pod set min size", "pod", klog.KObj(p))
		return fwk.NewStatus(fwk.Error, err.Error())
	}
	if XXX
// Check it is PodSet Compatible. otherwise return 
	// Count waiting nodes.
	// TODO: see if this could be made faster by using an indexer on the informer. Coscheduling plugin does this.
	seenPods, err := pl.podLister.List(labels.Everything())
	if err != nil {
		return fwk.NewStatus(fwk.UnschedulableAndUnresolvable, fmt.Sprintf("invalid pod group spec for groupo %v - min size not an integer", pgFullName))
	}
	pgSeenPods := 0
	for _, pp := range seenPods {
		if psutil.PodSetFullName(pp) == pgFullName {
			pgSeenPods += 1
		}
	}
	if pgSeenPods < pgMinSize {
		return fwk.NewStatus(fwk.Unschedulable, fmt.Sprintf("waiting for enough pods in pod group %v (seen: %v, min: %v)", pgFullName, pgSeenPods, pgMinSize))
	}
	return nil // There are enough pod to start scheduling them.
}

// New initializes a new plugin and returns it.
func New(c context.Context, _ runtime.Object, h framework.Handle, fts feature.Features) (framework.Plugin, error) {
	return &Gang{
		context:   c,
		handle:    h,
		podLister: h.SharedInformerFactory().Core().V1().Pods().Lister(),
	}, nil
}

// TODO: maybe it would help to hint that when a new pod arrives, we can re-enqueue these pods? See scheduling gates plugin for examples.
