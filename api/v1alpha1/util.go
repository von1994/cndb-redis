package v1alpha1

func enablePersistence(config map[string]string) {
	setConfigMapIfNotExist("appendonly", "yes", config)
	setConfigMapIfNotExist("auto-aof-rewrite-min-size", "536870912", config)
	setConfigMapIfNotExist("auto-aof-rewrite-percentage", "100", config)
	setConfigMapIfNotExist("repl-backlog-size", "62914560", config)
	setConfigMapIfNotExist("repl-diskless-sync", "yes", config)
	setConfigMapIfNotExist("aof-load-truncated", "yes", config)
	setConfigMapIfNotExist("stop-writes-on-bgsave-error", "no", config)
	setConfigMapIfNotExist("save", "900 1 300 10", config)
}

func setConfigMapIfNotExist(key, value string, config map[string]string) {
	if _, ok := config[key]; !ok {
		config[key] = value
	}
}

func disablePersistence(config map[string]string) {
	config["appendonly"] = "no"
	config["save"] = ""
}
