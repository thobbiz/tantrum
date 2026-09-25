package balancer

import (
	"net/http"
	"net/http/httputil"
	"sync/atomic"
	"time"
)

var healthClient = http.Client{Timeout: 2 * time.Second}

type Balancer struct {
	Backends []*Backend
	Counter  uint64
}

// NewBalancer creates a new Balancer with the given backends
func NewBalancer(backends []*Backend) *Balancer {
	return &Balancer{
		Backends: backends,
		Counter:  0,
	}
}

// NextBackend picks the next backend to handle the request using a round-robin algorithm
func (b *Balancer) NextBackend() *Backend {
	total := uint64(len(b.Backends))

	for range total {
		idx := atomic.AddUint64(&b.Counter, 1) % total
		backend := b.Backends[idx]
		if backend.IsAlive() {
			return backend
		}
	}
	return nil
}

// checkBackend sends a health check request to the backend and updates its alive status
func (b *Balancer) checkBackend(backend *Backend) {
	resp, err := healthClient.Get(backend.URL.String())
	if err != nil {
		backend.SetAlive(false)
		return
	}
	defer resp.Body.Close()
	backend.SetAlive(resp.StatusCode < 500)
}

// HealthCheck performs a health check on all backends at the specified interval
func (b *Balancer) HealthCheck(interval time.Duration) {
	// check immediately instead of waiting for 10 seconds
	for _, backend := range b.Backends {
		b.checkBackend(backend)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, backend := range b.Backends {
			b.checkBackend(backend)
		}
	}
}

// ServeHTTP satisfies the http.Handler interface
func (b *Balancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := b.NextBackend()
	if backend == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(backend.URL)
	proxy.ServeHTTP(w, r)
}
