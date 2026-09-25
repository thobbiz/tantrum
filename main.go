package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thobbiz/tantrum/balancer"
)

func main() {
	backendsFlag := flag.String("backends", "", "comma separated lis of backend URLs (e.g http://localhost:8081,http://localhost:8082)")
	intervalFlag := flag.Duration("health-interval", 0, "health-check interval, e.g 10s (default 10s)")
	flag.Parse()

	rawBackends := resolveBackends(*backendsFlag)
	if len(rawBackends) == 0 {
		log.Fatal("no backends configured: pass -backends or set the BACKENDS env var")
	}

	interval := resolveInterval(*intervalFlag)

	var backends []*balancer.Backend
	for _, raw := range rawBackends {
		b, err := balancer.NewBackend(raw)
		if err != nil {
			log.Fatalf("failed to create backend %q: %v", raw, err)
		}
		backends = append(backends, b)
	}

	lb := balancer.NewBalancer(backends)
	go lb.HealthCheck(interval)

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

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("load-balancing server stopped cleanly")
}

func resolveBackends(flagValue string) []string {
	raw := flagValue
	if raw == "" {
		raw = os.Getenv("BACKENDS")
	}

	if raw == "" {
		log.Println("No flag value read, defaulting to localhost value for local testing")
		raw = "http://localhost:8081,http://localhost:8082,http://localhost:8083"
	}

	var backends []string
	for part := range strings.SplitSeq(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			backends = append(backends, trimmed)
		}
	}
	return backends
}

func resolveInterval(flagValue time.Duration) time.Duration {
	if flagValue > 0 {
		return flagValue
	}

	if raw := os.Getenv("HEALTH_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			return d
		}
		log.Printf("invalid HEALTH_INTERVAL %q, falling back to default 10s", raw)
	}

	return 10 * time.Second
}
