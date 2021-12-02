package util

import (
	"fmt"
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

// GenerateName 生成标准名称
//  @param typeName
//  @param metaName
//  @return string
func GenerateName(typeName, metaName string) string {
	return fmt.Sprintf("%s%s-%s", BaseName, typeName, metaName)
}
