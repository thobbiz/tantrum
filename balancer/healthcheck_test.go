package balancer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckBackend_MarksAliveOn200(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	backend, err := NewBackend(srv.URL)
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}
	backend.SetAlive(false)

	lb := NewBalancer([]*Backend{backend})
	lb.checkBackend(backend)

	if !backend.IsAlive() {
		t.Fatal("expected backend to be marked alive after a 200 response")
	}
}

func TestCheckBackend_MarksDeadOn500(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
	defer srv.Close()

	backend, err := NewBackend(srv.URL)
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}

	lb := NewBalancer([]*Backend{backend})
	lb.checkBackend(backend)

	if backend.IsAlive() {
		t.Fatal("expected backend to be marked dead after a 500 response")
	}
}

func TestCheckBackend_MarksDeadOnConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := srv.URL
	srv.Close()

	backend, err := NewBackend(deadURL)
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}

	lb := NewBalancer([]*Backend{backend})
	lb.checkBackend(backend)

	if backend.IsAlive() {
		t.Fatal("expected backend to be marked dead when a connection is refused")
	}
}

func TestCheckBackend_RecoversFromDeadToAlive(t *testing.T) {
	up := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	backend, err := NewBackend(srv.URL)
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}

	lb := NewBalancer([]*Backend{backend})

	lb.checkBackend(backend)
	if backend.IsAlive() {
		t.Fatal("expected backend to be marked dead before recovery")
	}

	up = true
	lb.checkBackend(backend)
	if !backend.IsAlive() {
		t.Fatal("expected backend to be marked alive after recovery")
	}
}
