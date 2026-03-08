package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	_ "github.com/pfrederiksen/cloudnecromancer/internal/parser/services"
	"github.com/pfrederiksen/cloudnecromancer/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEvent(t *testing.T, eventID, eventName, eventSource string, eventTime time.Time, reqParams, respElems map[string]any) store.StoredEvent {
	t.Helper()
	raw := map[string]any{
		"eventID":   eventID,
		"eventTime": eventTime.Format(time.RFC3339),
		"eventName": eventName,
	}
	if reqParams != nil {
		raw["requestParameters"] = reqParams
	}
	if respElems != nil {
		raw["responseElements"] = respElems
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)
	return store.StoredEvent{
		EventID:     eventID,
		EventTime:   eventTime,
		EventName:   eventName,
		EventSource: eventSource,
		RawJSON:     string(b),
	}
}

func TestReplayFromEvents(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Hour)
	t2 := t0.Add(2 * time.Hour)
	t3 := t0.Add(3 * time.Hour)

	createInstance := makeEvent(t, "evt-1", "RunInstances", "ec2.amazonaws.com", t1,
		nil,
		map[string]any{
			"instancesSet": map[string]any{
				"items": []any{
					map[string]any{
						"instanceId":   "i-abc123",
						"instanceType": "t3.medium",
						"imageId":      "ami-12345",
						"vpcId":        "vpc-001",
					},
				},
			},
		},
	)

	updateInstance := makeEvent(t, "evt-2", "StopInstances", "ec2.amazonaws.com", t2,
		nil,
		map[string]any{
			"instancesSet": map[string]any{
				"items": []any{
					map[string]any{"instanceId": "i-abc123"},
				},
			},
		},
	)

	deleteInstance := makeEvent(t, "evt-3", "TerminateInstances", "ec2.amazonaws.com", t3,
		nil,
		map[string]any{
			"instancesSet": map[string]any{
				"items": []any{
					map[string]any{"instanceId": "i-abc123"},
				},
			},
		},
	)

	createBucket := makeEvent(t, "evt-4", "CreateBucket", "s3.amazonaws.com", t1,
		map[string]any{"bucketName": "my-bucket"},
		nil,
	)

	tests := []struct {
		name        string
		events      []store.StoredEvent
		at          time.Time
		includeDead bool
		wantTotal   int
		wantEC2     int
		wantS3      int
		wantState   map[string]string // resourceID → expected state
	}{
		{
			name:      "before any events",
			events:    []store.StoredEvent{createInstance, createBucket},
			at:        t0,
			wantTotal: 0,
		},
		{
			name:      "after create",
			events:    []store.StoredEvent{createInstance, createBucket},
			at:        t1.Add(time.Minute),
			wantTotal: 2,
			wantEC2:   1,
			wantS3:    1,
			wantState: map[string]string{"i-abc123": "active", "my-bucket": "active"},
		},
		{
			name:      "after update",
			events:    []store.StoredEvent{createInstance, updateInstance, createBucket},
			at:        t2.Add(time.Minute),
			wantTotal: 2,
			wantEC2:   1,
			wantState: map[string]string{"i-abc123": "active"},
		},
		{
			name:        "after delete without include-dead",
			events:      []store.StoredEvent{createInstance, updateInstance, deleteInstance, createBucket},
			at:          t3.Add(time.Minute),
			includeDead: false,
			wantTotal:   1,
			wantS3:      1,
		},
		{
			name:        "after delete with include-dead",
			events:      []store.StoredEvent{createInstance, updateInstance, deleteInstance, createBucket},
			at:          t3.Add(time.Minute),
			includeDead: true,
			wantTotal:   2,
			wantState:   map[string]string{"i-abc123": "terminated", "my-bucket": "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Filter events to only those before tt.at
			var filtered []store.StoredEvent
			for _, ev := range tt.events {
				if !ev.EventTime.After(tt.at) {
					filtered = append(filtered, ev)
				}
			}

			snap, err := ReplayFromEvents(filtered, ReplayOptions{
				At:          tt.at,
				AccountID:   "123456789012",
				IncludeDead: tt.includeDead,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, snap.Summary.TotalResources)

			if tt.wantEC2 > 0 {
				assert.Len(t, snap.Resources["ec2:instance"], tt.wantEC2)
			}
			if tt.wantS3 > 0 {
				assert.Len(t, snap.Resources["s3:bucket"], tt.wantS3)
			}

			if tt.wantState != nil {
				for resID, wantState := range tt.wantState {
					found := false
					for _, resources := range snap.Resources {
						for _, r := range resources {
							if r.ResourceID == resID {
								assert.Equal(t, wantState, r.State, "resource %s", resID)
								found = true
							}
						}
					}
					assert.True(t, found, "resource %s not found in snapshot", resID)
				}
			}
		})
	}
}

func TestReplayAttributeMerge(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	create := makeEvent(t, "evt-1", "RunInstances", "ec2.amazonaws.com", t1,
		nil,
		map[string]any{
			"instancesSet": map[string]any{
				"items": []any{
					map[string]any{
						"instanceId":   "i-merge",
						"instanceType": "t3.small",
						"imageId":      "ami-old",
					},
				},
			},
		},
	)

	update := makeEvent(t, "evt-2", "StopInstances", "ec2.amazonaws.com", t2,
		nil,
		map[string]any{
			"instancesSet": map[string]any{
				"items": []any{
					map[string]any{"instanceId": "i-merge"},
				},
			},
		},
	)

	snap, err := ReplayFromEvents([]store.StoredEvent{create, update}, ReplayOptions{
		At:        t2.Add(time.Minute),
		AccountID: "123456789012",
	})
	require.NoError(t, err)

	instances := snap.Resources["ec2:instance"]
	require.Len(t, instances, 1)
	// Original attributes should be preserved after update
	assert.Equal(t, "t3.small", instances[0].Attributes["instanceType"])
	assert.Equal(t, "ami-old", instances[0].Attributes["imageId"])
	// Update should have merged in stateChange
	assert.Equal(t, "stopped", instances[0].Attributes["stateChange"])
}

func TestReplayUnknownEventsSkipped(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	unknown := store.StoredEvent{
		EventID:     "evt-unknown",
		EventTime:   t1,
		EventName:   "SomeUnknownEvent",
		EventSource: "mystery.amazonaws.com",
		RawJSON:     `{"eventID":"evt-unknown","eventTime":"2026-01-01T01:00:00Z","eventName":"SomeUnknownEvent"}`,
	}

	snap, err := ReplayFromEvents([]store.StoredEvent{unknown}, ReplayOptions{
		At:        t1.Add(time.Minute),
		AccountID: "123456789012",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, snap.Summary.TotalResources)
}

func TestReplayRegisteredParsers(t *testing.T) {
	// Verify key events have registered parsers
	events := parser.RegisteredEvents()
	expected := []string{"RunInstances", "CreateRole", "CreateBucket", "CreateFunction20150331", "CreateDBInstance"}
	for _, e := range expected {
		_, err := parser.Lookup(e)
		assert.NoError(t, err, "expected parser for %s", e)
	}
	assert.GreaterOrEqual(t, len(events), 30, "expected at least 30 registered events")
}
