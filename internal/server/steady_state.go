package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/coreos/airlock/internal/herrors"
	"github.com/coreos/airlock/internal/lock"
)

var (
	steadyStateIncomingReqs = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "airlock_v1_steady_state_incoming_requests_total",
		Help: "Total number of incoming requests to /v1/steady-state.",
	})
)

const (
	// SteadyStateEndpoint is the endpoint for releasing a semaphore lock.
	SteadyStateEndpoint = "/v1/steady-state"
)

// SteadyState is the handler for the `/v1/steady-state` endpoint.
func (a *Airlock) SteadyState() http.Handler {
	prometheus.MustRegister(steadyStateIncomingReqs)

	handler := func(w http.ResponseWriter, req *http.Request) {
		if herr := a.steadyStateHandler(req); herr != nil {
			http.Error(w, herr.ToJSON(), herr.Code)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}

	return http.HandlerFunc(handler)
}

// steadyStateHandler contains logic to handle steady-state.
func (a *Airlock) steadyStateHandler(req *http.Request) *herrors.HTTPError {
	steadyStateIncomingReqs.Inc()
	logrus.Debug("got steady-state report")

	if a == nil {
		return &errNilAirlockServer
	}

	nodeIdentity, err := validateIdentity(req)
	if err != nil {
		msg := fmt.Sprintf("failed to validate client identity: %s", err.Error())
		logrus.Errorln(msg)
		herr := herrors.New(400, "invalid_client_identity", msg)
		return &herr
	}
	logrus.WithFields(logrus.Fields{
		"group": nodeIdentity.Group,
		"id":    nodeIdentity.ID,
	}).Debug("processing client steady-state report")

	slots, ok := a.LockGroups[nodeIdentity.Group]
	if !ok {
		msg := fmt.Sprintf("unknown group %q", nodeIdentity.Group)
		logrus.Errorln(msg)
		herr := herrors.New(400, "unknown_group", msg)
		return &herr
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.EtcdTxnTimeout)
	defer cancel()
	lockManager, err := lock.NewManager(ctx, a.EtcdEndpoints, nodeIdentity.Group, slots)
	if err != nil {
		msg := fmt.Sprintf("failed to initialize semaphore manager: %s", err.Error())
		logrus.Errorln(msg)
		herr := herrors.New(500, "failed_sem_init", msg)
		return &herr
	}
	defer lockManager.Close()

	sem, err := lockManager.UnlockIfHeld(ctx, nodeIdentity.ID)
	if err != nil {
		msg := fmt.Sprintf("failed to release any semaphore lock: %s", err.Error())
		logrus.Errorln(msg)
		herr := herrors.New(500, "failed_lock", err.Error())
		return &herr
	}

	// Update metrics.
	databaseLocksGauge.WithLabelValues(nodeIdentity.Group).Set(float64(len(sem.Holders)))
	databaseSlotsGauge.WithLabelValues(nodeIdentity.Group).Set(float64(sem.TotalSlots))

	logrus.WithFields(logrus.Fields{
		"group": nodeIdentity.Group,
		"id":    nodeIdentity.ID,
	}).Debug("steady-state confirmed")

	return nil
}
