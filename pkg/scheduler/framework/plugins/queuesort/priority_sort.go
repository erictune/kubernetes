/*
Copyright 2020 The Kubernetes Authors.

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

package queuesort

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/names"
)

// XXX Put these somewhere everyone can find.
const kPodGroupNameLabelKey = "scheduler.k8s.io/pod-group-name"
const kPodGroupMinSizeLabelKey = "scheduler.k8s.io/pod-group-min-size"

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = names.PrioritySort

// PrioritySort is a plugin that implements Priority based sorting.
type PrioritySort struct{}

var _ framework.QueueSortPlugin = &PrioritySort{}

// Name returns name of the plugin.
func (pl *PrioritySort) Name() string {
	return Name
}


// XXX Put these somewhere everyone can find.
func PodGroupFullName(pod *v1.Pod) string {
	pg, ok := pod.Labels[kPodGroupNameLabelKey]
	if !ok {
		return ""
	}
        parts := []string{pod.Namespace, pg}
	return strings.Join(parts, "/")
}

// Less is the function used by the activeQ heap algorithm to sort pods.
// It sorts pods based on their priority. When priorities are equal, it uses
// PodQueueInfo.timestamp.
func (pl *PrioritySort) Less(pInfo1, pInfo2 fwk.QueuedPodInfo) bool {
	p1 := corev1helpers.PodPriority(pInfo1.GetPodInfo().GetPod())
	p2 := corev1helpers.PodPriority(pInfo2.GetPodInfo().GetPod())
	// move to corev1helpers?
	g1 := PodGroupFullName(pInfo1.GetPodInfo().GetPod())
	g2 := PodGroupFullName(pInfo2.GetPodInfo().GetPod())
	// Sort gangs together (assuming they have the same priority)
	// and sort all gangs after all bare pods (which have empty string as their pod group identifier).
	// This approach is for experimentation only, as it might (a) bias to namespaces name, and (b) possible starvation of gangs by plain pods.
	// There are several possible workarounds to try later:
	// - sort pod group by oldest pod timestamp of any pod (requires periodic update of all pods in the queue)
	// - alternate between separate gang queue and non-gang queue, to allow bandwidth control of both types. ***
	// - offset the lexicograpical order by some periodically changing randomizer (and resort the pod queue on these epochs) to cause eventual
	//  fairness, but hard to diagnose performance.
	//  XXX enforce same prio for all in a gang at admission?
	return (p1 > p2) || (p1 == p2 && g1 < g2) || (p1 == p2 && pInfo1.GetTimestamp().Before(pInfo2.GetTimestamp()))
}

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &PrioritySort{}, nil
}
