//go:build integration

// Package integration holds tests that exercise real infrastructure. They are
// excluded from the default `go test ./...` by the `integration` build tag and
// run via `make test-integration`, which starts the dev infra, applies
// migrations, and sets INTEGRATION_DATABASE_URL.
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// connect dials the database named by INTEGRATION_DATABASE_URL. The test is
// skipped when the variable is unset so it never breaks a plain `go test`.
func connect(t *testing.T) *pgx.Conn {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTEGRATION_DATABASE_URL not set; run `make test-integration`")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// TestSchemaMigrated confirms the sample migrations have been applied.
func TestSchemaMigrated(t *testing.T) {
	conn := connect(t)
	for _, table := range []string{"users", "notes"} {
		var exists bool
		err := conn.QueryRow(context.Background(),
			`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)`,
			table).Scan(&exists)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q missing — run `make migrate-up`", table)
		}
	}
}

// TestNoteRoundTrip inserts and reads back a note against the real database,
// covering the schema the note domain depends on.
func TestNoteRoundTrip(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := conn.Exec(ctx,
		`INSERT INTO notes (id, title, body, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		id, "integration-title", "integration-body", now, now)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM notes WHERE id = $1`, id)
	})

	var title, body string
	if err := conn.QueryRow(ctx,
		`SELECT title, body FROM notes WHERE id = $1`, id).Scan(&title, &body); err != nil {
		t.Fatalf("select: %v", err)
	}
	if title != "integration-title" || body != "integration-body" {
		t.Errorf("round-trip mismatch: title=%q body=%q", title, body)
	}
}
