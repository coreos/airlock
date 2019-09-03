package herrors

import (
	"testing"
)

func TestJSON(t *testing.T) {
	herr := New(500, "generic_kind", "generic value")
	out := herr.ToJSON()
	expected := "{\"kind\":\"generic_kind\",\"value\":\"generic value\"}"
	if out != expected {
		t.Errorf("unexpected JSON: %s", out)
	}
}
