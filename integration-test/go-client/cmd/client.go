package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	env "github.com/caarlos0/env/v11"
	lib "github.com/cassis163/eureka-go-client"
	"github.com/cassis163/eureka-go-client/integration-test/go-client/internal"
)

const ttl = 3

type Env struct {
	ContainerIP string `env:"CONTAINER_IP"`
	EurekaURL   string `env:"EUREKA_CLIENT_SERVICE_URL_DEFAULTZONE"`
	AppID       string `env:"APP_ID" default:"go-client"`
	Port        int    `env:"PORT" default:"8080"`
}

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg Env
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	eurekaClient, err := lib.NewClient([]string{cfg.EurekaURL}, cfg.AppID, cfg.ContainerIP, cfg.Port)
	if err != nil {
		log.Fatalf("Failed to create Eureka client: %v", err)
	} else {
		log.Printf("Eureka client created successfully for app ID: %s", cfg.AppID)
	}

	instance, err := eurekaClient.RegisterInstance(context.Background(), net.ParseIP(cfg.ContainerIP), ttl, false)
	if err != nil {
		log.Fatalf("Failed to register instance: %v", err)
	} else {
		log.Printf("Instance registered successfully with ID: %s", instance.ID)
	}

    var wg sync.WaitGroup
    var server *http.Server

    wg.Add(1)
    go func() {
        defer wg.Done()
        periodicallySendHeartbeat(ctx, eurekaClient, time.Duration(ttl)*time.Second)
    }()

    wg.Add(1)
    go func() {
        defer wg.Done()
        server = internal.NewServer(":8080")
    }()

    wg.Wait()

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

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("failed to shut down server: %v", err)
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
