package common

import (
	"github.com/von1994/cndb-redis/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetAffinity 设置默认的Affinity
//  @param affinity
//  @param labels
//  @return *corev1.Affinity
func GetAffinity(affinity *corev1.Affinity, labels map[string]string) *corev1.Affinity {
	if affinity != nil {
		return affinity
	}

	// Return a SOFT anti-affinity
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						TopologyKey: util.HostnameTopologyKey,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
					},
				},
			},
		},
	}
}

// GetSecurityContext 设置默认的SecurityContext
// TODO: Add default setting
//  @param secctx
//  @return *corev1.PodSecurityContext
func GetSecurityContext(secctx *corev1.PodSecurityContext) *corev1.PodSecurityContext {
	if secctx != nil {
		return secctx
	}

	return nil
}

// PullPolicy 设置默认的镜像PullPolicy
//  @param specPolicy
//  @return corev1.PullPolicy
func PullPolicy(specPolicy corev1.PullPolicy) corev1.PullPolicy {
	if specPolicy == "" {
		return corev1.PullAlways
	}
	return specPolicy
}
