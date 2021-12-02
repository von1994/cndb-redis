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
	"github.com/von1994/cndb-redis/controllers/common"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/cache"
	"github.com/von1994/cndb-redis/controllers/redisstandalone/service"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	"github.com/von1994/cndb-redis/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
)

// RedisStandaloneReconciler reconciles a RedisStandalone object
type RedisStandaloneReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	handler *StandaloneHandler
}

var _ reconcile.Reconciler = &RedisStandaloneReconciler{}

// NewRedisStandaloneReconciler returns a new reconcile.Reconciler
func NewRedisStandaloneReconciler(mgr manager.Manager) *RedisStandaloneReconciler {
	// Create kubernetes service.
	k8sService := k8s.New(mgr.GetClient(), log.Log)

	// Create the redis clients
	redisClient := redis.New()

	// Create internal services.
	rcService := service.NewRedisStandaloneKubeClient(k8sService, log.Log)
	rcChecker := service.NewRedisStandaloneChecker(k8sService, redisClient, log.Log)
	rcHealer := service.NewRedisStandaloneHealer(k8sService, redisClient, log.Log)

	handler := &StandaloneHandler{
		k8sServices: k8sService,
		rcService:   rcService,
		rcChecker:   rcChecker,
		rcHealer:    rcHealer,
		metaCache:   new(cache.MetaMap),
		eventsCli:   k8s.NewEvent(mgr.GetEventRecorderFor(common.RedsiStandaloneControllerName), log.Log),
		logger:      log.Log,
	}

	return &RedisStandaloneReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), handler: handler}
}

//+kubebuilder:rbac:groups=redis.lovelycat.io,resources=redisstandalones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=redis.lovelycat.io,resources=redisstandalones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=redis.lovelycat.io,resources=redisstandalones/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisSentinel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *RedisStandaloneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	reqLogger := log.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling RedisStandalone")

	// Fetch the RedisStandalone instance
	instance := &redisv1alpha1.RedisStandalone{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("RedisStandalone delete")
			instance.Namespace = req.NamespacedName.Namespace
			instance.Name = req.NamespacedName.Name
			r.handler.metaCache.Del(instance)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	reqLogger.V(5).Info(fmt.Sprintf("RedisStandalone Spec:\n %+v", instance.Spec))

	if err = r.handler.Do(instance); err != nil {
		switch err.Error() {
		case common.NeedRequeueImmediatelyMsg:
			return reconcile.Result{Requeue: true}, nil
		case common.NeedRequeueMsg:
			return reconcile.Result{RequeueAfter: time.Duration(common.ReconcileTime) * time.Second}, nil
		default:
			reqLogger.Error(err, "Reconcile handler")
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(common.ReconcileTime) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisStandaloneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if !util.ShouldManage(e.Object) {
				return false
			}
			log.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call CreateFunc")
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if !util.ShouldManage(e.Object) {
				return false
			}
			log.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call DeleteFunc")
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if !util.ShouldManage(e.ObjectNew) {
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
		For(&redisv1alpha1.RedisStandalone{}).
		WithEventFilter(pred).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: common.MaxConcurrentReconciles,
			Reconciler:              nil,
			RateLimiter:             nil,
			Log:                     nil,
			CacheSyncTimeout:        0,
		}).
		Complete(r)
}
