package balancer

import (
	"net/url"
	"testing"
)

// newTestBackend builds a Backend without using NewBackend
func newTestBackend(t *testing.T, rawURL string) *Backend {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse test URL %q: %v", rawURL, err)
	}
	return &Backend{URL: parsed, alive: true}
}

func TestNextBackend_CyclesInRoundRobinOrder(t *testing.T) {
	b1 := newTestBackend(t, "http://backend-1")
	b2 := newTestBackend(t, "http://backend-2")
	b3 := newTestBackend(t, "http://backend-3")
	lb := NewBalancer([]*Backend{b1, b2, b3})

	expect := []*Backend{b2, b3, b1, b2, b3, b1}

	for i, expectedBackend := range expect {
		got := lb.NextBackend()
		if got != expectedBackend {
			t.Errorf("pick %d: got backend %s, expected %s", i, got.URL, expectedBackend.URL)
		}
	}
}

func TestNextBackend_SkipsDeadBackends(t *testing.T) {
	b1 := newTestBackend(t, "http://backend-1")
	b2 := newTestBackend(t, "http://backend-2")
	b3 := newTestBackend(t, "http://backend-3")
	b2.SetAlive(false)

	lb := NewBalancer([]*Backend{b1, b2, b3})

	for i := range 10 {
		got := lb.NextBackend()
		if got == b2 {
			t.Fatalf("pick %d: NextBackend returned a dead backend (%s)", i, got.URL)
		}
	}
}

func TestNextBackend_ReturnsNilWhenAllDead(t *testing.T) {
	b1 := newTestBackend(t, "http://backend-1")
	b2 := newTestBackend(t, "http://backend-2")
	b1.SetAlive(false)
	b2.SetAlive(false)

	lb := NewBalancer([]*Backend{b1, b2})

	if got := lb.NextBackend(); got != nil {
		t.Fatalf("expected nil when all backends are dead, got %s", got.URL)
	}
}

func TestNextBackend_ReturnNilWithNoBackend(t *testing.T) {
	lb := NewBalancer(nil)

	if got := lb.NextBackend(); got != nil {
		t.Fatalf("expected nil when there are no backend, got %s", got.URL)
	}
}
