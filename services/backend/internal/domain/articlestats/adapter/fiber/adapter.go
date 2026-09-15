package fiber

import (
	gf "github.com/gofiber/fiber/v3"
	fibercmp "github.com/sunkek/samsara-components/fiber"

	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats"
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
//	@Success	200	{object}	github_com_sunkek_samsara-template_backend_internal_domain_articlestats_model.Stats
//	@Router		/stats [get]
func (a *Adapter) handleGet(ctx gf.Ctx) error {
	s, err := a.svc.Get(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(s)
}
