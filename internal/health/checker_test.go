package health_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/health"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(_ context.Context) error { return m.err }

func newTestChecker(p health.Pinger) (*health.Checker, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	logger := slog.Default()
	return health.NewChecker(p, logger, reg), reg
}

func TestLiveness_AlwaysUp(t *testing.T) {
	c, _ := newTestChecker(&mockPinger{err: errors.New("db down")})

	result := c.Liveness(context.Background())
	if result.Status != "up" {
		t.Fatalf("expected status up, got %s", result.Status)
	}
	if result.Checks != nil {
		t.Fatalf("expected no checks, got %v", result.Checks)
	}
}

func TestReadiness_PostgresUp(t *testing.T) {
	c, reg := newTestChecker(&mockPinger{})

	result := c.Readiness(context.Background())
	if result.Status != "up" {
		t.Fatalf("expected status up, got %s", result.Status)
	}
	pg, ok := result.Checks["postgres"]
	if !ok {
		t.Fatal("missing postgres check")
	}
	if pg.Status != "up" {
		t.Fatalf("expected postgres up, got %s", pg.Status)
	}

	gauge := testGauge(t, reg, "scheduler_health_check_up", "postgres")
	if gauge != 1 {
		t.Fatalf("expected gauge 1, got %f", gauge)
	}
}

func TestReadiness_PostgresDown(t *testing.T) {
	c, reg := newTestChecker(&mockPinger{err: errors.New("connection refused")})

	result := c.Readiness(context.Background())
	if result.Status != "down" {
		t.Fatalf("expected status down, got %s", result.Status)
	}
	pg := result.Checks["postgres"]
	if pg.Status != "down" {
		t.Fatalf("expected postgres down, got %s", pg.Status)
	}
	if pg.Error == "" {
		t.Fatal("expected error message")
	}

	gauge := testGauge(t, reg, "scheduler_health_check_up", "postgres")
	if gauge != 0 {
		t.Fatalf("expected gauge 0, got %f", gauge)
	}
}

func testGauge(t *testing.T, reg *prometheus.Registry, name, depLabel string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "dependency" && lp.GetValue() == depLabel {
					return m.GetGauge().GetValue()
				}
			}
		}
	}
	t.Fatalf("metric %s{dependency=%q} not found", name, depLabel)
	return 0
}

// Silence the unused import lint for testutil if we only use Gather above.
var _ = testutil.ToFloat64
