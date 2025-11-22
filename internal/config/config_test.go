package config

import "testing"

func TestCurrentEnvironment(t *testing.T) {
	if got := CurrentEnvironment(); got != DefaultEnvironment {
		t.Fatalf("CurrentEnvironment() = %q, want %q", got, DefaultEnvironment)
	}
}
