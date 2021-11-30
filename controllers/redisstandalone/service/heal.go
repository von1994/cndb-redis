package service

import (
	"github.com/go-logr/logr"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
)

type RedisStandaloneHeal interface {
}

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
