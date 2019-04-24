package server

import (
	"errors"

	"github.com/coreos/airlock/internal/config"
)

var (
	// errNilAirlockServer is returned on nil server
	errNilAirlockServer = errors.New("nil Airlock server")
)

// Airlock is the main service
type Airlock struct {
	config.Settings
}
