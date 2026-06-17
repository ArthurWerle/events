package model

import "time"

type Status string

const (
	STATUS_PENDING    Status = "pending"
	STATUS_PROCESSING Status = "processing"
	STATUS_DONE       Status = "done"
	STATUS_FAILED     Status = "failed"
)

func (s Status) IsValid() bool {
	var validStatuses = map[Status]bool{
		STATUS_PENDING:    true,
		STATUS_PROCESSING: true,
		STATUS_DONE:       true,
		STATUS_FAILED:     true,
	}

	return validStatuses[s]
}

type Event struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
	Status    Status    `json:"status"`
}
