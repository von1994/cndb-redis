package v1alpha1

const (
	maxNameLength = 48

	defaultRedisNumber    = 3
	defaultSentinelNumber = 3

	defaultStandaloneRedisNumber = 1

	defaultRedisImage = "harbor.enmotech.com/cndb-redis/redis:5.0.4-alpine"

	defaultRedisExporterImage = "harbor.enmotech.com/cndb-redis/redis-exporter:1.31.4"
	defaultImagePullPolicy    = "IfNotPresent"

	defaultSlavePriority = "1"
)
