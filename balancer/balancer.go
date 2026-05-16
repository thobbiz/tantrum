package balancer

import (
	"net/http"
	"net/http/httputil"
	"sync/atomic"
	"time"
)

type Balancer struct {
	Backends []*Backend
	Counter  uint64
}

func NewBalancer(backends []*Backend) *Balancer {
	return &Balancer{
		Backends: backends,
		Counter:  0,
	}
}

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

func (b *Balancer) HealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, backend := range b.Backends {
			resp, err := http.Get(backend.URL.String())
			if err != nil || resp.StatusCode >= 500 {
				backend.SetAlive(false)
			} else {
				backend.SetAlive(true)
			}
		}
	}
}

func (b *Balancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := b.NextBackend()
	if backend == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(backend.URL)
	proxy.ServeHTTP(w, r)
}
