# Eureka Go Client
A Netflix Eureka client for Golang.

## Core Features
* Zero dependencies
* Supports [Eureka's REST operations](https://github.com/netflix/eureka/wiki/eureka-rest-operations)
* Failover if multiple Eureka server URLs are provided

## Getting Started
1. Get the package: `go get github.com/cassis163/eureka-go-client`

2. Use it:
```go
package main

import (
	"context"
	"log"
	"net"
	"time"

	eurekaClient "github.com/cassis163/eureka-go-client/pkg"
)

const (
	appID = "my-app"
	ip    = "10.5.0.50"
	ttl   = 3
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := eurekaClient.NewClient(
		[]string{"http://localhost:8761/eureka/"}, // Eureka server URLs, multiple can be provided if failover is desired
		appID,                                     // Your application's ID
		ip,                                        // Your application's IP address
		8080,                                      // Your application's port
	)
	if err != nil {
		log.Fatalf("Failed to create Eureka client: %v", err)
	}
    log.Printf("Eureka client created successfully for app ID: %s", appID)

	instance, err := client.RegisterInstance(ctx, net.ParseIP(ip), ttl, false)
	if err != nil {
		log.Fatalf("Failed to register instance: %v", err)
	}
    log.Printf("Instance registered successfully with ID: %s", instance.ID)

    var wg sync.WaitGroup

    wg.Add(1)
    go func() {
        defer wg.Done()
        periodicallySendHeartbeat(ctx, eurekaClient, time.Duration(ttl)*time.Second)
    }()

    <- ctx.Done()

    log.Println("Shutdown signal received, starting graceful shutdown...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

    wg.Wait()

    if err := eurekaClient.UnregisterInstance(shutdownCtx); err != nil {
        log.Printf("failed to unregister Eureka instance: %v", err)
    } else {
        log.Printf("Instance unregistered: %s", cfg.AppID)
    }

    log.Println("Shutdown complete.")
}

func periodicallySendHeartbeat(ctx context.Context, client lib.ClientAPI, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// optional: send one immediately
	send := func() {
		// give each request its own deadline
		hbCtx, cancel := context.WithTimeout(ctx, interval/2)
		defer cancel()

		if err := client.Heartbeat(hbCtx); err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
			return
		}
		log.Printf("Heartbeat sent for instance ID: %s", client.InstanceID())
	}

	send()

	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping heartbeat loop: %v", ctx.Err())
			return
		case <-ticker.C:
			send()
		}
	}
}
```
