package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

// RawEvent holds one CloudTrail event in a flat struct suitable for storage.
type RawEvent struct {
	EventID     string
	EventTime   time.Time
	EventName   string
	EventSource string
	Region      string
	RawJSON     string
}

// Fetcher retrieves CloudTrail events from one or more regions.
type Fetcher struct {
	client    CloudTrailAPI
	showProgress bool
}

// NewFetcher creates a Fetcher that calls the given CloudTrailAPI.
// If showProgress is true a progress bar is displayed per region.
func NewFetcher(client CloudTrailAPI, showProgress bool) *Fetcher {
	return &Fetcher{client: client, showProgress: showProgress}
}

// FetchEvents retrieves CloudTrail events across all specified regions
// between startTime and endTime. Events are deduplicated by EventID.
func (f *Fetcher) FetchEvents(ctx context.Context, startTime, endTime time.Time, regions []string) ([]RawEvent, error) {
	var (
		mu     sync.Mutex
		seen   = make(map[string]struct{})
		result []RawEvent
	)

	g, gctx := errgroup.WithContext(ctx)

	for _, region := range regions {
		region := region // capture
		g.Go(func() error {
			events, err := f.fetchRegion(gctx, startTime, endTime, region)
			if err != nil {
				return fmt.Errorf("fetch region %s: %w", region, err)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, ev := range events {
				if _, dup := seen[ev.EventID]; !dup {
					seen[ev.EventID] = struct{}{}
					result = append(result, ev)
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

func (f *Fetcher) fetchRegion(ctx context.Context, startTime, endTime time.Time, region string) ([]RawEvent, error) {
	var (
		events    []RawEvent
		nextToken *string
		bar       *progressbar.ProgressBar
	)

	if f.showProgress {
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription(fmt.Sprintf("  [%s]", region)),
			progressbar.OptionSetWidth(30),
			progressbar.OptionSpinnerType(14),
		)
	}

	for {
		input := &cloudtrail.LookupEventsInput{
			StartTime: &startTime,
			EndTime:   &endTime,
			NextToken: nextToken,
		}

		output, err := f.client.LookupEvents(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("LookupEvents: %w", err)
		}

		for _, ev := range output.Events {
			raw := RawEvent{
				Region: region,
			}
			if ev.EventId != nil {
				raw.EventID = *ev.EventId
			}
			if ev.EventTime != nil {
				raw.EventTime = *ev.EventTime
			}
			if ev.EventName != nil {
				raw.EventName = *ev.EventName
			}
			if ev.EventSource != nil {
				raw.EventSource = *ev.EventSource
			}
			if ev.CloudTrailEvent != nil {
				raw.RawJSON = *ev.CloudTrailEvent
			}
			events = append(events, raw)
		}

		if bar != nil {
			_ = bar.Add(len(output.Events))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	if bar != nil {
		_ = bar.Finish()
	}

	return events, nil
}
