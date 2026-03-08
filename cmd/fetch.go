package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/spf13/cobra"

	internalaws "github.com/pfrederiksen/cloudnecromancer/internal/aws"
	"github.com/pfrederiksen/cloudnecromancer/internal/store"
)

var (
	fetchAccountID string
	fetchRegion    string
	fetchRegions   string
	fetchStart     string
	fetchEnd       string
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch CloudTrail events and store them in DuckDB",
	RunE:  runFetch,
}

func init() {
	fetchCmd.Flags().StringVar(&fetchAccountID, "account-id", "", "AWS account ID (required)")
	fetchCmd.Flags().StringVar(&fetchRegion, "region", "us-east-1", "Single region to fetch from")
	fetchCmd.Flags().StringVar(&fetchRegions, "regions", "", "Comma-separated list of regions (overrides --region)")
	fetchCmd.Flags().StringVar(&fetchStart, "start", "", "Start time in RFC3339 format (required)")
	fetchCmd.Flags().StringVar(&fetchEnd, "end", "", "End time in RFC3339 format (required)")

	_ = fetchCmd.MarkFlagRequired("account-id")
	_ = fetchCmd.MarkFlagRequired("start")
	_ = fetchCmd.MarkFlagRequired("end")

	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	startTime, err := time.Parse(time.RFC3339, fetchStart)
	if err != nil {
		return fmt.Errorf("invalid --start: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, fetchEnd)
	if err != nil {
		return fmt.Errorf("invalid --end: %w", err)
	}

	regions := []string{fetchRegion}
	if fetchRegions != "" {
		regions = strings.Split(fetchRegions, ",")
		for i := range regions {
			regions[i] = strings.TrimSpace(regions[i])
		}
	}

	ctx := context.Background()

	// Load AWS config
	cfgOpts := []func(*config.LoadOptions) error{}
	if profile != "" {
		cfgOpts = append(cfgOpts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	client := cloudtrail.NewFromConfig(cfg)

	// Fetch events
	fetcher := internalaws.NewFetcher(client, !quiet)
	fmt.Fprintf(os.Stderr, "Fetching events from %d region(s): %s\n", len(regions), strings.Join(regions, ", "))
	fmt.Fprintf(os.Stderr, "Time range: %s to %s\n", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	events, err := fetcher.FetchEvents(ctx, startTime, endTime, regions)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	// Store events
	db, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	inserted, err := db.InsertEvents(events, fetchAccountID)
	if err != nil {
		return fmt.Errorf("insert events: %w", err)
	}

	// Gather services for summary
	serviceSet := make(map[string]struct{})
	for _, ev := range events {
		serviceSet[ev.EventSource] = struct{}{}
	}

	fmt.Fprintf(os.Stderr, "\nDone! %d events fetched, %d new events stored, %d services covered\n",
		len(events), inserted, len(serviceSet))
	if len(events) > 0 {
		minT, maxT := events[0].EventTime, events[0].EventTime
		for _, ev := range events[1:] {
			if ev.EventTime.Before(minT) {
				minT = ev.EventTime
			}
			if ev.EventTime.After(maxT) {
				maxT = ev.EventTime
			}
		}
		fmt.Fprintf(os.Stderr, "Date range: %s to %s\n", minT.Format(time.RFC3339), maxT.Format(time.RFC3339))
	}

	return nil
}
