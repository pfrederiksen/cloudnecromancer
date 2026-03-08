package services

import (
	"fmt"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
)

func init() {
	parser.Register(&rdsParser{})
}

type rdsParser struct{}

func (p *rdsParser) Service() string { return "rds" }

func (p *rdsParser) SupportedEvents() []string {
	return []string{
		"CreateDBInstance",
		"DeleteDBInstance",
		"ModifyDBInstance",
		"CreateDBCluster",
		"DeleteDBCluster",
	}
}

func (p *rdsParser) Parse(event map[string]any) (*parser.ResourceDelta, error) {
	eventID, eventTime, eventName, err := parseEvent(event)
	if err != nil {
		return nil, fmt.Errorf("rds parser: %w", err)
	}

	reqParams := getMap(event, "requestParameters")

	delta := &parser.ResourceDelta{
		EventID:   eventID,
		EventTime: eventTime,
		Service:   "rds",
	}

	switch eventName {
	case "CreateDBInstance":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "db_instance"
		delta.ResourceID = getString(reqParams, "dBInstanceIdentifier")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "engine"); v != "" {
			delta.Attributes["engine"] = v
		}
		if v := getString(reqParams, "dBInstanceClass"); v != "" {
			delta.Attributes["dBInstanceClass"] = v
		}

	case "DeleteDBInstance":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "db_instance"
		delta.ResourceID = getString(reqParams, "dBInstanceIdentifier")

	case "ModifyDBInstance":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "db_instance"
		delta.ResourceID = getString(reqParams, "dBInstanceIdentifier")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "dBInstanceClass"); v != "" {
			delta.Attributes["dBInstanceClass"] = v
		}

	case "CreateDBCluster":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "db_cluster"
		delta.ResourceID = getString(reqParams, "dBClusterIdentifier")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "engine"); v != "" {
			delta.Attributes["engine"] = v
		}

	case "DeleteDBCluster":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "db_cluster"
		delta.ResourceID = getString(reqParams, "dBClusterIdentifier")

	default:
		return nil, fmt.Errorf("rds parser: unsupported event %s", eventName)
	}

	return delta, nil
}
