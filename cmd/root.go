package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const banner = `
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 ░  ☠  CloudNecromancer  ☠  ░
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 Raising the dead since 2026
`

var (
	dbPath  string
	profile string
	quiet   bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "cloudnecromancer",
	Short: "Reconstruct AWS infrastructure from CloudTrail events",
	Long:  "CloudNecromancer resurrects point-in-time snapshots of AWS infrastructure by replaying CloudTrail event chains.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !quiet {
			fmt.Fprint(os.Stderr, banner)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "./necromancer.db", "Path to DuckDB database file")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "AWS profile to use")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress banner and non-essential output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
