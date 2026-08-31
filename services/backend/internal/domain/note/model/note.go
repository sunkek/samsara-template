package model

import "time"

// Note is the sample domain entity. Replace it with your own aggregate; it
// exists to demonstrate the domain → interface → adapter wiring end to end.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInput is the parameter object for creating a Note. Lives in model so
// both the domain layer and the REST adapter can reference it without forming
// an import cycle.
type CreateInput struct {
	Title string
	Body  string
}
