package export

import (
	"fmt"
	"io"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
)

// Exporter writes a Snapshot in a specific output format.
type Exporter interface {
	Export(snapshot *engine.Snapshot, w io.Writer) error
}

// GetExporter returns the exporter for the given format name.
func GetExporter(format string) (Exporter, error) {
	switch format {
	case "json":
		return &JSONExporter{}, nil
	case "hcl", "terraform", "tf":
		return &HCLExporter{}, nil
	case "ocsf":
		return &OCSFExporter{}, nil
	case "csv":
		return &CSVExporter{}, nil
	case "cloudformation", "cfn":
		return &CloudFormationExporter{}, nil
	case "cdk":
		return &CDKExporter{}, nil
	case "pulumi":
		return &PulumiExporter{}, nil
	default:
		return nil, fmt.Errorf("unknown export format: %s (supported: json, terraform, cloudformation, cdk, pulumi, ocsf, csv)", format)
	}
}
