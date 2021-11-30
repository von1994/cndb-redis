package controllers

import (
	"fmt"
	"github.com/go-logr/logr"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/cache"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/service"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisStandaloneHandler 包含处理调谐所需要的各种客户端
type RedisStandaloneHandler struct {
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
func (r *RedisStandaloneHandler) Do(rc *redisv1alpha1.RedisStandalone) error {
	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("handler doing")
	if modified, err := rc.Validate(); err != nil {
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
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("CheckAndHeal...")
	r.eventsCli.CheckCluster(rc)
	if err := r.CheckAndHeal(meta); err != nil {
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("SetReadyCondition...")
	r.eventsCli.HealthCluster(rc)
	rc.Status.SetReadyCondition("Cluster ok")
	if updateFailedErr := r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc); updateFailedErr != nil {
		return updateFailedErr
	}

	return nil
}

func (r *RedisStandaloneHandler) updateStatus(meta *cache.Meta) error {
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
			r.eventsCli.UpdateClusterStatus(rc, meta.Message)
			rc.Status.SetUpgradingCondition(meta.Message)
		default:
			r.eventsCli.UpdateClusterStatus(rc, meta.Message)
			rc.Status.SetUpdatingCondition(meta.Message)
		}
		return r.k8sServices.UpdateStandaloneStatus(rc.Namespace, rc)
	}
	return nil
}

func (r *RedisStandaloneHandler) createOwnerReferences(rc *redisv1alpha1.RedisStandalone) []metav1.OwnerReference {
	rcvk := redisv1alpha1.VersionKind(redisv1alpha1.RedisStandaloneKind)
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(rc, rcvk),
	}
}

// getLabels merges all the labels (dynamic and operator static ones).
func (r *RedisStandaloneHandler) getLabels(rc *redisv1alpha1.RedisStandalone) map[string]string {
	dynLabels := map[string]string{
		redisv1alpha1.LabelNameKey: fmt.Sprintf("%s%c%s", rc.Namespace, '_', rc.Name),
	}
	return util.MergeLabels(defaultLabels, dynLabels, rc.Labels)
}
