// Package rabbitmq adapts incoming note.created deliveries to the notestats
// projection service. It is the inbound transport for the read model.
package rabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	notemodel "github.com/sunkek/samsara-template/backend/internal/domain/note/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/notestats"
)

// Consumer decodes note.created events and applies them to the projection.
type Consumer struct {
	svc notestats.Service
}

func NewConsumer(svc notestats.Service) *Consumer {
	return &Consumer{svc: svc}
}

// Handle is the message handler passed to the rabbitmq component's Subscribe.
// Returning nil acks the delivery; returning an error nacks it with requeue.
func (c *Consumer) Handle(d amqp.Delivery) error {
	var e notemodel.NoteCreatedEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		// Poison message: ack (drop) it rather than requeue forever. A
		// production system would route it to a dead-letter exchange instead.
		return nil
	}
	// The delivery carries no caller context; use Background. Processing errors
	// (e.g. DB down) are returned so the broker requeues for a later retry.
	if err := c.svc.ApplyNoteCreated(context.Background(), e); err != nil {
		return err
	}
	metrics.EventConsumed()
	return nil
}
