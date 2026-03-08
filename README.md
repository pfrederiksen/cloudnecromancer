# CloudNecromancer

Reconstruct point-in-time AWS infrastructure snapshots by replaying CloudTrail events.

Given any historical timestamp, CloudNecromancer resurrects every resource that existed at that moment — EC2 instances, IAM roles, S3 buckets, Lambda functions, security groups, VPCs, RDS databases, and more — from create/modify/delete event chains stored in CloudTrail.

```
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 ░  ☠  CloudNecromancer  ☠  ░
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 Raising the dead since 2026
```

## Use Cases

- **Incident response** — "What was running at 3am before the breach?"
- **Compliance audits** — Point-in-time inventory for any past date
- **Post-incident timelines** — Full infrastructure state reconstruction
- **Drift analysis** — Compare two timestamps to see what changed

## Install

### Homebrew (macOS / Linux)

```bash
brew tap pfrederiksen/tap
brew install cloudnecromancer
```

### From release binaries

Download the latest release for your platform from [Releases](https://github.com/pfrederiksen/cloudnecromancer/releases).

| Platform | Architecture |
|----------|-------------|
| macOS | Intel (amd64), Apple Silicon (arm64) |
| Linux | amd64 |

### From source

```bash
go install github.com/pfrederiksen/cloudnecromancer@latest
```

> **Note:** Requires CGO and DuckDB C library headers (`build-essential` on Linux).

### Build locally

```bash
git clone https://github.com/pfrederiksen/cloudnecromancer.git
cd cloudnecromancer
make build
# Binary at ./bin/cloudnecromancer
```

## Quick Start

```bash
# 1. Fetch CloudTrail events into a local database
cloudnecromancer fetch \
  --account-id 123456789012 \
  --regions us-east-1,us-west-2 \
  --start 2026-01-01T00:00:00Z \
  --end 2026-03-01T00:00:00Z

# 2. Resurrect infrastructure at a specific point in time
cloudnecromancer resurrect --at 2026-02-15T03:00:00Z

# 3. Compare two points in time
cloudnecromancer diff \
  --from 2026-01-01T00:00:00Z \
  --to 2026-02-15T03:00:00Z
```

## Commands

### `fetch` — Pull CloudTrail events into local DB

```bash
cloudnecromancer fetch \
  --account-id 123456789012 \
  --region us-east-1 \
  --start 2026-01-01T00:00:00Z \
  --end 2026-03-01T00:00:00Z \
  [--regions us-east-1,us-west-2,eu-west-1] \
  [--profile my-aws-profile] \
  [--db ./necromancer.db]
```

Fetches CloudTrail management events across one or more regions concurrently and stores them in a local DuckDB database. Events are deduplicated by event ID, so re-running fetch for overlapping time ranges is safe.

**Example output:**

```
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 ░  ☠  CloudNecromancer  ☠  ░
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 Raising the dead since 2026

Fetching CloudTrail events...
 us-east-1  ████████████████████████████████ 100% | 12,847 events
 us-west-2  ████████████████████████████████ 100% |  3,291 events

Summary:
  Events fetched:  16,138
  Services:        ec2, iam, s3, lambda, rds
  Date range:      2026-01-01 to 2026-03-01
  Database:        ./necromancer.db (24 MB)
```

### `resurrect` — Reconstruct infrastructure at a point in time

```bash
cloudnecromancer resurrect \
  --at 2026-02-15T03:00:00Z \
  [--services ec2,iam,s3] \
  [--region us-east-1] \
  [--format json|terraform|cloudformation|cdk|pulumi|ocsf|csv] \
  [--output ./snapshot.json] \
  [--include-dead] \
  [--ritual] \
  [--db ./necromancer.db]
```

Replays all events up to the given timestamp and reconstructs the state of every resource. The `--ritual` flag adds an animated ASCII skull with a "RAISING THE DEAD..." typewriter effect.

**Example JSON output** (`--format json`):

```json
{
  "timestamp": "2026-02-15T03:00:00Z",
  "account_id": "123456789012",
  "regions": ["us-east-1", "us-west-2"],
  "resources": {
    "ec2:instance": [
      {
        "resource_id": "i-0abc123def456789",
        "state": "running",
        "attributes": {
          "instance_type": "t3.medium",
          "image_id": "ami-0abcdef1234567890",
          "vpc_id": "vpc-0123456789abcdef0",
          "subnet_id": "subnet-0123456789abcdef0"
        },
        "created_at": "2026-01-10T14:22:00Z",
        "last_modified": "2026-02-01T09:15:00Z"
      }
    ],
    "iam:role": [
      {
        "resource_id": "WebAppRole",
        "state": "active",
        "attributes": {
          "attached_policies": [
            "arn:aws:iam::123456789012:policy/S3ReadOnly",
            "arn:aws:iam::aws:policy/CloudWatchLogsFullAccess"
          ]
        },
        "created_at": "2026-01-05T10:00:00Z",
        "last_modified": "2026-01-20T16:30:00Z"
      }
    ],
    "s3:bucket": [
      {
        "resource_id": "prod-data-lake-2026",
        "state": "active",
        "attributes": {
          "versioning": "Enabled",
          "public_access_block": true
        },
        "created_at": "2026-01-02T08:00:00Z",
        "last_modified": "2026-01-15T12:00:00Z"
      }
    ]
  },
  "summary": {
    "total_resources": 47,
    "by_service": {
      "ec2": 23,
      "iam": 12,
      "s3": 5,
      "lambda": 4,
      "rds": 3
    },
    "by_state": {
      "active": 42,
      "running": 5
    }
  }
}
```

**Example HCL output** (`--format hcl`):

```hcl
# RECONSTRUCTED — verify before applying
# Generated by CloudNecromancer at 2026-02-15T03:00:00Z

import {
  to = aws_instance.i_0abc123def456789
  id = "i-0abc123def456789"
}

resource "aws_instance" "i_0abc123def456789" {
  instance_type = "t3.medium"
  ami           = "ami-0abcdef1234567890"
  subnet_id     = "subnet-0123456789abcdef0"
}

import {
  to = aws_iam_role.WebAppRole
  id = "WebAppRole"
}

resource "aws_iam_role" "WebAppRole" {
}

import {
  to = aws_s3_bucket.prod_data_lake_2026
  id = "prod-data-lake-2026"
}

resource "aws_s3_bucket" "prod_data_lake_2026" {
}
```

**Example CloudFormation output** (`--format cloudformation`):

```json
{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Reconstructed by CloudNecromancer at 2026-02-15T03:00:00Z -- verify before deploying",
  "Resources": {
    "Ec2Instancei0abc123def456789": {
      "Type": "AWS::EC2::Instance",
      "Properties": {
        "InstanceType": "t3.medium",
        "ImageId": "ami-0abcdef1234567890",
        "SubnetId": "subnet-0123456789abcdef0"
      }
    },
    "S3Bucketproddatalake2026": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "BucketName": "prod-data-lake-2026"
      }
    }
  }
}
```

**Example CDK output** (`--format cdk`):

```typescript
// RECONSTRUCTED -- verify before deploying
// Generated by CloudNecromancer at 2026-02-15T03:00:00Z

import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';

export class CloudNecromancerStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    new ec2.CfnInstance(this, "i-0abc123def456789", {
      instanceType: "t3.medium",
      imageId: "ami-0abcdef1234567890",
      subnetId: "subnet-0123456789abcdef0",
    });

    new s3.CfnBucket(this, "prod-data-lake-2026", {
      bucketName: "prod-data-lake-2026",
    });
  }
}
```

**Example Pulumi output** (`--format pulumi`):

```typescript
// RECONSTRUCTED -- verify before deploying
// Generated by CloudNecromancer at 2026-02-15T03:00:00Z

import * as aws from "@pulumi/aws";

const i_0abc123def456789 = new aws.ec2.Instance("i-0abc123def456789", {
    instanceType: "t3.medium",
    ami: "ami-0abcdef1234567890",
    subnetId: "subnet-0123456789abcdef0",
});

const prod_data_lake_2026 = new aws.s3.Bucket("prod-data-lake-2026", {
    bucket: "prod-data-lake-2026",
});
```

**Example CSV output** (`--format csv`):

```
resource_id,resource_type,service,state,region,account_id,created_at,last_modified,snapshot_timestamp,attributes_json
i-0abc123def456789,instance,ec2,running,us-east-1,123456789012,2026-01-10T14:22:00Z,2026-02-01T09:15:00Z,2026-02-15T03:00:00Z,"{""instance_type"":""t3.medium""}"
WebAppRole,role,iam,active,us-east-1,123456789012,2026-01-05T10:00:00Z,2026-01-20T16:30:00Z,2026-02-15T03:00:00Z,"{""attached_policies"":[""arn:aws:iam::123456789012:policy/S3ReadOnly""]}"
```

### `diff` — Compare infrastructure between two timestamps

```bash
cloudnecromancer diff \
  --from 2026-01-01T00:00:00Z \
  --to 2026-02-15T03:00:00Z \
  [--format table|json] \
  [--db ./necromancer.db]
```

Generates snapshots at both timestamps and reports what changed.

**Example output** (default table format):

```
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 ░  ☠  CloudNecromancer  ☠  ░
 ░░░░░░░░░░░░░░░░░░░░░░░░░░
 Raising the dead since 2026

Diff: 2026-01-01T00:00:00Z → 2026-02-15T03:00:00Z

+ ADDED (12 resources)
  + ec2:instance     i-0abc123def456789    t3.medium    us-east-1
  + ec2:instance     i-0def456789abc123    t3.large     us-west-2
  + s3:bucket        prod-data-lake-2026                us-east-1
  + lambda:function  process-orders        python3.12   us-east-1
  ...

- REMOVED (3 resources)
  - ec2:instance     i-0old999888777666    t2.micro     us-east-1
  - iam:role         LegacyAdminRole                    us-east-1
  - s3:bucket        temp-migration-2025                us-east-1

~ MODIFIED (8 resources)
  ~ iam:role         WebAppRole            us-east-1
      attached_policies: +arn:aws:iam::123456789012:policy/S3ReadOnly
  ~ ec2:security_group  sg-0123456789abcdef0    us-east-1
      ingress: +0.0.0.0/0:443
```

### `export` — Re-export an existing snapshot

```bash
cloudnecromancer export \
  --input ./snapshot.json \
  --format hcl \
  --output ./snapshot.tf
```

### `info` — Show database stats

```bash
cloudnecromancer info [--db ./necromancer.db]
```

**Example output:**

```
Database: ./necromancer.db
  Events:     16,138
  Date range: 2026-01-01T00:00:00Z to 2026-03-01T00:00:00Z
  Services:   ec2, iam, s3, lambda, rds
  Regions:    us-east-1, us-west-2
  Size:       24 MB
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--db` | `./necromancer.db` | Path to DuckDB database file |
| `--profile` | *(default chain)* | AWS profile to use |
| `--quiet` | `false` | Suppress banner and non-essential output |
| `--verbose` | `false` | Enable verbose logging |

## AWS Permissions

CloudNecromancer requires read-only access to CloudTrail. See [AWS_PERMISSIONS.md](AWS_PERMISSIONS.md) for the minimal IAM policy.

Quick version:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudtrail:LookupEvents",
        "cloudtrail:GetTrailStatus",
        "cloudtrail:DescribeTrails"
      ],
      "Resource": "*"
    }
  ]
}
```

Credentials are resolved via the standard AWS SDK credential chain: environment variables, `~/.aws/credentials`, IAM instance role, etc.

## Output Formats

| Format | Flag | Description |
|--------|------|-------------|
| JSON | `--format json` | Full snapshot with nested resource attributes |
| Terraform | `--format terraform` | Terraform-importable HCL with `import` + `resource` blocks (aliases: `hcl`, `tf`) |
| CloudFormation | `--format cloudformation` | AWS CloudFormation JSON template (alias: `cfn`) |
| CDK | `--format cdk` | AWS CDK TypeScript stack using L1 constructs |
| Pulumi | `--format pulumi` | Pulumi TypeScript program using `@pulumi/aws` |
| OCSF | `--format ocsf` | OCSF Inventory Info events (class_uid 5001), newline-delimited JSON |
| CSV | `--format csv` | Splunk lookup table format |

### Using CSV with Splunk

Upload the CSV output as a Splunk lookup table, then correlate with CloudTrail logs:

```spl
| inputlookup cloudnecromancer_lookup.csv
| join resource_id [search index=cloudtrail earliest=-30d]
```

## Supported Services

| Service | Create | Update | Delete |
|---------|--------|--------|--------|
| EC2 (instances, VPCs, subnets, SGs, IGWs) | Yes | Yes | Yes |
| IAM (roles, users, policies) | Yes | Yes | Yes |
| S3 (buckets, policies, versioning) | Yes | Yes | Yes |
| Lambda (functions) | Yes | Yes | Yes |
| RDS (instances, clusters) | Yes | Yes | Yes |

## How It Works

1. **Fetch** — CloudNecromancer pulls CloudTrail management events via `LookupEvents` and stores them in an embedded DuckDB database. Multi-region fetches run concurrently.

2. **Parse** — Each CloudTrail event is routed through a service-specific parser (registered at startup) that extracts a `ResourceDelta`: the action (create/update/delete), resource ID, and relevant attributes.

3. **Replay** — To reconstruct state at time T, the engine queries all events before T (ordered chronologically) and applies each delta to an in-memory resource map. Creates insert, updates merge attributes, deletes mark resources as terminated.

4. **Export** — The final snapshot is serialized in the requested format.

## Development

```bash
make build    # Build binary to ./bin/cloudnecromancer
make test     # Run all tests
make lint     # Run golangci-lint
make snapshot # Build cross-platform binaries (GoReleaser)
```

### Adding a new service parser

1. Create `internal/parser/services/myservice.go`
2. Implement the `Parser` interface
3. Register in `init()` with `parser.Register(&MyServiceParser{})`
4. Add test fixtures to `testdata/`
5. Add table-driven tests in `internal/parser/services/myservice_test.go`

### Adding a new exporter

1. Create `internal/export/myformat.go`
2. Implement the `Exporter` interface (`Export(snapshot, writer) error`)
3. Register it in `GetExporter()` in `internal/export/exporter.go`
4. Add tests in `internal/export/export_test.go`
5. Update `--format` flag descriptions in `cmd/resurrect.go` and `cmd/export.go`

## License

MIT
