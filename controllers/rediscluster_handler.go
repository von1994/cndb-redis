package controllers

import (
	"fmt"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/controllers/rediscluster/clustercache"
	"github.com/von1994/cndb-redis/controllers/rediscluster/service"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/metrics"
	"github.com/von1994/cndb-redis/pkg/util"
)

var (
	defaultLabels = map[string]string{
		redisv1alpha1.LabelManagedByKey: redisv1alpha1.OperatorName,
	}
)

// RedisClusterHandler is the RedisCluster handler. This handler will create the required
// resources that a RedisCluster needs.
type RedisClusterHandler struct {
	k8sServices k8s.Services
	rcService   service.RedisClusterClient
	rcChecker   service.RedisClusterCheck
	rcHealer    service.RedisClusterHeal
	metaCache   *clustercache.MetaMap
	eventsCli   k8s.Event
	logger      logr.Logger
}

// Do will ensure the RedisCluster is in the expected state and update the RedisCluster status.
func (r *RedisClusterHandler) Do(rc *redisv1alpha1.RedisCluster) error {
	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("handler doing")
	// validate may change the spec.
	if modified, err := rc.Validate(); err != nil {
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		return err
	} else if modified {
		return immediatelyNeedRequeueErr
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info("validated and no updates")

	// diff new and new RedisCluster, then update status
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
		if updateFailedErr := r.k8sServices.UpdateClusterStatus(rc.Namespace, rc); updateFailedErr != nil {
			return updateFailedErr
		}
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("CheckAndHeal...")
	r.eventsCli.CheckCluster(rc)
	if err := r.CheckAndHeal(meta); err != nil {
		metrics.ClusterMetrics.SetClusterError(rc.Namespace, rc.Name)
		if err.Error() != needRequeueMsg {
			r.eventsCli.FailedCluster(rc, err.Error())
			rc.Status.SetFailedCondition(err.Error())
			if updateFailedErr := r.k8sServices.UpdateClusterStatus(rc.Namespace, rc); updateFailedErr != nil {
				return updateFailedErr
			}
			return err
		}
		// if user delete statefulset or deployment, set status
		status := rc.Status.Conditions
		if len(status) > 0 && status[0].Type == redisv1alpha1.ClusterConditionHealthy {
			r.eventsCli.CreateCluster(rc)
			rc.Status.SetCreateCondition("redis server or sentinel server be removed by user, restart")
			if updateFailedErr := r.k8sServices.UpdateClusterStatus(rc.Namespace, rc); updateFailedErr != nil {
				return updateFailedErr
			}
		}
		return err
	}

	r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).V(2).Info("SetReadyCondition...")
	r.eventsCli.HealthCluster(rc)
	rc.Status.SetReadyCondition("Cluster ok")
	if updateFailedErr := r.k8sServices.UpdateClusterStatus(rc.Namespace, rc); updateFailedErr != nil {
		return updateFailedErr
	}
	metrics.ClusterMetrics.SetClusterOK(rc.Namespace, rc.Name)

	return nil
}

func (r *RedisClusterHandler) updateStatus(meta *clustercache.Meta) (err error) {
	rc := meta.Obj

	if meta.State != clustercache.Check {
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
		return r.k8sServices.UpdateClusterStatus(rc.Namespace, rc)
	}
	return nil
}

// getLabels merges all the labels (dynamic and operator static ones).
func (r *RedisClusterHandler) getLabels(rc *redisv1alpha1.RedisCluster) map[string]string {
	dynLabels := map[string]string{
		redisv1alpha1.LabelNameKey: fmt.Sprintf("%s%c%s", rc.Namespace, '_', rc.Name),
	}
	return util.MergeLabels(defaultLabels, dynLabels, rc.Labels)
}

func (r *RedisClusterHandler) createOwnerReferences(rc *redisv1alpha1.RedisCluster) []metav1.OwnerReference {
	rcvk := redisv1alpha1.VersionKind(redisv1alpha1.Kind)
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(rc, rcvk),
	}
}
