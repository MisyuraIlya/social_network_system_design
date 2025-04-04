package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"api-gateway/configs"
	"api-gateway/internal/gateway"
)

func main() {
	cfg := configs.LoadConfig()
	cfg.Print()

	router := gateway.InitRoutes(cfg)

	server := &http.Server{
		Addr:    cfg.AppPort,
		Handler: router,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("API Gateway is running at %s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down API Gateway...")
}
