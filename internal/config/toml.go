package config

import (
	"time"

	"github.com/BurntSushi/toml"
)

// tomlConfig is the top-level TOML configuration fragment
type tomlConfig struct {
	Service *serviceSection
	Etcd3   *etcd3Section
	Lock    *lockSection
}

// serviceSection holds the optional `service` fragment
type serviceSection struct {
	Address *string
	Port    *uint64
}

// etcd3Section holds the optional `etcd3` fragment
type etcd3Section struct {
	Endpoints    []string
	TxnTimeoutMs *uint64 `toml:"transaction_timeout_ms"`
}

// lockSection holds the optional `lock` fragment
type lockSection struct {
	DefaultGroupName *string `toml:"default_group_name"`
	DefaultSlots     *uint64 `toml:"default_slots"`
	Groups           []lockGroupSection
}

// lockGroupSection is a `lock.groups` entry
type lockGroupSection struct {
	Name  string
	Slots *uint64
}

// parseConfig tries to parse and merge TOML config and default settings
func parseConfig(fpath string, defaults Settings) (Settings, error) {
	cfg := tomlConfig{}
	if _, err := toml.DecodeFile(fpath, &cfg); err != nil {
		return Settings{}, err
	}
	runSettings := defaults
	mergeToml(&runSettings, cfg)

	return runSettings, nil
}

// mergeToml applies a TOML configuration fragment on top of existing settings
func mergeToml(settings *Settings, cfg tomlConfig) {
	if settings == nil {
		return
	}

	if cfg.Service != nil {
		mergeService(settings, *cfg.Service)
	}
	if cfg.Etcd3 != nil {
		mergeEtcd(settings, *cfg.Etcd3)
	}
	if cfg.Lock != nil {
		mergeLock(settings, *cfg.Lock)
	}
}

func mergeService(settings *Settings, cfg serviceSection) {
	if settings == nil {
		return
	}

	if cfg.Address != nil {
		settings.ServiceAddress = *cfg.Address
	}
	if cfg.Port != nil {
		settings.ServicePort = *cfg.Port
	}
}

func mergeEtcd(settings *Settings, cfg etcd3Section) {
	if settings == nil {
		return
	}

	if len(cfg.Endpoints) != 0 {
		settings.EtcdEndpoints = append(settings.EtcdEndpoints, cfg.Endpoints...)
	}
	if cfg.TxnTimeoutMs != nil {
		settings.EtcdTxnTimeout = time.Duration(*cfg.TxnTimeoutMs) * time.Millisecond
	}
}

func mergeLock(settings *Settings, cfg lockSection) {
	if settings == nil {
		return
	}

	baseName := "default"
	baseSlots := uint64(1)

	if cfg.DefaultGroupName != nil {
		baseName = *cfg.DefaultGroupName
	}
	if cfg.DefaultSlots != nil {
		baseSlots = *cfg.DefaultSlots
	}

	for _, group := range cfg.Groups {
		slots := baseSlots
		if group.Slots != nil {
			slots = *group.Slots
		}
		settings.LockGroups[group.Name] = slots
	}

	settings.LockGroups[baseName] = baseSlots
}
