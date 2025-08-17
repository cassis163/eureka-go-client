# Eureka Go Client
A Netflix Eureka client for Golang.

## Core Features
* Zero dependencies
* Supports [Eureka's REST operations](https://github.com/netflix/eureka/wiki/eureka-rest-operations)
* Failover if multiple Eureka server URLs are provided

## Getting Started
1. Get the package: `go get github.com/cassis163/eureka-go-client/client`

2. Use it:
```go
package main

import (
	"context"
	"log"
	"net"
	"time"

	eurekaClient "github.com/cassis163/eureka-go-client/client/pkg"
)

const (
	appID = "my-app"
	ip    = "10.5.0.50"
	ttl   = 3
)

func main() {
	client, err := eurekaClient.NewClient(
		[]string{"http://localhost:8761/eureka"}, // Eureka server URLs, multiple can be provided if failover is desired
		appID,                                    // Your application's ID
		ip,                                       // Your application's IP address
		8080,                                     // Your application's port
	)
	if err != nil {
		log.Fatalf("Failed to create Eureka client: %v", err)
	} else {
		log.Printf("Eureka client created successfully for app ID: %s", appID)
	}

	instance, err := client.RegisterInstance(context.Background(), net.ParseIP(ip), ttl, false)
	if err != nil {
		log.Fatalf("Failed to register instance: %v", err)
	} else {
		log.Printf("Instance registered successfully with ID: %s", instance.ID)
	}

	go periodicallySendHeartbeat(client, instance.ID, time.Duration(ttl)*time.Second)
	startServer()
}

func periodicallySendHeartbeat(client eurekaClient.ClientAPI, instanceID string, interval time.Duration) {
	for {
		err := client.Heartbeat(context.Background(), instanceID)
		if err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
		} else {
			log.Printf("Heartbeat sent for instance ID: %s", instanceID)
		}
		time.Sleep(interval)
	}
}
```
