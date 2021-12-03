package service

import (
	"fmt"
	"github.com/go-logr/logr"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	"github.com/von1994/cndb-redis/pkg/util"
)

type RedisStandaloneHeal interface {
	SetRedisCustomConfig(ip string, rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) error
}

var _ RedisStandaloneHeal = &RedisStandaloneHealer{}

// RedisStandaloneHealer is our implementation of RedisSentinelCheck intercace
type RedisStandaloneHealer struct {
	k8sService  k8s.Services
	redisClient redis.Client
	logger      logr.Logger
}

// NewRedisStandaloneHealer creates an object of the RedisSentinelChecker struct
func NewRedisStandaloneHealer(k8sService k8s.Services, redisClient redis.Client, logger logr.Logger) *RedisStandaloneHealer {
	return &RedisStandaloneHealer{
		k8sService:  k8sService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// SetRedisCustomConfig will call redis to set the configuration given in config
func (r *RedisStandaloneHealer) SetRedisCustomConfig(ip string, rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) error {
	if len(rc.Spec.Config) == 0 && len(auth.Password) == 0 {
		return nil
	}

	//if len(auth.Password) != 0 {
	//	rc.Spec.Config["requirepass"] = auth.Password
	//	rc.Spec.Config["masterauth"] = auth.Password
	//}

	r.logger.V(2).Info(fmt.Sprintf("setting the custom config on redis %s: %v", ip, rc.Spec.Config))

	return r.redisClient.SetCustomRedisConfig(ip, rc.Spec.Config, auth)
}
