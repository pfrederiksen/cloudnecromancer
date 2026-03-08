package parser_test

import (
	"testing"

	"github.com/pfrederiksen/cloudnecromancer/internal/parser"
	// Import services to trigger init() registrations.
	_ "github.com/pfrederiksen/cloudnecromancer/internal/parser/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisteredEventsContainsAllExpected(t *testing.T) {
	registered := parser.RegisteredEvents()

	expectedEvents := []string{
		// EC2
		"RunInstances", "TerminateInstances", "StartInstances", "StopInstances",
		"CreateSecurityGroup", "DeleteSecurityGroup", "AuthorizeSecurityGroupIngress",
		"CreateVpc", "DeleteVpc", "CreateSubnet", "CreateInternetGateway", "AttachInternetGateway",
		// IAM
		"CreateRole", "DeleteRole", "AttachRolePolicy", "DetachRolePolicy",
		"CreateUser", "DeleteUser", "CreatePolicy", "DeletePolicy",
		// S3
		"CreateBucket", "DeleteBucket", "PutBucketPolicy", "PutBucketVersioning", "PutPublicAccessBlock",
		// Lambda
		"CreateFunction20150331", "UpdateFunctionCode20150331v2", "DeleteFunction20150331",
		// RDS
		"CreateDBInstance", "DeleteDBInstance", "ModifyDBInstance", "CreateDBCluster", "DeleteDBCluster",
	}

	for _, event := range expectedEvents {
		assert.Contains(t, registered, event, "event %s should be registered", event)
	}

	assert.Len(t, registered, len(expectedEvents), "total registered events should match expected count")
}

func TestLookupReturnsCorrectParser(t *testing.T) {
	tests := []struct {
		eventName   string
		wantService string
	}{
		{"RunInstances", "ec2"},
		{"TerminateInstances", "ec2"},
		{"CreateRole", "iam"},
		{"CreateBucket", "s3"},
		{"CreateFunction20150331", "lambda"},
		{"CreateDBInstance", "rds"},
	}

	for _, tc := range tests {
		t.Run(tc.eventName, func(t *testing.T) {
			p, err := parser.Lookup(tc.eventName)
			require.NoError(t, err)
			assert.Equal(t, tc.wantService, p.Service())
		})
	}
}

func TestLookupUnknownEventReturnsError(t *testing.T) {
	_, err := parser.Lookup("NonExistentEvent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no parser registered")
}
