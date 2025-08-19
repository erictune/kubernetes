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

	"k8s.io/apimachinery/pkg/runtime"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/names"
	psutil "k8s.io/kubernetes/pkg/scheduler/util/podset"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = names.PrioritySort

// PrioritySort is a plugin that implements Priority based sorting.
type PrioritySort struct {
	enablePodSetScheduling bool
}

var _ framework.QueueSortPlugin = &PrioritySort{}

// Name returns name of the plugin.
func (pl *PrioritySort) Name() string {
	return Name
}

// Less is the function used by the activeQ heap algorithm to sort pods.
// It sorts pods based on their priority. When priorities are equal, it uses
// PodQueueInfo.timestamp.
func (pl *PrioritySort) Less(pInfo1, pInfo2 fwk.QueuedPodInfo) bool {
	p1 := corev1helpers.PodPriority(pInfo1.GetPodInfo().GetPod())
	p2 := corev1helpers.PodPriority(pInfo2.GetPodInfo().GetPod())

	if pl.enablePodSetScheduling {
		// For PodSet scheduling to work, we need to sort all the pods of a podset consecutively.
		// We do that by sorting using the podset name as the secondary key. Priority is still the primary key.  All pods in a podset
		// have to have the same priority or they will be unschedulable.
		//
		// To avoid risk of normal pods while the podset feature is in alpha, we sort standard pods before all podset pods, at a given
		// priority band, by sorting in descending order on the podset name, and using "" as the value for regular pods.
		//
		// There is still some _possible_ unfairness between podsets (based on their namespace name) and is it _remotely possible_ that
		// plain pods starve podsets for a long time.  We could fix this by using the timestamp of the (oldest/newest/first-seen) pod of
		// the podset.  This could be done by tracking that value in QueuedPodInfo.
		g1 := psutil.PodSetFullName(pInfo1.GetPodInfo().GetPod())
		g2 := psutil.PodSetFullName(pInfo2.GetPodInfo().GetPod())
		return (p1 > p2) || (p1 == p2 && g1 < g2) || (p1 == p2 && g1 == g2 && pInfo1.GetTimestamp().Before(pInfo2.GetTimestamp()))

	}
	return (p1 > p2) || (p1 == p2 && pInfo1.GetTimestamp().Before(pInfo2.GetTimestamp()))
}

// XXX TODO delete these and use the ones defined in the utility/

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	// TODO: set enablePodSetScheduling based on an additional parameter "fts feature.Features"
	// However that means everywhere that uses RegisterQueueSortPlugin needs to change.
	// So, hardcoding it on for prototyping.
	return &PrioritySort{
		enablePodSetScheduling: true,
	}, nil
}
