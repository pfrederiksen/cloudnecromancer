package services

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/" + name + ".json")
	require.NoError(t, err, "failed to read fixture %s", name)
	var event map[string]any
	require.NoError(t, json.Unmarshal(data, &event), "failed to unmarshal fixture %s", name)
	return event
}

func TestEC2Parser(t *testing.T) {
	p := &ec2Parser{}

	tests := []struct {
		name         string
		fixture      string
		wantAction   parser.Action
		wantResType  string
		wantResID    string
		wantService  string
		wantAttrKeys []string
	}{
		{
			name:         "RunInstances creates instance",
			fixture:      "RunInstances",
			wantAction:   parser.ActionCreate,
			wantResType:  "instance",
			wantResID:    "i-0abc123def456ghij",
			wantService:  "ec2",
			wantAttrKeys: []string{"instanceType", "imageId", "subnetId", "vpcId"},
		},
		{
			name:        "TerminateInstances deletes instance",
			fixture:     "TerminateInstances",
			wantAction:  parser.ActionDelete,
			wantResType: "instance",
			wantResID:   "i-0abc123def456ghij",
			wantService: "ec2",
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
			assert.Equal(t, tc.wantService, delta.Service)
			assert.NotEmpty(t, delta.EventID)
			assert.False(t, delta.EventTime.IsZero())

			for _, key := range tc.wantAttrKeys {
				assert.Contains(t, delta.Attributes, key, "missing attribute %s", key)
			}
		})
	}
}

func TestEC2ParserRunInstancesAttributes(t *testing.T) {
	p := &ec2Parser{}
	event := loadFixture(t, "RunInstances")
	delta, err := p.Parse(event)
	require.NoError(t, err)

	assert.Equal(t, "t3.micro", delta.Attributes["instanceType"])
	assert.Equal(t, "ami-0abcdef1234567890", delta.Attributes["imageId"])
	assert.Equal(t, "subnet-0bb1c79de3EXAMPLE", delta.Attributes["subnetId"])
	assert.Equal(t, "vpc-0123456789abcdef0", delta.Attributes["vpcId"])
}

func TestEC2ParserSupportedEvents(t *testing.T) {
	p := &ec2Parser{}
	assert.Equal(t, "ec2", p.Service())
	events := p.SupportedEvents()
	assert.Contains(t, events, "RunInstances")
	assert.Contains(t, events, "TerminateInstances")
	assert.Contains(t, events, "CreateVpc")
	assert.Contains(t, events, "CreateSecurityGroup")
	assert.Len(t, events, 12)
}

func TestEC2ParserInvalidEvent(t *testing.T) {
	p := &ec2Parser{}
	_, err := p.Parse(map[string]any{})
	assert.Error(t, err)
}
