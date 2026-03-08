package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
	"github.com/pfrederiksen/cloudnecromancer/internal/export"
)

var (
	exportInput  string
	exportFormat string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Re-export an existing snapshot in a different format",
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportInput, "input", "", "Input snapshot JSON file (required)")
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Output format: json, terraform, cloudformation, cdk, pulumi, ocsf, csv")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: stdout)")
	_ = exportCmd.MarkFlagRequired("input")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(exportInput)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var snap engine.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("parse snapshot: %w", err)
	}

	w := os.Stdout
	if exportOutput != "" {
		f, err := os.OpenFile(exportOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer f.Close()
		w = f
	}

	exp, err := export.GetExporter(exportFormat)
	if err != nil {
		return err
	}

	return exp.Export(&snap, w)
}
