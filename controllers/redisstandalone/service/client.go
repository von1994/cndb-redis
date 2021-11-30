package service

import (
	"github.com/go-logr/logr"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redisstandalone"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RedisStandaloneClient interface {
	EnsureRedisStatefulset(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisHeadlessService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisMonitorService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisShutdownConfigMap(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisConfigMap(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error
}

type RedisStandaloneKubeClient struct {
	K8SService k8s.Services
	logger     logr.Logger
}

func NewRedisStandaloneKubeClient(k8sService k8s.Services, logger logr.Logger) *RedisStandaloneKubeClient {
	return &RedisStandaloneKubeClient{
		K8SService: k8sService,
		logger:     logger,
	}
}

func shouldUpdateRedis(expectResource, containterResource corev1.ResourceRequirements, expectSize, replicas int32) bool {
	if expectSize != replicas {
		return true
	}
	if result := containterResource.Requests.Cpu().Cmp(*expectResource.Requests.Cpu()); result != 0 {
		return true
	}
	if result := containterResource.Requests.Memory().Cmp(*expectResource.Requests.Memory()); result != 0 {
		return true
	}
	if result := containterResource.Limits.Cpu().Cmp(*expectResource.Limits.Cpu()); result != 0 {
		return true
	}
	if result := containterResource.Limits.Memory().Cmp(*expectResource.Limits.Memory()); result != 0 {
		return true
	}
	return false
}

func exporterChanged(rc *redisv1alpha1.RedisStandalone, sts *appsv1.StatefulSet) bool {
	if rc.Spec.Exporter.Enabled {
		for _, container := range sts.Spec.Template.Spec.Containers {
			if container.Name == common.ExporterContainerName {
				return false
			}
		}
		return true
	} else {
		for _, container := range sts.Spec.Template.Spec.Containers {
			if container.Name == common.ExporterContainerName {
				return true
			}
		}
		return false
	}
}

func (r *RedisStandaloneKubeClient) EnsureRedisStatefulset(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	oldSs, err := r.K8SService.GetStatefulSet(rc.Namespace, redisstandalone.GetRedisName(rc))
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			ss := generateRedisStatefulSet(rc, labels, ownerRefs)
			return r.K8SService.CreateStatefulSet(rc.Namespace, ss)
		}
		return err
	}

	if shouldUpdateRedis(rc.Spec.Resources, oldSs.Spec.Template.Spec.Containers[0].Resources,
		rc.Spec.Size, *oldSs.Spec.Replicas) || exporterChanged(rc, oldSs) {
		ss := generateRedisStatefulSet(rc, labels, ownerRefs)
		return r.K8SService.UpdateStatefulSet(rc.Namespace, ss)
	}

	return nil
}

func (r *RedisStandaloneKubeClient) EnsureRedisService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateRedisService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

func (r *RedisStandaloneKubeClient) EnsureRedisHeadlessService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateHeadlessRedisService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

func (r *RedisStandaloneKubeClient) EnsureRedisMonitorService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateRedisMonitorService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

func (r *RedisStandaloneKubeClient) EnsureRedisShutdownConfigMap(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if rc.Spec.ShutdownConfigMap != "" {
		if _, err := r.K8SService.GetConfigMap(rc.Namespace, rc.Spec.ShutdownConfigMap); err != nil {
			return err
		}
	} else {
		cm := generateRedisShutdownConfigMap(rc, labels, ownerRefs)
		return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
	}
	return nil
}

func (r *RedisStandaloneKubeClient) EnsureRedisConfigMap(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateRedisConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}
