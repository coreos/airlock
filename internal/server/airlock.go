package server

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/coreos/airlock/internal/config"
	"github.com/coreos/airlock/internal/herrors"
	"github.com/coreos/airlock/internal/lock"
)

var (
	// errNilAirlockServer is returned on nil server
	errNilAirlockServer = herrors.New(500, "nil_server", "nil Airlock server")

	// configGroupsGauge holds a metrics gauge with number of configured groups.
	configGroupsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "airlock_config_groups",
		Help: "Total number of configured groups.",
	})
	// configSlotsGauge holds a metrics gauge with per-group configured slots.
	configSlotsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "airlock_config_semaphore_slots",
		Help: "Total number of configured slots per group.",
	}, []string{"group"})
	// databaseLocksGauge holds a metrics gauge with per-group lock-holders status.
	databaseLocksGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "airlock_database_semaphore_lock_holders",
		Help: "Total number of locked slots per group, in the database.",
	}, []string{"group"})
	// databaseSlotsGauge holds a metrics gauge with per-group slots limit.
	databaseSlotsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "airlock_database_semaphore_slots",
		Help: "Total number of slots per group, in the database.",
	}, []string{"group"})
)

// Airlock is the main service
type Airlock struct {
	config.Settings
}

// RegisterMetrics registers all server-related metrics.
func (a *Airlock) RegisterMetrics() error {
	if a == nil {
		return errors.New("nil Airlock")
	}

	collectors := []prometheus.Collector{
		configGroupsGauge,
		configSlotsGauge,
		databaseLocksGauge,
		databaseSlotsGauge,
	}
	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			return err
		}
	}

	configGroupsGauge.Set(float64(len(a.LockGroups)))
	for group, maxSlots := range a.LockGroups {
		configSlotsGauge.WithLabelValues(group).Set(float64(maxSlots))
	}

	return nil
}

// RunConsistencyChecker runs a continuous checker for consistency between configuration
// and remote state.
func (a *Airlock) RunConsistencyChecker(ctx context.Context) {
	for {
		for group, maxSlots := range a.LockGroups {
			a.checkConsistency(ctx, group, maxSlots)
		}

		pause := time.NewTimer(time.Minute)
		select {
		case <-ctx.Done():
			break
		case <-pause.C:
			continue
		}
	}
}

// checkConsistencytakes takes care of polling etcd, exposing the shared state as metrics,
// and warning if it detects a mismatch with the service configuration.
func (a *Airlock) checkConsistency(ctx context.Context, group string, maxSlots uint64) {
	if a == nil {
		logrus.Error("consistency check, nil Airlock")
		return
	}

	innerCtx, cancel := context.WithTimeout(ctx, a.EtcdTxnTimeout)
	defer cancel()

	// TODO(lucab): re-arrange so that the manager can be re-used.
	manager, err := lock.NewManager(innerCtx, a.EtcdEndpoints, group, maxSlots)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"reason": err.Error(),
		}).Warn("consistency check, manager creation failed")
		return
	}
	semaphore, err := manager.FetchSemaphore(innerCtx)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"reason": err.Error(),
		}).Warn("consistency check, semaphore fetch failed")
		return
	}

	// Update metrics.
	databaseLocksGauge.WithLabelValues(group).Set(float64(len(semaphore.Holders)))
	databaseSlotsGauge.WithLabelValues(group).Set(float64(semaphore.TotalSlots))

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
