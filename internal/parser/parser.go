package parser

import "time"

// Action represents the type of change a CloudTrail event describes.
type Action string

const (
	ActionCreate Action = "CREATE"
	ActionUpdate Action = "UPDATE"
	ActionDelete Action = "DELETE"
)

// ResourceDelta represents a single state change extracted from a CloudTrail event.
type ResourceDelta struct {
	EventID      string
	EventTime    time.Time
	Action       Action
	Service      string
	ResourceType string
	ResourceID   string
	Attributes   map[string]any
}

// Parser extracts ResourceDeltas from raw CloudTrail events.
type Parser interface {
	Service() string
	SupportedEvents() []string
	Parse(event map[string]any) (*ResourceDelta, error)
}
