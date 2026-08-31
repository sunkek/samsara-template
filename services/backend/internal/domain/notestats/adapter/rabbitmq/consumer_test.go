package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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

// stubFn is a Service whose apply step the test controls.
type stubFn struct {
	applyFn func(context.Context) error
}

func (s *stubFn) ApplyNoteCreated(ctx context.Context, _ notemodel.NoteCreatedEvent) error {
	return s.applyFn(ctx)
}
func (s *stubFn) Get(context.Context) (statsmodel.Stats, error) { return statsmodel.Stats{}, nil }

func validDelivery(t *testing.T) amqp.Delivery {
	t.Helper()
	body, err := json.Marshal(notemodel.NoteCreatedEvent{NoteID: "n1", Title: "hello"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return amqp.Delivery{Body: body}
}

// A projection write must not run unbounded: a hung database would otherwise
// hold this consumer for as long as the connection survives.
func TestHandleBoundsTheProjectionWrite(t *testing.T) {
	var gotDeadline bool
	svc := &stubFn{applyFn: func(ctx context.Context) error {
		_, gotDeadline = ctx.Deadline()
		return nil
	}}

	if err := NewConsumer(svc).Handle(validDelivery(t)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !gotDeadline {
		t.Error("the projection write got a context with no deadline")
	}
}

// A failing write is requeued, but not instantly: an immediate nack on a
// persistent failure spins as fast as the broker can redeliver.
func TestHandleBacksOffBeforeRequeue(t *testing.T) {
	svc := &stubFn{applyFn: func(context.Context) error { return errors.New("db down") }}

	start := time.Now()
	err := NewConsumer(svc).Handle(validDelivery(t))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("want an error so the delivery is requeued")
	}
	if elapsed < requeueBackoff {
		t.Errorf("returned after %v, want at least the %v backoff", elapsed, requeueBackoff)
	}
}
