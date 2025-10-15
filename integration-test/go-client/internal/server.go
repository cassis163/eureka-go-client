package internal

import (
	"fmt"
	"log"
	"net/http"
)

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func StartServer() {
	http.HandleFunc("/hello-world", helloWorldHandler)
	http.HandleFunc("/health", healthCheckHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
