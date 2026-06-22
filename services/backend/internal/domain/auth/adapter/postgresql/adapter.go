package postgresql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sunkek/mishap"
	pgcmp "github.com/sunkek/samsara-components/postgresql"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// uniqueViolation is the Postgres SQLSTATE for a unique-constraint conflict.
const uniqueViolation = "23505"

type Adapter struct {
	pg *pgcmp.Component
}

func New(pg *pgcmp.Component) *Adapter {
	return &Adapter{pg: pg}
}

func (a *Adapter) InsertUser(ctx context.Context, u model.User) (model.User, error) {
	const q = `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, password_hash, created_at, updated_at`
	var out model.User
	err := a.pg.Get(ctx, &out, q, u.ID, u.Email, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return model.User{}, mishap.New("email already registered", e.Conflict)
		}
		return model.User{}, mishap.Wrap(err, "insert user")
	}
	return out, nil
}

func (a *Adapter) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	const q = `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
	return a.getOne(ctx, q, email)
}

func (a *Adapter) GetUserByID(ctx context.Context, id string) (model.User, error) {
	const q = `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
	return a.getOne(ctx, q, id)
}

func (a *Adapter) getOne(ctx context.Context, q string, arg any) (model.User, error) {
	var out model.User
	if err := a.pg.Get(ctx, &out, q, arg); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, mishap.New("user not found", e.NotFound)
		}
		return model.User{}, mishap.Wrap(err, "get user")
	}
	return out, nil
}
