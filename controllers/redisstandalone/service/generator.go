package service

import (
	"fmt"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redisstandalone"
	"github.com/von1994/cndb-redis/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getRedisCommand(rc *redisv1alpha1.RedisStandalone) []string {
	if len(rc.Spec.Command) > 0 {
		return rc.Spec.Command
	}

	cmds := []string{
		"redis-server",
		"/etc/redis/redis.conf",
	}

	return cmds
}

func getRedisDataVolume(rc *redisv1alpha1.RedisStandalone) *corev1.Volume {
	// This will find the volumed desired by the user. If no volume defined
	// an EmptyDir will be used by default
	switch {
	case rc.Spec.Storage.PersistentVolumeClaim != nil:
		return nil
	case rc.Spec.Storage.EmptyDir != nil:
		return &corev1.Volume{
			Name: redisStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: rc.Spec.Storage.EmptyDir,
			},
		}
	default:
		return &corev1.Volume{
			Name: redisStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
	}
}

func getRedisDataVolumeName(rc *redisv1alpha1.RedisStandalone) string {
	switch {
	case rc.Spec.Storage.PersistentVolumeClaim != nil:
		return rc.Spec.Storage.PersistentVolumeClaim.Name
	case rc.Spec.Storage.EmptyDir != nil:
		return redisStorageVolumeName
	default:
		return redisStorageVolumeName
	}
}

func getRedisVolumeMounts(rc *redisv1alpha1.RedisStandalone) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      redisConfigurationVolumeName,
			MountPath: "/etc/redis",
		},
		//{
		//	Name:      redisShutdownConfigurationVolumeName,
		//	MountPath: "/redis-shutdown",
		//},
		//{
		//	Name:      getRedisDataVolumeName(rc),
		//	MountPath: "/data",
		//},
	}

	return volumeMounts
}

func getRedisVolumes(rc *redisv1alpha1.RedisStandalone) []corev1.Volume {
	//shutdownConfigMapName := redisstandalone.GetRedisShutdownConfigMapName(rc)
	//
	//executeMode := int32(0744)
	readMode := int32(0644)
	volumes := []corev1.Volume{
		{
			Name: redisConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: redisstandalone.GetRedisName(rc)},
				DefaultMode:          &readMode,
			},
			}},
		//{
		//	Name: redisShutdownConfigurationVolumeName,
		//	VolumeSource: corev1.VolumeSource{
		//		ConfigMap: &corev1.ConfigMapVolumeSource{
		//			LocalObjectReference: corev1.LocalObjectReference{
		//				Name: shutdownConfigMapName,
		//			},
		//			DefaultMode: &executeMode,
		//		},
		//	},
		//},
	}

	dataVolume := getRedisDataVolume(rc)
	if dataVolume != nil {
		volumes = append(volumes, *dataVolume)
	}

	return volumes
}

func generateSelectorLabels(component, name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":   util.AppLabel,
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/name":      name,
	}
}

func generateRedisService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	name := redisstandalone.GetRedisName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	redisTargetPort := intstr.FromInt(6379)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       6379,
					Protocol:   corev1.ProtocolTCP,
					Name:       "redis",
					TargetPort: redisTargetPort,
				},
			},
			Selector: labels,
		},
	}
}

func generateHeadlessRedisService(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	name := redisstandalone.GetRedisName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	redisTargetPort := intstr.FromInt(6379)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name + "-headless",
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{
					Port:       6379,
					Protocol:   corev1.ProtocolTCP,
					Name:       "redis",
					TargetPort: redisTargetPort,
				},
			},
			Selector: labels,
		},
	}
}

func generateRedisStatefulSet(rc *redisv1alpha1.RedisStandalone, labels map[string]string,
	ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := redisstandalone.GetRedisName(rc)
	namespace := rc.Namespace

	spec := rc.Spec
	redisCommand := getRedisCommand(rc)
	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))

	volumeMounts := getRedisVolumeMounts(rc)
	volumes := getRedisVolumes(rc)

	probeArg := "redis-cli -h $(hostname)"
	if spec.Password != "" {
		probeArg = fmt.Sprintf("%s -a '%s' ping", probeArg, spec.Password)
	} else {
		probeArg = fmt.Sprintf("%s ping", probeArg)
	}

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &spec.Size,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: "RollingUpdate",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: rc.Spec.Annotations,
				},
				Spec: corev1.PodSpec{
					Affinity:         common.GetAffinity(rc.Spec.Affinity, labels),
					Tolerations:      rc.Spec.ToleRations,
					NodeSelector:     rc.Spec.NodeSelector,
					SecurityContext:  common.GetSecurityContext(rc.Spec.SecurityContext),
					ImagePullSecrets: rc.Spec.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Name:            "redis",
							Image:           rc.Spec.Image,
							ImagePullPolicy: common.PullPolicy(rc.Spec.ImagePullPolicy),
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volumeMounts,
							Command:      redisCommand,
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: graceTime,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											probeArg,
										},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: graceTime,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											probeArg,
										},
									},
								},
							},
							Resources: rc.Spec.Resources,
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "/redis-shutdown/shutdown.sh"},
									},
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if rc.Spec.Storage.PersistentVolumeClaim != nil {
		if !rc.Spec.Storage.KeepAfterDeletion {
			// Set an owner reference so the persistent volumes are deleted when the rc is
			rc.Spec.Storage.PersistentVolumeClaim.OwnerReferences = ownerRefs
		}
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			*rc.Spec.Storage.PersistentVolumeClaim,
		}
	}

	//if rc.Spec.Exporter.Enabled {
	//	exporter := createRedisExporterContainer(rc)
	//	ss.Spec.Template.Spec.Containers = append(ss.Spec.Template.Spec.Containers, exporter)
	//}

	return ss
}

func generateRedisConfigMap(rc *redisv1alpha1.RedisStandalone, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	name := redisstandalone.GetRedisName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	redisConfigFileContent := common.RedisBasicConfig
	if rc.Spec.Password != "" {
		redisConfigFileContent = fmt.Sprintf("%s\nrequirepass %s\nmasterauth %s\n", redisConfigFileContent, rc.Spec.Password, rc.Spec.Password)
	}

	for k, v := range rc.Spec.Config {
		redisConfigFileContent = fmt.Sprintf("%s\n%s %s", redisConfigFileContent, k, v)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{
			util.RedisConfigFileName: redisConfigFileContent,
		},
	}
}
