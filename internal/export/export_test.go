package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pfrederiksen/cloudnecromancer/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSnapshot() *engine.Snapshot {
	return &engine.Snapshot{
		Timestamp: time.Date(2026, 2, 15, 3, 0, 0, 0, time.UTC),
		AccountID: "123456789012",
		Regions:   []string{"us-east-1"},
		Resources: map[string][]engine.Resource{
			"ec2:instance": {
				{
					ResourceID:   "i-abc123",
					State:        "active",
					Attributes:   map[string]any{"instanceType": "t3.medium", "imageId": "ami-12345"},
					CreatedAt:    time.Date(2026, 1, 10, 14, 22, 0, 0, time.UTC),
					LastModified: time.Date(2026, 2, 1, 9, 15, 0, 0, time.UTC),
				},
			},
			"s3:bucket": {
				{
					ResourceID:   "my-bucket",
					State:        "active",
					Attributes:   map[string]any{"versioning": "Enabled"},
					CreatedAt:    time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC),
					LastModified: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		Summary: engine.Summary{
			TotalResources: 2,
			ByService:      map[string]int{"ec2": 1, "s3": 1},
			ByState:        map[string]int{"active": 2},
		},
	}
}

func TestJSONExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &JSONExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	var parsed engine.Snapshot
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "123456789012", parsed.AccountID)
	assert.Equal(t, 2, parsed.Summary.TotalResources)
	assert.Len(t, parsed.Resources["ec2:instance"], 1)
	assert.Equal(t, "i-abc123", parsed.Resources["ec2:instance"][0].ResourceID)
}

func TestHCLExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &HCLExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	output := buf.String()
	// Should contain import block
	assert.Contains(t, output, "import {")
	assert.Contains(t, output, `id = "i-abc123"`)
	// Should contain resource block with mapped attribute names
	assert.Contains(t, output, `resource "aws_instance"`)
	assert.Contains(t, output, "instance_type")
	// Should contain the generated header
	assert.Contains(t, output, "RECONSTRUCTED")
	// S3 bucket should also be present
	assert.Contains(t, output, `resource "aws_s3_bucket"`)
}

func TestHCLExporterUnknownType(t *testing.T) {
	snap := &engine.Snapshot{
		Timestamp: time.Now(),
		Resources: map[string][]engine.Resource{
			"mystery:widget": {
				{ResourceID: "w-123", State: "active", Attributes: map[string]any{"foo": "bar"}},
			},
		},
	}
	var buf bytes.Buffer
	exp := &HCLExporter{}
	require.NoError(t, exp.Export(snap, &buf))
	output := buf.String()
	assert.Contains(t, output, "Unsupported resource type: mystery:widget")
	assert.Contains(t, output, "w-123")
	assert.Contains(t, output, "omitted for safety")
}

func TestHCLExporterInjectionPrevention(t *testing.T) {
	t.Run("malicious attribute key rejected", func(t *testing.T) {
		snap := &engine.Snapshot{
			Timestamp: time.Now(),
			Resources: map[string][]engine.Resource{
				"ec2:instance": {
					{
						ResourceID: "i-safe",
						State:      "active",
						Attributes: map[string]any{
							"safe_key":                           "good",
							"bad = \"pwned\"\n}\nresource \"x\"": "evil",
						},
					},
				},
			},
		}
		var buf bytes.Buffer
		exp := &HCLExporter{}
		require.NoError(t, exp.Export(snap, &buf))
		output := buf.String()
		assert.NotContains(t, output, "pwned")
		assert.NotContains(t, output, "evil")
	})

	t.Run("newline in typeKey sanitized in comment", func(t *testing.T) {
		snap := &engine.Snapshot{
			Timestamp: time.Now(),
			Resources: map[string][]engine.Resource{
				"evil\nresource \"null\" \"x\" {": {
					{ResourceID: "w-1", State: "active", Attributes: map[string]any{}},
				},
			},
		}
		var buf bytes.Buffer
		exp := &HCLExporter{}
		require.NoError(t, exp.Export(snap, &buf))
		output := buf.String()
		// Verify every non-empty line containing the injected text starts with #
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "null") {
				assert.True(t, strings.HasPrefix(line, "#"), "injected content escaped comment: %s", line)
			}
		}
	})
}

func TestOCSFExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &OCSFExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	// Each resource gets one NDJSON line
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	for _, line := range lines {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		// Required OCSF fields
		assert.Equal(t, float64(5001), event["class_uid"])
		assert.Equal(t, "Inventory Info", event["class_name"])
		assert.NotNil(t, event["metadata"])
		assert.NotNil(t, event["cloud"])
		assert.NotNil(t, event["resource"])
	}
}

func TestCSVExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &CSVExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Header + 2 data rows
	assert.Len(t, records, 3)
	assert.Equal(t, "resource_id", records[0][0])
	assert.Equal(t, "attributes_json", records[0][9])

	// Verify data rows exist (order may vary due to map iteration)
	ids := []string{records[1][0], records[2][0]}
	assert.Contains(t, ids, "i-abc123")
	assert.Contains(t, ids, "my-bucket")
}

func TestCloudFormationExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &CloudFormationExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	var template map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &template))
	assert.Equal(t, "2010-09-09", template["AWSTemplateFormatVersion"])
	assert.Contains(t, template["Description"].(string), "CloudNecromancer")

	resources, ok := template["Resources"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, resources, 2) // instance + bucket

	// Verify EC2 instance resource
	found := false
	for _, res := range resources {
		r := res.(map[string]any)
		if r["Type"] == "AWS::EC2::Instance" {
			props := r["Properties"].(map[string]any)
			assert.Equal(t, "t3.medium", props["InstanceType"])
			found = true
		}
	}
	assert.True(t, found, "expected AWS::EC2::Instance in template")
}

func TestCDKExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &CDKExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	output := buf.String()
	assert.Contains(t, output, "import * as cdk from 'aws-cdk-lib'")
	assert.Contains(t, output, "CloudNecromancerStack")
	assert.Contains(t, output, "CfnInstance")
	assert.Contains(t, output, `"t3.medium"`)
	assert.Contains(t, output, "CfnBucket")
}

func TestPulumiExporter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer
	exp := &PulumiExporter{}
	require.NoError(t, exp.Export(snap, &buf))

	output := buf.String()
	assert.Contains(t, output, `import * as aws from "@pulumi/aws"`)
	assert.Contains(t, output, "aws.ec2.Instance")
	assert.Contains(t, output, `"t3.medium"`)
	assert.Contains(t, output, "aws.s3.Bucket")
}

func TestGetExporter(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"json", false},
		{"hcl", false},
		{"ocsf", false},
		{"csv", false},
		{"cloudformation", false},
		{"cfn", false},
		{"cdk", false},
		{"pulumi", false},
		{"xml", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			exp, err := GetExporter(tt.format)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, exp)
			}
		})
	}
}
