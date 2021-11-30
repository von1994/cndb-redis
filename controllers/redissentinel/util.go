package redissentinel

import (
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/util"
)

// const
const (
	BaseName               = "redis"
	SentinelName           = "-sentinel"
	SentinelRoleName       = "sentinel"
	SentinelConfigFileName = "sentinel.conf"
	RedisConfigFileName    = "redis.conf"
	RedisName              = "-cluster"
	RedisShutdownName      = "r-s"
	RedisRoleName          = "redis"
	AppLabel               = "redis-cluster"
	HostnameTopologyKey    = "kubernetes.io/hostname"
)

// GetRedisShutdownConfigMapName returns the name for redis configmap
func GetRedisShutdownConfigMapName(rc *redisv1alpha1.RedisSentinel) string {
	if rc.Spec.ShutdownConfigMap != "" {
		return rc.Spec.ShutdownConfigMap
	}
	return GetRedisShutdownName(rc)
}

// GetRedisName returns the name for redis resources
func GetRedisName(rc *redisv1alpha1.RedisSentinel) string {
	return util.GenerateName(RedisName, rc.Name)
}

// GetRedisShutdownName returns the name for redis resources
func GetRedisShutdownName(rc *redisv1alpha1.RedisSentinel) string {
	return util.GenerateName(RedisShutdownName, rc.Name)
}

// GetSentinelName returns the name for sentinel resources
func GetSentinelName(rc *redisv1alpha1.RedisSentinel) string {
	return util.GenerateName(SentinelName, rc.Name)
}

// GetSentinelReadinessCm  生成sentinel readiness configmap名称
//  @param rc
//  @return string
func GetSentinelReadinessCm(rc *redisv1alpha1.RedisSentinel) string {
	return util.GenerateName("-sentinel-readiness", rc.Name)
}

// GetSentinelHeadlessSvc  生成sentinel headless svc名称
//  @param rc
//  @return string
func GetSentinelHeadlessSvc(rc *redisv1alpha1.RedisSentinel) string {
	return util.GenerateName("-sentinel-headless", rc.Name)
}
