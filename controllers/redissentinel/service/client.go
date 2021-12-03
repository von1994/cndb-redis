package service

import (
	"github.com/go-logr/logr"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redissentinel"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/util"
)

// RedisSentinelClient has the minimumm methods that a Redis cluster controller needs to satisfy
// in order to talk with K8s
type RedisSentinelClient interface {
	EnsureSentinelService(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelHeadlessService(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelConfigMap(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelProbeConfigMap(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelStatefulset(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisStatefulset(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisService(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisMonitorService(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisShutdownConfigMap(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisConfigMap(RedisSentinel *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureNotPresentRedisService(RedisSentinel *redisv1alpha1.RedisSentinel) error
}

var _ RedisSentinelClient = &RedisSentinelKubeClient{}

// RedisSentinelKubeClient implements the required methods to talk with kubernetes
type RedisSentinelKubeClient struct {
	K8SService k8s.Services
	logger     logr.Logger
}

// NewRedisSentinelKubeClient creates a new RedisSentinelKubeClient
func NewRedisSentinelKubeClient(k8sService k8s.Services, logger logr.Logger) *RedisSentinelKubeClient {
	return &RedisSentinelKubeClient{
		K8SService: k8sService,
		logger:     logger,
	}
}

func generateSelectorLabels(component, name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":   util.AppLabel,
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/name":      name,
	}
}

// EnsureSentinelService makes sure the sentinel service exists
func (r *RedisSentinelKubeClient) EnsureSentinelService(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateSentinelService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureSentinelHeadlessService makes sure the sentinel headless service exists
func (r *RedisSentinelKubeClient) EnsureSentinelHeadlessService(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := newHeadLessSvcForCR(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureSentinelConfigMap makes sure the sentinel configmap exists
func (r *RedisSentinelKubeClient) EnsureSentinelConfigMap(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateSentinelConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}

// EnsureSentinelProbeConfigMap makes sure the sentinel configmap exists
func (r *RedisSentinelKubeClient) EnsureSentinelProbeConfigMap(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateSentinelReadinessProbeConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}

// EnsureSentinelStatefulset makes sure the sentinel deployment exists in the desired state
func (r *RedisSentinelKubeClient) EnsureSentinelStatefulset(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := r.ensurePodDisruptionBudget(rc, util.SentinelName, util.SentinelRoleName, labels, ownerRefs); err != nil {
		return err
	}

	oldSs, err := r.K8SService.GetStatefulSet(rc.Namespace, redissentinel.GetSentinelName(rc))
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			ss := generateSentinelStatefulSet(rc, labels, ownerRefs)
			return r.K8SService.CreateStatefulSet(rc.Namespace, ss)
		}
		return err
	}

	if shouldUpdateRedis(rc.Spec.Sentinel.Resources, oldSs.Spec.Template.Spec.Containers[0].Resources, rc.Spec.Sentinel.Replicas, *oldSs.Spec.Replicas) {
		ss := generateSentinelStatefulSet(rc, labels, ownerRefs)
		return r.K8SService.UpdateStatefulSet(rc.Namespace, ss)
	}
	return nil
}

// EnsureRedisStatefulset makes sure the redis statefulset exists in the desired state
func (r *RedisSentinelKubeClient) EnsureRedisStatefulset(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := r.ensurePodDisruptionBudget(rc, util.RedisName, util.RedisRoleName, labels, ownerRefs); err != nil {
		return err
	}

	oldSs, err := r.K8SService.GetStatefulSet(rc.Namespace, redissentinel.GetRedisName(rc))
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

func exporterChanged(rc *redisv1alpha1.RedisSentinel, sts *appsv1.StatefulSet) bool {
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

// EnsureRedisConfigMap makes sure the sentinel configmap exists
func (r *RedisSentinelKubeClient) EnsureRedisConfigMap(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateRedisConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}

// EnsureRedisShutdownConfigMap makes sure the redis configmap with shutdown script exists
func (r *RedisSentinelKubeClient) EnsureRedisShutdownConfigMap(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
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

// EnsureRedisService makes sure the redis statefulset exists
func (r *RedisSentinelKubeClient) EnsureRedisService(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateRedisService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureRedisMonitorService makes sure the redis statefulset exists
func (r *RedisSentinelKubeClient) EnsureRedisMonitorService(rc *redisv1alpha1.RedisSentinel, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateRedisMonitorService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureNotPresentRedisService makes sure the redis service is not present
func (r *RedisSentinelKubeClient) EnsureNotPresentRedisService(rc *redisv1alpha1.RedisSentinel) error {
	name := redissentinel.GetRedisName(rc)
	namespace := rc.Namespace
	// If the service exists (no get error), delete it
	if _, err := r.K8SService.GetService(namespace, name); err == nil {
		return r.K8SService.DeleteService(namespace, name)
	}
	return nil
}

// EnsureRedisStatefulset makes sure the pdb exists in the desired state
func (r *RedisSentinelKubeClient) ensurePodDisruptionBudget(rc *redisv1alpha1.RedisSentinel, name string, component string, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	name = util.GenerateName(name, rc.Name)
	namespace := rc.Namespace

	minAvailable := intstr.FromInt(2)
	labels = util.MergeLabels(labels, generateSelectorLabels(component, rc.Name))

	pdb := generatePodDisruptionBudget(name, namespace, labels, ownerRefs, minAvailable)

	return r.K8SService.CreateIfNotExistsPodDisruptionBudget(namespace, pdb)
}
