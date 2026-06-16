package model

import "time"

type Status string

const (
	Pending    Status = "pending"
	Processing Status = "processing"
	Done       Status = "done"
	Failed     Status = "failed"
)

func (s Status) IsValid() bool {
	var validStatuses = map[Status]bool{
		Pending:    true,
		Processing: true,
		Done:       true,
		Failed:     true,
	}

	return validStatuses[s]
}

type Event struct {
	ID        uint
	CreatedAt time.Time
	Payload   string
	Status    Status
}
