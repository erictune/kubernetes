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
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	// "k8s.io/apimachinery/pkg/runtime"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
	"k8s.io/kubernetes/test/utils/ktesting"
)

type MockPodLister struct {
	l []*corev1.Pod
}

func (mpl *MockPodLister) List(selector labels.Selector) (ret []*corev1.Pod, err error) {
	return mpl.l, nil
}
func (mpl *MockPodLister) Pods(namespace string) listerscorev1.PodNamespaceLister {
	panic("not implemented")
}

// New initializes a new plugin for testing and returns it.
func NewTestPlugin(c context.Context, allpods []*corev1.Pod, fts feature.Features) (framework.Plugin, error) {
	return &Gang{
		context:   c,
		handle:    nil,
		podLister: &MockPodLister{l: allpods},
	}, nil
}


func TestPreEnqueue(t *testing.T) {
	pSolo := st.MakePod().Name("p").Namespace("ns1").Obj()
	pSoloAffAntiAffSpread := st.MakePod().Name("p").Namespace("ns1").PodAffinity("foo", nil, st.PodAffinityWithRequiredReq).SpreadConstraint(1, "foo", corev1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj()
	pGang1 := st.MakePod().Name("pgang-1").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").Obj()
	pGang2 := st.MakePod().Name("pgang-2").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").Obj()
	pGang3 := st.MakePod().Name("pgang-3").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").Obj()
	pGangWithPA1 := st.MakePod().Name("pgang-1").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").PodAffinity("foo", nil, st.PodAffinityWithRequiredReq).Obj()
	pGangWithPA2 := st.MakePod().Name("pgang-2").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").PodAffinity("foo", nil, st.PodAffinityWithRequiredReq).Obj()
	pGangWithPAA1 := st.MakePod().Name("pgang-1").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").PodAntiAffinity("foo", nil, st.PodAntiAffinityWithRequiredPreferredReq).Obj()
	pGangWithPAA2 := st.MakePod().Name("pgang-2").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").PodAntiAffinity("foo", nil, st.PodAntiAffinityWithRequiredPreferredReq).Obj()
	pGangWithHardTSC1 := st.MakePod().Name("pgang-1").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").SpreadConstraint(5, "foo", corev1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj()
	pGangWithHardTSC2 := st.MakePod().Name("pgang-2").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").SpreadConstraint(5, "foo", corev1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj()
	pGangWithSoftTSC1 := st.MakePod().Name("pgang-1").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").SpreadConstraint(5, "foo", corev1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj()
	pGangWithSoftTSC2 := st.MakePod().Name("pgang-2").Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo").SpreadConstraint(5, "foo", corev1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj()

	tests := []struct {
		name    string
		pod     *corev1.Pod
		allpods []*corev1.Pod
		want    *fwk.Status
	}{
		{
			name:    "pod not in a pod group; no other pods present.",
			pod:     pSolo,
			allpods: []*corev1.Pod{pSolo},
			want:    nil,
		},
		{
			name:    "pod not in a pod group; one pod from a podgroup is present.",
			pod:     pSolo,
			allpods: []*corev1.Pod{pSolo, pGang1},
			want:    nil,
		},
		{
			name:    "pod in a pod group; no other pods present.",
			pod:     pGang1,
			allpods: []*corev1.Pod{pGang1},
			want:    fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in pod group ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name:    "pod in a pod group; unrelated pods are in the system, but no peers",
			pod:     pGang1,
			allpods: []*corev1.Pod{pSolo, pGang1},
			want:    fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in pod group ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name:    "pod in a pod group; one peers present.",
			pod:     pGang1,
			allpods: []*corev1.Pod{pGang2, pSolo, pGang1},
			want:    nil,
		},
		{
			name:    "pod in a pod group; two peers present.",
			pod:     pGang1,
			allpods: []*corev1.Pod{pGang2, pSolo, pGang1, pGang3},
			want:    nil,
		},
		{
			name:    "pod not in a pod group with affinity; no other pods present; affinity, anti-affinity, topology spreading.",
			pod:     pSoloAffAntiAffSpread,
			allpods: []*corev1.Pod{pSoloAffAntiAffSpread},
			want:    nil,
		},
		{
			name:    "pod in a sufficient pod group with affinity",
			pod:     pGangWithPA1,
			allpods: []*corev1.Pod{pGangWithPA1, pGangWithPA2},
			want:    fwk.NewStatus(fwk.UnschedulableAndUnresolvable, "Pods with spec.affinity.podAffinity may not use pod group scheduling."),
		},
		{
			name:    "pod in a sufficient pod group with anti-affinity",
			pod:     pGangWithPAA1,
			allpods: []*corev1.Pod{pGangWithPAA1, pGangWithPAA2},
			want:    fwk.NewStatus(fwk.UnschedulableAndUnresolvable, "Pods with spec.affinity.podAntiAffinity may not use pod group scheduling."),
		},
		{
			name:    "pod in a sufficient pod group with hard spreading",
			pod:     pGangWithHardTSC1,
			allpods: []*corev1.Pod{pGangWithHardTSC1, pGangWithHardTSC2},
			want:    fwk.NewStatus(fwk.UnschedulableAndUnresolvable, "Pods with a spec.topologySpreadConstraint with DoNotSchedule may not pod group scheduling."),
		},
		{
			name:    "pod in a sufficient pod group with soft spreading",
			pod:     pGangWithSoftTSC1,
			allpods: []*corev1.Pod{pGangWithSoftTSC1, pGangWithSoftTSC2},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			p, err := NewTestPlugin(ctx, tt.allpods, feature.Features{})
			if err != nil {
				t.Fatalf("Creating plugin: %v", err)
			}

			got := p.(framework.PreEnqueuePlugin).PreEnqueue(ctx, tt.pod)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected status (-want, +got):\n%s", diff)
			}
		})
	}
}
