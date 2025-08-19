package podset

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
)

func TestIsPodSetPod(t *testing.T) {
	pPlain := st.MakePod().Name("plain").Namespace("ns1").Obj()
	pNameOnly := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetNameLabelKey, "foo").Obj()
	pSizeOnly := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetSizeLabelKey, "2").Obj()
	pBoth := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetNameLabelKey, "foo").Label(PodSetSizeLabelKey, "2").Obj()

	tests := []struct {
		name string
		pod  *v1.Pod
		want bool
	}{
		{
			name: "plain pod.",
			pod:  pPlain,
			want: false,
		},
		{
			name: "name only.",
			pod:  pNameOnly,
			want: true,
		},
		{
			name: "size only.",
			pod:  pSizeOnly,
			want: false,
		},
		{
			name: "both.",
			pod:  pBoth,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPodSetPod(tt.pod)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("unexpected difference (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestPodSetSize(t *testing.T) {
	pPlain := st.MakePod().Name("plain").Namespace("ns1").Obj()
	pNameOnly := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetNameLabelKey, "foo").Obj()
	pSizeOnly := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetSizeLabelKey, "2").Obj()
	pBoth := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetNameLabelKey, "foo").Label(PodSetSizeLabelKey, "2").Obj()
	pNonIntSize := st.MakePod().Name("plain").Namespace("ns1").Label(PodSetNameLabelKey, "foo").Label(PodSetSizeLabelKey, "Twooo").Obj()

	tests := []struct {
		name    string
		pod     *v1.Pod
		wantRes int
		wantErr error
	}{
		{
			name:    "plain pod.",
			pod:     pPlain,
			wantRes: 0,
			wantErr: nil,
		},
		{
			name:    "name only.",
			pod:     pNameOnly,
			wantRes: 0,
			wantErr: nil,
		},
		{
			name:    "size only.",
			pod:     pSizeOnly,
			wantRes: 2,
			wantErr: nil,
		},
		{
			name:    "both.",
			pod:     pBoth,
			wantRes: 2,
			wantErr: nil,
		},
		{
			name:    "non-int size.",
			pod:     pNonIntSize,
			wantRes: 0,
			wantErr: ErrInvalidSize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, gotErr := PodSetSize(tt.pod)
			if diff := cmp.Diff(tt.wantRes, gotRes); diff != "" {
				t.Errorf("unexpected difference (-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantErr, gotErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("unexpected difference (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestPodSetName(t *testing.T) {
	pPlain := st.MakePod().Name("plain").Namespace("ns1").Obj()
	pName := st.MakePod().Name("psname").Namespace("ns1").Label(PodSetNameLabelKey, "foo").Obj()
	tests := []struct {
		name string
		pod  *v1.Pod
		want string
	}{
		{
			name: "plain pod.",
			pod:  pPlain,
			want: "",
		},
		{
			name: "podset name.",
			pod:  pName,
			want: "foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PodSetLabel(tt.pod)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected status (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestIsPodSetCompatible(t *testing.T) {
	pPlain := st.MakePod().Name("plain").Namespace("ns1").Obj()
	pPA := st.MakePod().Name("pa").Namespace("ns1").PodAffinity("foo", nil, st.PodAffinityWithRequiredReq).Obj()
	pPAA := st.MakePod().Name("paa").Namespace("ns1").PodAntiAffinity("foo", nil, st.PodAntiAffinityWithRequiredPreferredReq).Obj()
	pHardTSC := st.MakePod().Name("phtsc").Namespace("ns1").SpreadConstraint(5, "foo", v1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj()
	pSoftTSC := st.MakePod().Name("pstsc").Namespace("ns1").SpreadConstraint(5, "foo", v1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj()

	tests := []struct {
		name string
		pod  *v1.Pod
		want error
	}{
		{
			name: "plain pod.",
			pod:  pPlain,
			want: nil,
		},
		{
			name: "affinity pod.",
			pod:  pPA,
			want: ErrPodSetUsesAffinity,
		},
		{
			name: "anti-affinity pod.",
			pod:  pPAA,
			want: ErrPodSetUsesAntiAffinity,
		},
		{
			name: "pod with hard spreading.",
			pod:  pHardTSC,
			want: ErrPodSetUsesHardSpreading,
		},
		{
			name: "pod with soft spreading.",
			pod:  pSoftTSC,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPodSetCompatible(tt.pod)
			if tt.want == nil && got != nil {
				t.Errorf("unexpected status: want is nil, got is:\n%s", got.Error())
			}
			if tt.want != nil && got == nil {
				t.Errorf("unexpected status: want is non-nil, got is nil:\n%s", tt.want.Error())
			}
			if tt.want != nil && got != nil {
				if diff := cmp.Diff(tt.want.Error(), got.Error()); diff != "" {
					t.Errorf("unexpected status (-want, +got):\n%s", diff)
				}
			}
		})
	}
}

func TestEquivalentForFeasibility(t *testing.T) {
	vol1 := v1.Volume{
		VolumeSource: v1.VolumeSource{
			AWSElasticBlockStore: &v1.AWSElasticBlockStoreVolumeSource{
				VolumeID: "foo",
			},
		},
	}
	vol2 := v1.Volume{
		VolumeSource: v1.VolumeSource{
			GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{
				PDName: "foo",
			},
		},
	}
	// Resource Claim {Template} string names.
	rcGlobal := "rc-global"      // e.g. some cluster global resource that is claimable by many pods.
	rctOneGpu := "rct-one-gpu"   // e.g. a ResourceClaimTemplate for any one device of some type such as a GPU.
	rctTwoGpus := "rct-two-gpus" // e.g. a different ResourceClaimTemplate from above.
	rcThisGpu := "rc-this-gpu"   // e.g. a claim for a specific device id
	rcThatGpu := "rc-that-gpus"  // e.g. a claim for a different specific device id

	// Resource Requests
	res1_2 := map[v1.ResourceName]string{"cpu": "1000", "memory": "2G"}
	res2_4 := map[v1.ResourceName]string{"cpu": "2000", "memory": "4G"}
	res1_2_3 := map[v1.ResourceName]string{"cpu": "1000", "memory": "2G", "nvidia.com/gpu": "3"}
	resBE := map[v1.ResourceName]string{}

	portHTTP := []v1.ContainerPort{{ContainerPort: 80, Protocol: "TCP"}}
	portHTTPS := []v1.ContainerPort{{ContainerPort: 443, Protocol: "TCP"}}

	time1 := metav1.Now()
	time2 := metav1.Now()

	var ten int32 = 10
	var hundred int32 = 100

	tests := []struct {
		name    string
		p, q    *v1.Pod
		wantErr bool
	}{
		{
			name:    "identical pods",
			p:       st.MakePod().Name("p").Obj(),
			q:       st.MakePod().Name("p2").Obj(),
			wantErr: false,
		},
		{
			name:    "different namespace",
			p:       st.MakePod().Name("p").Namespace("foo").Obj(),
			q:       st.MakePod().Name("p2").Namespace("bar").Obj(),
			wantErr: true,
		},
		{
			name:    "different scheduler name",
			p:       st.MakePod().Name("p").SchedulerName("foo").Obj(),
			q:       st.MakePod().Name("p2").SchedulerName("bar").Obj(),
			wantErr: true,
		},
		{
			name:    "different tolerations",
			p:       st.MakePod().Name("p").Toleration("foo").Obj(),
			q:       st.MakePod().Name("p2").Toleration("bar").Obj(),
			wantErr: true,
		},
		{
			name:    "same tolerations",
			p:       st.MakePod().Name("p").Toleration("foo").Obj(),
			q:       st.MakePod().Name("p2").Toleration("foo").Obj(),
			wantErr: false,
		},
		{
			name:    "same node affinity",
			p:       st.MakePod().Name("p").NodeAffinityIn("node", []string{"node-a", "node-y"}, st.NodeSelectorTypeMatchExpressions).Obj(),
			q:       st.MakePod().Name("p2").NodeAffinityIn("node", []string{"node-a", "node-y"}, st.NodeSelectorTypeMatchExpressions).Obj(),
			wantErr: false,
		},
		{
			name: "different node affinity",
			p:    st.MakePod().Name("p").NodeAffinityIn("node", []string{"node-a", "node-y"}, st.NodeSelectorTypeMatchExpressions).Obj(),
			q:    st.MakePod().Name("p2").NodeAffinityIn("node", []string{"node-b", "node-z"}, st.NodeSelectorTypeMatchExpressions).Obj(),

			wantErr: true,
		},
		{
			name:    "different node affinity 2",
			p:       st.MakePod().Name("p").NodeAffinityIn("node", []string{"node-a", "node-y"}, st.NodeSelectorTypeMatchExpressions).Obj(),
			q:       st.MakePod().Name("p2").Obj(),
			wantErr: true,
		},
		{
			name:    "different node affinity 3",
			p:       st.MakePod().Name("p").Obj(),
			q:       st.MakePod().Name("p2").NodeAffinityIn("node", []string{"node-a", "node-y"}, st.NodeSelectorTypeMatchExpressions).NodeAffinityIn("zone", []string{"zone-a"}, st.NodeSelectorTypeMatchExpressions).Obj(),
			wantErr: true,
		},
		{
			name:    "different topology spread constraints",
			p:       st.MakePod().Name("p").Namespace("ns1").SpreadConstraint(3, "foo", v1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj(),
			q:       st.MakePod().Name("p2").Namespace("ßns1").SpreadConstraint(5, "foo", v1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj(),
			wantErr: true,
		},
		{
			name:    "different topology spread constraints 2",
			p:       st.MakePod().Name("p").Namespace("ns1").SpreadConstraint(5, "foo", v1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj(),
			q:       st.MakePod().Name("p2").Namespace("ns1").SpreadConstraint(5, "bar", v1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj(),
			wantErr: true,
		},
		{
			name:    "different topology spread constraints 3",
			p:       st.MakePod().Name("p").Namespace("ns1").SpreadConstraint(5, "foo", v1.DoNotSchedule, nil, nil, nil, nil, []string{}).Obj(),
			q:       st.MakePod().Name("p2").Namespace("ns1").SpreadConstraint(5, "foo", v1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj(),
			wantErr: true,
		},
		{
			name:    "same topology spread constraints",
			p:       st.MakePod().Name("p").Namespace("ns1").SpreadConstraint(5, "foo", v1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj(),
			q:       st.MakePod().Name("p2").Namespace("ns1").SpreadConstraint(5, "foo", v1.ScheduleAnyway, nil, nil, nil, nil, []string{}).Obj(),
			wantErr: false,
		},
		{
			name:    "different volumes",
			p:       st.MakePod().Volume(vol1).Obj(),
			q:       st.MakePod().Volume(vol2).Obj(),
			wantErr: true,
		},
		{
			name:    "different volumes 2",
			p:       st.MakePod().Volume(vol1).Obj(),
			q:       st.MakePod().Obj(),
			wantErr: true,
		},
		{
			name:    "same volumes",
			p:       st.MakePod().Volume(vol1).Obj(),
			q:       st.MakePod().Volume(vol1).Obj(),
			wantErr: false,
		},
		// Resource Claims
		{
			// Same ResourceClaim string on all pods of a podset is supported. Meaning: every pod wants to use the same resource slice.
			// Feasibility result should not differ between identical pods both with the same ResourceClaim string.
			name:    "same resourceClaim",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).Obj(),
			wantErr: false,
		},
		{
			// It probably doesn't matter if only the "Name" differs, but let's be restrictive until we know there is an need to allow it.
			name:    "different resourceClaims named",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c2", ResourceClaimName: &rcGlobal}).Obj(),
			wantErr: true,
		},
		{
			// Different claims make feasibility checks different, so not allowed.
			name:    "different resourceClaims referenced",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcThisGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcThatGpu}).Obj(),
			wantErr: true,
		},
		// Resource Claim Templates
		{
			// Same ResourceClaimTemplate string on all pods of a podset is supported. Meaning: every pod wants to use a differently-named but
			// identical claim.  Feasibility result should not differ between otherwise identical pods.
			name:    "same resourceClaimTemplate",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimTemplateName: &rctOneGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimTemplateName: &rctOneGpu}).Obj(),
			wantErr: false,
		},
		{
			// It probably doesn't matter if only the "Name" differs, since it is (presumably) only used as a merge key, but
			// until there is a need, we don't allow it to differ.
			name:    "same resourceClaimTemplate 2",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimTemplateName: &rctOneGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c2", ResourceClaimTemplateName: &rctOneGpu}).Obj(),
			wantErr: true,
		},
		{
			// Not allowed.  Different resourceClaimTemplates might not result in the same feasibility check outcomes.
			name:    "different resourceClaimTemplates",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimTemplateName: &rctOneGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimTemplateName: &rctTwoGpus}).Obj(),
			wantErr: true,
		},
		// Multiple Resource Claim{Template}s
		{
			// Not allowed.  Mix of claim and template.
			name:    "mixed resourceClaim and Template",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rctOneGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimTemplateName: &rctOneGpu}).Obj(),
			wantErr: true,
		},
		{
			// Not allowed.  Second one different.
			name:    "mixed resourceClaim and Template",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).PodResourceClaims(v1.PodResourceClaim{Name: "c2", ResourceClaimName: &rctOneGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).PodResourceClaims(v1.PodResourceClaim{Name: "c2", ResourceClaimName: &rctTwoGpus}).Obj(),
			wantErr: true,
		},
		{
			// Not allowed.  Different number of items.
			name:    "mixed resourceClaim and Template",
			p:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).PodResourceClaims(v1.PodResourceClaim{Name: "c2", ResourceClaimName: &rctOneGpu}).Obj(),
			q:       st.MakePod().Namespace("foo").PodResourceClaims(v1.PodResourceClaim{Name: "c", ResourceClaimName: &rcGlobal}).Obj(),
			wantErr: true,
		},
		// Containers
		{
			// Different containers resulting in different total resources is definitely not allowed.
			name:    "different number of containers",
			p:       st.MakePod().Req(res1_2).Obj(),
			q:       st.MakePod().Req(res1_2).Req(res1_2).Obj(),
			wantErr: true,
		},
		{
			// We could maybe support different containers that add up to the same resources, but the current implementation does not.
			name:    "different number of containers 2",
			p:       st.MakePod().Req(res2_4).Obj(),
			q:       st.MakePod().Req(res1_2).Req(res1_2).Obj(),
			wantErr: true,
		},
		{
			// One container per pod with the same resources.
			name:    "same number of the same containers",
			p:       st.MakePod().Req(res1_2).Obj(),
			q:       st.MakePod().Req(res1_2).Obj(),
			wantErr: false,
		},
		{
			// Two container per pod with the same resources.
			name:    "same number of the same containers 2",
			p:       st.MakePod().Req(res1_2).Req(res2_4).Req(res1_2).Obj(),
			q:       st.MakePod().Req(res1_2).Req(res2_4).Req(res1_2).Obj(),
			wantErr: false,
		},
		{
			name:    "differ by custom resource ",
			p:       st.MakePod().Req(res1_2).Obj(),
			q:       st.MakePod().Req(res1_2_3).Obj(),
			wantErr: true,
		},
		{
			name:    "BE QOS Container on one pod",
			p:       st.MakePod().Req(res1_2).Req(resBE).Obj(),
			q:       st.MakePod().Req(res1_2).Obj(),
			wantErr: false,
		},
		{
			// Different size init containers are not allowed, for simplicity, though in some cases this does not affect feasibility like in this test case.
			name:    "different size of init containers",
			p:       st.MakePod().InitReq(res1_2).Req(res2_4).Obj(),
			q:       st.MakePod().InitReq(res2_4).Req(res2_4).Req(res1_2).Obj(),
			wantErr: true,
		},
		{
			// Different number init containers are not allowed, for simplicity, though in some cases this does not affect feasibility like in this test case.
			name:    "different number of init containers",
			p:       st.MakePod().InitReq(res1_2).InitReq(res1_2).Req(res2_4).Obj(),
			q:       st.MakePod().Req(res2_4).Req(res1_2).Obj(),
			wantErr: true,
		},
		{
			// Same init and regular containers is allowed.
			name:    "same init and regular containers",
			p:       st.MakePod().InitReq(res1_2).Req(res2_4).Obj(),
			q:       st.MakePod().InitReq(res1_2).Req(res2_4).Obj(),
			wantErr: false,
		},
		// Ports
		{
			// Different host port is not allowed.
			name:    "different host port",
			p:       st.MakePod().HostPort(8080).Obj(),
			q:       st.MakePod().HostPort(443).Obj(),
			wantErr: true,
		},
		{
			// Same host port is allowed.
			name:    "same host port",
			p:       st.MakePod().HostPort(8080).Obj(),
			q:       st.MakePod().HostPort(8080).Obj(),
			wantErr: false,
		},
		{
			// Different container port is not allowed.
			name:    "different container port",
			p:       st.MakePod().ContainerPort(portHTTP).Obj(),
			q:       st.MakePod().ContainerPort(portHTTPS).Obj(),
			wantErr: true,
		},
		{
			// Same container port is allowed.
			name:    "same container port",
			p:       st.MakePod().ContainerPort(portHTTP).Obj(),
			q:       st.MakePod().ContainerPort(portHTTP).Obj(),
			wantErr: false,
		},
		{
			// Different init container port is not allowed.
			name:    "different container port",
			p:       st.MakePod().InitContainerPort(false, portHTTP).Obj(),
			q:       st.MakePod().InitContainerPort(false, portHTTPS).Obj(),
			wantErr: true,
		},
		{
			// Same init container port is allowed.
			name:    "same container port",
			p:       st.MakePod().InitContainerPort(false, portHTTP).Obj(),
			q:       st.MakePod().InitContainerPort(false, portHTTP).Obj(),
			wantErr: false,
		},
		{
			// Different Image is allowed (e.g. leader vs worker with otherwise identical requirements.). Image name does not
			// affect feasibility (though it may affect scoring a little.)
			name:    "different image name",
			p:       st.MakePod().Container("repo.example/dir/image:v1").Obj(),
			q:       st.MakePod().Container("repo.example/dir/image:v2").Obj(),
			wantErr: false,
		},
		{
			// Different Image is allowed (e.g. leader vs worker with otherwise identical requirements.). Image name does not
			// affect feasibility (though it may affect scoring a little.)
			name:    "different image name",
			p:       st.MakePod().Container("repo.example/dir/image:v1").Obj(),
			q:       st.MakePod().Container("repo.example/dir/image:v2").Obj(),
			wantErr: false,
		},
		{
			name:    "different priority",
			p:       st.MakePod().Priority(ten).Obj(),
			q:       st.MakePod().Priority(hundred).Obj(),
			wantErr: true,
		},
		{
			name:    "same priority",
			p:       st.MakePod().Priority(ten).Obj(),
			q:       st.MakePod().Priority(ten).Obj(),
			wantErr: false,
		},
		{
			// Different creation timestamps are totally expected for pods within a podset.
			name:    "different creationTimestamp",
			p:       st.MakePod().CreationTimestamp(time1).Obj(),
			q:       st.MakePod().CreationTimestamp(time2).Obj(),
			wantErr: false,
		},
		{
			// Pods can transiently have different stages of termination.
			name:    "different Termination state",
			p:       st.MakePod().Terminating().Obj(),
			q:       st.MakePod().Obj(),
			wantErr: false,
		},
		{
			// While bound, pods may be bound to different nodes.
			name:    "different nodenames state",
			p:       st.MakePod().Node("n1").Obj(),
			q:       st.MakePod().Node("n2").Obj(),
			wantErr: false,
		},
		{
			// some pods may be bound and some not - rare, but don't reject the entire podset if this happens..
			name:    "different boundness state",
			p:       st.MakePod().Node("n1").Obj(),
			q:       st.MakePod().Obj(),
			wantErr: false,
		},
		// XXX TODO:
		// PreemptionPolicy() from wrappers.go
		// Overhead()
		// UID()

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EquivalentForFeasibility(tt.p, tt.q)
			if (err != nil) != tt.wantErr {
				t.Errorf("EquivalentForFeasibility() error = %v, wantErr %v", safeErr(err), tt.wantErr)
			}
		})
		// Relationship is reflexive.
		t.Run(tt.name+"_reverse", func(t *testing.T) {
			err := EquivalentForFeasibility(tt.q, tt.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("EquivalentForFeasibility() error = %v, wantErr %v", safeErr(err), tt.wantErr)
			}
		})
	}
}

func safeErr(err error) string {
	if err != nil {
		return err.Error()
	}
	return "NIL"
}
