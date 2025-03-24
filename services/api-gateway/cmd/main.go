package main

import (
	"fmt"
	"log"
	"net/http"

	"api-gateway/configs"
	"api-gateway/internal/gateway"
)

func main() {
	cfg := configs.LoadConfig()
	cfg.Print()

	mux := http.NewServeMux()

	gwHandler := gateway.NewGatewayHandler(cfg)
	gwHandler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}

	fmt.Printf("API Gateway listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Gateway server failed: %v", err)
	}
}
