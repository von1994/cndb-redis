package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"
)

// Validate set the values by default if not defined and checks if the values given are valid
func (r *RedisStandalone) Validate() (bool, error) {
	rCopy := r.DeepCopy()

	if len(r.Name) > maxNameLength {
		return false, fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	if r.Spec.Size == 0 {
		r.Spec.Size = defaultStandaloneRedisNumber
	} else if r.Spec.Size != defaultStandaloneRedisNumber {
		return false, errors.New("standalone mode, number of redis in spec must be set 1")
	}

	if r.Spec.Image == "" {
		r.Spec.Image = defaultRedisImage
	}

	if r.Spec.Config == nil {
		r.Spec.Config = make(map[string]string)
	}

	if r.Spec.Exporter.Enabled {
		if r.Spec.Exporter.Image == "" {
			r.Spec.Exporter.Image = defaultRedisExporterImage
		}
		if r.Spec.Exporter.ImagePullPolicy == "" {
			r.Spec.Exporter.ImagePullPolicy = defaultImagePullPolicy
		}
	}

	// 增加持久化所需要的配置
	if !r.Spec.DisablePersistence {
		enablePersistence(r.Spec.Config)
	} else {
		disablePersistence(r.Spec.Config)
	}

	return !reflect.DeepEqual(r, rCopy), nil
}
