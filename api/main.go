// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/main.go
// PURPOSE: Main entry point for the API
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: config, store, httpx
// USED BY: N/A
// RULES: Graceful shutdown
// DO NOT: N/A
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fairdrop/api/internal/config"
	"fairdrop/api/internal/httpx"
	"fairdrop/api/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	rdb := store.NewRedisClient(cfg.RedisURL)
	defer rdb.Client.Close()

	router := httpx.NewRouter(cfg, rdb)
	mux := router.Routes()

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		fmt.Printf("Starting server on port %s...\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen and serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
