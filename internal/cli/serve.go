package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/coreos/airlock/internal/server"
	"github.com/coreos/airlock/internal/status"
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

	if runSettings == nil {
		return errors.New("nil runSettings")
	}
	airlock := server.Airlock{Settings: *runSettings}

	stopCh := make(chan os.Signal, 4)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	// Status service.
	if runSettings.StatusEnabled {
		if err := airlock.RegisterMetrics(); err != nil {
			return err
		}

		statusMux := http.NewServeMux()
		statusMux.Handle(status.MetricsEndpoint, status.Metrics())
		statusService := http.Server{
			Addr:    fmt.Sprintf("%s:%d", runSettings.StatusAddress, runSettings.StatusPort),
			Handler: statusMux,
		}
		go runService(stopCh, &statusService, airlock)
		defer statusService.Close()

		logrus.WithFields(logrus.Fields{
			"address": runSettings.StatusAddress,
			"port":    runSettings.StatusPort,
		}).Info("status service")
	} else {
		logrus.Warn("status service disabled")
	}

	// Main service.
	serviceMux := http.NewServeMux()
	serviceMux.Handle(server.PreRebootEndpoint, airlock.PreReboot())
	serviceMux.Handle(server.SteadyStateEndpoint, airlock.SteadyState())
	mainService := http.Server{
		Addr:    fmt.Sprintf("%s:%d", runSettings.ServiceAddress, runSettings.ServicePort),
		Handler: serviceMux,
	}
	logrus.WithFields(logrus.Fields{
		"address": runSettings.ServiceAddress,
		"port":    runSettings.ServicePort,
	}).Info("main service")
	go runService(stopCh, &mainService, airlock)
	defer mainService.Close()

	// Background consistency checker.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go airlock.RunConsistencyChecker(ctx)

	<-stopCh
	return nil
}

// runService runs an HTTP service
func runService(stopCh chan os.Signal, service *http.Server, airlock server.Airlock) {
	if err := service.ListenAndServe(); err != nil {
		logrus.WithFields(logrus.Fields{
			"reason": err,
		}).Error("service failure")
	}
	stopCh <- os.Interrupt
}
