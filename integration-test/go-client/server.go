package main

import (
    "fmt"
    "log"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}

func startServer() {
    http.HandleFunc("/hello-world", handler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
