// seed inserts a test user and 20 jobs into the local dev database.
// Run: go run ./cmd/seed
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/infrastructure/postgres"
)

const seedEmail = "seed@test.local"

type jobSpec struct {
	key     string
	url     string
	method  string
	retries int
	backoff string
}

var jobs = []jobSpec{
	// Happy path — should complete successfully
	{"seed-001", "https://httpbin.org/post", "POST", 3, "exponential"},
	{"seed-002", "https://httpbin.org/post", "POST", 3, "exponential"},
	{"seed-003", "https://httpbin.org/post", "POST", 3, "exponential"},
	{"seed-004", "https://httpbin.org/get", "GET", 3, "exponential"},
	{"seed-005", "https://httpbin.org/get", "GET", 3, "exponential"},

	// Will fail — server returns 500, triggers retries
	{"seed-006", "https://httpbin.org/status/500", "POST", 3, "exponential"},
	{"seed-007", "https://httpbin.org/status/500", "POST", 2, "linear"},
	{"seed-008", "https://httpbin.org/status/503", "POST", 3, "exponential"},

	// Will fail — not found
	{"seed-009", "https://httpbin.org/status/404", "GET", 1, "linear"},
	{"seed-010", "https://httpbin.org/status/404", "GET", 1, "linear"},

	// Will timeout — httpbin delays the response longer than our timeout
	{"seed-011", "https://httpbin.org/delay/35", "GET", 2, "exponential"},
	{"seed-012", "https://httpbin.org/delay/35", "GET", 2, "exponential"},

	// Mixed methods
	{"seed-013", "https://httpbin.org/put", "PUT", 3, "exponential"},
	{"seed-014", "https://httpbin.org/patch", "PATCH", 3, "exponential"},
	{"seed-015", "https://httpbin.org/delete", "DELETE", 3, "exponential"},

	// More happy path
	{"seed-016", "https://httpbin.org/post", "POST", 3, "exponential"},
	{"seed-017", "https://httpbin.org/post", "POST", 3, "exponential"},
	{"seed-018", "https://httpbin.org/get", "GET", 0, "exponential"},
	{"seed-019", "https://httpbin.org/get", "GET", 0, "exponential"},
	{"seed-020", "https://httpbin.org/post", "POST", 3, "linear"},
}

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set — run: direnv allow")
	}

	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	// Upsert test user
	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email)
		VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET updated_at = NOW()
		RETURNING id`,
		seedEmail,
	).Scan(&userID)
	if err != nil {
		pool.Close()
		log.Fatalf("upsert user: %v", err)
	}

	scheduledAt := time.Now().Add(time.Minute)

	// Insert jobs, skip any that already exist (idempotent re-runs)
	var inserted, skipped int
	var jobIDs []string

	for _, spec := range jobs {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO jobs (
				user_id, idempotency_key, url, method, headers,
				timeout_seconds, status, scheduled_at, max_retries, backoff
			) VALUES ($1, $2, $3, $4, '{}', 30, 'pending', $5, $6, $7)
			ON CONFLICT (user_id, idempotency_key) DO NOTHING
			RETURNING id`,
			userID, spec.key, spec.url, spec.method,
			scheduledAt, spec.retries, spec.backoff,
		).Scan(&id)
		if err != nil {
			pool.Close()
			log.Fatalf("insert job %s: %v", spec.key, err)
		}
		if id == "" {
			skipped++
		} else {
			jobIDs = append(jobIDs, id)
			inserted++
		}
	}

	pool.Close()

	fmt.Println("Seed complete")
	fmt.Println()
	fmt.Printf("  User:         %s\n", seedEmail)
	fmt.Printf("  User ID:      %s\n", userID)
	fmt.Printf("  Jobs created: %d  (skipped %d already existing)\n", inserted, skipped)
	fmt.Printf("  Scheduled at: %s  (~1 minute from now)\n", scheduledAt.Format(time.RFC3339))
	fmt.Println()

	if len(jobIDs) > 0 {
		fmt.Println("  Sample job IDs:")
		limit := 5
		if len(jobIDs) < limit {
			limit = len(jobIDs)
		}
		for _, id := range jobIDs[:limit] {
			fmt.Printf("    %s\n", id)
		}
	}

	fmt.Println()
	fmt.Println("How to test:")
	fmt.Println()
	fmt.Println("  Step 1 — get a JWT for the seed user:")
	fmt.Println()
	fmt.Printf("    curl -s -X POST http://localhost:8080/auth/magic-link \\\n")
	fmt.Printf("      -H 'Content-Type: application/json' \\\n")
	fmt.Printf("      -d '{\"email\":\"%s\"}'\n", seedEmail)
	fmt.Println()
	fmt.Println("    # Copy the token from the server log, then:")
	fmt.Println()
	fmt.Println("    curl -s 'http://localhost:8080/auth/verify?token=TOKEN'")
	fmt.Println("    # → {\"token\":\"eyJ...\"}")
	fmt.Println()
	fmt.Println("  Step 2 — query a job (use any ID from above):")
	fmt.Println()
	fmt.Println("    export JWT=eyJ...")
	fmt.Println("    curl -s http://localhost:8080/jobs/JOB_ID -H \"Authorization: Bearer $JWT\"")
	fmt.Println()
	fmt.Println("  Step 3 — wait ~1 minute for the scheduler to execute them, then check attempts:")
	fmt.Println()
	fmt.Println("    curl -s http://localhost:8080/jobs/JOB_ID/attempts -H \"Authorization: Bearer $JWT\"")
	fmt.Println()
	fmt.Println("  What to expect:")
	fmt.Println("    seed-001..005, 013..020  →  complete (2xx from httpbin)")
	fmt.Println("    seed-006..010            →  fail after retries (4xx/5xx)")
	fmt.Println("    seed-011..012            →  fail with timeout error (35s delay > 30s timeout)")
}
