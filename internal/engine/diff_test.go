package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	from := &Snapshot{
		Timestamp: t1,
		Resources: map[string][]Resource{
			"ec2:instance": {
				{ResourceID: "i-stays", State: "active", Attributes: map[string]any{"instanceType": "t3.small"}},
				{ResourceID: "i-removed", State: "active", Attributes: map[string]any{"instanceType": "t2.micro"}},
				{ResourceID: "i-changed", State: "active", Attributes: map[string]any{"instanceType": "t3.small", "imageId": "ami-old"}},
			},
		},
	}

	to := &Snapshot{
		Timestamp: t2,
		Resources: map[string][]Resource{
			"ec2:instance": {
				{ResourceID: "i-stays", State: "active", Attributes: map[string]any{"instanceType": "t3.small"}},
				{ResourceID: "i-new", State: "active", Attributes: map[string]any{"instanceType": "t3.large"}},
				{ResourceID: "i-changed", State: "active", Attributes: map[string]any{"instanceType": "t3.medium", "imageId": "ami-old"}},
			},
		},
	}

	result := Diff(from, to)

	t.Run("added", func(t *testing.T) {
		assert.Len(t, result.Added, 1)
		assert.Equal(t, "i-new", result.Added[0].ResourceID)
		assert.Equal(t, "ec2:instance", result.Added[0].TypeKey)
	})

	t.Run("removed", func(t *testing.T) {
		assert.Len(t, result.Removed, 1)
		assert.Equal(t, "i-removed", result.Removed[0].ResourceID)
	})

	t.Run("modified", func(t *testing.T) {
		assert.Len(t, result.Modified, 1)
		assert.Equal(t, "i-changed", result.Modified[0].ResourceID)
		assert.Len(t, result.Modified[0].Changes, 1)
		assert.Equal(t, "instanceType", result.Modified[0].Changes[0].Key)
		assert.Equal(t, "t3.small", result.Modified[0].Changes[0].OldValue)
		assert.Equal(t, "t3.medium", result.Modified[0].Changes[0].NewValue)
	})
}

func TestDiffEmpty(t *testing.T) {
	snap := &Snapshot{
		Timestamp: time.Now(),
		Resources: map[string][]Resource{},
	}
	result := Diff(snap, snap)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.Empty(t, result.Modified)
}
