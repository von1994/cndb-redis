package common

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/von1994/cndb-redis/pkg/metrics"
	"github.com/von1994/cndb-redis/pkg/util"
)

// const
const (
	NeedRequeueMsg            = "need requeue"
	NeedRequeueImmediatelyMsg = "need requeue immediately"
	RedisBasicConfig          = `bind 0.0.0.0
protected-mode yes
tcp-backlog 511
timeout 0
tcp-keepalive 300
daemonize no
supervised no
pidfile /var/run/redis.pid`
	RedisClusterControllerName    = "redis-cluster-controller"
	RedsiStandaloneControllerName = "redis-standalone-controller"
)

var (
	// ErrNeedRequeue 表示需要等待然后重试并且不抛出错误
	ErrNeedRequeue = errors.New(NeedRequeueMsg)

	// ErrNeedRequeueImmediately 表示需要立即重试并且不抛出错误
	ErrNeedRequeueImmediately = errors.New(NeedRequeueImmediatelyMsg)

	ControllerFlagSet *pflag.FlagSet
	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 4.
	MaxConcurrentReconciles int
	// ReconcileTime is the delay between reconciliations. Defaults to 60s.
	ReconcileTime int
)

func init() {
	ControllerFlagSet = pflag.NewFlagSet("controller", pflag.ExitOnError)
	ControllerFlagSet.IntVar(&MaxConcurrentReconciles, "ctr-maxconcurrent", 4, "the maximum number of concurrent Reconciles which can be run. Defaults to 4.")
	ControllerFlagSet.IntVar(&ReconcileTime, "ctr-reconciletime", 10, "")
	namespace := util.ReplaceSymbol(util.GetEnvOrDefault("MY_POD_NAMESPACE", "default"))
	metrics.InitPrometheusMetrics(namespace, prometheus.NewRegistry())
}
