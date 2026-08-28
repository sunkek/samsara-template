package e

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sunkek/mishap"
)

// Every code in this package must map to a status deliberately. A code with no
// case falls through to 500, which is a silent way to turn a client error into
// a server error — so the map is asserted exhaustively, by hand.
func TestHTTPStatusMapsEveryCode(t *testing.T) {
	want := map[mishap.Code]int{
		NotFound:   404,
		Conflict:   409,
		Validation: 400,
		Forbidden:  403,
		JWT:        401,
		RateLimit:  429,
		Internal:   500,
	}
	for code, status := range want {
		t.Run(string(code), func(t *testing.T) {
			if got := HTTPStatus(mishap.New("boom", code)); got != status {
				t.Errorf("HTTPStatus(%s) = %d, want %d", code, got, status)
			}
		})
	}
}

func TestHTTPStatusEdges(t *testing.T) {
	t.Run("nil is a success", func(t *testing.T) {
		if got := HTTPStatus(nil); got != 200 {
			t.Errorf("HTTPStatus(nil) = %d, want 200", got)
		}
	})

	t.Run("a plain error is a server error", func(t *testing.T) {
		if got := HTTPStatus(errors.New("something broke")); got != 500 {
			t.Errorf("HTTPStatus(plain) = %d, want 500", got)
		}
	})

	t.Run("an unknown code is a server error", func(t *testing.T) {
		if got := HTTPStatus(mishap.New("boom", mishap.Code("ERR_NOT_A_REAL_CODE"))); got != 500 {
			t.Errorf("HTTPStatus(unknown code) = %d, want 500", got)
		}
	})

	t.Run("the code survives wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", mishap.Wrap(mishap.New("gone", NotFound), "inner"))
		if got := HTTPStatus(wrapped); got != 404 {
			t.Errorf("HTTPStatus(wrapped NotFound) = %d, want 404 — wrapping must not lose the code", got)
		}
	})
}
