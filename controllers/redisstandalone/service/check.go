package service

import (
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	goredis "github.com/go-redis/redis"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redisstandalone"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	"github.com/von1994/cndb-redis/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"net"
)

// RedisStandaloneCheck defines the intercace able to check the correct status of a redis cluster
type RedisStandaloneCheck interface {
	CheckRedisNumber(rc *redisv1alpha1.RedisStandalone) error
	CheckRedisConfig(rc *redisv1alpha1.RedisStandalone, addr string, auth *util.AuthConfig) error
	GetRedisesIPs(rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) ([]string, error)
}

var _ RedisStandaloneCheck = &RedisStandaloneChecker{}

// RedisStandaloneChecker is our implementation of RedisSentinelCheck intercace
type RedisStandaloneChecker struct {
	k8sService  k8s.Services
	redisClient redis.Client
	logger      logr.Logger
}

// NewRedisStandaloneChecker creates an object of the RedisSentinelChecker struct
func NewRedisStandaloneChecker(k8sService k8s.Services, redisClient redis.Client, logger logr.Logger) *RedisStandaloneChecker {
	return &RedisStandaloneChecker{
		k8sService:  k8sService,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (r *RedisStandaloneChecker) GetRedisesIPs(rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) ([]string, error) {
	redises := []string{}
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, redisstandalone.GetRedisName(rc))
	if err != nil {
		return nil, err
	}
	for _, rp := range rps.Items {
		if rp.Status.Phase == corev1.PodRunning { // Only work with running pods
			redises = append(redises, rp.Status.PodIP)
		}
	}
	return redises, nil
}

func (r *RedisStandaloneChecker) CheckRedisConfig(rc *redisv1alpha1.RedisStandalone, addr string, auth *util.AuthConfig) error {
	client := goredis.NewClient(&goredis.Options{
		Addr:     net.JoinHostPort(addr, "6379"),
		Password: auth.Password,
		DB:       0,
	})
	defer client.Close()
	configs, err := r.redisClient.GetAllRedisConfig(client)
	if err != nil {
		return err
	}

	for key, value := range rc.Spec.Config {
		var err error
		if _, ok := common.ParseConfigMap[key]; ok {
			value, err = util.ParseRedisMemConf(value)
			if err != nil {
				r.logger.Error(err, "redis config format err", "key", key, "value", value)
				continue
			}
		}
		if value != configs[key] {
			return fmt.Errorf("%s configs conflict, expect: %s, current: %s", key, value, configs[key])
		}
	}
	return nil

}

func (r *RedisStandaloneChecker) CheckRedisNumber(rc *redisv1alpha1.RedisStandalone) error {
	ss, err := r.k8sService.GetStatefulSet(rc.Namespace, redisstandalone.GetRedisName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Size != *ss.Spec.Replicas {
		return errors.New("number of redis pods differ from specification")
	}
	if rc.Spec.Size != ss.Status.ReadyReplicas {
		return errors.New("waiting all of redis pods become ready")
	}
	return nil
}
