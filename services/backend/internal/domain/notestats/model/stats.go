package model

import "time"

// Stats is the note read-model: a projection maintained by consuming
// note.created events. It demonstrates a simple CQRS-style read model kept
// separate from the note write-model.
type Stats struct {
	TotalCount int64     `json:"total_count"`
	LastNoteID string    `json:"last_note_id"`
	LastTitle  string    `json:"last_title"`
	UpdatedAt  time.Time `json:"updated_at"`
}
