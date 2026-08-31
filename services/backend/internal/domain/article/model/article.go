package model

import "time"

// Article is the sample entity for the optional-infrastructure domain: the same
// shape as a Note, cached on read and announced on write.
type Article struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInput is the parameter object for creating a Article. Lives in model so
// both the domain layer and the REST adapter can reference it without forming
// an import cycle.
type CreateInput struct {
	Title string
	Body  string
}

// ArticleCreatedEvent is the payload published when an article is created. It
// is the shared contract between the RabbitMQ publisher and the consumer
// worker, so it lives in model where both can import it.
type ArticleCreatedEvent struct {
	ArticleID string    `json:"article_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
