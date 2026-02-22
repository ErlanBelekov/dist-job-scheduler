package main

import (
	"context"
	"log"

	"github.com/ErlanBelekov/dist-job-scheduler/config"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/infrastructure/postgres"
	httptransport "github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	log.Println("db connected")

	r := httptransport.NewRouter()
	r.Run(":" + cfg.Port)
}
