package balancer

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	alive bool
	mux   sync.RWMutex
}

func NewBackend(rawURL string) (*Backend, error) {
	parsedUrl, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	b := &Backend{
		URL:   parsedUrl,
		alive: true,
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for backend %s: %v", parsedUrl, err)
		b.SetAlive(false)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
	b.Proxy = proxy

	return b, nil
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()

	b.alive = alive
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.alive
}
