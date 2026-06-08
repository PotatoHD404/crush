package netguard

import (
	"net/http"
	"strings"
	"testing"
)

func TestBlocksForbiddenHost(t *testing.T) {
	Install()
	// A forbidden external host must be rejected at dial time by the guard. Without the
	// guard this would attempt a real connection to charm.land.
	_, err := http.Get("https://catwalk.charm.land/v2/providers")
	if err == nil {
		t.Fatal("expected request to catwalk.charm.land to be blocked")
	}
	if !strings.Contains(err.Error(), "netguard") {
		t.Fatalf("expected a netguard block error, got: %v", err)
	}
}

func TestAllowsLoopback(t *testing.T) {
	Install()
	// Loopback must pass the guard. Port 9 is almost certainly closed, so we expect a
	// connection error — but NOT a netguard block.
	_, err := http.Get("http://127.0.0.1:9/")
	if err != nil && strings.Contains(err.Error(), "netguard") {
		t.Fatalf("loopback must not be blocked by netguard, got: %v", err)
	}
}
