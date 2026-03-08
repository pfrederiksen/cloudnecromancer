package services

import (
	"fmt"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
)

func init() {
	parser.Register(&iamParser{})
}

type iamParser struct{}

func (p *iamParser) Service() string { return "iam" }

func (p *iamParser) SupportedEvents() []string {
	return []string{
		"CreateRole",
		"DeleteRole",
		"AttachRolePolicy",
		"DetachRolePolicy",
		"CreateUser",
		"DeleteUser",
		"CreatePolicy",
		"DeletePolicy",
	}
}

func (p *iamParser) Parse(event map[string]any) (*parser.ResourceDelta, error) {
	eventID, eventTime, eventName, err := parseEvent(event)
	if err != nil {
		return nil, fmt.Errorf("iam parser: %w", err)
	}

	reqParams := getMap(event, "requestParameters")
	respElems := getMap(event, "responseElements")

	delta := &parser.ResourceDelta{
		EventID:   eventID,
		EventTime: eventTime,
		Service:   "iam",
	}

	switch eventName {
	case "CreateRole":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "role"
		delta.ResourceID = getString(reqParams, "roleName")
		delta.Attributes = make(map[string]any)
		role := getMap(respElems, "role")
		if v := getString(role, "arn"); v != "" {
			delta.Attributes["arn"] = v
		}

	case "DeleteRole":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "role"
		delta.ResourceID = getString(reqParams, "roleName")

	case "AttachRolePolicy":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "role"
		delta.ResourceID = getString(reqParams, "roleName")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "policyArn"); v != "" {
			delta.Attributes["policyArn"] = v
		}

	case "DetachRolePolicy":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "role"
		delta.ResourceID = getString(reqParams, "roleName")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "policyArn"); v != "" {
			delta.Attributes["policyArn"] = v
		}

	case "CreateUser":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "user"
		delta.ResourceID = getString(reqParams, "userName")
		delta.Attributes = make(map[string]any)
		user := getMap(respElems, "user")
		if v := getString(user, "arn"); v != "" {
			delta.Attributes["arn"] = v
		}

	case "DeleteUser":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "user"
		delta.ResourceID = getString(reqParams, "userName")

	case "CreatePolicy":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "policy"
		policy := getMap(respElems, "policy")
		delta.ResourceID = getString(policy, "arn")
		delta.Attributes = make(map[string]any)
		if v := getString(policy, "policyName"); v != "" {
			delta.Attributes["policyName"] = v
		}

	case "DeletePolicy":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "policy"
		delta.ResourceID = getString(reqParams, "policyArn")

	default:
		return nil, fmt.Errorf("iam parser: unsupported event %s", eventName)
	}

	return delta, nil
}
