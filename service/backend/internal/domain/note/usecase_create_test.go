package note

import (
	"context"
	"errors"
	"testing"

	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// stubDB is an in-memory note.DB. The Service/DB ports exist precisely so the
// use cases can be tested without a real database — this is the pattern to copy
// for your own domains.
type stubDB struct {
	inserted  model.Note
	insertErr error
}

func (s *stubDB) Insert(_ context.Context, n model.Note) (model.Note, error) {
	if s.insertErr != nil {
		return model.Note{}, s.insertErr
	}
	s.inserted = n
	return n, nil
}
func (s *stubDB) List(context.Context) ([]model.Note, error)      { return nil, nil }
func (s *stubDB) Get(context.Context, string) (model.Note, error) { return model.Note{}, nil }

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
			got, err := New(db, NoopCache{}, NoopEvents{}).Create(context.Background(), tt.in)
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
	_, err := New(&stubDB{insertErr: dbErr}, NoopCache{}, NoopEvents{}).Create(context.Background(), model.CreateInput{Title: "x"})
	if !errors.Is(err, dbErr) {
		t.Fatalf("want wrapped db error, got %v", err)
	}
}

// stubEvents records the last published note.
type stubEvents struct {
	calls int
	last  model.Note
}

func (s *stubEvents) NoteCreated(_ context.Context, n model.Note) error {
	s.calls++
	s.last = n
	return nil
}

func TestCreatePublishesEvent(t *testing.T) {
	ev := &stubEvents{}
	got, err := New(&stubDB{}, NoopCache{}, ev).Create(context.Background(), model.CreateInput{Title: "hello"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ev.calls != 1 {
		t.Fatalf("want 1 event published, got %d", ev.calls)
	}
	if ev.last.ID != got.ID || ev.last.Title != "hello" {
		t.Errorf("published note = %+v, want id=%s title=hello", ev.last, got.ID)
	}
}
