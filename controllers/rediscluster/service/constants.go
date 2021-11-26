package service

// variables refering to the redis exporter port
const (
	// redis exporter default listen 9121
	exporterPort                 = 9121
	exporterPortName             = "http-metrics"
	exporterContainerName        = "redis-exporter"
	exporterDefaultRequestCPU    = "50m"
	exporterDefaultLimitCPU      = "100m"
	exporterDefaultRequestMemory = "50Mi"
	exporterDefaultLimitMemory   = "200Mi"

	redisPasswordEnv = "REDIS_PASSWORD"

	redisShutdownConfigurationVolumeName = "redis-shutdown-config"
	redisStorageVolumeName               = "redis-data"

	graceTime = 30

	useLabelKey       = "used-for"
	monitorLabelValue = "monitor"
)
