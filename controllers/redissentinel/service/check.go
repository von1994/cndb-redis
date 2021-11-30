package service

import (
	"errors"
	"fmt"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redissentinel"
	"net"
	"time"

	"github.com/go-logr/logr"
	goredis "github.com/go-redis/redis"
	corev1 "k8s.io/api/core/v1"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	"github.com/von1994/cndb-redis/pkg/util"
)

// RedisSentinelCheck defines the intercace able to check the correct status of a redis cluster
type RedisSentinelCheck interface {
	CheckRedisNumber(rc *redisv1alpha1.RedisSentinel) error
	CheckSentinelNumber(rc *redisv1alpha1.RedisSentinel) error
	EnsureSentinelReady(rc *redisv1alpha1.RedisSentinel) error
	CheckSentinelReadyReplicas(rc *redisv1alpha1.RedisSentinel) error
	CheckAllSlavesFromMaster(master string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	CheckSentinelNumberInMemory(sentinel string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	CheckSentinelSlavesNumberInMemory(sentinel string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error
	CheckSentinelMonitor(sentinel string, monitor string, auth *util.AuthConfig) error
	GetMasterIP(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) (string, error)
	GetNumberMasters(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) (int, error)
	GetRedisesIPs(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) ([]string, error)
	GetSentinelsIPs(rc *redisv1alpha1.RedisSentinel) ([]string, error)
	GetMinimumRedisPodTime(rc *redisv1alpha1.RedisSentinel) (time.Duration, error)
	CheckRedisConfig(rc *redisv1alpha1.RedisSentinel, addr string, auth *util.AuthConfig) error
}

var parseConfigMap = map[string]int8{
	"maxmemory":                  0,
	"proto-max-bulk-len":         0,
	"client-query-buffer-limit":  0,
	"repl-backlog-size":          0,
	"auto-aof-rewrite-min-size":  0,
	"active-defrag-ignore-bytes": 0,
	"hash-max-ziplist-entries":   0,
	"hash-max-ziplist-value":     0,
	"stream-node-max-bytes":      0,
	"set-max-intset-entries":     0,
	"zset-max-ziplist-entries":   0,
	"zset-max-ziplist-value":     0,
	"hll-sparse-max-bytes":       0,
	// TODO parse client-output-buffer-limit
	//"client-output-buffer-limit": 0,
}

// RedisSentinelChecker is our implementation of RedisSentinelCheck intercace
type RedisSentinelChecker struct {
	k8sService  k8s.Services
	redisClient redis.Client
	logger      logr.Logger
}

// NewRedisSentinelChecker creates an object of the RedisSentinelChecker struct
func NewRedisSentinelChecker(k8sService k8s.Services, redisClient redis.Client, logger logr.Logger) *RedisSentinelChecker {
	return &RedisSentinelChecker{
		k8sService:  k8sService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// CheckRedisConfig check current redis config is same as custom config
func (r *RedisSentinelChecker) CheckRedisConfig(rc *redisv1alpha1.RedisSentinel, addr string, auth *util.AuthConfig) error {
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
		if _, ok := parseConfigMap[key]; ok {
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

// CheckRedisNumber controls that the number of deployed redis is the same than the requested on the spec
func (r *RedisSentinelChecker) CheckRedisNumber(rc *redisv1alpha1.RedisSentinel) error {
	ss, err := r.k8sService.GetStatefulSet(rc.Namespace, redissentinel.GetRedisName(rc))
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

// CheckSentinelNumber controls that the number of deployed sentinel is the same than the requested on the spec
func (r *RedisSentinelChecker) CheckSentinelNumber(rc *redisv1alpha1.RedisSentinel) error {
	d, err := r.k8sService.GetStatefulSet(rc.Namespace, redissentinel.GetSentinelName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Sentinel.Replicas != *d.Spec.Replicas {
		return errors.New("number of sentinel pods differ from specification")
	}
	return nil
}

// EnsureSentinelReady 通过检查目标副本数和ready副本数是否相等来判断sentinel是否ready
//  @receiver r
//  @param rc
//  @return error
func (r *RedisSentinelChecker) EnsureSentinelReady(rc *redisv1alpha1.RedisSentinel) error {
	d, err := r.k8sService.GetStatefulSet(rc.Namespace, redissentinel.GetSentinelName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Sentinel.Replicas != d.Status.ReadyReplicas {
		r.logger.Info(fmt.Sprintf("waiting redis cluster %s all sentinel pods become ready", rc.Name))
		return common.ErrNeedRequeue
	}
	return nil
}

// CheckSentinelReadyReplicas controls that the number of deployed sentinel ready pod is the same than the requested on the spec
func (r *RedisSentinelChecker) CheckSentinelReadyReplicas(rc *redisv1alpha1.RedisSentinel) error {
	d, err := r.k8sService.GetStatefulSet(rc.Namespace, redissentinel.GetSentinelName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Sentinel.Replicas != d.Status.ReadyReplicas {
		return errors.New("waiting all of sentinel pods become ready")
	}
	return nil
}

// CheckAllSlavesFromMaster controls that all slaves have the same master (the real one)
func (r *RedisSentinelChecker) CheckAllSlavesFromMaster(master string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	rips, err := r.GetRedisesIPs(rc, auth)
	if err != nil {
		return err
	}
	for _, rip := range rips {
		slave, err := r.redisClient.GetSlaveMasterIP(rip, auth)
		if err != nil {
			return err
		}
		if slave != "" && slave != master {
			return fmt.Errorf("slave %s don't have the master %s, has %s", rip, master, slave)
		}
	}
	return nil
}

// CheckSentinelNumberInMemory controls that sentinels have only the living sentinels on its memory.
func (r *RedisSentinelChecker) CheckSentinelNumberInMemory(sentinel string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	nSentinels, err := r.redisClient.GetNumberSentinelsInMemory(sentinel, auth)
	if err != nil {
		return err
	} else if nSentinels != rc.Spec.Sentinel.Replicas {
		return errors.New("sentinels in memory mismatch")
	}
	return nil
}

// CheckSentinelSlavesNumberInMemory controls that sentinels have only the spected slaves number.
func (r *RedisSentinelChecker) CheckSentinelSlavesNumberInMemory(sentinel string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	nSlaves, err := r.redisClient.GetNumberSentinelSlavesInMemory(sentinel, auth)
	if err != nil {
		return err
	} else if nSlaves != rc.Spec.Size-1 {
		return errors.New("sentinel's slaves in memory mismatch")
	}
	return nil
}

// CheckSentinelMonitor controls if the sentinels are monitoring the expected master
func (r *RedisSentinelChecker) CheckSentinelMonitor(sentinel string, monitor string, auth *util.AuthConfig) error {
	actualMonitorIP, err := r.redisClient.GetSentinelMonitor(sentinel, auth)
	if err != nil {
		return err
	}
	if actualMonitorIP != monitor {
		return errors.New("the monitor on the sentinel config does not match with the expected one")
	}
	return nil
}

// GetMasterIP connects to all redis and returns the master of the redis cluster
func (r *RedisSentinelChecker) GetMasterIP(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) (string, error) {
	rips, err := r.GetRedisesIPs(rc, auth)
	if err != nil {
		return "", err
	}
	masters := []string{}
	for _, rip := range rips {
		master, err := r.redisClient.IsMaster(rip, auth)
		if err != nil {
			return "", err
		}
		if master {
			masters = append(masters, rip)
		}
	}

	if len(masters) != 1 {
		return "", errors.New("number of redis nodes known as master is different than 1")
	}
	return masters[0], nil
}

// GetNumberMasters returns the number of redis nodes that are working as a master
func (r *RedisSentinelChecker) GetNumberMasters(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) (int, error) {
	nMasters := 0
	rips, err := r.GetRedisesIPs(rc, auth)
	if err != nil {
		return nMasters, err
	}
	for _, rip := range rips {
		master, err := r.redisClient.IsMaster(rip, auth)
		if err != nil {
			return nMasters, err
		}
		if master {
			nMasters++
		}
	}
	return nMasters, nil
}

// GetRedisesIPs returns the IPs of the Redis nodes
func (r *RedisSentinelChecker) GetRedisesIPs(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) ([]string, error) {
	redises := []string{}
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, redissentinel.GetRedisName(rc))
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

// GetSentinelsIPs returns the IPs of the Sentinel nodes
func (r *RedisSentinelChecker) GetSentinelsIPs(rc *redisv1alpha1.RedisSentinel) ([]string, error) {
	sentinels := []string{}
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, redissentinel.GetSentinelName(rc))
	if err != nil {
		return nil, err
	}
	for _, sp := range rps.Items {
		if sp.Status.Phase == corev1.PodRunning { // Only work with running pods
			sentinels = append(sentinels, sp.Status.PodIP)
		}
	}
	return sentinels, nil
}

// GetMinimumRedisPodTime returns the minimum time a pod is alive
func (r *RedisSentinelChecker) GetMinimumRedisPodTime(rc *redisv1alpha1.RedisSentinel) (time.Duration, error) {
	minTime := 100000 * time.Hour // More than ten years
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, redissentinel.GetRedisName(rc))
	if err != nil {
		return minTime, err
	}
	for _, redisNode := range rps.Items {
		if redisNode.Status.StartTime == nil {
			continue
		}
		start := redisNode.Status.StartTime.Round(time.Second)
		alive := time.Since(start)
		r.logger.V(2).Info(fmt.Sprintf("pod %s has been alive for %.f seconds", redisNode.Status.PodIP, alive.Seconds()))
		if alive < minTime {
			minTime = alive
		}
	}
	return minTime, nil
}
