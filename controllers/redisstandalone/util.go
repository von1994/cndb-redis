package redisstandalone

import (
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/util"
)

// const
const (
	RedisName           = "-standalone"
	RedisShutdownName   = "r-s"
	//HostnameTopologyKey = "kubernetes.io/hostname"
)

// GetRedisName returns the name for redis resources
func GetRedisName(rc *redisv1alpha1.RedisStandalone) string {
	return util.GenerateName(RedisName, rc.Name)
}

// GetRedisShutdownConfigMapName 返回ShutDownConfigMap Name
//  @param rc
//  @return string
func GetRedisShutdownConfigMapName(rc *redisv1alpha1.RedisStandalone) string {
	if rc.Spec.ShutdownConfigMap != "" {
		return rc.Spec.ShutdownConfigMap
	}
	return GetRedisShutdownName(rc)
}

// GetRedisShutdownName 返回默认的返回ShutDownConfigMap Name
//  @param rc
//  @return string
func GetRedisShutdownName(rc *redisv1alpha1.RedisStandalone) string {
	return util.GenerateName(RedisShutdownName, rc.Name)
}
