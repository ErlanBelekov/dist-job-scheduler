package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// CheckResult represents the health of a single dependency.
type CheckResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// HealthResult is the top-level health response.
type HealthResult struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks,omitempty"`
}

// Checker verifies that all dependencies are reachable.
type Checker struct {
	db     Pinger
	logger *slog.Logger
	gauge  *prometheus.GaugeVec
}

// NewChecker creates a health checker and registers its Prometheus gauge.
func NewChecker(db Pinger, logger *slog.Logger, reg prometheus.Registerer) *Checker {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scheduler",
		Name:      "health_check_up",
		Help:      "Whether a dependency is reachable. 1 = up, 0 = down.",
	}, []string{"dependency"})
	reg.MustRegister(gauge)

	return &Checker{
		db:     db,
		logger: logger.With("component", "health"),
		gauge:  gauge,
	}
}

// Liveness returns a simple "up" response if the process is running.
func (c *Checker) Liveness(_ context.Context) HealthResult {
	return HealthResult{Status: "up"}
}

// Readiness pings every dependency and reports per-check status.
func (c *Checker) Readiness(ctx context.Context) HealthResult {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	result := HealthResult{
		Status: "up",
		Checks: make(map[string]CheckResult),
	}

	if err := c.db.Ping(checkCtx); err != nil {
		c.logger.Warn("postgres health check failed", "error", err)
		result.Status = "down"
		result.Checks["postgres"] = CheckResult{Status: "down", Error: err.Error()}
		c.gauge.WithLabelValues("postgres").Set(0)
	} else {
		result.Checks["postgres"] = CheckResult{Status: "up"}
		c.gauge.WithLabelValues("postgres").Set(1)
	}

	return result
}
