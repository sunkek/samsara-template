package fiber

import (
	gf "github.com/gofiber/fiber/v3"
	fibercmp "github.com/sunkek/samsara-components/fiber"

	"github.com/sunkek/samsara-template/backend/internal/domain/notestats"
)

// Adapter exposes the note read-model (projection) over HTTP.
type Adapter struct {
	svc notestats.Service
}

func New(f *fibercmp.Component, svc notestats.Service) *Adapter {
	a := &Adapter{svc: svc}
	f.Register(func(r gf.Router) {
		r.Get("/stats", a.handleGet)
	})
	return a
}

// handleGet godoc
//
//	@Summary	Get note statistics (event-projected read model)
//	@Tags		stats
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	model.Stats
//	@Router		/stats [get]
func (a *Adapter) handleGet(ctx gf.Ctx) error {
	s, err := a.svc.Get(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(s)
}
