package memory

import (
	"context"
	"testing"
	"time"
)

func TestRevokeThenIsRevoked(t *testing.T) {
	a := New()
	ctx := context.Background()

	got, err := a.IsRevoked(ctx, "jti-1")
	if err != nil || got {
		t.Fatalf("unrevoked token: got (%v, %v), want (false, nil)", got, err)
	}

	if err := a.Revoke(ctx, "jti-1", time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	got, err = a.IsRevoked(ctx, "jti-1")
	if err != nil || !got {
		t.Fatalf("revoked token: got (%v, %v), want (true, nil)", got, err)
	}
}

func TestEntryExpires(t *testing.T) {
	a := New()
	now := time.Now()
	a.now = func() time.Time { return now }
	ctx := context.Background()

	if err := a.Revoke(ctx, "jti-1", time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	now = now.Add(time.Minute)

	got, err := a.IsRevoked(ctx, "jti-1")
	if err != nil || got {
		t.Fatalf("expired entry: got (%v, %v), want (false, nil)", got, err)
	}
	if len(a.revoked) != 0 {
		t.Fatalf("expired entry not dropped: %d left", len(a.revoked))
	}
}
