package services

import (
	"testing"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIAMParser(t *testing.T) {
	p := &iamParser{}

	tests := []struct {
		name         string
		fixture      string
		wantAction   parser.Action
		wantResType  string
		wantResID    string
		wantAttrKeys []string
	}{
		{
			name:         "CreateRole creates role",
			fixture:      "CreateRole",
			wantAction:   parser.ActionCreate,
			wantResType:  "role",
			wantResID:    "my-service-role",
			wantAttrKeys: []string{"arn"},
		},
		{
			name:        "DeleteRole deletes role",
			fixture:     "DeleteRole",
			wantAction:  parser.ActionDelete,
			wantResType: "role",
			wantResID:   "my-service-role",
		},
		{
			name:         "AttachRolePolicy updates role",
			fixture:      "AttachRolePolicy",
			wantAction:   parser.ActionUpdate,
			wantResType:  "role",
			wantResID:    "my-service-role",
			wantAttrKeys: []string{"policyArn"},
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
			assert.Equal(t, "iam", delta.Service)
			assert.NotEmpty(t, delta.EventID)
			assert.False(t, delta.EventTime.IsZero())

			for _, key := range tc.wantAttrKeys {
				assert.Contains(t, delta.Attributes, key, "missing attribute %s", key)
			}
		})
	}
}

func TestIAMParserCreateRoleAttributes(t *testing.T) {
	p := &iamParser{}
	event := loadFixture(t, "CreateRole")
	delta, err := p.Parse(event)
	require.NoError(t, err)

	assert.Equal(t, "arn:aws:iam::123456789012:role/my-service-role", delta.Attributes["arn"])
}

func TestIAMParserAttachRolePolicyAttributes(t *testing.T) {
	p := &iamParser{}
	event := loadFixture(t, "AttachRolePolicy")
	delta, err := p.Parse(event)
	require.NoError(t, err)

	assert.Equal(t, "arn:aws:iam::aws:policy/AWSLambdaBasicExecutionRole", delta.Attributes["policyArn"])
}

func TestIAMParserSupportedEvents(t *testing.T) {
	p := &iamParser{}
	assert.Equal(t, "iam", p.Service())
	events := p.SupportedEvents()
	assert.Len(t, events, 8)
	assert.Contains(t, events, "CreateRole")
	assert.Contains(t, events, "DeleteRole")
	assert.Contains(t, events, "AttachRolePolicy")
	assert.Contains(t, events, "DetachRolePolicy")
	assert.Contains(t, events, "CreateUser")
	assert.Contains(t, events, "DeleteUser")
	assert.Contains(t, events, "CreatePolicy")
	assert.Contains(t, events, "DeletePolicy")
}
