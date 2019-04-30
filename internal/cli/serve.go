package cli

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/coreos/airlock/internal/server"
)

var (
	cmdServe = &cobra.Command{
		Use:  "serve",
		RunE: runServe,
	}
)

// runServe runs the main HTTP service
func runServe(cmd *cobra.Command, cmdArgs []string) error {
	logrus.WithFields(logrus.Fields{
		"groups": runSettings.LockGroups,
	}).Debug("lock groups")

	logrus.WithFields(logrus.Fields{
		"address": runSettings.ServiceAddress,
		"port":    runSettings.ServicePort,
	}).Info("starting service")

	if runSettings == nil {
		return errors.New("nil runSettings")
	}
	airlock := server.Airlock{*runSettings}

	http.Handle(server.PreRebootEndpoint, airlock.PreReboot())
	http.Handle(server.SteadyStateEndpoint, airlock.SteadyState())

	listenAddr := fmt.Sprintf("%s:%d", runSettings.ServiceAddress, runSettings.ServicePort)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		return err
	}

	return nil
}
