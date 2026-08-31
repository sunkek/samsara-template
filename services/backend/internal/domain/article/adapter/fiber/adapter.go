package fiber

import (
	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"

	fibercmp "github.com/sunkek/samsara-components/fiber"
	"github.com/sunkek/samsara-template/backend/internal/common/e"

	"github.com/sunkek/samsara-template/backend/internal/domain/article"
	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// Adapter exposes the article domain over HTTP. It depends on the inbound port
// (article.Service), so routes are registered with a live handler immediately —
// no two-phase injection, no nil checks.
type Adapter struct {
	svc article.Service
}

func New(f *fibercmp.Component, svc article.Service) *Adapter {
	a := &Adapter{svc: svc}
	f.Register(a.routes)
	return a
}

// routes is the adapter's route table. It is a method rather than a closure
// inside New so tests can mount the real routes on a bare router — the
// component only applies registered funcs when it starts and binds a port.
func (a *Adapter) routes(r gf.Router) {
	g := r.Group("/articles")
	g.Post("/", a.handleCreate)
	g.Get("/", a.handleList)
	g.Get("/:id", a.handleGet)
}

type createReq struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// handleCreate godoc
//
//	@Summary	Create an article
//	@Tags		articles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		createReq	true	"article"
//	@Success	201		{object}	model.Article
//	@Router		/articles [post]
func (a *Adapter) handleCreate(ctx gf.Ctx) error {
	var req createReq
	if err := ctx.Bind().Body(&req); err != nil {
		return mishap.Wrap(err, "bind body", mishap.WithCode(e.Validation))
	}
	n, err := a.svc.Create(ctx.Context(), model.CreateInput{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		return err
	}
	return ctx.Status(gf.StatusCreated).JSON(n)
}

// handleList godoc
//
//	@Summary	List articles
//	@Tags		articles
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}	model.Article
//	@Router		/articles [get]
func (a *Adapter) handleList(ctx gf.Ctx) error {
	articles, err := a.svc.List(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(articles)
}

// handleGet godoc
//
//	@Summary	Get an article by id
//	@Tags		articles
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"article id"
//	@Success	200	{object}	model.Article
//	@Router		/articles/{id} [get]
func (a *Adapter) handleGet(ctx gf.Ctx) error {
	n, err := a.svc.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return err
	}
	return ctx.JSON(n)
}
