package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pfrederiksen/cloudnecromancer/internal/store"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show statistics about the local event store",
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	db, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	stats, err := db.Stats()
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	if stats.EventCount == 0 {
		fmt.Fprintln(os.Stderr, "Event store is empty. Run 'fetch' first.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "Database:   %s\n", dbPath)
	fmt.Fprintf(os.Stdout, "Events:     %d\n", stats.EventCount)
	fmt.Fprintf(os.Stdout, "Date range: %s to %s\n",
		stats.MinTime.Format(time.RFC3339), stats.MaxTime.Format(time.RFC3339))
	fmt.Fprintf(os.Stdout, "Services:   %s\n", strings.Join(stats.Services, ", "))

	return nil
}
