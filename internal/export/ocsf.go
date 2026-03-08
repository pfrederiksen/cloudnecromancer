package export

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
)

// OCSFExporter writes a Snapshot as OCSF Inventory Info events (class_uid 5001),
// one per line (newline-delimited JSON).
type OCSFExporter struct{}

// Export writes each resource as an OCSF Inventory Info event.
func (e *OCSFExporter) Export(snapshot *engine.Snapshot, w io.Writer) error {
	enc := json.NewEncoder(w)
	for typeKey, resources := range snapshot.Resources {
		for _, res := range resources {
			event := buildOCSFEvent(snapshot, typeKey, res)
			if err := enc.Encode(event); err != nil {
				return fmt.Errorf("ocsf encode: %w", err)
			}
		}
	}
	return nil
}

func buildOCSFEvent(snapshot *engine.Snapshot, typeKey string, res engine.Resource) map[string]any {
	return map[string]any{
		"class_uid":  5001,
		"class_name": "Inventory Info",
		"category_uid": 5,
		"category_name": "Discovery",
		"severity_id": 1,
		"activity_id": 1,
		"activity_name": "Log",
		"time": snapshot.Timestamp.Format(time.RFC3339),
		"metadata": map[string]any{
			"version": "1.1.0",
			"product": map[string]any{
				"name":      "CloudNecromancer",
				"vendor_name": "cloudnecromancer",
			},
		},
		"cloud": map[string]any{
			"provider": "AWS",
			"account": map[string]any{
				"uid": snapshot.AccountID,
			},
			"region": firstRegion(snapshot.Regions),
		},
		"resource": map[string]any{
			"uid":  res.ResourceID,
			"type": typeKey,
			"data": res.Attributes,
		},
		"status":    res.State,
		"status_id": statusID(res.State),
	}
}

func firstRegion(regions []string) string {
	if len(regions) > 0 {
		return regions[0]
	}
	return ""
}

func statusID(state string) int {
	switch state {
	case "active", "running":
		return 1
	case "terminated":
		return 2
	default:
		return 99
	}
}
