package fiber

import (
	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"
	fibercmp "github.com/sunkek/samsara-components/fiber"

	"github.com/sunkek/samsara-template/backend/internal/domain/note"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Adapter exposes the note domain over HTTP. It depends on the inbound port
// (note.Service), so routes are registered with a live handler immediately —
// no two-phase injection, no nil checks.
type Adapter struct {
	svc note.Service
}

func New(f *fibercmp.Component, svc note.Service) *Adapter {
	a := &Adapter{svc: svc}
	f.Register(func(r gf.Router) {
		g := r.Group("/notes")
		g.Post("/", a.handleCreate)
		g.Get("/", a.handleList)
		g.Get("/:id", a.handleGet)
	})
	return a
}

type createReq struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// handleCreate godoc
//
//	@Summary	Create a note
//	@Tags		notes
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		createReq	true	"note"
//	@Success	201		{object}	model.Note
//	@Router		/notes [post]
func (a *Adapter) handleCreate(ctx gf.Ctx) error {
	var req createReq
	if err := ctx.Bind().Body(&req); err != nil {
		return mishap.Wrap(err, "bind body")
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
//	@Summary	List notes
//	@Tags		notes
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}	model.Note
//	@Router		/notes [get]
func (a *Adapter) handleList(ctx gf.Ctx) error {
	notes, err := a.svc.List(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(notes)
}

// handleGet godoc
//
//	@Summary	Get a note by id
//	@Tags		notes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"note id"
//	@Success	200	{object}	model.Note
//	@Router		/notes/{id} [get]
func (a *Adapter) handleGet(ctx gf.Ctx) error {
	n, err := a.svc.Get(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return err
	}
	return ctx.JSON(n)
}
