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

package gang

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/names"
	//corev1 "k8s.io/api/core/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

const (
	// PodGroupNameLabelKey is the label key used to identify the pod group name.
	PodGroupNameLabelKey = "scheduler.k8s.io/pod-group-name"
	// PodGroupMinSizeLabelKey is the label key used to identify the minimum number of pods in a pod group.
	PodGroupMinSizeLabelKey = "scheduler.k8s.io/pod-group-min-size"
)

// Name of the plugin used in the plugin registry and configurations.
const Name = names.Gang

// PodGroupFullName returns the full name of the pod group for a given pod.
func PodGroupFullName(pod *v1.Pod) string {
	pg, ok := pod.Labels[PodGroupNameLabelKey]
	if !ok {
		return ""
	}
	parts := []string{pod.Namespace, pg}
	return strings.Join(parts, "/")
}

// PodGroupMinSize returns the minimum size of the pod group for a given pod.
func PodGroupMinSize(pod *v1.Pod) (int, error) {
	pgmsStr, ok := pod.Labels[PodGroupMinSizeLabelKey]
	if !ok {
		return 0, nil // No minimum.
	}
	pgmn, err := strconv.Atoi(pgmsStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for pod group min size: %w", err)
	}
	return pgmn, nil
}

// Gang checks if a Pod is part of a pod group that requires gang scheduling
// and if so, if the minimum number of pods for that group have been seen by
// the scheduler's informers.
type Gang struct {
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
	pgFullName := PodGroupFullName(p)
	if pgFullName == "" {
		return nil
	}
	pgMinSize, err := PodGroupMinSize(p)
	if err != nil {
		klog.ErrorS(err, "Failed to get pod group min size", "pod", klog.KObj(p))
		return fwk.NewStatus(fwk.Error, err.Error())
	}
	if p.Spec.Affinity != nil {
		if p.Spec.Affinity.PodAffinity != nil {
			if len(p.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 ||
				len(p.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
				return fwk.NewStatus(
					fwk.UnschedulableAndUnresolvable,
					fmt.Sprintf("Pods with spec.affinity.podAffinity may not use pod group scheduling."))
			}
		}
		if p.Spec.Affinity.PodAntiAffinity != nil {
			if len(p.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 ||
				len(p.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
				return fwk.NewStatus(
					fwk.UnschedulableAndUnresolvable,
					fmt.Sprintf("Pods with spec.affinity.podAntiAffinity may not use pod group scheduling."))
			}
		}
	}
	for _, tsc := range p.Spec.TopologySpreadConstraints {
		if tsc.WhenUnsatisfiable == v1.DoNotSchedule {
			return fwk.NewStatus(
				fwk.UnschedulableAndUnresolvable,
				fmt.Sprintf("Pods with a spec.topologySpreadConstraint with DoNotSchedule may not pod group scheduling."))
		}
	}

	// Count waiting nodes.
	// TODO: see if this could be made faster by using an indexer on the informer. Coscheduling plugin does this.
	seenPods, err := pl.podLister.List(labels.Everything())
	if err != nil {
		return fwk.NewStatus(fwk.UnschedulableAndUnresolvable, fmt.Sprintf("invalid pod group spec for groupo %v - min size not an integer", pgFullName))
	}
	pgSeenPods := 0
	for _, pp := range seenPods {
		if PodGroupFullName(pp) == pgFullName {
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
