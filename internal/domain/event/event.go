package event

import "time"

// Event is a domain occurrence worth recording or reacting to.
type Event interface {
	Name() string
	OccurredAt() time.Time
}
