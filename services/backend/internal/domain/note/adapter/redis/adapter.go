// Package redis implements the note domain's Cache outbound port on top of the
// samsara Redis component. Values are JSON-encoded. A miss reports
// found=false with no error; a Redis or decode failure reports the error, which
// the domain logs and treats as a miss. Cache hit/miss metrics are recorded
// here, so a build wired with NoopCache emits none.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	rediscmp "github.com/sunkek/samsara-components/redis"

	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
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
	var n model.Note
	found, err := a.getJSON(ctx, noteKeyPrefix+id, &n)
	recordLookup(found)
	if err != nil {
		return model.Note{}, false, err
	}
	return n, found, nil
}

func (a *Adapter) SetNote(ctx context.Context, n model.Note) error {
	return a.setJSON(ctx, noteKeyPrefix+n.ID, n)
}

func (a *Adapter) GetList(ctx context.Context) ([]model.Note, bool, error) {
	var notes []model.Note
	found, err := a.getJSON(ctx, listKey, &notes)
	recordLookup(found)
	if err != nil {
		return nil, false, err
	}
	return notes, found, nil
}

func (a *Adapter) SetList(ctx context.Context, notes []model.Note) error {
	return a.setJSON(ctx, listKey, notes)
}

func (a *Adapter) InvalidateList(ctx context.Context) error {
	_, err := a.rdb.Del(ctx, listKey)
	return err
}

// getJSON fetches key and decodes it into dst. A miss reports found=false with
// no error; a decode failure reports the error, which the domain treats as a
// miss and logs.
func (a *Adapter) getJSON(ctx context.Context, key string, dst any) (bool, error) {
	s, err := a.rdb.Get(ctx, key)
	if err != nil {
		if errors.Is(err, rediscmp.ErrNil) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(s), dst); err != nil {
		return false, err
	}
	return true, nil
}

// recordLookup counts hits and misses here rather than in the domain: whether a
// lookup hit is a fact about this cache, and a build wired with NoopCache has
// no cache to report on, so it should emit nothing at all.
func recordLookup(found bool) {
	if found {
		metrics.CacheHit()
		return
	}
	metrics.CacheMiss()
}

func (a *Adapter) setJSON(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return a.rdb.Set(ctx, key, b, a.ttl)
}
