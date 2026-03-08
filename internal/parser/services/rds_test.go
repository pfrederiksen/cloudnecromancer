package services

import (
	"testing"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRDSParser(t *testing.T) {
	p := &rdsParser{}

	tests := []struct {
		name         string
		fixture      string
		wantAction   parser.Action
		wantResType  string
		wantResID    string
		wantAttrKeys []string
	}{
		{
			name:         "CreateDBInstance creates db_instance",
			fixture:      "CreateDBInstance",
			wantAction:   parser.ActionCreate,
			wantResType:  "db_instance",
			wantResID:    "my-app-db",
			wantAttrKeys: []string{"engine", "dBInstanceClass"},
		},
		{
			name:        "DeleteDBInstance deletes db_instance",
			fixture:     "DeleteDBInstance",
			wantAction:  parser.ActionDelete,
			wantResType: "db_instance",
			wantResID:   "my-app-db",
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
			assert.Equal(t, "rds", delta.Service)
			assert.NotEmpty(t, delta.EventID)
			assert.False(t, delta.EventTime.IsZero())

			for _, key := range tc.wantAttrKeys {
				assert.Contains(t, delta.Attributes, key, "missing attribute %s", key)
			}
		})
	}
}

func TestRDSParserCreateDBInstanceAttributes(t *testing.T) {
	p := &rdsParser{}
	event := loadFixture(t, "CreateDBInstance")
	delta, err := p.Parse(event)
	require.NoError(t, err)

	assert.Equal(t, "postgres", delta.Attributes["engine"])
	assert.Equal(t, "db.t3.medium", delta.Attributes["dBInstanceClass"])
}

func TestRDSParserSupportedEvents(t *testing.T) {
	p := &rdsParser{}
	assert.Equal(t, "rds", p.Service())
	events := p.SupportedEvents()
	assert.Len(t, events, 5)
	assert.Contains(t, events, "CreateDBInstance")
	assert.Contains(t, events, "DeleteDBInstance")
	assert.Contains(t, events, "ModifyDBInstance")
	assert.Contains(t, events, "CreateDBCluster")
	assert.Contains(t, events, "DeleteDBCluster")
}
