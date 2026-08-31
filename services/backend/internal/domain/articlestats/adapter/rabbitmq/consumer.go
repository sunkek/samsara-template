// Package rabbitmq adapts incoming article.created deliveries to the
// articlestats projection service. It is the inbound transport for the read model.
package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats"
)

// handleTimeout bounds one projection write. A delivery carries no caller
// context, so without a deadline a hung database blocks this consumer for as
// long as the connection survives — a stall with no error and no signal.
const handleTimeout = 10 * time.Second

// requeueBackoff is the pause before an unprocessable delivery goes back to the
// broker. RabbitMQ redelivers a nacked message immediately, so a persistent
// failure (the database being down) otherwise becomes a hot loop: nack,
// redeliver, fail, nack, at whatever rate the broker can sustain. The pause
// costs nothing in the healthy path, which never reaches it.
const requeueBackoff = time.Second

// Consumer decodes article.created events and applies them to the projection.
type Consumer struct {
	svc articlestats.Service
}

func NewConsumer(svc articlestats.Service) *Consumer {
	return &Consumer{svc: svc}
}

// Handle is the message handler passed to the rabbitmq component's Subscribe.
// Returning nil acks the delivery; returning an error nacks it with requeue.
func (c *Consumer) Handle(d amqp.Delivery) error {
	var e articlemodel.ArticleCreatedEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		// Poison message: ack (drop) it rather than requeue forever. A
		// production system would route it to a dead-letter exchange instead.
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), handleTimeout)
	defer cancel()

	if err := c.svc.ApplyArticleCreated(ctx, e); err != nil {
		// Requeue for a later retry, but not instantly — see requeueBackoff.
		time.Sleep(requeueBackoff)
		return err
	}
	metrics.EventConsumed()
	return nil
}
