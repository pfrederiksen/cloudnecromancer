package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
	"github.com/pfrederiksen/cloudnecromancer/internal/store"
)

var (
	diffFrom   string
	diffTo     string
	diffFormat string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare infrastructure between two timestamps",
	Long:  "Generates snapshots at two points in time and reports what changed.",
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffFrom, "from", "", "Start timestamp (RFC3339, required)")
	diffCmd.Flags().StringVar(&diffTo, "to", "", "End timestamp (RFC3339, required)")
	diffCmd.Flags().StringVar(&diffFormat, "format", "table", "Output format: table, json")
	_ = diffCmd.MarkFlagRequired("from")
	_ = diffCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	fromTime, err := time.Parse(time.RFC3339, diffFrom)
	if err != nil {
		return fmt.Errorf("invalid --from timestamp: %w", err)
	}
	toTime, err := time.Parse(time.RFC3339, diffTo)
	if err != nil {
		return fmt.Errorf("invalid --to timestamp: %w", err)
	}

	st, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()

	fromOpts := engine.ReplayOptions{At: fromTime}
	toOpts := engine.ReplayOptions{At: toTime}

	fromSnap, err := engine.Replay(st, fromOpts)
	if err != nil {
		return fmt.Errorf("replay --from: %w", err)
	}

	toSnap, err := engine.Replay(st, toOpts)
	if err != nil {
		return fmt.Errorf("replay --to: %w", err)
	}

	result := engine.Diff(fromSnap, toSnap)

	switch diffFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	default:
		printDiffTable(result, fromTime, toTime)
	}

	return nil
}

func printDiffTable(result *engine.DiffResult, from, to time.Time) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Fprintf(os.Stderr, "\nDiff: %s → %s\n\n", from.Format(time.RFC3339), to.Format(time.RFC3339))

	if len(result.Added) > 0 {
		fmt.Fprintf(os.Stderr, "%s (%d resources)\n", green.Render("+ ADDED"), len(result.Added))
		for _, e := range result.Added {
			fmt.Fprintf(os.Stderr, "  %s %-20s %s\n", green.Render("+"), e.TypeKey, e.ResourceID)
		}
		fmt.Fprintln(os.Stderr)
	}

	if len(result.Removed) > 0 {
		fmt.Fprintf(os.Stderr, "%s (%d resources)\n", red.Render("- REMOVED"), len(result.Removed))
		for _, e := range result.Removed {
			fmt.Fprintf(os.Stderr, "  %s %-20s %s\n", red.Render("-"), e.TypeKey, e.ResourceID)
		}
		fmt.Fprintln(os.Stderr)
	}

	if len(result.Modified) > 0 {
		fmt.Fprintf(os.Stderr, "%s (%d resources)\n", yellow.Render("~ MODIFIED"), len(result.Modified))
		for _, e := range result.Modified {
			fmt.Fprintf(os.Stderr, "  %s %-20s %s\n", yellow.Render("~"), e.TypeKey, e.ResourceID)
			for _, c := range e.Changes {
				old := fmt.Sprintf("%v", c.OldValue)
				new := fmt.Sprintf("%v", c.NewValue)
				if c.OldValue == nil {
					fmt.Fprintf(os.Stderr, "      %s: %s\n", c.Key, green.Render("+"+new))
				} else if c.NewValue == nil {
					fmt.Fprintf(os.Stderr, "      %s: %s\n", c.Key, red.Render("-"+old))
				} else {
					fmt.Fprintf(os.Stderr, "      %s: %s → %s\n", c.Key, dim.Render(old), yellow.Render(new))
				}
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	if len(result.Added) == 0 && len(result.Removed) == 0 && len(result.Modified) == 0 {
		fmt.Fprintln(os.Stderr, "No changes detected.")
	}
}
