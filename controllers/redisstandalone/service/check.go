package service

import (
	"github.com/go-logr/logr"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
)

// RedisStandaloneCheck defines the intercace able to check the correct status of a redis cluster
type RedisStandaloneCheck interface {
}

// RedisClusterChecker is our implementation of RedisClusterCheck intercace
type RedisStandaloneChecker struct {
	k8sService  k8s.Services
	redisClient redis.Client
	logger      logr.Logger
}

// NewRedisStandaloneChecker creates an object of the RedisClusterChecker struct
func NewRedisStandaloneChecker(k8sService k8s.Services, redisClient redis.Client, logger logr.Logger) *RedisStandaloneChecker {
	return &RedisStandaloneChecker{
		k8sService:  k8sService,
		redisClient: redisClient,
		logger:      logger,
	}
}
