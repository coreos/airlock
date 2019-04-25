package cli

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cmdServe = &cobra.Command{
		Use:  "serve",
		RunE: runServe,
	}
)

func runServe(cmd *cobra.Command, cmdArgs []string) error {
	logrus.WithFields(logrus.Fields{
		"groups": runSettings.LockGroups,
	}).Debug("lock groups")

	logrus.WithFields(logrus.Fields{
		"address": runSettings.ServiceAddress,
		"port":    runSettings.ServicePort,
	}).Info("starting service")

	return nil
}
