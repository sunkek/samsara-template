// Package redis implements the note domain's Cache outbound port on top of the
// samsara Redis component. Values are JSON-encoded; reads return found=false on
// a miss or any error so the domain falls back to the database.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	rediscmp "github.com/sunkek/samsara-components/redis"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

const (
	noteKeyPrefix = "note:cache:item:"
	listKey       = "note:cache:list"
)

type Adapter struct {
	rdb rediscmp.Client
	ttl time.Duration
}

// New builds the cache adapter. ttl is the entry lifetime (0 = no expiry).
func New(rdb rediscmp.Client, ttl time.Duration) *Adapter {
	return &Adapter{rdb: rdb, ttl: ttl}
}

func (a *Adapter) GetNote(ctx context.Context, id string) (model.Note, bool, error) {
	return a.getJSON(ctx, noteKeyPrefix+id, &model.Note{})
}

func (a *Adapter) SetNote(ctx context.Context, n model.Note) error {
	return a.setJSON(ctx, noteKeyPrefix+n.ID, n)
}

func (a *Adapter) GetList(ctx context.Context) ([]model.Note, bool, error) {
	var notes []model.Note
	s, err := a.rdb.Get(ctx, listKey)
	if err != nil {
		if errors.Is(err, rediscmp.ErrNil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if err := json.Unmarshal([]byte(s), &notes); err != nil {
		return nil, false, err
	}
	return notes, true, nil
}

func (a *Adapter) SetList(ctx context.Context, notes []model.Note) error {
	return a.setJSON(ctx, listKey, notes)
}

func (a *Adapter) InvalidateList(ctx context.Context) error {
	_, err := a.rdb.Del(ctx, listKey)
	return err
}

// getJSON fetches and decodes a single note. dst must be *model.Note.
func (a *Adapter) getJSON(ctx context.Context, key string, dst *model.Note) (model.Note, bool, error) {
	s, err := a.rdb.Get(ctx, key)
	if err != nil {
		if errors.Is(err, rediscmp.ErrNil) {
			return model.Note{}, false, nil
		}
		return model.Note{}, false, err
	}
	if err := json.Unmarshal([]byte(s), dst); err != nil {
		return model.Note{}, false, err
	}
	return *dst, true, nil
}

func (a *Adapter) setJSON(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return a.rdb.Set(ctx, key, b, a.ttl)
}
