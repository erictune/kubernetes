/*
Copyright 2022 The Kubernetes Authors.

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

type MockPodLister struct{
	l []*corev1.Pod
}

func (mpl *MockPodLister) List(selector labels.Selector) (ret []*corev1.Pod, err error) {
	return mpl.l, nil
}
func (mpl *MockPodLister) Pods(namespace string) listerscorev1.PodNamespaceLister {
	panic("not implemented")
}

// New initializes a new plugin for testing and returns it.
func NewTestPlugin(c context.Context, allpods  []*corev1.Pod, fts feature.Features) (framework.Plugin, error) {
	return &Gang{
		context:   c,
		handle:    nil,
		podLister: &MockPodLister{ l: allpods },
	}, nil
}

func makeGangOfPods(pgName string, nPods int) []*corev1.Pod {
	g := []*corev1.Pod{}
	for i := range(nPods) {
		g = append(g, st.MakePod().Name(fmt.Sprintf("p-%v", i)).Namespace("ns1").Label(PodGroupMinSizeLabelKey, "2").Label(PodGroupNameLabelKey, "grp-foo") .Obj(),
		)
	}
	return g
}

func TestPreEnqueue(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		allpods  []*corev1.Pod
		want *fwk.Status
	}{
		{
			name: "pod does not belong to a pod group, and no other pods present.",
			pod:  st.MakePod().Name("p").Namespace("ns1").Obj(),
			allpods:  []*corev1.Pod{ st.MakePod().Name("p").Obj() },
			want: nil,
		},
		{
			name: "pod does not belong to a pod group, but one pod from a podgroup is present.",
			pod:  st.MakePod().Name("p").Namespace("ns1").Obj(),
			allpods:  append( []*corev1.Pod{ st.MakePod().Name("p").Obj() }, makeGangOfPods("gangpod", 1)...),
			want: nil,
		},
		{
			name: "pod belongs to a pod group and is the only pod around",
			pod:  makeGangOfPods("gangpod", 1)[0],
			allpods:  makeGangOfPods("gangpod", 1),
			want: fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in pod group ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name: "pod belongs to a pod group and unrelated pods are in the system, but no peers",
			pod:  makeGangOfPods("gangpod", 1)[0],
			allpods:  append( makeGangOfPods("gangpod", 1), st.MakePod().Name("p").Namespace("ns1").Obj(), st.MakePod().Name("p2").Obj()),
			want: fwk.NewStatus(fwk.Unschedulable, "waiting for enough pods in pod group ns1/grp-foo (seen: 1, min: 2)"),
		},
		{
			name: "pod belongs to a pod group and two peers are in the system, and the minimum is 2 total",
			pod:  makeGangOfPods("gangpod", 1)[0],
			allpods:  makeGangOfPods("gangpod", 2),
			want: nil,
		},
		{
			name: "pod belongs to a pod group and three peers are in the system, and the minimum is 2 total",
			pod:  makeGangOfPods("gangpod", 1)[0],
			allpods:  makeGangOfPods("gangpod", 3),
			want: nil,
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
