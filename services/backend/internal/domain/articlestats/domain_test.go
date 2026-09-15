package articlestats

import (
	"context"
	"errors"
	"testing"
	"time"

	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats/model"
)

// The read-model's use cases are deliberately thin — the projection logic is
// the SQL upsert, which internal/integration exercises against a real database.
// What is worth pinning here is the wiring: the consumer's event reaches the
// port unchanged, an empty projection is a value rather than an error, and a
// store failure is propagated rather than swallowed into a zero Stats that a
// caller would render as "no articles yet".

type stubDB struct {
	stats    model.Stats
	getErr   error
	applyErr error

	gotEvent articlemodel.ArticleCreatedEvent
	applies  int
}

func (s *stubDB) RecordArticleCreated(_ context.Context, e articlemodel.ArticleCreatedEvent) error {
	s.applies++
	s.gotEvent = e
	return s.applyErr
}

func (s *stubDB) Get(context.Context) (model.Stats, error) { return s.stats, s.getErr }

func TestApplyArticleCreatedPassesTheEventThrough(t *testing.T) {
	db := &stubDB{}
	d := New(db)

	ev := articlemodel.ArticleCreatedEvent{
		ArticleID: "a1",
		Title:     "hello",
		CreatedAt: time.Now().UTC(),
	}
	if err := d.ApplyArticleCreated(context.Background(), ev); err != nil {
		t.Fatalf("ApplyArticleCreated: %v", err)
	}
	if db.applies != 1 {
		t.Errorf("store was called %d times, want 1", db.applies)
	}
	if db.gotEvent != ev {
		t.Errorf("store got %+v, want %+v", db.gotEvent, ev)
	}
}

// The consumer acks on nil and requeues on error, so a swallowed store failure
// here loses the event permanently.
func TestApplyArticleCreatedPropagatesStoreErrors(t *testing.T) {
	want := errors.New("store down")
	d := New(&stubDB{applyErr: want})

	err := d.ApplyArticleCreated(context.Background(), articlemodel.ArticleCreatedEvent{ArticleID: "a1"})
	if !errors.Is(err, want) {
		t.Fatalf("ApplyArticleCreated error = %v, want %v", err, want)
	}
}

func TestGetReturnsTheProjection(t *testing.T) {
	want := model.Stats{TotalCount: 7, LastArticleID: "a7", LastTitle: "seventh"}
	d := New(&stubDB{stats: want})

	got, err := d.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Errorf("Get = %+v, want %+v", got, want)
	}
}

// No events yet is a zero projection, not a 404: the adapter translates
// pgx.ErrNoRows into an empty Stats, and the domain must not turn that back
// into an error.
func TestGetEmptyProjectionIsNotAnError(t *testing.T) {
	d := New(&stubDB{})

	got, err := d.Get(context.Background())
	if err != nil {
		t.Fatalf("Get on an empty projection returned %v, want nil", err)
	}
	if got != (model.Stats{}) {
		t.Errorf("Get = %+v, want the zero Stats", got)
	}
}

func TestGetPropagatesStoreErrors(t *testing.T) {
	want := errors.New("store down")
	d := New(&stubDB{getErr: want})

	if _, err := d.Get(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Get error = %v, want %v", err, want)
	}
}
