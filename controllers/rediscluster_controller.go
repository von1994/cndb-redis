/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/von1994/cndb-redis/controllers/rediscluster/clustercache"
	"github.com/von1994/cndb-redis/controllers/rediscluster/service"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	"github.com/von1994/cndb-redis/pkg/metrics"
	"github.com/von1994/cndb-redis/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	controllerFlagSet *pflag.FlagSet
	// maxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 4.
	maxConcurrentReconciles int
	// reconcileTime is the delay between reconciliations. Defaults to 60s.
	reconcileTime int
)

func init() {
	controllerFlagSet = pflag.NewFlagSet("controller", pflag.ExitOnError)
	controllerFlagSet.IntVar(&maxConcurrentReconciles, "ctr-maxconcurrent", 4, "the maximum number of concurrent Reconciles which can be run. Defaults to 4.")
	controllerFlagSet.IntVar(&reconcileTime, "ctr-reconciletime", 60, "")
	namespace := util.ReplaceSymbol(util.GetEnvOrDefault("MY_POD_NAMESPACE", "default"))
	metrics.InitPrometheusMetrics(namespace, prometheus.NewRegistry())
}

// RedisClusterReconciler reconciles a RedisCluster object
type RedisClusterReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	handler *RedisClusterHandler
}

var _ reconcile.Reconciler = &RedisClusterReconciler{}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager) *RedisClusterReconciler {
	// Create kubernetes service.
	k8sService := k8s.New(mgr.GetClient(), log.Log)

	// Create the redis clients
	redisClient := redis.New()

	// Create internal services.
	rcService := service.NewRedisClusterKubeClient(k8sService, log.Log)
	rcChecker := service.NewRedisClusterChecker(k8sService, redisClient, log.Log)
	rcHealer := service.NewRedisClusterHealer(k8sService, redisClient, log.Log)

	handler := &RedisClusterHandler{
		k8sServices: k8sService,
		rcService:   rcService,
		rcChecker:   rcChecker,
		rcHealer:    rcHealer,
		metaCache:   new(clustercache.MetaMap),
		eventsCli:   k8s.NewEvent(mgr.GetEventRecorderFor("redis-operator"), log.Log),
		logger:      log.Log,
	}

	return &RedisClusterReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), handler: handler}
}

//+kubebuilder:rbac:groups=redis.lovelycat.io,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=redis.lovelycat.io,resources=redisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=redis.lovelycat.io,resources=redisclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *RedisClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	reqLogger := log.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling RedisCluster")

	// Fetch the RedisCluster instance
	instance := &redisv1alpha1.RedisCluster{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("RedisCluster delete")
			instance.Namespace = req.NamespacedName.Namespace
			instance.Name = req.NamespacedName.Name
			r.handler.metaCache.Del(instance)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	reqLogger.V(5).Info(fmt.Sprintf("RedisCluster Spec:\n %+v", instance.Spec))

	if err = r.handler.Do(instance); err != nil {
		switch err.Error() {
		case service.NeedRequeueImmediatelyMsg:
			return reconcile.Result{Requeue: true}, nil
		case service.NeedRequeueMsg:
			return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
		default:
			reqLogger.Error(err, "Reconcile handler")
			return reconcile.Result{}, err
		}
	}

	if err = r.handler.rcChecker.CheckSentinelReadyReplicas(instance); err != nil {
		reqLogger.Info(err.Error())
		return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// returns false if redisCluster is ignored (not managed) by this operator.
			if !shouldManage(e.Object) {
				return false
			}
			log.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call CreateFunc")
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// returns false if redisCluster is ignored (not managed) by this operator.
			if !shouldManage(e.Object) {
				return false
			}
			log.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call DeleteFunc")
			metrics.ClusterMetrics.DeleteCluster(e.Object.GetNamespace(), e.Object.GetName())
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// returns false if redisCluster is ignored (not managed) by this operator.
			if !shouldManage(e.ObjectNew) {
				return false
			}
			log.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).V(5).Info("Call UpdateFunc")
			// Ignore updates to CR status in which case metadata.Generation does not change
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				log.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).
					Info("Generation change return true", "old", e.ObjectOld, "new", e.ObjectNew)
				return true
			}
			return false
		},
		GenericFunc: nil,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.RedisCluster{}).
		WithEventFilter(pred).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
			Reconciler:              nil,
			RateLimiter:             nil,
			Log:                     nil,
			CacheSyncTimeout:        0,
		}).
		Complete(r)
}

func shouldManage(meta v1.Object) bool {
	if v, ok := meta.GetAnnotations()[util.AnnotationScope]; ok {
		if util.IsClusterScoped() {
			return v == util.AnnotationClusterScoped
		}
	} else {
		if !util.IsClusterScoped() {
			return true
		}
	}
	return false
}
