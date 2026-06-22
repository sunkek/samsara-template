package note

import (
	"context"
	"testing"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// countingDB records how often each port method is called so the tests can
// assert the cache short-circuits the database.
type countingDB struct {
	getCalls, listCalls int
	note                model.Note
	list                []model.Note
}

func (c *countingDB) Insert(_ context.Context, n model.Note) (model.Note, error) { return n, nil }
func (c *countingDB) Get(context.Context, string) (model.Note, error) {
	c.getCalls++
	return c.note, nil
}
func (c *countingDB) List(context.Context) ([]model.Note, error) {
	c.listCalls++
	return c.list, nil
}

// stubCache is a controllable Cache: it returns configured hits and counts
// writes/invalidations.
type stubCache struct {
	noteHit bool
	note    model.Note
	listHit bool
	list    []model.Note

	setNote    int
	setList    int
	invalidate int
}

func (s *stubCache) GetNote(context.Context, string) (model.Note, bool, error) {
	return s.note, s.noteHit, nil
}
func (s *stubCache) SetNote(context.Context, model.Note) error { s.setNote++; return nil }
func (s *stubCache) GetList(context.Context) ([]model.Note, bool, error) {
	return s.list, s.listHit, nil
}
func (s *stubCache) SetList(context.Context, []model.Note) error { s.setList++; return nil }
func (s *stubCache) InvalidateList(context.Context) error        { s.invalidate++; return nil }

func TestGetCacheHitSkipsDB(t *testing.T) {
	db := &countingDB{}
	cache := &stubCache{noteHit: true, note: model.Note{ID: "n1", Title: "cached"}}

	got, err := New(db, cache, NoopEvents{}).Get(context.Background(), "n1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "cached" {
		t.Errorf("title = %q, want cached value", got.Title)
	}
	if db.getCalls != 0 {
		t.Errorf("DB queried on cache hit: getCalls = %d", db.getCalls)
	}
}

func TestGetCacheMissReadsDBAndPopulates(t *testing.T) {
	db := &countingDB{note: model.Note{ID: "n1", Title: "fromdb"}}
	cache := &stubCache{} // miss

	got, err := New(db, cache, NoopEvents{}).Get(context.Background(), "n1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "fromdb" {
		t.Errorf("title = %q, want DB value", got.Title)
	}
	if db.getCalls != 1 {
		t.Errorf("want 1 DB get, got %d", db.getCalls)
	}
	if cache.setNote != 1 {
		t.Errorf("want cache populated on miss, setNote = %d", cache.setNote)
	}
}

func TestListCacheMissReadsDBAndPopulates(t *testing.T) {
	db := &countingDB{list: []model.Note{{ID: "n1"}}}
	cache := &stubCache{}

	got, err := New(db, cache, NoopEvents{}).List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("len = %d, want 1", len(got))
	}
	if db.listCalls != 1 || cache.setList != 1 {
		t.Errorf("listCalls = %d, setList = %d; want 1 and 1", db.listCalls, cache.setList)
	}
}

func TestCreateWarmsItemAndInvalidatesList(t *testing.T) {
	db := &countingDB{}
	cache := &stubCache{}

	if _, err := New(db, cache, NoopEvents{}).Create(context.Background(), model.CreateInput{Title: "x"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if cache.setNote != 1 {
		t.Errorf("want item warmed, setNote = %d", cache.setNote)
	}
	if cache.invalidate != 1 {
		t.Errorf("want list invalidated, invalidate = %d", cache.invalidate)
	}
}
