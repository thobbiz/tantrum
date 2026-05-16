package balancer

import "sync/atomic"

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
