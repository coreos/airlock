package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

// HTTPParams contains all parameters for a remote lock request.
type HTTPParams struct {
	ClientParams Params `json:"client_params"`
}

// Params contains client parameters for a remote lock request.
type Params struct {
	Group string `json:"group"`
	ID    string `json:"id"`
}

// NodeIdentity contains validated client identity from request parameters.
type NodeIdentity struct {
	Group string
	ID    string
}

// validateIdentity validates client request and parameters, returning its identity
func validateIdentity(req *http.Request) (*NodeIdentity, error) {
	if req.Header.Get("fleet-lock-protocol") != "true" {
		return nil, errors.New("wrong 'fleet-lock-protocol' header")
	}

	decoder := json.NewDecoder(req.Body)
	var input HTTPParams
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}

	if input.ClientParams.Group == "" {
		return nil, errors.New("empty client group")
	}

	if input.ClientParams.ID == "" {
		return nil, errors.New("empty client ID")
	}

	identity := NodeIdentity{
		Group: input.ClientParams.Group,
		ID:    input.ClientParams.ID,
	}

	return &identity, nil
}
