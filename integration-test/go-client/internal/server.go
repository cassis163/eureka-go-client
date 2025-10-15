package internal

import (
	"fmt"
	"net/http"
	"time"
)

func helloWorldHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "OK")
}

// NewServer builds an *http.Server that inherits ctx as the base context for connections.
func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", helloWorldHandler)
	mux.HandleFunc("/health", healthCheckHandler)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
