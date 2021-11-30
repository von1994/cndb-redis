package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultSentinelCustomConfig = []string{"down-after-milliseconds 5000", "failover-timeout 10000"}
)

// Validate set the values by default if not defined and checks if the values given are valid
func (r *RedisSentinel) Validate() (bool, error) {
	rCopy := r.DeepCopy()
	if len(r.Name) > maxNameLength {
		return false, fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	if r.Spec.Size == 0 {
		r.Spec.Size = defaultRedisNumber
	} else if r.Spec.Size < defaultRedisNumber {
		return false, errors.New("number of redis in spec is less than the minimum")
	}

	if r.Spec.Sentinel.Replicas == 0 {
		r.Spec.Sentinel.Replicas = defaultSentinelNumber
	} else if r.Spec.Sentinel.Replicas < defaultSentinelNumber {
		return false, errors.New("number of sentinels in spec is less than the minimum")
	}

	if r.Spec.Image == "" {
		r.Spec.Image = defaultRedisImage
	}

	if r.Spec.Sentinel.Image == "" {
		r.Spec.Sentinel.Image = defaultRedisImage
	}

	if r.Spec.Sentinel.Resources.Size() == 0 {
		r.Spec.Sentinel.Resources = defaultSentinelResource()
	}

	if r.Spec.Config == nil {
		r.Spec.Config = make(map[string]string)
	}

	r.Spec.Config["slave-priority"] = defaultSlavePriority

	if !r.Spec.DisablePersistence {
		enablePersistence(r.Spec.Config)
	} else {
		disablePersistence(r.Spec.Config)
	}

	if r.Spec.Exporter.Enabled {
		if r.Spec.Exporter.Image == "" {
			r.Spec.Exporter.Image = defaultRedisExporterImage
		}
		if r.Spec.Exporter.ImagePullPolicy == "" {
			r.Spec.Exporter.ImagePullPolicy = defaultImagePullPolicy
		}
	}

	return !reflect.DeepEqual(r, rCopy), nil
}

func defaultSentinelResource() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("20m"),
			v1.ResourceMemory: resource.MustParse("16Mi"),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("100m"),
			v1.ResourceMemory: resource.MustParse("60Mi"),
		},
	}
}
