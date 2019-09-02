package server

import (
	"github.com/coreos/airlock/internal/config"
	"github.com/coreos/airlock/internal/herrors"
)

var (
	// errNilAirlockServer is returned on nil server
	errNilAirlockServer = herrors.New(500, "nil_server", "nil Airlock server")
)

// Airlock is the main service
type Airlock struct {
	config.Settings
}
