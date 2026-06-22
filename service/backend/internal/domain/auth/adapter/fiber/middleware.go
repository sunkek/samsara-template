package fiber

import (
	"strings"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/common/logging"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// localsKey is an unexported type so the claims entry in fiber Locals cannot
// collide with keys set by other packages.
type localsKey struct{}

// claimsKey is the single instance used to store/read claims.
var claimsKey = localsKey{}

// Middleware returns a global middleware that requires a valid Bearer access
// token on every request whose path does not start with one of publicPrefixes.
// On success the verified claims are stored in the request Locals; read them
// with ClaimsFromContext. On failure it returns an e.JWT error, which the
// central error handler maps to 401.
//
// Pass the fully-prefixed public paths, e.g. cfg.Fiber.PathPrefix+"/auth".
//
// This deliberately does its own prefix check rather than samsara-components/
// fiber's ExcludeRoutes helper: that helper's match sense is inverted relative
// to its own doc comment, so wiring it here would protect exactly the routes it
// is meant to leave public. Keep the inline skipper.
func (a *Adapter) Middleware(publicPrefixes ...string) gf.Handler {
	return func(ctx gf.Ctx) error {
		path := ctx.Path()
		for _, p := range publicPrefixes {
			// Match on a path boundary so a public prefix like ".../auth"
			// cannot accidentally unguard a sibling route like ".../authz".
			if path == p || strings.HasPrefix(path, p+"/") {
				return ctx.Next()
			}
		}

		raw := ctx.Get("Authorization")
		token, ok := strings.CutPrefix(raw, "Bearer ")
		if !ok || token == "" {
			return mishap.New("missing or malformed Authorization header", e.JWT)
		}

		claims, err := a.svc.Verify(ctx.Context(), token)
		if err != nil {
			return err
		}

		ctx.Locals(claimsKey, claims)
		// Add user_id to the request-scoped logger so authenticated requests are
		// correlated by both request_id and user_id.
		ctx.SetContext(logging.Into(ctx.Context(), logging.From(ctx.Context()).With("user_id", claims.UserID)))
		return ctx.Next()
	}
}

// ClaimsFromContext returns the authenticated user's claims set by Middleware.
// ok is false on unauthenticated (public) routes.
func ClaimsFromContext(ctx gf.Ctx) (model.Claims, bool) {
	claims, ok := ctx.Locals(claimsKey).(model.Claims)
	return claims, ok
}
