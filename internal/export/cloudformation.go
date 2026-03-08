package export

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
)

// CloudFormationExporter writes a Snapshot as an AWS CloudFormation template (JSON).
type CloudFormationExporter struct{}

// cfnTypeMapping maps internal type keys to CloudFormation resource types.
var cfnTypeMapping = map[string]string{
	"ec2:instance":         "AWS::EC2::Instance",
	"ec2:security_group":   "AWS::EC2::SecurityGroup",
	"ec2:vpc":              "AWS::EC2::VPC",
	"ec2:subnet":           "AWS::EC2::Subnet",
	"ec2:internet_gateway": "AWS::EC2::InternetGateway",
	"iam:role":             "AWS::IAM::Role",
	"iam:user":             "AWS::IAM::User",
	"iam:policy":           "AWS::IAM::ManagedPolicy",
	"s3:bucket":            "AWS::S3::Bucket",
	"lambda:function":      "AWS::Lambda::Function",
	"rds:db_instance":      "AWS::RDS::DBInstance",
	"rds:db_cluster":       "AWS::RDS::DBCluster",
}

// cfnPropertyMapping maps our attribute names to CloudFormation property names.
var cfnPropertyMapping = map[string]map[string]string{
	"AWS::EC2::Instance": {
		"instanceType": "InstanceType",
		"imageId":      "ImageId",
		"subnetId":     "SubnetId",
	},
	"AWS::EC2::VPC": {
		"cidrBlock": "CidrBlock",
	},
	"AWS::EC2::Subnet": {
		"vpcId":            "VpcId",
		"cidrBlock":        "CidrBlock",
		"availabilityZone": "AvailabilityZone",
	},
	"AWS::EC2::SecurityGroup": {
		"groupName":   "GroupName",
		"vpcId":       "VpcId",
		"description": "GroupDescription",
	},
	"AWS::Lambda::Function": {
		"functionName": "FunctionName",
		"runtime":      "Runtime",
		"handler":      "Handler",
		"role":         "Role",
	},
	"AWS::RDS::DBInstance": {
		"dBInstanceIdentifier": "DBInstanceIdentifier",
		"engine":               "Engine",
		"dBInstanceClass":      "DBInstanceClass",
	},
	"AWS::S3::Bucket": {
		"bucketName": "BucketName",
	},
}

// Export writes the snapshot as a CloudFormation template.
func (e *CloudFormationExporter) Export(snapshot *engine.Snapshot, w io.Writer) error {
	template := map[string]any{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description": fmt.Sprintf("Reconstructed by CloudNecromancer at %s -- verify before deploying",
			snapshot.Timestamp.Format(time.RFC3339)),
		"Resources": buildCFNResources(snapshot),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(template); err != nil {
		return fmt.Errorf("cloudformation encode: %w", err)
	}
	return nil
}

func buildCFNResources(snapshot *engine.Snapshot) map[string]any {
	resources := make(map[string]any)

	for typeKey, resList := range snapshot.Resources {
		cfnType, known := cfnTypeMapping[typeKey]
		if !known {
			continue
		}

		propMapping := cfnPropertyMapping[cfnType]

		for _, res := range resList {
			logicalID := cfnLogicalID(typeKey, res.ResourceID)
			properties := make(map[string]any)

			for attrKey, attrVal := range res.Attributes {
				cfnKey := attrKey
				if propMapping != nil {
					if mapped, ok := propMapping[attrKey]; ok {
						cfnKey = mapped
					}
				}
				properties[cfnKey] = attrVal
			}

			resources[logicalID] = map[string]any{
				"Type":       cfnType,
				"Properties": properties,
				"Metadata": map[string]any{
					"CloudNecromancer": map[string]any{
						"OriginalResourceId": res.ResourceID,
						"State":              res.State,
					},
				},
			}
		}
	}

	return resources
}

func cfnLogicalID(typeKey, resourceID string) string {
	// CloudFormation logical IDs must be alphanumeric
	parts := strings.SplitN(typeKey, ":", 2)
	prefix := ""
	if len(parts) == 2 {
		prefix = strings.Title(parts[0]) + strings.Title(parts[1]) //nolint:staticcheck
	} else {
		prefix = strings.Title(typeKey) //nolint:staticcheck
	}

	safeID := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, resourceID)

	return prefix + safeID
}
