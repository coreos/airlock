package server

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/coreos/airlock/internal/config"
	"github.com/coreos/airlock/internal/herrors"
)

var (
	// errNilAirlockServer is returned on nil server
	errNilAirlockServer = herrors.New(500, "nil_server", "nil Airlock server")

	// DatabaseLocksGauge holds a metrics gauge with per-group lock-holders status.
	DatabaseLocksGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "airlock_database_semaphore_lock_holders",
		Help: "Total number of locked slots per group, in the database.",
	}, []string{"group"})
	// DatabaseSlotsGauge holds a metrics gauge with per-group slots limit.
	DatabaseSlotsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
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
	for _, collector := range []prometheus.Collector{DatabaseLocksGauge, DatabaseSlotsGauge} {
		if err := prometheus.Register(collector); err != nil {
			return err
		}
	}
	return nil
}
