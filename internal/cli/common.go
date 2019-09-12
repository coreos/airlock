package cli

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/coreos/airlock/internal/config"
)

var (
	// airlockCmd is the top-level cobra command for `airlock`
	airlockCmd = &cobra.Command{
		Use:               "airlock",
		Short:             "Update/reboot manager, with distributed locking based on etcd3",
		PersistentPreRunE: commonSetup,
	}
	cmdEx = &cobra.Command{
		Use:   "ex",
		Short: "Experimental commands",
	}
	cmdGet = &cobra.Command{
		Use:   "get",
		Short: "Introspect live state",
	}

	verbosity   int
	configPath  string
	runSettings *config.Settings
)

// Init initializes the CLI environment for airlock
func Init() (*cobra.Command, error) {
	airlockCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/airlock/config.toml", "path to configuration file")
	airlockCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "increase verbosity level")

	cmdGet.AddCommand(cmdGetSlots)
	cmdEx.AddCommand(cmdGet)
	airlockCmd.AddCommand(cmdServe, cmdEx)

	return airlockCmd, nil
}

// commonSetup perform actions commons to all CLI subcommands
func commonSetup(cmd *cobra.Command, cmdArgs []string) error {
	if configPath == "" {
		return errors.New("empty path to configuration file")
	}
	logrus.SetLevel(verbosityLevel(verbosity))

	cfg, err := config.Parse(configPath)
	if err != nil {
		return err
	}
	if err := validateSettings(cfg); err != nil {
		return err
	}
	runSettings = &cfg

	logrus.WithFields(logrus.Fields{
		"endpoints": runSettings.EtcdEndpoints,
	}).Debug("etcd3 configuration")

	return nil
}

// verbosityLevel parses `-v` count into logrus log-level
func verbosityLevel(verbCount int) logrus.Level {
	switch verbCount {
	case 0:
		return logrus.WarnLevel
	case 1:
		return logrus.InfoLevel
	case 2:
		return logrus.DebugLevel
	default:
		return logrus.TraceLevel
	}

}

// validateSettings sanity-checks all settings
func validateSettings(cfg config.Settings) error {
	if len(cfg.EtcdEndpoints) == 0 {
		return errors.New("no etcd3 endpoints configured")
	}
	if len(cfg.LockGroups) == 0 {
		return errors.New("no lock-groups configured")
	}

	return nil
}
