package rabbitmq

import (
	"context"
	"encoding/json"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	notemodel "github.com/sunkek/samsara-template/backend/internal/domain/note/model"
	statsmodel "github.com/sunkek/samsara-template/backend/internal/domain/notestats/model"
)

type stubSvc struct {
	applied int
	last    notemodel.NoteCreatedEvent
}

func (s *stubSvc) ApplyNoteCreated(_ context.Context, e notemodel.NoteCreatedEvent) error {
	s.applied++
	s.last = e
	return nil
}
func (s *stubSvc) Get(context.Context) (statsmodel.Stats, error) { return statsmodel.Stats{}, nil }

func TestHandleValidEvent(t *testing.T) {
	body, _ := json.Marshal(notemodel.NoteCreatedEvent{NoteID: "n1", Title: "hello"})
	svc := &stubSvc{}
	if err := NewConsumer(svc).Handle(amqp.Delivery{Body: body}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if svc.applied != 1 || svc.last.NoteID != "n1" {
		t.Errorf("apply count=%d last=%+v", svc.applied, svc.last)
	}
}

func TestHandlePoisonMessageDropped(t *testing.T) {
	svc := &stubSvc{}
	// Invalid JSON must be acked (nil) and not applied, so it does not requeue.
	if err := NewConsumer(svc).Handle(amqp.Delivery{Body: []byte("{not json")}); err != nil {
		t.Fatalf("poison message should be dropped (nil), got %v", err)
	}
	if svc.applied != 0 {
		t.Errorf("poison message must not be applied, applied=%d", svc.applied)
	}
}
