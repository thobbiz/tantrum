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

func NewBackend(rawURL string) (*Backend, error) {
	parsedUrl, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &Backend{
		URL:   parsedUrl,
		alive: true,
	}, nil
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
