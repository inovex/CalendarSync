package config

type AdapterConfig struct {
	config Adapter
}

func NewAdapterConfig(config Adapter) AdapterConfig {
	return AdapterConfig{
		config: config,
	}
}

func (cfg AdapterConfig) Adapter() Adapter {
	return cfg.config
}
