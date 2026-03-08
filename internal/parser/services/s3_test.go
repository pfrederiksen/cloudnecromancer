package services

import (
	"testing"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3Parser(t *testing.T) {
	p := &s3Parser{}

	tests := []struct {
		name        string
		fixture     string
		wantAction  parser.Action
		wantResType string
		wantResID   string
	}{
		{
			name:        "CreateBucket creates bucket",
			fixture:     "CreateBucket",
			wantAction:  parser.ActionCreate,
			wantResType: "bucket",
			wantResID:   "my-app-data-bucket",
		},
		{
			name:        "DeleteBucket deletes bucket",
			fixture:     "DeleteBucket",
			wantAction:  parser.ActionDelete,
			wantResType: "bucket",
			wantResID:   "my-app-data-bucket",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := loadFixture(t, tc.fixture)
			delta, err := p.Parse(event)
			require.NoError(t, err)

			assert.Equal(t, tc.wantAction, delta.Action)
			assert.Equal(t, tc.wantResType, delta.ResourceType)
			assert.Equal(t, tc.wantResID, delta.ResourceID)
			assert.Equal(t, "s3", delta.Service)
			assert.NotEmpty(t, delta.EventID)
			assert.False(t, delta.EventTime.IsZero())
		})
	}
}

func TestS3ParserSupportedEvents(t *testing.T) {
	p := &s3Parser{}
	assert.Equal(t, "s3", p.Service())
	events := p.SupportedEvents()
	assert.Len(t, events, 5)
	assert.Contains(t, events, "CreateBucket")
	assert.Contains(t, events, "DeleteBucket")
	assert.Contains(t, events, "PutBucketPolicy")
	assert.Contains(t, events, "PutBucketVersioning")
	assert.Contains(t, events, "PutPublicAccessBlock")
}
