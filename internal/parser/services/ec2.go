package services

import (
	"fmt"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
)

func init() {
	parser.Register(&ec2Parser{})
}

type ec2Parser struct{}

func (p *ec2Parser) Service() string { return "ec2" }

func (p *ec2Parser) SupportedEvents() []string {
	return []string{
		"RunInstances",
		"TerminateInstances",
		"StartInstances",
		"StopInstances",
		"CreateSecurityGroup",
		"DeleteSecurityGroup",
		"AuthorizeSecurityGroupIngress",
		"CreateVpc",
		"DeleteVpc",
		"CreateSubnet",
		"CreateInternetGateway",
		"AttachInternetGateway",
	}
}

func (p *ec2Parser) Parse(event map[string]any) (*parser.ResourceDelta, error) {
	eventID, eventTime, eventName, err := parseEvent(event)
	if err != nil {
		return nil, fmt.Errorf("ec2 parser: %w", err)
	}

	reqParams := getMap(event, "requestParameters")
	respElems := getMap(event, "responseElements")

	delta := &parser.ResourceDelta{
		EventID:   eventID,
		EventTime: eventTime,
		Service:   "ec2",
	}

	switch eventName {
	case "RunInstances":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "instance"
		delta.Attributes = make(map[string]any)
		instancesSet := getMap(respElems, "instancesSet")
		items := getSlice(instancesSet, "items")
		if len(items) > 0 {
			item, _ := items[0].(map[string]any)
			if item != nil {
				delta.ResourceID = getString(item, "instanceId")
				if v := getString(item, "instanceType"); v != "" {
					delta.Attributes["instanceType"] = v
				}
				if v := getString(item, "imageId"); v != "" {
					delta.Attributes["imageId"] = v
				}
				if v := getString(item, "subnetId"); v != "" {
					delta.Attributes["subnetId"] = v
				}
				if v := getString(item, "vpcId"); v != "" {
					delta.Attributes["vpcId"] = v
				}
			}
		}

	case "TerminateInstances":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "instance"
		instancesSet := getMap(respElems, "instancesSet")
		items := getSlice(instancesSet, "items")
		if len(items) > 0 {
			item, _ := items[0].(map[string]any)
			if item != nil {
				delta.ResourceID = getString(item, "instanceId")
			}
		}

	case "StartInstances":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "instance"
		delta.Attributes = map[string]any{"stateChange": "started"}
		instancesSet := getMap(respElems, "instancesSet")
		items := getSlice(instancesSet, "items")
		if len(items) > 0 {
			item, _ := items[0].(map[string]any)
			if item != nil {
				delta.ResourceID = getString(item, "instanceId")
			}
		}

	case "StopInstances":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "instance"
		delta.Attributes = map[string]any{"stateChange": "stopped"}
		instancesSet := getMap(respElems, "instancesSet")
		items := getSlice(instancesSet, "items")
		if len(items) > 0 {
			item, _ := items[0].(map[string]any)
			if item != nil {
				delta.ResourceID = getString(item, "instanceId")
			}
		}

	case "CreateSecurityGroup":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "security_group"
		delta.ResourceID = getString(respElems, "groupId")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "groupName"); v != "" {
			delta.Attributes["groupName"] = v
		}
		if v := getString(reqParams, "vpcId"); v != "" {
			delta.Attributes["vpcId"] = v
		}

	case "DeleteSecurityGroup":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "security_group"
		delta.ResourceID = getString(reqParams, "groupId")

	case "AuthorizeSecurityGroupIngress":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "security_group"
		delta.ResourceID = getString(reqParams, "groupId")

	case "CreateVpc":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "vpc"
		vpc := getMap(respElems, "vpc")
		delta.ResourceID = getString(vpc, "vpcId")
		delta.Attributes = make(map[string]any)
		if v := getString(vpc, "cidrBlock"); v != "" {
			delta.Attributes["cidrBlock"] = v
		}

	case "DeleteVpc":
		delta.Action = parser.ActionDelete
		delta.ResourceType = "vpc"
		delta.ResourceID = getString(reqParams, "vpcId")

	case "CreateSubnet":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "subnet"
		subnet := getMap(respElems, "subnet")
		delta.ResourceID = getString(subnet, "subnetId")
		delta.Attributes = make(map[string]any)
		if v := getString(subnet, "vpcId"); v != "" {
			delta.Attributes["vpcId"] = v
		}
		if v := getString(subnet, "cidrBlock"); v != "" {
			delta.Attributes["cidrBlock"] = v
		}
		if v := getString(subnet, "availabilityZone"); v != "" {
			delta.Attributes["availabilityZone"] = v
		}

	case "CreateInternetGateway":
		delta.Action = parser.ActionCreate
		delta.ResourceType = "internet_gateway"
		igw := getMap(respElems, "internetGateway")
		delta.ResourceID = getString(igw, "internetGatewayId")

	case "AttachInternetGateway":
		delta.Action = parser.ActionUpdate
		delta.ResourceType = "internet_gateway"
		delta.ResourceID = getString(reqParams, "internetGatewayId")
		delta.Attributes = make(map[string]any)
		if v := getString(reqParams, "vpcId"); v != "" {
			delta.Attributes["vpcId"] = v
		}

	default:
		return nil, fmt.Errorf("ec2 parser: unsupported event %s", eventName)
	}

	return delta, nil
}
