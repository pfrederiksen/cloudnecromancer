package export

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
)

// JSONExporter writes a Snapshot as indented JSON.
type JSONExporter struct{}

// Export writes the snapshot as pretty-printed JSON.
func (e *JSONExporter) Export(snapshot *engine.Snapshot, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snapshot); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}
