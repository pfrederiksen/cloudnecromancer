package engine

import "fmt"

// DiffResult contains the differences between two snapshots.
type DiffResult struct {
	From    Snapshot      `json:"from"`
	To      Snapshot      `json:"to"`
	Added   []DiffEntry   `json:"added"`
	Removed []DiffEntry   `json:"removed"`
	Modified []ModifiedEntry `json:"modified"`
}

// DiffEntry represents a resource that was added or removed.
type DiffEntry struct {
	TypeKey    string         `json:"type_key"` // "ec2:instance"
	ResourceID string         `json:"resource_id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// ModifiedEntry represents a resource that changed between two timestamps.
type ModifiedEntry struct {
	TypeKey    string            `json:"type_key"`
	ResourceID string            `json:"resource_id"`
	Changes    []AttributeChange `json:"changes"`
}

// AttributeChange describes a single attribute difference.
type AttributeChange struct {
	Key      string `json:"key"`
	OldValue any    `json:"old_value,omitempty"`
	NewValue any    `json:"new_value,omitempty"`
}

// Diff compares two snapshots and returns what was added, removed, and modified.
func Diff(from, to *Snapshot) *DiffResult {
	result := &DiffResult{
		From: *from,
		To:   *to,
	}

	// Build lookup maps: "typeKey:resourceID" → Resource
	fromMap := indexResources(from)
	toMap := indexResources(to)

	// Added: in `to` but not in `from`
	for key, res := range toMap {
		typeKey := typeKeyFromIndex(key)
		if _, exists := fromMap[key]; !exists {
			result.Added = append(result.Added, DiffEntry{
				TypeKey:    typeKey,
				ResourceID: res.ResourceID,
				Attributes: res.Attributes,
			})
		}
	}

	// Removed: in `from` but not in `to`
	for key, res := range fromMap {
		typeKey := typeKeyFromIndex(key)
		if _, exists := toMap[key]; !exists {
			result.Removed = append(result.Removed, DiffEntry{
				TypeKey:    typeKey,
				ResourceID: res.ResourceID,
				Attributes: res.Attributes,
			})
		}
	}

	// Modified: in both, but attributes differ
	for key, fromRes := range fromMap {
		toRes, exists := toMap[key]
		if !exists {
			continue
		}
		changes := diffAttributes(fromRes.Attributes, toRes.Attributes)
		if len(changes) > 0 {
			typeKey := typeKeyFromIndex(key)
			result.Modified = append(result.Modified, ModifiedEntry{
				TypeKey:    typeKey,
				ResourceID: fromRes.ResourceID,
				Changes:    changes,
			})
		}
	}

	return result
}

func indexResources(snap *Snapshot) map[string]Resource {
	m := make(map[string]Resource)
	for typeKey, resources := range snap.Resources {
		for _, res := range resources {
			key := typeKey + ":" + res.ResourceID
			m[key] = res
		}
	}
	return m
}

func typeKeyFromIndex(indexKey string) string {
	// indexKey is "service:type:resourceID", we want "service:type"
	colons := 0
	for i, c := range indexKey {
		if c == ':' {
			colons++
			if colons == 2 {
				return indexKey[:i]
			}
		}
	}
	return indexKey
}

func diffAttributes(from, to map[string]any) []AttributeChange {
	var changes []AttributeChange

	// Check for changed or removed keys
	for k, oldVal := range from {
		newVal, exists := to[k]
		if !exists {
			changes = append(changes, AttributeChange{Key: k, OldValue: oldVal})
		} else if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			changes = append(changes, AttributeChange{Key: k, OldValue: oldVal, NewValue: newVal})
		}
	}

	// Check for added keys
	for k, newVal := range to {
		if _, exists := from[k]; !exists {
			changes = append(changes, AttributeChange{Key: k, NewValue: newVal})
		}
	}

	return changes
}

