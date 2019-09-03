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
	preRebootIncomingReqs = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "airlock_v1_pre_reboot_incoming_requests_total",
		Help: "Total number of incoming requests to /v1/pre-reboot.",
	})
)

const (
	// PreRebootEndpoint is the endpoint for requesting a semaphore lock.
	PreRebootEndpoint = "/v1/pre-reboot"
)

// PreReboot is the handler for the `/v1/pre-reboot` endpoint.
func (a *Airlock) PreReboot() http.Handler {
	prometheus.MustRegister(preRebootIncomingReqs)

	handler := func(w http.ResponseWriter, req *http.Request) {
		if herr := a.preRebootHandler(req); herr != nil {
			http.Error(w, herr.ToJSON(), herr.Code)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}

	return http.HandlerFunc(handler)
}

// preRebootHandler contains pre-reboot handling logic
func (a *Airlock) preRebootHandler(req *http.Request) *herrors.HTTPError {
	preRebootIncomingReqs.Inc()
	logrus.Debug("got pre-reboot request")

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
		"uuid":  nodeIdentity.UUID,
	}).Debug("processing client pre-reboot request")

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

	err = lockManager.RecursiveLock(ctx, nodeIdentity.UUID)
	if err != nil {
		msg := fmt.Sprintf("failed to lock semaphore: %s", err.Error())
		logrus.Errorln(msg)
		herr := herrors.New(500, "failed_lock", err.Error())
		return &herr
	}

	logrus.WithFields(logrus.Fields{
		"group": nodeIdentity.Group,
		"uuid":  nodeIdentity.UUID,
	}).Debug("givin green-flag to pre-reboot request")

	return nil
}
