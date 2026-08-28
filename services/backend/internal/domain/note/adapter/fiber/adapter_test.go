package fiber

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// stubService is a note.Service returning whatever the test configures.
type stubService struct {
	note    model.Note
	list    []model.Note
	err     error
	gotID   string
	gotIn   model.CreateInput
	creates int
}

func (s *stubService) Create(_ context.Context, in model.CreateInput) (model.Note, error) {
	s.creates++
	s.gotIn = in
	return s.note, s.err
}
func (s *stubService) List(context.Context) ([]model.Note, error) { return s.list, s.err }
func (s *stubService) Get(_ context.Context, id string) (model.Note, error) {
	s.gotID = id
	return s.note, s.err
}

// appWithRoutes mounts the adapter's real route table, so a wrong verb or path
// in the adapter fails these tests.
func appWithRoutes(t *testing.T, svc *stubService) *gf.App {
	t.Helper()
	app := gf.New(gf.Config{
		ErrorHandler: func(c gf.Ctx, err error) error { return c.SendStatus(e.HTTPStatus(err)) },
	})
	(&Adapter{svc: svc}).routes(app)
	return app
}

func send(t *testing.T, app *gf.App, method, path, body string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// A create returns 201 with the created note, not 200 — clients rely on the
// status to tell a create from a fetch.
func TestCreateReturns201AndTheNote(t *testing.T) {
	svc := &stubService{note: model.Note{ID: "n-1", Title: "written"}}
	resp := send(t, appWithRoutes(t, svc), http.MethodPost, "/notes", `{"title":"written","body":"b"}`)

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if svc.gotIn.Title != "written" || svc.gotIn.Body != "b" {
		t.Errorf("service got %+v, want the request body decoded into CreateInput", svc.gotIn)
	}

	var got model.Note
	decode(t, resp, &got)
	if got.ID != "n-1" {
		t.Errorf("body id = %q, want the created note", got.ID)
	}
}

// The error code a use case returns decides the status. A validation failure
// must not surface as a 500.
func TestUseCaseErrorCodeDecidesTheStatus(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		method string
		path   string
		want   int
	}{
		{"validation on create", mishap.New("title is required", e.Validation), http.MethodPost, "/notes", 400},
		{"missing note on get", mishap.New("no such note", e.NotFound), http.MethodGet, "/notes/n-1", 404},
		{"unmapped failure on list", mishap.New("db down", e.Internal), http.MethodGet, "/notes", 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := appWithRoutes(t, &stubService{err: tc.err})
			body := ""
			if tc.method == http.MethodPost {
				body = `{"title":""}`
			}
			if got := send(t, app, tc.method, tc.path, body).StatusCode; got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

// The id in the path must reach the use case, and a list must come back as a
// JSON array.
func TestReadsReachTheServiceAndSerialise(t *testing.T) {
	t.Run("get passes the path id through", func(t *testing.T) {
		svc := &stubService{note: model.Note{ID: "n-42"}}
		resp := send(t, appWithRoutes(t, svc), http.MethodGet, "/notes/n-42", "")

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if svc.gotID != "n-42" {
			t.Errorf("service got id %q, want n-42", svc.gotID)
		}
	})

	t.Run("list serialises as an array", func(t *testing.T) {
		svc := &stubService{list: []model.Note{{ID: "a"}, {ID: "b"}}}
		resp := send(t, appWithRoutes(t, svc), http.MethodGet, "/notes", "")

		var got []model.Note
		decode(t, resp, &got)
		if len(got) != 2 {
			t.Errorf("got %d notes, want 2", len(got))
		}
	})
}

// A malformed body is the client's fault, not the server's.
func TestMalformedBodyIsARequestError(t *testing.T) {
	svc := &stubService{}
	resp := send(t, appWithRoutes(t, svc), http.MethodPost, "/notes", `{"title":`)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		t.Fatalf("status = %d, want a failure for an undecodable body", resp.StatusCode)
	}
	if svc.creates != 0 {
		t.Errorf("Create called %d times, want 0 — a body that will not decode must not reach the use case", svc.creates)
	}
}

func decode(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}
