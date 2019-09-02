package herrors

import (
	"encoding/json"
)

// HTTPError is the error type used by the main HTTP service.
type HTTPError struct {
	// Code is the HTTP code to return.
	Code int `json:"-"`
	// Kind is a machine-friendly error description.
	Kind string `json:"kind"`
	// Value is a human-friendly error description.
	Value string `json:"value"`
}

// New builds a new HTTPError.
func New(code int, kind string, value string) HTTPError {
	outKind := kind
	if outKind == "" {
		outKind = "generic_error"
	}

	outValue := value
	if outValue == "" {
		outValue = "generic error"
	}

	return HTTPError{
		Code:  code,
		Kind:  outKind,
		Value: outValue,
	}

}

// FromError converts an error.
func FromError(err error) HTTPError {
	outValue := err.Error()
	if outValue == "" {
		outValue = "generic error"
	}

	return HTTPError{
		Code:  500,
		Kind:  "generic_error",
		Value: outValue,
	}
}

// ToJSON converts an error to JSON.
func (herr HTTPError) ToJSON() string {
	out, err := json.Marshal(herr)
	if err != nil {
		return ""
	}
	return string(out)
}
