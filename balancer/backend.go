package balancer

import (
	"net/url"
	"sync"
)

type Backend struct {
	URL   *url.URL
	alive bool
	mux   sync.RWMutex
}

func (B *Backend) SetAlive(alive bool) {
	B.mux.Lock()
	defer B.mux.Unlock()

	B.alive = alive
}

func (B *Backend) IsAlive() bool {
	B.mux.RLock()
	defer B.mux.RUnlock()
	return B.alive

}
