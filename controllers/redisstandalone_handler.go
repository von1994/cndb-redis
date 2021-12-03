package controllers

import (
	"fmt"
	"github.com/go-logr/logr"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/cache"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/service"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/metrics"
	"github.com/von1994/cndb-redis/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StandaloneHandler 包含处理调谐所需要的各种客户端
type StandaloneHandler struct {
	k8sServices k8s.Services
	rcService   service.RedisStandaloneClient
	rcChecker   service.RedisStandaloneCheck
	rcHealer    service.RedisStandaloneHeal
	metaCache   *cache.MetaMap
	eventsCli   k8s.Event
	logger      logr.Logger
}

// Do 处理对资源的调谐逻辑
//  @receiver r
//  @param rc
//  @return error
func (r *StandaloneHandler) Do(rc *redisv1alpha1.RedisStandalone) error {
	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("handler doing")
	if modified, err := rc.Validate(); err != nil {
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		return err
	} else if modified {
		if err := r.k8sServices.UpdateStandaloneSpec(rc.Namespace, rc); err != nil {
			return err
		}
		return common.ErrNeedRequeueImmediately
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("validated and no updates")

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
		if updateFailedErr := r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc); updateFailedErr != nil {
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
			if updateFailedErr := r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc); updateFailedErr != nil {
				return updateFailedErr
			}
			return err
		}
		status := rc.Status.Conditions
		if len(status) > 0 && status[0].Type == redisv1alpha1.ClusterConditionHealthy {
			r.eventsCli.CreateCluster(rc)
			rc.Status.SetCreateCondition("redis server or sentinel server be removed by user, restart")
			if updateFailedErr := r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc); updateFailedErr != nil {
				return updateFailedErr
			}
		}
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("SetReadyCondition...")
	r.eventsCli.HealthCluster(rc)
	rc.Status.SetReadyCondition("Cluster ok")
	if updateFailedErr := r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc); updateFailedErr != nil {
		return updateFailedErr
	}
	metrics.ClusterMetrics.SetClusterOK(rc.Namespace, rc.Name)
	return nil
}

func (r *StandaloneHandler) updateStatus(meta *cache.Meta) error {
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
		return r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc)
	}
	return nil
}

func (r *StandaloneHandler) createOwnerReferences(rc *redisv1alpha1.RedisStandalone) []metav1.OwnerReference {
	rcvk := redisv1alpha1.VersionKind(redisv1alpha1.RedisStandaloneKind)
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(rc, rcvk),
	}
}

// getLabels merges all the labels (dynamic and operator static ones).
func (r *StandaloneHandler) getLabels(rc *redisv1alpha1.RedisStandalone) map[string]string {
	dynLabels := map[string]string{
		redisv1alpha1.LabelNameKey: fmt.Sprintf("%s%c%s", rc.Namespace, '_', rc.Name),
	}
	return util.MergeLabels(common.DefaultLabels, dynLabels, rc.Labels)
}

func (r *StandaloneHandler) setRedisConfig(meta *cache.Meta) error {
	redises, err := r.rcChecker.GetRedisesIPs(meta.Obj)
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

func (r *StandaloneHandler) CheckAndHeal(meta *cache.Meta) error {
	if err := r.rcChecker.CheckRedisNumber(meta.Obj); err != nil {
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).V(2).Info("number of redis mismatch, this could be for a change on the statefulset")
		r.eventsCli.UpdateSentinelStatus(meta.Obj, "wait for all redis server start")
		return common.ErrNeedRequeue
	}

	if err := r.setRedisConfig(meta); err != nil {
		return err
	}
	return nil
}

// Ensure the RedisSentinel's components are correct.
func (r *StandaloneHandler) Ensure(rc *redisv1alpha1.RedisStandalone, labels map[string]string, or []metav1.OwnerReference) error {
	if err := r.rcService.EnsureRedisService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisHeadlessService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisShutdownConfigMap(rc, labels, or); err != nil {
		return err
	}
	if rc.Spec.Exporter.Enabled {
		if err := r.rcService.EnsureRedisMonitorService(rc, labels, or); err != nil {
			return err
		}
	}
	if err := r.rcService.EnsureRedisStatefulset(rc, labels, or); err != nil {
		return err
	}
	return nil
}
