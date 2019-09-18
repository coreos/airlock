package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/coreos/airlock/internal/lock"
	"github.com/coreos/airlock/internal/server"
	"github.com/coreos/airlock/internal/status"
)

var (
	cmdServe = &cobra.Command{
		Use:  "serve",
		RunE: runServe,
	}

	configGroups = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "airlock_config_groups",
		Help: "Total number of configured groups.",
	})
	configSlots = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "airlock_config_semaphore_slots",
		Help: "Total number of configured slots per group.",
	}, []string{"group"})
)

// runServe runs the main HTTP service
func runServe(cmd *cobra.Command, cmdArgs []string) error {
	logrus.WithFields(logrus.Fields{
		"groups": runSettings.LockGroups,
	}).Debug("lock groups")

	if runSettings == nil {
		return errors.New("nil runSettings")
	}
	airlock := server.Airlock{*runSettings}

	stopCh := make(chan os.Signal)
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
		go runService(stopCh, statusService, airlock)
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
	go runService(stopCh, mainService, airlock)
	defer mainService.Close()

	// Background consistency checker.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go checkConsistency(ctx, airlock)

	<-stopCh
	return nil
}

// runService runs an HTTP service
func runService(stopCh chan os.Signal, service http.Server, airlock server.Airlock) {
	if err := service.ListenAndServe(); err != nil {
		logrus.WithFields(logrus.Fields{
			"reason": err,
		}).Error("service failure")
	}
	stopCh <- os.Interrupt
}

// checkConsistency continuously checks for consistency between configuration and remote state.
//
// It takes care of polling etcd, exposing the shared state as metrics, and warning if
// it detects a mismatch with the service configuration.
func checkConsistency(ctx context.Context, service server.Airlock) {
	prometheus.MustRegister(configGroups)
	prometheus.MustRegister(configSlots)

	configGroups.Set(float64(len(service.LockGroups)))
	for group, maxSlots := range service.LockGroups {
		configSlots.WithLabelValues(group).Set(float64(maxSlots))
	}

	// Consistency-checking logic, with its own scope for defers.
	checkAndLog := func() {
		for group, maxSlots := range service.LockGroups {
			innerCtx, cancel := context.WithTimeout(ctx, service.EtcdTxnTimeout)
			defer cancel()

			// TODO(lucab): re-arrange so that the manager can be re-used.
			manager, err := lock.NewManager(innerCtx, service.EtcdEndpoints, group, maxSlots)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"reason": err.Error(),
				}).Warn("consistency check, manager creation failed")
				continue
			}
			semaphore, err := manager.FetchSemaphore(innerCtx)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"reason": err.Error(),
				}).Warn("consistency check, semaphore fetch failed")
				continue
			}

			// Update metrics.
			server.DatabaseLocksGauge.WithLabelValues(group).Set(float64(len(semaphore.Holders)))
			server.DatabaseSlotsGauge.WithLabelValues(group).Set(float64(semaphore.TotalSlots))

			// Log any inconsistencies.
			if semaphore.TotalSlots != maxSlots {
				logrus.WithFields(logrus.Fields{
					"config":   maxSlots,
					"database": semaphore.TotalSlots,
					"group":    group,
				}).Warn("semaphore max slots consistency check failed")
			}
			if semaphore.TotalSlots < uint64(len(semaphore.Holders)) {
				logrus.WithFields(logrus.Fields{
					"group":  group,
					"holder": len(semaphore.Holders),
					"slots":  semaphore.TotalSlots,
				}).Warn("semaphore locks consistency check failed")
			}
		}
	}

	for {
		checkAndLog()

		pause := time.NewTimer(time.Minute)
		select {
		case <-ctx.Done():
			break
		case <-pause.C:
			continue
		}
	}
}
