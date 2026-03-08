package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
)

// MockCloudTrailClient implements aws.CloudTrailAPI for testing.
// It returns pre-configured responses in order, and can inject errors.
type MockCloudTrailClient struct {
	mu        sync.Mutex
	responses []MockResponse
	callIndex int
	Calls     []LookupEventsCall
}

// MockResponse holds a single canned response (or error) for LookupEvents.
type MockResponse struct {
	Output *cloudtrail.LookupEventsOutput
	Err    error
}

// LookupEventsCall records the arguments of one LookupEvents invocation.
type LookupEventsCall struct {
	Input *cloudtrail.LookupEventsInput
}

// NewMockCloudTrailClient creates a mock with the given sequence of responses.
func NewMockCloudTrailClient(responses ...MockResponse) *MockCloudTrailClient {
	return &MockCloudTrailClient{responses: responses}
}

// LookupEvents satisfies CloudTrailAPI. It returns the next response from the
// configured list. If the list is exhausted it returns an error.
func (m *MockCloudTrailClient) LookupEvents(ctx context.Context, params *cloudtrail.LookupEventsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.LookupEventsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, LookupEventsCall{Input: params})

	if m.callIndex >= len(m.responses) {
		return nil, fmt.Errorf("mock: no more responses configured (call #%d)", m.callIndex)
	}

	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp.Output, resp.Err
}

// CallCount returns how many times LookupEvents was called.
func (m *MockCloudTrailClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}
