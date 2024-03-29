package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Phase of the RF status
type Phase string

// Condition saves the state information of the redis cluster
type Condition struct {
	// Status of cluster condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime string      `json:"lastUpdateTime,omitempty"`
	LastUpdateAt   metav1.Time `json:"-"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// ConditionType defines the condition that the RF can have
type ConditionType string

const (
	ClusterConditionAvailable   ConditionType = "Available"
	ClusterConditionHealthy     ConditionType = "Healthy"
	ClusterConditionRunning                   = "Running"
	ClusterConditionCreating                  = "Creating"
	ClusterConditionRecovering                = "Recovering"
	ClusterConditionScaling                   = "Scaling"
	ClusterConditionScalingDown               = "ScalingDown"
	ClusterConditionUpgrading                 = "Upgrading"
	ClusterConditionUpdating                  = "Updating"
	ClusterConditionFailed                    = "Failed"
)

// RedisSentinelStatus defines the observed state of RedisSentinel
// +k8s:openapi-gen=true
type RedisSentinelStatus struct {
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions []Condition `json:"conditions,omitempty"`
	MasterIP   string      `json:"masterIP,omitempty"`
	SentinelIP string      `json:"sentinelIP,omitempty"`
}

func (cs *RedisSentinelStatus) DescConditionsByTime() {
	sort.Slice(cs.Conditions, func(i, j int) bool {
		return cs.Conditions[i].LastUpdateAt.Time.After(cs.Conditions[j].LastUpdateAt.Time)
	})
}

func (cs *RedisSentinelStatus) SetScalingUpCondition(message string) {
	c := newClusterCondition(ClusterConditionScaling, corev1.ConditionTrue, "Scaling up", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) SetCreateCondition(message string) {
	c := newClusterCondition(ClusterConditionCreating, corev1.ConditionTrue, "Creating", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) SetScalingDownCondition(message string) {
	c := newClusterCondition(ClusterConditionScaling, corev1.ConditionTrue, "Scaling down", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) SetUpgradingCondition(message string) {
	c := newClusterCondition(ClusterConditionUpgrading, corev1.ConditionTrue,
		"Cluster upgrading", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) SetUpdatingCondition(message string) {
	c := newClusterCondition(ClusterConditionUpdating, corev1.ConditionTrue,
		"Cluster updating", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) SetReadyCondition(message string) {
	c := newClusterCondition(ClusterConditionHealthy, corev1.ConditionTrue, "Cluster available", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) SetFailedCondition(message string) {
	c := newClusterCondition(ClusterConditionFailed, corev1.ConditionTrue,
		"Cluster failed", message)
	cs.setClusterCondition(*c)
}

func (cs *RedisSentinelStatus) ClearCondition(t ConditionType) {
	pos, _ := getClusterCondition(cs, t)
	if pos == -1 {
		return
	}
	cs.Conditions = append(cs.Conditions[:pos], cs.Conditions[pos+1:]...)
}

func (cs *RedisSentinelStatus) setClusterCondition(c Condition) {
	pos, cp := getClusterCondition(cs, c.Type)
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

func getClusterCondition(status *RedisSentinelStatus, t ConditionType) (int, *Condition) {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

func newClusterCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
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
