package fiber

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats/model"
)

type stubService struct {
	stats model.Stats
	err   error
}

func (s *stubService) ApplyArticleCreated(context.Context, articlemodel.ArticleCreatedEvent) error {
	return s.err
}
func (s *stubService) Get(context.Context) (model.Stats, error) { return s.stats, s.err }

func app(t *testing.T, svc *stubService) *gf.App {
	t.Helper()
	a := gf.New(gf.Config{
		ErrorHandler: func(c gf.Ctx, err error) error { return c.SendStatus(e.HTTPStatus(err)) },
	})
	(&Adapter{svc: svc}).routes(a)
	return a
}

func get(t *testing.T, a *gf.App) *http.Response {
	t.Helper()
	resp, err := a.Test(httptest.NewRequest(http.MethodGet, "/stats", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestGetReturnsTheProjection(t *testing.T) {
	want := model.Stats{TotalCount: 3, LastArticleID: "a3", LastTitle: "third"}
	resp := get(t, app(t, &stubService{stats: want}))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got model.Stats
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got != want {
		t.Errorf("body = %+v, want %+v", got, want)
	}
}

// An empty projection is a 200 with zeroes, not a 404 — there is always exactly
// one read model, it just may have seen no events yet.
func TestGetEmptyProjectionIs200(t *testing.T) {
	resp := get(t, app(t, &stubService{}))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestGetErrorMapsToStatus(t *testing.T) {
	resp := get(t, app(t, &stubService{err: mishap.New("boom", e.Internal)}))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}
