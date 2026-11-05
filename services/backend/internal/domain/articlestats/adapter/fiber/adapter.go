package fiber

import (
	gf "github.com/gofiber/fiber/v3"
	fibercmp "github.com/sunkek/samsara-components/fiber"

	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats/model"
)

// Adapter exposes the article read-model (projection) over HTTP.
type Adapter struct {
	svc articlestats.Service
}

func New(f *fibercmp.Component, svc articlestats.Service) *Adapter {
	a := &Adapter{svc: svc}
	f.Register(a.routes)
	return a
}

// routes is the adapter's route table, a method rather than a closure so tests
// can mount the real routes on a bare router (see the note adapter).
func (a *Adapter) routes(r gf.Router) {
	r.Get("/stats", a.handleGet)
}

// handleGet godoc
//
//	@Summary	Get article statistics (event-projected read model)
//	@Tags		stats
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	model.Stats
//	@Router		/stats [get]
func (a *Adapter) handleGet(ctx gf.Ctx) error {
	// Declared with its type so this file imports model: swag resolves the
	// model.Stats in the annotation above through the file's imports only, and
	// a module-path-qualified name there would break when a fork renames the
	// module.
	var s model.Stats
	s, err := a.svc.Get(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(s)
}
