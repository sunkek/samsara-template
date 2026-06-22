package fiber

import (
	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"
	fibercmp "github.com/sunkek/samsara-components/fiber"

	"github.com/sunkek/samsara-template/backend/internal/domain/auth"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// Adapter exposes the auth domain over HTTP and provides the auth middleware.
type Adapter struct {
	svc auth.Service
}

// New registers the auth routes. Any middlewares passed in mw are applied to
// the /auth group only (e.g. a rate limiter on the credential endpoints).
func New(f *fibercmp.Component, svc auth.Service, mw ...gf.Handler) *Adapter {
	a := &Adapter{svc: svc}
	f.Register(func(r gf.Router) {
		g := r.Group("/auth")
		for _, m := range mw {
			g.Use(m)
		}
		g.Post("/register", a.handleRegister)
		g.Post("/login", a.handleLogin)
		g.Post("/refresh", a.handleRefresh)
		g.Post("/logout", a.handleLogout)
	})
	return a
}

type credentialsReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// handleRegister godoc
//
//	@Summary	Register a new user
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		credentialsReq	true	"credentials"
//	@Success	201		{object}	model.User
//	@Router		/auth/register [post]
func (a *Adapter) handleRegister(ctx gf.Ctx) error {
	var req credentialsReq
	if err := ctx.Bind().Body(&req); err != nil {
		return mishap.Wrap(err, "bind body")
	}
	u, err := a.svc.Register(ctx.Context(), model.RegisterInput{Email: req.Email, Password: req.Password})
	if err != nil {
		return err
	}
	return ctx.Status(gf.StatusCreated).JSON(u)
}

// handleLogin godoc
//
//	@Summary	Log in and obtain tokens
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		credentialsReq	true	"credentials"
//	@Success	200		{object}	model.Tokens
//	@Router		/auth/login [post]
func (a *Adapter) handleLogin(ctx gf.Ctx) error {
	var req credentialsReq
	if err := ctx.Bind().Body(&req); err != nil {
		return mishap.Wrap(err, "bind body")
	}
	tokens, err := a.svc.Login(ctx.Context(), model.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		return err
	}
	return ctx.JSON(tokens)
}

// handleRefresh godoc
//
//	@Summary	Exchange a refresh token for a new token pair
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		refreshReq	true	"refresh token"
//	@Success	200		{object}	model.Tokens
//	@Router		/auth/refresh [post]
func (a *Adapter) handleRefresh(ctx gf.Ctx) error {
	var req refreshReq
	if err := ctx.Bind().Body(&req); err != nil {
		return mishap.Wrap(err, "bind body")
	}
	tokens, err := a.svc.Refresh(ctx.Context(), req.RefreshToken)
	if err != nil {
		return err
	}
	return ctx.JSON(tokens)
}

// handleLogout godoc
//
//	@Summary	Revoke a refresh token
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body	refreshReq	true	"refresh token"
//	@Success	204		"no content"
//	@Router		/auth/logout [post]
func (a *Adapter) handleLogout(ctx gf.Ctx) error {
	var req refreshReq
	if err := ctx.Bind().Body(&req); err != nil {
		return mishap.Wrap(err, "bind body")
	}
	if err := a.svc.Logout(ctx.Context(), req.RefreshToken); err != nil {
		return err
	}
	return ctx.SendStatus(gf.StatusNoContent)
}
