package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// RedisStandaloneStatus defines the observed state of RedisStandalone
// +k8s:openapi-gen=true
type RedisStandaloneStatus struct {
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions []Condition `json:"conditions,omitempty"`
	MasterIP   string      `json:"masterIP,omitempty"`
	SentinelIP string      `json:"sentinelIP,omitempty"`
}

func (cs *RedisStandaloneStatus) DescConditionsByTime() {
	sort.Slice(cs.Conditions, func(i, j int) bool {
		return cs.Conditions[i].LastUpdateAt.Time.After(cs.Conditions[j].LastUpdateAt.Time)
	})
}

func (cs *RedisStandaloneStatus) SetScalingUpCondition(message string) {
	c := newClusterCondition(ClusterConditionScaling, corev1.ConditionTrue, "Scaling up", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) SetCreateCondition(message string) {
	c := newClusterCondition(ClusterConditionCreating, corev1.ConditionTrue, "Creating", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) SetScalingDownCondition(message string) {
	c := newClusterCondition(ClusterConditionScaling, corev1.ConditionTrue, "Scaling down", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) SetUpgradingCondition(message string) {
	c := newClusterCondition(ClusterConditionUpgrading, corev1.ConditionTrue,
		"Cluster upgrading", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) SetUpdatingCondition(message string) {
	c := newClusterCondition(ClusterConditionUpdating, corev1.ConditionTrue,
		"Cluster updating", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) SetReadyCondition(message string) {
	c := newClusterCondition(ClusterConditionHealthy, corev1.ConditionTrue, "Cluster available", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) SetFailedCondition(message string) {
	c := newClusterCondition(ClusterConditionFailed, corev1.ConditionTrue,
		"Cluster failed", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisStandaloneStatus) ClearCondition(t ConditionType) {
	pos, _ := getStandaloneCondition(cs, t)
	if pos == -1 {
		return
	}
	cs.Conditions = append(cs.Conditions[:pos], cs.Conditions[pos+1:]...)
}

func (cs *RedisStandaloneStatus) setClusterCondition(c Condition) {
	pos, cp := getStandaloneCondition(cs, c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		now := time.Now()
		nowString := now.Format(time.RFC3339)
		cs.Conditions[pos].LastUpdateAt = metav1.Now()
		cs.Conditions[pos].LastUpdateTime = nowString
		return
	}

	if cp != nil {
		cs.Conditions[pos] = c
	} else {
		cs.Conditions = append(cs.Conditions, c)
	}
}

func getStandaloneCondition(status *RedisStandaloneStatus, t ConditionType) (int, *Condition) {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

func newStandaloneCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
	now := time.Now()
	nowString := now.Format(time.RFC3339)
	return &Condition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     nowString,
		LastUpdateAt:       metav1.Now(),
		LastTransitionTime: nowString,
		Reason:             reason,
		Message:            message,
	}
}
