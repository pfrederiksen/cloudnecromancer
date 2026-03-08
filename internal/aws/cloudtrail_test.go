package aws_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	cttypes "github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	internalaws "github.com/pfrederiksen/cloudnecromancer/internal/aws"
	"github.com/pfrederiksen/cloudnecromancer/internal/aws/mocks"
)

func strPtr(s string) *string { return &s }
func timePtr(t time.Time) *time.Time { return &t }

func makeEvent(id, name, source string, t time.Time) cttypes.Event {
	return cttypes.Event{
		EventId:         strPtr(id),
		EventName:       strPtr(name),
		EventSource:     strPtr(source),
		EventTime:       timePtr(t),
		CloudTrailEvent: strPtr(`{"detail":"` + id + `"}`),
	}
}

func TestFetchEvents(t *testing.T) {
	baseTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		regions    []string
		responses  []mocks.MockResponse
		wantCount  int
		wantErr    bool
	}{
		{
			name:    "single page single region",
			regions: []string{"us-east-1"},
			responses: []mocks.MockResponse{
				{
					Output: &cloudtrail.LookupEventsOutput{
						Events: []cttypes.Event{
							makeEvent("evt-1", "CreateBucket", "s3.amazonaws.com", baseTime),
							makeEvent("evt-2", "PutObject", "s3.amazonaws.com", baseTime.Add(time.Minute)),
						},
					},
				},
			},
			wantCount: 2,
		},
		{
			name:    "pagination across two pages",
			regions: []string{"us-east-1"},
			responses: []mocks.MockResponse{
				{
					Output: &cloudtrail.LookupEventsOutput{
						Events: []cttypes.Event{
							makeEvent("evt-1", "CreateBucket", "s3.amazonaws.com", baseTime),
						},
						NextToken: strPtr("page2"),
					},
				},
				{
					Output: &cloudtrail.LookupEventsOutput{
						Events: []cttypes.Event{
							makeEvent("evt-2", "PutObject", "s3.amazonaws.com", baseTime.Add(time.Minute)),
						},
					},
				},
			},
			wantCount: 2,
		},
		{
			name:    "multi-region dedup",
			regions: []string{"us-east-1", "eu-west-1"},
			responses: []mocks.MockResponse{
				// us-east-1 page
				{
					Output: &cloudtrail.LookupEventsOutput{
						Events: []cttypes.Event{
							makeEvent("evt-1", "CreateBucket", "s3.amazonaws.com", baseTime),
							makeEvent("evt-2", "PutObject", "s3.amazonaws.com", baseTime.Add(time.Minute)),
						},
					},
				},
				// eu-west-1 page — evt-1 is a duplicate
				{
					Output: &cloudtrail.LookupEventsOutput{
						Events: []cttypes.Event{
							makeEvent("evt-1", "CreateBucket", "s3.amazonaws.com", baseTime),
							makeEvent("evt-3", "DeleteBucket", "s3.amazonaws.com", baseTime.Add(2*time.Minute)),
						},
					},
				},
			},
			wantCount: 3, // evt-1, evt-2, evt-3
		},
		{
			name:    "API error",
			regions: []string{"us-east-1"},
			responses: []mocks.MockResponse{
				{Err: fmt.Errorf("throttled")},
			},
			wantErr: true,
		},
		{
			name:    "empty result",
			regions: []string{"us-east-1"},
			responses: []mocks.MockResponse{
				{Output: &cloudtrail.LookupEventsOutput{}},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockCloudTrailClient(tt.responses...)
			fetcher := internalaws.NewFetcher(mock, false)

			start := baseTime.Add(-time.Hour)
			end := baseTime.Add(time.Hour)

			events, err := fetcher.FetchEvents(context.Background(), start, end, tt.regions)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != tt.wantCount {
				t.Errorf("got %d events, want %d", len(events), tt.wantCount)
			}
		})
	}
}

func TestFetchEvents_PaginationCallCount(t *testing.T) {
	baseTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	mock := mocks.NewMockCloudTrailClient(
		mocks.MockResponse{
			Output: &cloudtrail.LookupEventsOutput{
				Events:    []cttypes.Event{makeEvent("e1", "X", "s3.amazonaws.com", baseTime)},
				NextToken: strPtr("tok"),
			},
		},
		mocks.MockResponse{
			Output: &cloudtrail.LookupEventsOutput{
				Events:    []cttypes.Event{makeEvent("e2", "Y", "s3.amazonaws.com", baseTime)},
				NextToken: strPtr("tok2"),
			},
		},
		mocks.MockResponse{
			Output: &cloudtrail.LookupEventsOutput{
				Events: []cttypes.Event{makeEvent("e3", "Z", "s3.amazonaws.com", baseTime)},
			},
		},
	)

	fetcher := internalaws.NewFetcher(mock, false)
	events, err := fetcher.FetchEvents(context.Background(), baseTime.Add(-time.Hour), baseTime.Add(time.Hour), []string{"us-east-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("got %d events, want 3", len(events))
	}
	if mock.CallCount() != 3 {
		t.Errorf("got %d API calls, want 3", mock.CallCount())
	}
}
