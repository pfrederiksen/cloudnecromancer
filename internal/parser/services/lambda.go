package services

import (
	"fmt"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
)

func init() {
	parser.Register(&lambdaParser{})
}

type lambdaParser struct{}

func (p *lambdaParser) Service() string { return "lambda" }

func (p *lambdaParser) SupportedEvents() []string {
	return []string{
		"CreateFunction20150331",
		"UpdateFunctionCode20150331v2",
		"DeleteFunction20150331",
	}
}

func (p *lambdaParser) Parse(event map[string]any) (*parser.ResourceDelta, error) {
	eventID, eventTime, eventName, err := parseEvent(event)
	if err != nil {
		return nil, fmt.Errorf("lambda parser: %w", err)
	}

	reqParams := getMap(event, "requestParameters")
	respElems := getMap(event, "responseElements")

	delta := &parser.ResourceDelta{
		EventID:   eventID,
		EventTime: eventTime,
		Service:   "lambda",
	}

	switch eventName {
	case "CreateFunction20150331":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "function"
		delta.ResourceID = getString(reqParams, "functionName")
		delta.Attributes = make(map[string]any)
		if v := getString(respElems, "functionArn"); v != "" {
			delta.Attributes["functionArn"] = v
		}
		if v := getString(reqParams, "runtime"); v != "" {
			delta.Attributes["runtime"] = v
		}
		if v := getString(reqParams, "handler"); v != "" {
			delta.Attributes["handler"] = v
		}
		if v := getString(reqParams, "role"); v != "" {
			delta.Attributes["role"] = v
		}

	case "UpdateFunctionCode20150331v2":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "function"
		delta.ResourceID = getString(reqParams, "functionName")

	case "DeleteFunction20150331":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "function"
		delta.ResourceID = getString(reqParams, "functionName")

	default:
		return nil, fmt.Errorf("lambda parser: unsupported event %s", eventName)
	}

	return delta, nil
}
