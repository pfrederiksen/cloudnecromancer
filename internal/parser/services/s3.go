package services

import (
	"fmt"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
)

func init() {
	parser.Register(&s3Parser{})
}

type s3Parser struct{}

func (p *s3Parser) Service() string { return "s3" }

func (p *s3Parser) SupportedEvents() []string {
	return []string{
		"CreateBucket",
		"DeleteBucket",
		"PutBucketPolicy",
		"PutBucketVersioning",
		"PutPublicAccessBlock",
	}
}

func (p *s3Parser) Parse(event map[string]any) (*parser.ResourceDelta, error) {
	eventID, eventTime, eventName, err := parseEvent(event)
	if err != nil {
		return nil, fmt.Errorf("s3 parser: %w", err)
	}

	reqParams := getMap(event, "requestParameters")

	delta := &parser.ResourceDelta{
		EventID:   eventID,
		EventTime: eventTime,
		Service:   "s3",
	}

	switch eventName {
	case "CreateBucket":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "bucket"
		delta.ResourceID = getString(reqParams, "bucketName")

	case "DeleteBucket":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "bucket"
		delta.ResourceID = getString(reqParams, "bucketName")

	case "PutBucketPolicy":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "bucket"
		delta.ResourceID = getString(reqParams, "bucketName")
		delta.Attributes = map[string]any{"change": "policy"}

	case "PutBucketVersioning":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "bucket"
		delta.ResourceID = getString(reqParams, "bucketName")
		delta.Attributes = map[string]any{"change": "versioning"}

	case "PutPublicAccessBlock":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "bucket"
		delta.ResourceID = getString(reqParams, "bucketName")
		delta.Attributes = map[string]any{"change": "publicAccessBlock"}

	default:
		return nil, fmt.Errorf("s3 parser: unsupported event %s", eventName)
	}

	return delta, nil
}
