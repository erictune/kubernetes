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

package podsets

import (
	"context"
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
	psutil "k8s.io/kubernetes/pkg/scheduler/util/podset"
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
	return &PodSets{
		context:   c,
		handle:    nil,
		podLister: &MockPodLister{l: allpods},
	}, nil
}

func TestPreEnqueue(t *testing.T) {
	// This test does not try to cover all possible cases for pod equivalence or for pod compatibility.  Those are covered in
	// the k8s.io/pkg/scheduler/util/podset unit tests.
	pSolo := st.MakePod().Name("p").Namespace("ns1").Obj()
	pSolo2 := st.MakePod().Name("q").Namespace("ns1").Obj()
	pPodSet := st.MakePod().Name("ps-1").Namespace("ns1").Label(psutil.PodSetSizeLabelKey, "2").Label(psutil.PodSetNameLabelKey, "grp-foo").Obj()
	pPodSet2 := st.MakePod().Name("ps-2").Namespace("ns1").Label(psutil.PodSetSizeLabelKey, "2").Label(psutil.PodSetNameLabelKey, "grp-foo").Obj()
	pPodSetNotIdentical := st.MakePod().Name("ps-3").Namespace("ns2").Label(psutil.PodSetSizeLabelKey, "2").Label(psutil.PodSetNameLabelKey, "grp-foo").Obj()
	pPodSetIncompatible := st.MakePod().Name("ps-1").Namespace("ns1").Label(psutil.PodSetSizeLabelKey, "2").Label(psutil.PodSetNameLabelKey, "grp-foo").PodAffinity("foo", nil, st.PodAffinityWithRequiredReq).Obj()

	tests := []struct {
		name    string
		pod     *corev1.Pod
		allpods []*corev1.Pod
		want    *fwk.Status
	}{
		{
			name:    "pod not in a podset; no other pods present.",
			pod:     pSolo,
			allpods: []*corev1.Pod{pSolo},
			want:    nil,
		},
		{
			name:    "pod not in a podset; one pod from a podgroup is present.",
			pod:     pSolo,
			allpods: []*corev1.Pod{pSolo, pPodSet},
			want:    nil,
		},
		{
			name:    "pod in a podset; no other pods present.",
			pod:     pPodSet,
			allpods: []*corev1.Pod{pPodSet},
			want:    fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in podset ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name:    "pod in a podset; unrelated pods are in the system, but no peers",
			pod:     pPodSet,
			allpods: []*corev1.Pod{pSolo, pPodSet, pSolo2},
			want:    fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in podset ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name:    "both pods from a podset of size 2",
			pod:     pPodSet,
			allpods: []*corev1.Pod{pPodSet, pPodSet2},
			want:    nil,
		},
		{
			name:    "pod asking for podset but uses disallowed spec",
			pod:     pPodSetIncompatible,
			allpods: []*corev1.Pod{pPodSetIncompatible},
			want:    fwk.NewStatus(fwk.UnschedulableAndUnresolvable, "pod incompatible with podset scheduling:  pods with spec.affinity.podAffinity may not be in a podset"),
		},
		{
			name:    "pod asking for podset, but does not match the other pod in the podset",
			pod:     pPodSet,
			allpods: []*corev1.Pod{pPodSet, pPodSetNotIdentical},
			want:    fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in podset ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name:    "pod asking for podset, but does not match the other pod in the podset, reversed",
			pod:     pPodSet,
			allpods: []*corev1.Pod{pPodSetNotIdentical, pPodSet},
			want:    fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in podset ns1/grp-foo (seen: 1, min: 2)"),
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
