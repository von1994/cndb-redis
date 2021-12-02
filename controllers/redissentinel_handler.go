package controllers

import (
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redissentinel/cache"
	"github.com/von1994/cndb-redis/controllers/redissentinel/service"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/metrics"
	"github.com/von1994/cndb-redis/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	checkInterval = 5 * time.Second
	timeOut       = 30 * time.Second
)

// SentinelHandler is the RedisSentinel handler. This handler will create the required
// resources that a RedisSentinel needs.
type SentinelHandler struct {
	k8sServices k8s.Services
	rcService   service.RedisSentinelClient
	rcChecker   service.RedisSentinelCheck
	rcHealer    service.RedisSentinelHeal
	metaCache   *cache.MetaMap
	eventsCli   k8s.Event
	logger      logr.Logger
}

// Do will ensure the RedisSentinel is in the expected state and update the RedisSentinel status.
func (r *SentinelHandler) Do(rc *redisv1alpha1.RedisSentinel) error {
	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("handler doing")
	// validate may change the spec.
	// TODO: 考虑使用validatingWebhook和mutatingWebhook实现
	if modified, err := rc.Validate(); err != nil {
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		return err
	} else if modified {
		if err := r.k8sServices.UpdateSentinelSpec(rc.Namespace, rc); err != nil {
			return err
		}
		return common.ErrNeedRequeueImmediately
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("validated and no updates")

	// diff new and new RedisSentinel, then update status
	meta := r.metaCache.Cache(rc)
	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(3).
		Info(fmt.Sprintf("meta status:%s, mes:%s, state:%s", meta.Status, meta.Message, meta.State))
	if err := r.updateStatus(meta); err != nil {
		r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(5).
			Info(fmt.Sprintf("update status failed due to %s", err.Error()))
		return err
	}

	// Create owner refs so the objects manager by this handler have ownership to the
	// received rc.
	oRefs := r.createOwnerReferences(rc)

	// Create the labels every object derived from this need to have.
	labels := r.getLabels(rc)

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("Ensure...")
	r.eventsCli.EnsureCluster(rc)
	if err := r.Ensure(meta.Obj, labels, oRefs); err != nil {
		r.eventsCli.FailedCluster(rc, err.Error())
		rc.Status.SetFailedCondition(err.Error())
		if updateFailedErr := r.k8sServices.UpdateSentinelStatus(rc.Namespace, rc); updateFailedErr != nil {
			return updateFailedErr
		}
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("CheckAndHeal...")
	r.eventsCli.CheckCluster(rc)
	if err := r.CheckAndHeal(meta); err != nil {
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		if err.Error() != common.NeedRequeueMsg {
			r.eventsCli.FailedCluster(rc, err.Error())
			rc.Status.SetFailedCondition(err.Error())
			if updateFailedErr := r.k8sServices.UpdateSentinelStatus(rc.Namespace, rc); updateFailedErr != nil {
				return updateFailedErr
			}
			return err
		}
		// if user delete statefulset or deployment, set status
		status := rc.Status.Conditions
		if len(status) > 0 && status[0].Type == redisv1alpha1.ClusterConditionHealthy {
			r.eventsCli.CreateCluster(rc)
			rc.Status.SetCreateCondition("redis server or sentinel server be removed by user, restart")
			if updateFailedErr := r.k8sServices.UpdateSentinelStatus(rc.Namespace, rc); updateFailedErr != nil {
				return updateFailedErr
			}
		}
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("SetReadyCondition...")
	r.eventsCli.HealthCluster(rc)
	rc.Status.SetReadyCondition("Cluster ok")
	if updateFailedErr := r.k8sServices.UpdateSentinelStatus(rc.Namespace, rc); updateFailedErr != nil {
		return updateFailedErr
	}
	metrics.ClusterMetrics.SetClusterOK(rc.Namespace, rc.Name)

	return nil
}

func (r *SentinelHandler) updateStatus(meta *cache.Meta) (err error) {
	rc := meta.Obj

	if meta.State != cache.Check {
		// Password change is not allowed
		//rc.Spec.Redis.Password = rc.Spec.Redis.Password
		switch meta.Status {
		case redisv1alpha1.ClusterConditionCreating:
			r.eventsCli.CreateCluster(rc)
			rc.Status.SetCreateCondition(meta.Message)
		case redisv1alpha1.ClusterConditionScaling:
			r.eventsCli.NewSlaveAdd(rc, meta.Message)
			rc.Status.SetScalingUpCondition(meta.Message)
		case redisv1alpha1.ClusterConditionScalingDown:
			r.eventsCli.SlaveRemove(rc, meta.Message)
			rc.Status.SetScalingDownCondition(meta.Message)
		case redisv1alpha1.ClusterConditionUpgrading:
			r.eventsCli.UpdateSentinelStatus(rc, meta.Message)
			rc.Status.SetUpgradingCondition(meta.Message)
		default:
			r.eventsCli.UpdateSentinelStatus(rc, meta.Message)
			rc.Status.SetUpdatingCondition(meta.Message)
		}
		return r.k8sServices.UpdateSentinelStatus(rc.Namespace, rc)
	}
	return nil
}

// getLabels merges all the labels (dynamic and operator static ones).
func (r *SentinelHandler) getLabels(rc *redisv1alpha1.RedisSentinel) map[string]string {
	dynLabels := map[string]string{
		redisv1alpha1.LabelNameKey: fmt.Sprintf("%s%c%s", rc.Namespace, '_', rc.Name),
	}
	return util.MergeLabels(common.DefaultLabels, dynLabels, rc.Labels)
}

func (r *SentinelHandler) createOwnerReferences(rc *redisv1alpha1.RedisSentinel) []metav1.OwnerReference {
	rcvk := redisv1alpha1.VersionKind(redisv1alpha1.RedisSentinelKind)
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(rc, rcvk),
	}
}

// CheckAndHeal Check the health of the cluster and heal,
// Waiting Number of ready redis is equal as the set on the RedisSentinel spec
// Waiting Number of ready sentinel is equal as the set on the RedisSentinel spec
// Check only one master
// Number of redis master is 1
// All redis slaves have the same master
// Set Custom Redis config
// All sentinels points to the same redis master
// Sentinel has not death nodes
// Sentinel knows the correct slave number
func (r *SentinelHandler) CheckAndHeal(meta *cache.Meta) error {
	if err := r.rcChecker.CheckRedisNumber(meta.Obj); err != nil {
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).V(2).Info("number of redis mismatch, this could be for a change on the statefulset")
		r.eventsCli.UpdateSentinelStatus(meta.Obj, "wait for all redis server start")
		return common.ErrNeedRequeue
	}
	if err := r.rcChecker.CheckSentinelNumber(meta.Obj); err != nil {
		r.eventsCli.FailedCluster(meta.Obj, err.Error())
		return nil
	}

	nMasters, err := r.rcChecker.GetNumberMasters(meta.Obj, meta.Auth)
	if err != nil {
		return err
	}
	switch nMasters {
	case 0:
		r.eventsCli.UpdateSentinelStatus(meta.Obj, "set master")
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).V(2).Info("no master find, fixing...")
		redisesIP, err := r.rcChecker.GetRedisesIPs(meta.Obj, meta.Auth)
		if err != nil {
			return err
		}
		if len(redisesIP) == 1 {
			if err := r.rcHealer.MakeMaster(redisesIP[0], meta.Auth); err != nil {
				return err
			}
			break
		}
		minTime, err := r.rcChecker.GetMinimumRedisPodTime(meta.Obj)
		if err != nil {
			return err
		}
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(fmt.Sprintf("time %.f more than expected. Not even one master, fixing...", minTime.Round(time.Second).Seconds()))
		if err := r.rcHealer.SetOldestAsMaster(meta.Obj, meta.Auth); err != nil {
			return err
		}
	case 1:
		break
	default:
		return errors.New("more than one master, fix manually")
	}

	master, err := r.rcChecker.GetMasterIP(meta.Obj, meta.Auth)
	if err != nil {
		return err
	}
	if err := r.rcChecker.CheckAllSlavesFromMaster(master, meta.Obj, meta.Auth); err != nil {
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
		if err := r.rcHealer.SetMasterOnAll(master, meta.Obj, meta.Auth); err != nil {
			return err
		}
	}

	if err = r.setRedisConfig(meta); err != nil {
		return err
	}

	sentinels, err := r.rcChecker.GetSentinelsIPs(meta.Obj)
	if err != nil {
		return err
	}
	for _, sip := range sentinels {
		if err := r.rcChecker.CheckSentinelMonitor(sip, master, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
			if err := r.rcHealer.NewSentinelMonitor(sip, master, meta.Obj, meta.Auth); err != nil {
				return err
			}
		}
	}
	for _, sip := range sentinels {
		if err := r.rcChecker.CheckSentinelSlavesNumberInMemory(sip, meta.Obj, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).
				Info("restoring sentinel ...", "sentinel", sip, "reason", err.Error())
			if err := r.rcHealer.RestoreSentinel(sip, meta.Auth); err != nil {
				return err
			}
			if err := r.waitRestoreSentinelSlavesOK(sip, meta.Obj, meta.Auth); err != nil {
				r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
				return err
			}
		}
	}
	for _, sip := range sentinels {
		if err := r.rcChecker.CheckSentinelNumberInMemory(sip, meta.Obj, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).
				Info("restoring sentinel ...", "sentinel", sip, "reason", err.Error())
			if err := r.rcHealer.RestoreSentinel(sip, meta.Auth); err != nil {
				return err
			}
		}
	}

	if err = r.setSentinelConfig(meta, sentinels); err != nil {
		return err
	}

	// Ensure sentinel redises are all ready
	if err = r.rcChecker.EnsureSentinelReady(meta.Obj); err != nil {
		return err
	}

	return nil
}

func (r *SentinelHandler) setRedisConfig(meta *cache.Meta) error {
	redises, err := r.rcChecker.GetRedisesIPs(meta.Obj, meta.Auth)
	if err != nil {
		return err
	}
	for _, rip := range redises {
		if err := r.rcChecker.CheckRedisConfig(meta.Obj, rip, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
			r.eventsCli.UpdateSentinelStatus(meta.Obj, "set custom config for redis server")
			if err := r.rcHealer.SetRedisCustomConfig(rip, meta.Obj, meta.Auth); err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO do as set redis config
func (r *SentinelHandler) setSentinelConfig(meta *cache.Meta, sentinels []string) error {
	if meta.State == cache.Check {
		return nil
	}

	for _, sip := range sentinels {
		if err := r.rcHealer.SetSentinelCustomConfig(sip, meta.Obj, meta.Auth); err != nil {
			return err
		}
	}
	return nil
}

func (r *SentinelHandler) waitRestoreSentinelSlavesOK(sentinel string, rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	timer := time.NewTimer(timeOut)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("wait for resetore sentinel slave timeout")
		default:
			if err := r.rcChecker.CheckSentinelSlavesNumberInMemory(sentinel, rc, auth); err != nil {
				r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info(err.Error())
				time.Sleep(checkInterval)
			} else {
				return nil
			}
		}
	}
}

// Ensure the RedisSentinel's components are correct.
func (r *SentinelHandler) Ensure(rc *redisv1alpha1.RedisSentinel, labels map[string]string, or []metav1.OwnerReference) error {
	if err := r.rcService.EnsureRedisService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelHeadlessService(rc, labels, or); err != nil {
		return err
	}
	if rc.Spec.Exporter.Enabled {
		if err := r.rcService.EnsureRedisMonitorService(rc, labels, or); err != nil {
			return err
		}
	}
	if err := r.rcService.EnsureSentinelConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelProbeConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisShutdownConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisStatefulset(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelStatefulset(rc, labels, or); err != nil {
		return err
	}

	return nil
}
