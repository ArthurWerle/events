package model

import "time"

type EventExecution struct {
	ID          uint       `json:"id"`
	EventID     uint       `json:"event_id"`
	AttemptedAt time.Time  `json:"attempted_at"`
	StatusCode  *int       `json:"status_code"`
	Error       *string    `json:"error"`
	DurationMs  *int       `json:"duration_ms"`
}
