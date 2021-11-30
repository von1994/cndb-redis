package controllers

import (
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the RedisCluster's components are correct.
func (r *RedisStandaloneHandler) Ensure(rc *redisv1alpha1.RedisStandalone, labels map[string]string, or []metav1.OwnerReference) error {
	if err := r.rcService.EnsureRedisService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisHeadlessService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisStatefulset(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisConfigMap(rc, labels, or); err != nil {
		return err
	}

	return nil
}

func (r *RedisStandaloneHandler) CheckAndHeal(meta *cache.Meta) error {
	return nil
}
