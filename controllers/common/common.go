package common

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
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
	RedisSentinelControllerName   = "redis-sentinel-controller"
	RedsiStandaloneControllerName = "redis-standalone-controller"

	ExporterPort                 = 9121
	ExporterPortName             = "http-metrics"
	ExporterContainerName        = "redis-exporter"
	ExporterDefaultRequestCPU    = "50m"
	ExporterDefaultLimitCPU      = "100m"
	ExporterDefaultRequestMemory = "50Mi"
	ExporterDefaultLimitMemory   = "200Mi"

	RedisPasswordEnv = "REDIS_PASSWORD"

	RedisConfigurationVolumeName         = "redis-conf"
	RedisShutdownConfigurationVolumeName = "redis-shutdown-config"
	RedisStorageVolumeName               = "redis-data"

	GraceTime = 30

	UseLabelKey       = "used-for"
	MonitorLabelValue = "monitor"
)

var (
	// ErrNeedRequeue 表示需要等待然后重试并且不抛出错误
	ErrNeedRequeue = errors.New(NeedRequeueMsg)

	// ErrNeedRequeueImmediately 表示需要立即重试并且不抛出错误
	ErrNeedRequeueImmediately = errors.New(NeedRequeueImmediatelyMsg)

	// ControllerFlagSet 设置controller flagset
	ControllerFlagSet *pflag.FlagSet
	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 4.
	MaxConcurrentReconciles int
	// ReconcileTime is the delay between reconciliations. Defaults to 60s.
	ReconcileTime int

	// DefaultLabels 增加operator专用标签
	DefaultLabels = map[string]string{
		redisv1alpha1.LabelManagedByKey: redisv1alpha1.OperatorName,
	}

	// ParseConfigMap 特殊处理的redis设置
	ParseConfigMap = map[string]int8{
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
)

func init() {
	ControllerFlagSet = pflag.NewFlagSet("controller", pflag.ExitOnError)
	ControllerFlagSet.IntVar(&MaxConcurrentReconciles, "ctr-maxconcurrent", 4, "the maximum number of concurrent Reconciles which can be run. Defaults to 4.")
	ControllerFlagSet.IntVar(&ReconcileTime, "ctr-reconciletime", 10, "")
	namespace := util.ReplaceSymbol(util.GetEnvOrDefault("MY_POD_NAMESPACE", "default"))
	metrics.InitPrometheusMetrics(namespace, prometheus.NewRegistry())
}
