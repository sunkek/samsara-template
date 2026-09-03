// Package redis implements the article domain's Cache outbound port on top of
// the samsara Redis component. Values are JSON-encoded. A miss reports
// found=false with no error; a Redis or decode failure reports the error, which
// the domain logs and treats as a miss. Cache hit/miss metrics are recorded
// here rather than in the domain: whether a lookup hit is a fact about this
// cache.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	rediscmp "github.com/sunkek/samsara-components/redis"

	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

const (
	articleKeyPrefix = "article:cache:item:"
	listKey          = "article:cache:list"
)

type Adapter struct {
	rdb rediscmp.KV
	ttl time.Duration
}

// New builds the cache adapter. ttl is the entry lifetime (0 = no expiry).
func New(rdb rediscmp.KV, ttl time.Duration) *Adapter {
	return &Adapter{rdb: rdb, ttl: ttl}
}

func (a *Adapter) GetArticle(ctx context.Context, id string) (model.Article, bool, error) {
	var n model.Article
	found, err := a.getJSON(ctx, articleKeyPrefix+id, &n)
	recordLookup(found)
	if err != nil {
		return model.Article{}, false, err
	}
	return n, found, nil
}

func (a *Adapter) SetArticle(ctx context.Context, n model.Article) error {
	return a.setJSON(ctx, articleKeyPrefix+n.ID, n)
}

func (a *Adapter) GetList(ctx context.Context) ([]model.Article, bool, error) {
	var articles []model.Article
	found, err := a.getJSON(ctx, listKey, &articles)
	recordLookup(found)
	if err != nil {
		return nil, false, err
	}
	return articles, found, nil
}

func (a *Adapter) SetList(ctx context.Context, articles []model.Article) error {
	return a.setJSON(ctx, listKey, articles)
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
// lookup hit is a fact about this cache, not about the use case that asked.
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
