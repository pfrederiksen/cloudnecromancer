package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
	"github.com/pfrederiksen/cloudnecromancer/internal/export"
	"github.com/pfrederiksen/cloudnecromancer/internal/store"
)

var (
	resurrectAt       string
	resurrectServices string
	resurrectRegion   string
	resurrectFormat   string
	resurrectOutput   string
	resurrectDead     bool
	resurrectRitual   bool
)

var resurrectCmd = &cobra.Command{
	Use:   "resurrect",
	Short: "Reconstruct infrastructure at a point in time",
	Long:  "Replays CloudTrail events to rebuild the state of all AWS resources at the specified timestamp.",
	RunE:  runResurrect,
}

func init() {
	resurrectCmd.Flags().StringVar(&resurrectAt, "at", "", "Point-in-time timestamp (RFC3339, required)")
	resurrectCmd.Flags().StringVar(&resurrectServices, "services", "", "Comma-separated service filter (e.g. ec2,iam,s3)")
	resurrectCmd.Flags().StringVar(&resurrectRegion, "region", "", "Region filter")
	resurrectCmd.Flags().StringVar(&resurrectFormat, "format", "json", "Output format: json, terraform, cloudformation, cdk, pulumi, ocsf, csv")
	resurrectCmd.Flags().StringVar(&resurrectOutput, "output", "", "Output file path (default: stdout)")
	resurrectCmd.Flags().BoolVar(&resurrectDead, "include-dead", false, "Include terminated/deleted resources")
	resurrectCmd.Flags().BoolVar(&resurrectRitual, "ritual", false, "Show animated resurrection ritual")
	_ = resurrectCmd.MarkFlagRequired("at")
	rootCmd.AddCommand(resurrectCmd)
}

func runResurrect(cmd *cobra.Command, args []string) error {
	at, err := time.Parse(time.RFC3339, resurrectAt)
	if err != nil {
		return fmt.Errorf("invalid --at timestamp: %w", err)
	}

	if resurrectRitual && !quiet {
		printRitual()
	}

	st, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()

	var services []string
	if resurrectServices != "" {
		services = strings.Split(resurrectServices, ",")
	}

	var regions []string
	if resurrectRegion != "" {
		regions = strings.Split(resurrectRegion, ",")
	}

	opts := engine.ReplayOptions{
		At:          at,
		Services:    services,
		Regions:     regions,
		IncludeDead: resurrectDead,
	}

	snap, err := engine.Replay(st, opts)
	if err != nil {
		return fmt.Errorf("resurrection failed: %w", err)
	}

	// Determine output writer
	w := os.Stdout
	if resurrectOutput != "" {
		f, err := os.OpenFile(resurrectOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	// Select exporter
	exp, err := export.GetExporter(resurrectFormat)
	if err != nil {
		return err
	}

	if err := exp.Export(snap, w); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "\nSnapshot at %s: %d resources across %d services\n",
			at.Format(time.RFC3339), snap.Summary.TotalResources, len(snap.Summary.ByService))
	}

	return nil
}

func printRitual() {
	skull := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Render(`
    ╔═══════════════════════════╗
    ║   ☠  RAISING THE DEAD  ☠  ║
    ╚═══════════════════════════╝
`)
	fmt.Fprint(os.Stderr, skull)
	// Typewriter effect
	msg := "Summoning infrastructure from the beyond..."
	for _, c := range msg {
		fmt.Fprint(os.Stderr, string(c))
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Fprintln(os.Stderr)
}
