package article

import (
	"context"
	"errors"
	"testing"

	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// countingDB records how often each port method is called so the tests can
// assert the cache short-circuits the database.
type countingDB struct {
	getCalls, listCalls int
	article             model.Article
	list                []model.Article
}

func (c *countingDB) Insert(_ context.Context, n model.Article) (model.Article, error) { return n, nil }
func (c *countingDB) Get(context.Context, string) (model.Article, error) {
	c.getCalls++
	return c.article, nil
}
func (c *countingDB) List(context.Context) ([]model.Article, error) {
	c.listCalls++
	return c.list, nil
}

// stubCache is a controllable Cache: it returns configured hits and counts
// writes/invalidations.
type stubCache struct {
	articleHit bool
	article    model.Article
	listHit    bool
	list       []model.Article
	getErr     error

	setArticle int
	setList    int
	invalidate int
}

func (s *stubCache) GetArticle(context.Context, string) (model.Article, bool, error) {
	return s.article, s.articleHit, s.getErr
}
func (s *stubCache) SetArticle(context.Context, model.Article) error { s.setArticle++; return nil }
func (s *stubCache) GetList(context.Context) ([]model.Article, bool, error) {
	return s.list, s.listHit, s.getErr
}
func (s *stubCache) SetList(context.Context, []model.Article) error { s.setList++; return nil }
func (s *stubCache) InvalidateList(context.Context) error           { s.invalidate++; return nil }

func TestGetCacheHitSkipsDB(t *testing.T) {
	db := &countingDB{}
	cache := &stubCache{articleHit: true, article: model.Article{ID: "n1", Title: "cached"}}

	got, err := New(db, cache, nopEvents{}).Get(context.Background(), "n1")
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
	db := &countingDB{article: model.Article{ID: "n1", Title: "fromdb"}}
	cache := &stubCache{} // miss

	got, err := New(db, cache, nopEvents{}).Get(context.Background(), "n1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "fromdb" {
		t.Errorf("title = %q, want DB value", got.Title)
	}
	if db.getCalls != 1 {
		t.Errorf("want 1 DB get, got %d", db.getCalls)
	}
	if cache.setArticle != 1 {
		t.Errorf("want cache populated on miss, setArticle = %d", cache.setArticle)
	}
}

func TestListCacheMissReadsDBAndPopulates(t *testing.T) {
	db := &countingDB{list: []model.Article{{ID: "n1"}}}
	cache := &stubCache{}

	got, err := New(db, cache, nopEvents{}).List(context.Background())
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

	if _, err := New(db, cache, nopEvents{}).Create(context.Background(), model.CreateInput{Title: "x"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if cache.setArticle != 1 {
		t.Errorf("want item warmed, setArticle = %d", cache.setArticle)
	}
	if cache.invalidate != 1 {
		t.Errorf("want list invalidated, invalidate = %d", cache.invalidate)
	}
}

// A cache that is down must not fail the request: reads fall through to the
// database, which is the whole point of the port being best-effort.
func TestCacheReadErrorFallsBackToDB(t *testing.T) {
	down := errors.New("redis unreachable")

	t.Run("get", func(t *testing.T) {
		db := &countingDB{article: model.Article{ID: "n1", Title: "from db"}}
		d := New(db, &stubCache{getErr: down}, nopEvents{})

		got, err := d.Get(context.Background(), "n1")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Title != "from db" {
			t.Errorf("title = %q, want the database value", got.Title)
		}
		if db.getCalls != 1 {
			t.Errorf("db.Get called %d times, want 1", db.getCalls)
		}
	})

	t.Run("list", func(t *testing.T) {
		db := &countingDB{list: []model.Article{{ID: "n1"}}}
		d := New(db, &stubCache{getErr: down}, nopEvents{})

		got, err := d.List(context.Background())
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d articles, want 1 from the database", len(got))
		}
		if db.listCalls != 1 {
			t.Errorf("db.List called %d times, want 1", db.listCalls)
		}
	})
}

// nopEvents is a publisher that drops everything: these tests are about the
// cache, and a publish failure must never be what makes one of them fail.
type nopEvents struct{}

func (nopEvents) ArticleCreated(context.Context, model.Article) error { return nil }
