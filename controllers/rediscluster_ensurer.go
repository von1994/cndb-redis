package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
)

// Ensure the RedisCluster's components are correct.
func (r *RedisClusterHandler) Ensure(rc *redisv1alpha1.RedisCluster, labels map[string]string, or []metav1.OwnerReference) error {
	if err := r.rcService.EnsureRedisService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelHeadlessService(rc, labels, or); err != nil {
		return err
	}
	if rc.Spec.Exporter.Enabled {
		if err := r.rcService.EnsureRedisMonitorService(rc, labels, or); err != nil {
			return err
		}
	}
	if err := r.rcService.EnsureSentinelConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelProbeConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisShutdownConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisStatefulset(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelStatefulset(rc, labels, or); err != nil {
		return err
	}

	return nil
}
