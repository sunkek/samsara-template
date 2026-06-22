// Package rabbitmq implements the note domain's Events outbound port by
// publishing JSON events to a RabbitMQ topic exchange via the samsara rabbitmq
// component.
package rabbitmq

import (
	"context"
	"encoding/json"

	rabbitcmp "github.com/sunkek/samsara-components/rabbitmq"

	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Publisher is the slice of the samsara rabbitmq component this adapter needs.
// *rabbitcmp.Component satisfies it; depending on the interface keeps the
// adapter unit-testable without a broker.
type Publisher interface {
	Publish(ctx context.Context, exchange, routingKey string, contentType rabbitcmp.ContentType, body []byte) error
}

type Adapter struct {
	pub            Publisher
	exchange       string
	noteCreatedKey string
}

// New builds the publisher adapter. The caller is responsible for declaring the
// exchange on the component (see cmd/main).
func New(pub Publisher, exchange, noteCreatedKey string) *Adapter {
	return &Adapter{pub: pub, exchange: exchange, noteCreatedKey: noteCreatedKey}
}

func (a *Adapter) NoteCreated(ctx context.Context, n model.Note) error {
	body, err := json.Marshal(model.NoteCreatedEvent{
		NoteID:    n.ID,
		Title:     n.Title,
		CreatedAt: n.CreatedAt,
	})
	if err != nil {
		return err
	}
	if err := a.pub.Publish(ctx, a.exchange, a.noteCreatedKey, rabbitcmp.ContentTypeJSON, body); err != nil {
		return err
	}
	metrics.EventPublished()
	return nil
}
