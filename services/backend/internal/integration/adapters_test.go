//go:build integration

// Adapter-level integration tests. db_test.go proves the migrations produce the
// schema; this file proves the hand-written SQL in each postgresql adapter
// actually runs against it.
//
// That distinction is the reason this file exists. The adapters are the one
// place where a typo cannot be caught by the compiler or by a unit test with a
// stub — the queries are strings, and docs/adr/0005 chooses them deliberately
// over a query builder. Something has to execute them.
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sunkek/mishap"
	pgcmp "github.com/sunkek/samsara-components/postgresql"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	authpg "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/postgresql"
	authmodel "github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
	notepg "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/postgresql"
	notemodel "github.com/sunkek/samsara-template/backend/internal/domain/note/model"
	// feat:if redis,rabbitmq
	articlepg "github.com/sunkek/samsara-template/backend/internal/domain/article/adapter/postgresql"
	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
	statspg "github.com/sunkek/samsara-template/backend/internal/domain/articlestats/adapter/postgresql"
	// feat:end
)

// component starts a real postgresql component against INTEGRATION_DATABASE_URL
// and stops it when the test ends. The adapters take *Component, so a stub is
// not an option here — which is the point.
func component(t *testing.T) *pgcmp.Component {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTEGRATION_DATABASE_URL not set; run `make test-integration`")
	}

	pg := pgcmp.New(pgcmp.Config{URI: dsn, ConnectTimeout: 10 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	started := make(chan error, 1)
	go func() { started <- pg.Start(ctx, func() { started <- nil }) }()
	select {
	case err := <-started:
		if err != nil {
			t.Fatalf("start postgresql component: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("postgresql component did not become ready: %v", ctx.Err())
	}

	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = pg.Stop(stopCtx)
	})
	return pg
}

// hasCode reports whether err carries the given mishap code. The adapters
// translate pgx.ErrNoRows into e.NotFound, and that translation is what the API
// turns into a 404 — so it is behaviour, not decoration.
func hasCode(err error, code mishap.Code) bool {
	m, ok := mishap.As(err)
	return ok && m.Code() == code
}

func TestNoteAdapter(t *testing.T) {
	pg := component(t)
	a := notepg.New(pg)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	want := notemodel.Note{
		ID:        uuid.NewString(),
		Title:     "integration",
		Body:      "written by TestNoteAdapter",
		CreatedAt: now,
		UpdatedAt: now,
	}

	got, err := a.Insert(ctx, want)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if got.ID != want.ID || got.Title != want.Title || got.Body != want.Body {
		t.Fatalf("Insert returned %+v, want id/title/body of %+v", got, want)
	}
	// The RETURNING clause must give back what was stored, not what was sent:
	// a column default or a trigger would show up here.
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("Insert created_at = %v, want %v", got.CreatedAt, want.CreatedAt)
	}

	fetched, err := a.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.Title != want.Title {
		t.Errorf("Get title = %q, want %q", fetched.Title, want.Title)
	}

	list, err := a.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !containsNote(list, want.ID) {
		t.Errorf("List did not return the inserted note %s (%d rows)", want.ID, len(list))
	}

	_, err = a.Get(ctx, uuid.NewString())
	if err == nil {
		t.Fatal("Get with an unknown id returned no error")
	}
	if !hasCode(err, e.NotFound) {
		t.Errorf("Get with an unknown id = %v, want code %s", err, e.NotFound)
	}
}

func containsNote(notes []notemodel.Note, id string) bool {
	for _, n := range notes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func TestAuthAdapter(t *testing.T) {
	pg := component(t)
	a := authpg.New(pg)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	want := authmodel.User{
		ID:           uuid.NewString(),
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "not-a-real-hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	got, err := a.InsertUser(ctx, want)
	if err != nil {
		t.Fatalf("InsertUser: %v", err)
	}
	if got.Email != want.Email {
		t.Fatalf("InsertUser email = %q, want %q", got.Email, want.Email)
	}

	byEmail, err := a.GetUserByEmail(ctx, want.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	// The hash must survive the round trip: login compares against this column,
	// so a query that forgets it would fail authentication for every user.
	if byEmail.PasswordHash != want.PasswordHash {
		t.Errorf("GetUserByEmail password_hash = %q, want %q", byEmail.PasswordHash, want.PasswordHash)
	}

	byID, err := a.GetUserByID(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID.Email != want.Email {
		t.Errorf("GetUserByID email = %q, want %q", byID.Email, want.Email)
	}

	if _, err := a.GetUserByEmail(ctx, uuid.NewString()+"@example.com"); !hasCode(err, e.NotFound) {
		t.Errorf("GetUserByEmail for an unknown address = %v, want code %s", err, e.NotFound)
	}
	if _, err := a.GetUserByID(ctx, uuid.NewString()); !hasCode(err, e.NotFound) {
		t.Errorf("GetUserByID for an unknown id = %v, want code %s", err, e.NotFound)
	}

	// The unique index on email is what stops two accounts sharing a login, so
	// a second insert must fail rather than succeed quietly.
	dup := want
	dup.ID = uuid.NewString()
	if _, err := a.InsertUser(ctx, dup); err == nil {
		t.Error("InsertUser with a duplicate email succeeded; the unique index is not doing its job")
	}
}

// feat:if redis,rabbitmq
func TestArticleAdapter(t *testing.T) {
	pg := component(t)
	a := articlepg.New(pg)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	want := articlemodel.Article{
		ID:        uuid.NewString(),
		Title:     "integration",
		Body:      "written by TestArticleAdapter",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := a.Insert(ctx, want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := a.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != want.Title {
		t.Errorf("Get title = %q, want %q", got.Title, want.Title)
	}
	if _, err := a.List(ctx); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := a.Get(ctx, uuid.NewString()); !hasCode(err, e.NotFound) {
		t.Errorf("Get with an unknown id = %v, want code %s", err, e.NotFound)
	}
}

// TestArticleStatsAdapter covers the projection's upsert. It is the one query
// in the repository with ON CONFLICT logic, so "the second event increments
// rather than replaces" is worth asserting rather than assuming.
func TestArticleStatsAdapter(t *testing.T) {
	pg := component(t)
	a := statspg.New(pg)
	ctx := context.Background()

	before, err := a.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	ev := articlemodel.ArticleCreatedEvent{
		ArticleID: uuid.NewString(),
		Title:     "first",
		CreatedAt: time.Now().UTC(),
	}
	if err := a.RecordArticleCreated(ctx, ev); err != nil {
		t.Fatalf("RecordArticleCreated: %v", err)
	}
	second := articlemodel.ArticleCreatedEvent{
		ArticleID: uuid.NewString(),
		Title:     "second",
		CreatedAt: time.Now().UTC(),
	}
	if err := a.RecordArticleCreated(ctx, second); err != nil {
		t.Fatalf("RecordArticleCreated (second): %v", err)
	}

	after, err := a.Get(ctx)
	if err != nil {
		t.Fatalf("Get after record: %v", err)
	}
	if after.TotalCount != before.TotalCount+2 {
		t.Errorf("TotalCount = %d, want %d", after.TotalCount, before.TotalCount+2)
	}
	if after.LastTitle != second.Title {
		t.Errorf("LastTitle = %q, want %q", after.LastTitle, second.Title)
	}
}

// feat:end
