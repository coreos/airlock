package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

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
		code, err := a.steadyStateHandler(req)
		if err != nil {
			http.Error(w, err.Error(), code)
		} else {
			w.WriteHeader(code)
		}
	}

	return http.HandlerFunc(handler)
}

// steadyStateHandler contains logic to handle steady-state.
func (a *Airlock) steadyStateHandler(req *http.Request) (int, error) {
	steadyStateIncomingReqs.Inc()
	logrus.Debug("got steady-state report")

	if a == nil {
		return 500, errNilAirlockServer
	}

	nodeIdentity, err := validateIdentity(req)
	if err != nil {
		logrus.Errorln("failed to validate client identity: ", err)
		return 400, err
	}
	logrus.WithFields(logrus.Fields{
		"group": nodeIdentity.Group,
		"uuid":  nodeIdentity.UUID,
	}).Debug("processing client steady-state report")

	slots, ok := a.LockGroups[nodeIdentity.Group]
	if !ok {
		err := fmt.Errorf("unknown group %q", nodeIdentity.Group)
		logrus.Errorln("unable to satisfy client request: ", err)
		return 400, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.EtcdTxnTimeout)
	defer cancel()
	lockManager, err := lock.NewManager(ctx, a.EtcdEndpoints, nodeIdentity.Group, slots)
	if err != nil {
		logrus.Errorln("failed to initialize semaphore manager: ", err)
		return 500, err
	}
	defer lockManager.Close()

	err = lockManager.UnlockIfHeld(ctx, nodeIdentity.UUID)
	if err != nil {
		logrus.Errorln("failed to release any semaphore lock: ", err)
		return 500, err
	}

	logrus.WithFields(logrus.Fields{
		"group": nodeIdentity.Group,
		"uuid":  nodeIdentity.UUID,
	}).Debug("steady-state confirmed")

	return http.StatusOK, nil
}
