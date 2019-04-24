package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/coreos/airlock/internal/lock"
)

const (
	// SteadyStateEndpoint is the endpoint for releasing a semaphore lock.
	SteadyStateEndpoint = "/v1/steady-state"
)

// SteadyState is the handler for the `/v1/steady-state` endpoint.
func (a *Airlock) SteadyState() http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		logrus.Debug("got steady-state report")
		if a == nil {
			http.Error(w, errNilAirlockServer.Error(), 500)
			return
		}

		nodeIdentity, err := validateIdentity(req)
		if err != nil {
			logrus.Errorln("failed to validate client identity: ", err)
			http.Error(w, err.Error(), 400)
			return
		}
		logrus.WithFields(logrus.Fields{
			"group": nodeIdentity.Group,
			"uuid":  nodeIdentity.UUID,
		}).Debug("processing client steady-state report")

		slots, ok := a.LockGroups[nodeIdentity.Group]
		if !ok {
			err := fmt.Errorf("unknown group %q", nodeIdentity.Group)
			logrus.Errorln("unable to satisfy client request: ", err)
			http.Error(w, err.Error(), 400)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), a.EtcdTxnTimeout)
		defer cancel()
		lockManager, err := lock.NewManager(ctx, a.EtcdEndpoints, nodeIdentity.Group, slots)
		if err != nil {
			logrus.Errorln("failed to initialize semaphore manager: ", err)
			http.Error(w, err.Error(), 500)
			return
		}
		defer lockManager.Close()

		err = lockManager.UnlockIfHeld(ctx, nodeIdentity.UUID)
		if err != nil {
			logrus.Errorln("failed to release any semaphore lock: ", err)
			http.Error(w, err.Error(), 500)
			return
		}

		logrus.WithFields(logrus.Fields{
			"group": nodeIdentity.Group,
			"uuid":  nodeIdentity.UUID,
		}).Debug("steady-state confirmed")
	}

	return http.HandlerFunc(handler)
}
