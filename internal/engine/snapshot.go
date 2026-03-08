package engine

import "time"

// Snapshot represents the reconstructed state of all AWS resources at a point in time.
type Snapshot struct {
	Timestamp time.Time                `json:"timestamp"`
	AccountID string                   `json:"account_id"`
	Regions   []string                 `json:"regions"`
	Resources map[string][]Resource    `json:"resources"` // key: "ec2:instance"
	Summary   Summary                  `json:"summary"`
}

// Resource represents a single AWS resource and its reconstructed state.
type Resource struct {
	ResourceID   string         `json:"resource_id"`
	State        string         `json:"state"`
	Attributes   map[string]any `json:"attributes"`
	CreatedAt    time.Time      `json:"created_at"`
	LastModified time.Time      `json:"last_modified"`
}

// Summary contains aggregate information about a snapshot.
type Summary struct {
	TotalResources int            `json:"total_resources"`
	ByService      map[string]int `json:"by_service"`
	ByState        map[string]int `json:"by_state"`
}
