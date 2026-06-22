package auth

import "time"

// Domain holds the auth use cases. It depends on the DB outbound port, a
// Revoker for the refresh-token denylist, and an internal token manager for
// JWT signing/parsing.
type Domain struct {
	db      DB
	revoker Revoker
	tok     tokenManager
}

// New builds the auth domain. revoker backs logout and refresh-token rotation
// and is required — wire the Redis adapter in production (a stub in tests) so
// revocation is always explicit and never silently a no-op. secret signs and
// verifies all tokens; it must be non-empty (config marks it required).
// accessTTL/refreshTTL set token lifetimes.
func New(db DB, revoker Revoker, secret string, accessTTL, refreshTTL time.Duration) *Domain {
	return &Domain{
		db:      db,
		revoker: revoker,
		tok: tokenManager{
			secret:     []byte(secret),
			accessTTL:  accessTTL,
			refreshTTL: refreshTTL,
		},
	}
}

var _ Service = (*Domain)(nil)
