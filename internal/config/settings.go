package config

import (
	"time"
)

// Settings stores runtime application configuration
type Settings struct {
	ServiceAddress string
	ServicePort    uint64

	EtcdEndpoints  []string
	EtcdTxnTimeout time.Duration

	LockGroups map[string]uint64
}

// Parse parses a TOML configuration file and default values
// into runtime settings
func Parse(fpath string) (Settings, error) {
	base := defaultSettings()

	settings, err := parseConfig(fpath, base)
	if err != nil {
		return Settings{}, err
	}

	// Make sure there is at least one reboot group
	if len(settings.LockGroups) == 0 {
		settings.LockGroups["default"] = 1
	}

	return settings, nil
}

// defaultSettings returns default settings for airlock commands
func defaultSettings() Settings {
	return Settings{
		ServiceAddress: "0.0.0.0",
		ServicePort:    9090,

		EtcdEndpoints:  []string{},
		EtcdTxnTimeout: time.Duration(3) * time.Second,

		LockGroups: make(map[string]uint64),
	}
}
