package service

import (
	"errors"
	"fmt"
	"github.com/von1994/cndb-redis/controllers/redissentinel"
	"sort"
	"strconv"

	"github.com/go-logr/logr"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	"github.com/von1994/cndb-redis/pkg/util"
)

// RedisSentinelHeal defines the interface able to fix the problems on the redis clusters
type RedisSentinelHeal interface {
	MakeMaster(ip string, auth *util.AuthConfig) error
	SetOldestAsMaster(RedisSentinel *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	SetMasterOnAll(masterIP string, RedisSentinel *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	NewSentinelMonitor(ip string, monitor string, RedisSentinel *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	RestoreSentinel(ip string, auth *util.AuthConfig) error
	SetSentinelCustomConfig(ip string, RedisSentinel *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	SetRedisCustomConfig(ip string, RedisSentinel *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
}

// RedisSentinelHealer is our implementation of RedisSentinelCheck intercace
type RedisSentinelHealer struct {
	k8sService  k8s.Services
	redisClient redis.Client
	logger      logr.Logger
}

// NewRedisSentinelHealer creates an object of the RedisSentinelChecker struct
func NewRedisSentinelHealer(k8sService k8s.Services, redisClient redis.Client, logger logr.Logger) *RedisSentinelHealer {
	return &RedisSentinelHealer{
		k8sService:  k8sService,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (r *RedisSentinelHealer) MakeMaster(ip string, auth *util.AuthConfig) error {
	return r.redisClient.MakeMaster(ip, auth)
}

// SetOldestAsMaster puts all redis to the same master, chosen by order of appearance
func (r *RedisSentinelHealer) SetOldestAsMaster(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	ssp, err := r.k8sService.GetStatefulSetPods(rc.Namespace, redissentinel.GetRedisName(rc))
	if err != nil {
		return err
	}
	if len(ssp.Items) < 1 {
		return errors.New("number of redis pods are 0")
	}

	// Order the pods so we start by the oldest one
	sort.Slice(ssp.Items, func(i, j int) bool {
		return ssp.Items[i].CreationTimestamp.Before(&ssp.Items[j].CreationTimestamp)
	})

	newMasterIP := ""
	for _, pod := range ssp.Items {
		if newMasterIP == "" {
			newMasterIP = pod.Status.PodIP
			r.logger.V(2).Info(fmt.Sprintf("new master is %s with ip %s", pod.Name, newMasterIP))
			if err := r.redisClient.MakeMaster(newMasterIP, auth); err != nil {
				return err
			}
		} else {
			r.logger.V(2).Info(fmt.Sprintf("making pod %s slave of %s", pod.Name, newMasterIP))
			if err := r.redisClient.MakeSlaveOf(pod.Status.PodIP, newMasterIP, auth); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetMasterOnAll puts all redis nodes as a slave of a given master
func (r *RedisSentinelHealer) SetMasterOnAll(masterIP string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	ssp, err := r.k8sService.GetStatefulSetPods(rc.Namespace, redissentinel.GetRedisName(rc))
	if err != nil {
		return err
	}
	for _, pod := range ssp.Items {
		if pod.Status.PodIP == masterIP {
			r.logger.V(2).Info(fmt.Sprintf("ensure pod %s is master", pod.Name))
			if err := r.redisClient.MakeMaster(masterIP, auth); err != nil {
				return err
			}
		} else {
			r.logger.V(2).Info(fmt.Sprintf("making pod %s slave of %s", pod.Name, masterIP))
			if err := r.redisClient.MakeSlaveOf(pod.Status.PodIP, masterIP, auth); err != nil {
				return err
			}
		}
	}
	return nil
}

// NewSentinelMonitor changes the master that Sentinel has to monitor
func (r *RedisSentinelHealer) NewSentinelMonitor(ip string, monitor string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	r.logger.V(2).Info("sentinel is not monitoring the correct master, changing...")
	quorum := strconv.Itoa(int(getQuorum(rc)))
	return r.redisClient.MonitorRedis(ip, monitor, quorum, auth)
}

// RestoreSentinel clear the number of sentinels on memory
func (r *RedisSentinelHealer) RestoreSentinel(ip string, auth *util.AuthConfig) error {
	r.logger.V(2).Info(fmt.Sprintf("restoring sentinel %s...", ip))
	return r.redisClient.ResetSentinel(ip, auth)
}

// SetSentinelCustomConfig will call sentinel to set the configuration given in config
func (r *RedisSentinelHealer) SetSentinelCustomConfig(ip string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	if len(rc.Spec.Sentinel.CustomConfig) == 0 {
		return nil
	}
	r.logger.V(2).Info(fmt.Sprintf("setting the custom config on sentinel %s: %v", ip, rc.Spec.Sentinel.CustomConfig))
	return r.redisClient.SetCustomSentinelConfig(ip, rc.Spec.Sentinel.CustomConfig, auth)
}

// SetRedisCustomConfig will call redis to set the configuration given in config
func (r *RedisSentinelHealer) SetRedisCustomConfig(ip string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
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
