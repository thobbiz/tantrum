package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thobbiz/load-balancer/balancer"
)

func main() {
	backend1, err := balancer.NewBackend("http://localhost:8081")
	if err != nil {
		log.Fatalf("failed to create backend: %v", err)
	}
	backend2, err := balancer.NewBackend("http://localhost:8082")
	if err != nil {
		log.Fatalf("failed to create backend: %v", err)
	}

	backend3, err := balancer.NewBackend("http://localhost:8083")
	if err != nil {
		log.Fatalf("failed to create backend: %v", err)
	}

	q := []*balancer.Backend{backend1, backend2, backend3}
	lb := balancer.NewBalancer(q)

	go lb.HealthCheck(10 * time.Second)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	go func() {
		log.Println("starting load-balancing server on 8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start load-balancing server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutdown signal received, draining connections")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("load-balancing server stopped cleanly")
}
