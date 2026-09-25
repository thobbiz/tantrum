package balancer

import (
	"bytes"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

var healthClient = http.Client{Timeout: 2 * time.Second}

var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
}

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

// nextUntried returns the next backend that has not been tried yet for a particular request
func (b *Balancer) nextUntried(tried map[*Backend]bool) *Backend {
	total := uint64(len(b.Backends))

	for range total {
		idx := atomic.AddUint64(&b.Counter, 1) % total
		backend := b.Backends[idx]
		if backend.IsAlive() && !tried[backend] {
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
	// Buffer the request body once so each retry attempt can replay it —
	// Backend.Proxy.ServeHTTP consumes r.Body, so reusing the same
	// *http.Request across attempts would send an empty body otherwise.
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
	}

	maxAttempts := len(b.Backends)
	if !idempotentMethods[r.Method] {
		maxAttempts = 1
	}

	tried := make(map[*Backend]bool)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		backend := b.nextUntried(tried)
		if backend == nil {
			break
		}
		tried[backend] = true

		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		buf := newBufferedResponse()
		backend.Proxy.ServeHTTP(buf, r)

		if buf.statusCode < 500 {
			for k, vv := range buf.Header() {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(buf.statusCode)
			w.Write(buf.body.Bytes())
			return
		}
	}

	http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
}
