package services

import (
	"testing"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLambdaParser(t *testing.T) {
	p := &lambdaParser{}

	tests := []struct {
		name         string
		fixture      string
		wantAction   parser.Action
		wantResType  string
		wantResID    string
		wantAttrKeys []string
	}{
		{
			name:         "CreateFunction creates function",
			fixture:      "CreateFunction20150331",
			wantAction:   parser.ActionCreate,
			wantResType:  "function",
			wantResID:    "my-processor-function",
			wantAttrKeys: []string{"functionArn", "runtime", "handler", "role"},
		},
		{
			name:        "DeleteFunction deletes function",
			fixture:     "DeleteFunction20150331",
			wantAction:  parser.ActionDelete,
			wantResType: "function",
			wantResID:   "my-processor-function",
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
			assert.Equal(t, "lambda", delta.Service)
			assert.NotEmpty(t, delta.EventID)
			assert.False(t, delta.EventTime.IsZero())

			for _, key := range tc.wantAttrKeys {
				assert.Contains(t, delta.Attributes, key, "missing attribute %s", key)
			}
		})
	}
}

func TestLambdaParserCreateFunctionAttributes(t *testing.T) {
	p := &lambdaParser{}
	event := loadFixture(t, "CreateFunction20150331")
	delta, err := p.Parse(event)
	require.NoError(t, err)

	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:my-processor-function", delta.Attributes["functionArn"])
	assert.Equal(t, "python3.12", delta.Attributes["runtime"])
	assert.Equal(t, "index.handler", delta.Attributes["handler"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/my-service-role", delta.Attributes["role"])
}

func TestLambdaParserSupportedEvents(t *testing.T) {
	p := &lambdaParser{}
	assert.Equal(t, "lambda", p.Service())
	events := p.SupportedEvents()
	assert.Len(t, events, 3)
	assert.Contains(t, events, "CreateFunction20150331")
	assert.Contains(t, events, "UpdateFunctionCode20150331v2")
	assert.Contains(t, events, "DeleteFunction20150331")
}
