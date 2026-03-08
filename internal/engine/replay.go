package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	_ "github.com/pfrederiksen/cloudnecromancer/internal/parser/services" // register parsers
	"github.com/pfrederiksen/cloudnecromancer/internal/store"
)

// ReplayOptions configures the resurrection engine.
type ReplayOptions struct {
	At          time.Time
	AccountID   string
	Services    []string
	Regions     []string
	IncludeDead bool
}

// Replay processes stored events up to the given timestamp and returns a
// point-in-time Snapshot of all resources.
func Replay(st *store.Store, opts ReplayOptions) (*Snapshot, error) {
	events, err := st.QueryEvents(opts.At, serviceFilters(opts.Services), opts.Regions)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	return ReplayFromEvents(events, opts)
}

// ReplayFromEvents builds a Snapshot from pre-loaded StoredEvents.
// Exported for testing without a real store.
func ReplayFromEvents(events []store.StoredEvent, opts ReplayOptions) (*Snapshot, error) {
	// resourceKey → *Resource
	state := make(map[string]*Resource)
	// resourceKey → "service:type" for grouping
	resourceTypes := make(map[string]string)

	for _, ev := range events {
		delta, err := parseStoredEvent(ev)
		if err != nil {
			// Skip events with no registered parser (unknown services)
			continue
		}

		key := resourceKey(delta)
		if key == "" {
			continue
		}
		typeKey := delta.Service + ":" + delta.ResourceType

		switch delta.Action {
		case parser.ActionCreate:
			state[key] = &Resource{
				ResourceID:   delta.ResourceID,
				State:        "active",
				Attributes:   copyAttrs(delta.Attributes),
				CreatedAt:    delta.EventTime,
				LastModified: delta.EventTime,
			}
			resourceTypes[key] = typeKey

		case parser.ActionUpdate:
			existing, ok := state[key]
			if !ok {
				// Update for a resource we haven't seen created — create it implicitly
				existing = &Resource{
					ResourceID:   delta.ResourceID,
					State:        "active",
					Attributes:   make(map[string]any),
					CreatedAt:    delta.EventTime,
					LastModified: delta.EventTime,
				}
				state[key] = existing
				resourceTypes[key] = typeKey
			}
			mergeAttrs(existing.Attributes, delta.Attributes)
			existing.LastModified = delta.EventTime

		case parser.ActionDelete:
			existing, ok := state[key]
			if ok {
				existing.State = "terminated"
				existing.LastModified = delta.EventTime
			}
		}
	}

	return buildSnapshot(state, resourceTypes, opts), nil
}

func parseStoredEvent(ev store.StoredEvent) (*parser.ResourceDelta, error) {
	p, err := parser.Lookup(ev.EventName)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(ev.RawJSON), &raw); err != nil {
		return nil, fmt.Errorf("unmarshal event %s: %w", ev.EventID, err)
	}

	return p.Parse(raw)
}

func resourceKey(delta *parser.ResourceDelta) string {
	if delta.ResourceID == "" {
		return ""
	}
	return delta.Service + ":" + delta.ResourceType + ":" + delta.ResourceID
}

func copyAttrs(src map[string]any) map[string]any {
	if src == nil {
		return make(map[string]any)
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeAttrs(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func buildSnapshot(state map[string]*Resource, resourceTypes map[string]string, opts ReplayOptions) *Snapshot {
	snap := &Snapshot{
		Timestamp: opts.At,
		AccountID: opts.AccountID,
		Regions:   opts.Regions,
		Resources: make(map[string][]Resource),
		Summary: Summary{
			ByService: make(map[string]int),
			ByState:   make(map[string]int),
		},
	}

	for key, res := range state {
		if res.State == "terminated" && !opts.IncludeDead {
			continue
		}

		typeKey := resourceTypes[key]
		snap.Resources[typeKey] = append(snap.Resources[typeKey], *res)
		snap.Summary.TotalResources++

		// Extract service from "service:type"
		svc := typeKey
		for i, c := range typeKey {
			if c == ':' {
				svc = typeKey[:i]
				break
			}
		}
		snap.Summary.ByService[svc]++
		snap.Summary.ByState[res.State]++
	}

	// Sort resources within each type for deterministic output
	for typeKey := range snap.Resources {
		sort.Slice(snap.Resources[typeKey], func(i, j int) bool {
			return snap.Resources[typeKey][i].ResourceID < snap.Resources[typeKey][j].ResourceID
		})
	}

	return snap
}

// serviceFilters converts user-friendly service names (e.g. "ec2") to
// CloudTrail event source format (e.g. "ec2.amazonaws.com") if needed.
// If the service already contains a dot, it's passed through as-is.
func serviceFilters(services []string) []string {
	if len(services) == 0 {
		return nil
	}
	result := make([]string, len(services))
	for i, s := range services {
		if len(s) > 0 && s[len(s)-1] != '.' {
			// Could be short name — pass through and let DuckDB filter
			result[i] = s
		} else {
			result[i] = s
		}
	}
	return result
}
