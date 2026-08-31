package article

import (
	"context"
	"errors"
	"testing"

	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// stubDB is an in-memory article.DB. The Service/DB ports exist precisely so the
// use cases can be tested without a real database — this is the pattern to copy
// for your own domains.
type stubDB struct {
	inserted  model.Article
	insertErr error
}

func (s *stubDB) Insert(_ context.Context, n model.Article) (model.Article, error) {
	if s.insertErr != nil {
		return model.Article{}, s.insertErr
	}
	s.inserted = n
	return n, nil
}
func (s *stubDB) List(context.Context) ([]model.Article, error)      { return nil, nil }
func (s *stubDB) Get(context.Context, string) (model.Article, error) { return model.Article{}, nil }

// codeOf extracts the mishap error code, or "" when err is nil / not a mishap.
func codeOf(err error) mishap.Code {
	if m, ok := mishap.As(err); ok {
		return m.Code()
	}
	return ""
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		in      model.CreateInput
		wantErr mishap.Code // "" means no error expected
	}{
		{"empty title", model.CreateInput{Title: "   "}, e.Validation},
		{"ok", model.CreateInput{Title: "hello", Body: "world"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &stubDB{}
			got, err := New(db, noopCache{}, nopEvents{}).Create(context.Background(), tt.in)
			if tt.wantErr != "" {
				if codeOf(err) != tt.wantErr {
					t.Fatalf("want code %q, got err %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID == "" {
				t.Error("expected a generated ID")
			}
			if got.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}
			if db.inserted.Title != tt.in.Title {
				t.Errorf("inserted title = %q, want %q", db.inserted.Title, tt.in.Title)
			}
		})
	}
}

func TestCreatePropagatesDBError(t *testing.T) {
	dbErr := errors.New("boom")
	_, err := New(&stubDB{insertErr: dbErr}, noopCache{}, nopEvents{}).Create(context.Background(), model.CreateInput{Title: "x"})
	if !errors.Is(err, dbErr) {
		t.Fatalf("want wrapped db error, got %v", err)
	}
}

// stubEvents records the last published article.
type stubEvents struct {
	calls int
	last  model.Article
}

func (s *stubEvents) ArticleCreated(_ context.Context, n model.Article) error {
	s.calls++
	s.last = n
	return nil
}

func TestCreatePublishesEvent(t *testing.T) {
	ev := &stubEvents{}
	got, err := New(&stubDB{}, noopCache{}, ev).Create(context.Background(), model.CreateInput{Title: "hello"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ev.calls != 1 {
		t.Fatalf("want 1 event published, got %d", ev.calls)
	}
	if ev.last.ID != got.ID || ev.last.Title != "hello" {
		t.Errorf("published article = %+v, want id=%s title=hello", ev.last, got.ID)
	}
}

// noopCache is a cache that misses on every read and drops every write, so the
// create tests exercise the use case rather than the cache.
type noopCache struct{}

func (noopCache) GetArticle(context.Context, string) (model.Article, bool, error) {
	return model.Article{}, false, nil
}
func (noopCache) SetArticle(context.Context, model.Article) error { return nil }
func (noopCache) GetList(context.Context) ([]model.Article, bool, error) {
	return nil, false, nil
}
func (noopCache) SetList(context.Context, []model.Article) error { return nil }
func (noopCache) InvalidateList(context.Context) error           { return nil }
