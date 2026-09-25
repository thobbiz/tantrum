// mock-backend is a throwaway HTTP server used to exercise tantrum locally or in docker-compose.
// It identifies itself in its responses so you can watch round robin in action, and exposes a
// deliberately slow endpoint for testing graceful shutdown.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello from backend on port %s\n", port)
	})

	// Hit this to simulate a slow request for graceful-shutdown testing.
	http.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprintf(w, "slow response from backend on port %s\n", port)
	})

	log.Printf("mock backend listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
