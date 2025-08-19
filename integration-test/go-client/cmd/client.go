package main

import (
	"context"
	"log"
	"net"
	"time"

	env "github.com/caarlos0/env/v11"
	lib "github.com/cassis163/eureka-go-client/client"
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

	go periodicallySendHeartbeat(eurekaClient, instance.ID, time.Duration(ttl)*time.Second)
	internal.StartServer()
}

func periodicallySendHeartbeat(client lib.ClientAPI, instanceID string, interval time.Duration) {
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
