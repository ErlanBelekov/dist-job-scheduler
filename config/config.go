package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	Env  string `env:"ENV" envDefault:"local" validate:"required,oneof=local staging production"`
	Port string `env:"PORT" envDefault:"8080" validate:"required"`

	DatabaseURL     string `env:"DATABASE_URL,required" validate:"required"`
	WorkerCount     int    `env:"WORKER_COUNT" envDefault:"5" validate:"min=1,max=100"`
	PollIntervalSec int    `env:"POLL_INTERVAL_SEC" envDefault:"1" validate:"min=1,max=60"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}
