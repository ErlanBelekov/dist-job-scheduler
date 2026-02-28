package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/config"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/infrastructure/postgres"
	httptransport "github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/handler"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger := newLogger(cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		stop()
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	jobRepo := postgres.NewJobRepository(pool)
	jobUsecase := usecase.NewJobUsecase(jobRepo)
	jobHandler := handler.NewJobHandler(jobUsecase, logger)

	srv := http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httptransport.NewRouter(jobHandler),
	}

	go func() {
		logger.Info("server started", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	logger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown", "error", err)
	}
}

func newLogger(env string) *slog.Logger {
	if env == "local" {
		return slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
