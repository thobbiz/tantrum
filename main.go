package main

import (
	"log"
	"net/http"
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

	log.Println("starting load-balancing server on 8080")
	err = srv.ListenAndServe()
	log.Fatal(err)
}
