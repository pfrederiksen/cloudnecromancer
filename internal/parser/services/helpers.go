package services

import (
	"fmt"
	"time"
)

// parseEvent extracts common fields from a raw CloudTrail event map.
func parseEvent(event map[string]any) (eventID string, eventTime time.Time, eventName string, err error) {
	eid, ok := event["eventID"].(string)
	if !ok || eid == "" {
		return "", time.Time{}, "", fmt.Errorf("missing or invalid eventID")
	}

	etStr, ok := event["eventTime"].(string)
	if !ok || etStr == "" {
		return "", time.Time{}, "", fmt.Errorf("missing or invalid eventTime")
	}

	et, err := time.Parse(time.RFC3339, etStr)
	if err != nil {
		return "", time.Time{}, "", fmt.Errorf("invalid eventTime format: %w", err)
	}

	en, ok := event["eventName"].(string)
	if !ok || en == "" {
		return "", time.Time{}, "", fmt.Errorf("missing or invalid eventName")
	}

	return eid, et, en, nil
}

// getString safely extracts a string value from a map.
func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

// getMap safely extracts a nested map from a map.
func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

// getSlice safely extracts a slice from a map.
func getSlice(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, _ := m[key].([]any)
	return v
}
