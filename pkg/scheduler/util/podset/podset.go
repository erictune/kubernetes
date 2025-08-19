package podset

// A podset is a set of pods that can move through the scheduling cycle as a unit.
// Each pod of a podset needs to be IsPodSetCompatible().
// Each pair of pods in a podset must be EquivalentForFeasibility().
// Membership in a podset is indicated by labels kPodSetNameLabelKey and kPodSetSizeLabelKey.
// TODO: change the second one to an annotation.
import (
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

const PodSetNameLabelKey = "alpha.scheduler.k8s.io/pod-set-name"
const PodSetSizeLabelKey = "alpha.scheduler.k8s.io/pod-set-size"

var ErrInvalidSize = fmt.Errorf("invalid value for podset size")
var ErrPodSetUsesAffinity = fmt.Errorf("pods with spec.affinity.podAffinity may not be in a podset")
var ErrPodSetUsesAntiAffinity = fmt.Errorf("pods with spec.affinity.podAntiAffinity may not be in a podset")
var ErrPodSetUsesHardSpreading = fmt.Errorf("pods with a DoNotSchedule spec.topologySpreadConstraint item may not be in a podset")

// True if the pod is trying to be a podset (but may have invalid values).
func IsPodSetPod(pod *v1.Pod) bool {
	_, ok := pod.Labels[PodSetNameLabelKey]
	return ok
}

func PodSetLabel(pod *v1.Pod) string {
	v, ok := pod.Labels[PodSetNameLabelKey]
	if !ok {
		return ""
	}
	return v
}

func PodSetFullName(pod *v1.Pod) string {
	pg, ok := pod.Labels[PodSetNameLabelKey]
	if !ok {
		return ""
	}
	parts := []string{pod.Namespace, pg}
	return strings.Join(parts, "/")
}

func PodSetSize(pod *v1.Pod) (int, error) {
	pssStr, ok := pod.Labels[PodSetSizeLabelKey]
	if !ok {
		return 0, nil
	}
	pss, err := strconv.Atoi(pssStr)
	if err != nil {
		return 0, ErrInvalidSize
	}
	return pss, nil
}

// True if a pod is compatible with using PodSet scheduling.
// If an error is returned, the Caller would typically treat the entire podset that this pod
// is part of as UnschedulableAndUnresolvable.
func IsPodSetCompatible(p *v1.Pod) error {
	if p.Spec.Affinity != nil {
		if p.Spec.Affinity.PodAffinity != nil {
			if len(p.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 ||
				len(p.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
				return ErrPodSetUsesAffinity
			}
		}
		if p.Spec.Affinity.PodAntiAffinity != nil {
			if len(p.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 ||
				len(p.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
				return ErrPodSetUsesAntiAffinity
			}
		}
	}
	for _, tsc := range p.Spec.TopologySpreadConstraints {
		if tsc.WhenUnsatisfiable == v1.DoNotSchedule {
			return ErrPodSetUsesHardSpreading
		}
	}
	return nil
}

// True if two pods are identical for purposes of being in a PodSet scheduling.
func EquivalentForFeasibility(p *v1.Pod, q *v1.Pod) error {
	if p.Namespace != q.Namespace {
		return fmt.Errorf("pods have different namespaces: %q vs %q", p.Namespace, q.Namespace)
	}
	if p.Spec.SchedulerName != q.Spec.SchedulerName {
		return fmt.Errorf("pods have different scheduler names: %q vs %q", p.Spec.SchedulerName, q.Spec.SchedulerName)
	}
	if p.Spec.Priority != q.Spec.Priority {
		return fmt.Errorf("pods have different priorities: %d vs %d", p.Spec.Priority, q.Spec.Priority)
	}
	if !equality.Semantic.DeepEqual(p.Spec.Tolerations, q.Spec.Tolerations) {
		return fmt.Errorf("pods have different tolerations")
	}

	var pNodeAffinity, qNodeAffinity *v1.NodeAffinity
	if p.Spec.Affinity != nil {
		pNodeAffinity = p.Spec.Affinity.NodeAffinity
	}
	if q.Spec.Affinity != nil {
		qNodeAffinity = q.Spec.Affinity.NodeAffinity
	}
	if !equality.Semantic.DeepEqual(pNodeAffinity, qNodeAffinity) {
		return fmt.Errorf("pods have different node affinities")
	}

	if !equality.Semantic.DeepEqual(p.Spec.TopologySpreadConstraints, q.Spec.TopologySpreadConstraints) {
		return fmt.Errorf("pods have different topology spread constraints")
	}
	if !equality.Semantic.DeepEqual(p.Spec.Volumes, q.Spec.Volumes) {
		return fmt.Errorf("pods have different volumes")
	}
	if !equality.Semantic.DeepEqual(p.Spec.ResourceClaims, q.Spec.ResourceClaims) {
		return fmt.Errorf("pods have different resource claims")
	}

	if err := containersEquivalentForFeasibility(p.Spec.InitContainers, q.Spec.InitContainers, "init"); err != nil {
		return err
	}
	if err := containersEquivalentForFeasibility(p.Spec.Containers, q.Spec.Containers, "regular"); err != nil {
		return err
	}

	return nil
}

func containersEquivalentForFeasibility(pContainers, qContainers []v1.Container, containerType string) error {
	if len(pContainers) != len(qContainers) {
		return fmt.Errorf("pods have different number of %s containers", containerType)
	}
	for i := range pContainers {
		if !equality.Semantic.DeepEqual(pContainers[i].Ports, qContainers[i].Ports) {
			return fmt.Errorf("pods have different ports in %s %q", containerType, pContainers[i].Name)
		}
		if !equality.Semantic.DeepEqual(pContainers[i].Resources, qContainers[i].Resources) {
			return fmt.Errorf("pods have different resources in %s %q", containerType, pContainers[i].Name)
		}
	}
	return nil
}
