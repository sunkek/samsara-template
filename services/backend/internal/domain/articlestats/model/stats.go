package model

import "time"

// Stats is the article read-model: a projection maintained by consuming
// article.created events. It demonstrates a simple CQRS-style read model kept
// separate from the article write-model.
type Stats struct {
	TotalCount    int64     `json:"total_count"`
	LastArticleID string    `json:"last_article_id"`
	LastTitle     string    `json:"last_title"`
	UpdatedAt     time.Time `json:"updated_at"`
}
