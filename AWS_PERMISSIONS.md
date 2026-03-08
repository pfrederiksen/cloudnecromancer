# AWS Permissions

CloudNecromancer requires read-only access to AWS CloudTrail. Below is the minimal IAM policy.

## Minimal IAM Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudNecromancerReadOnly",
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

## Actions Explained

| Action | Purpose |
|--------|---------|
| `cloudtrail:LookupEvents` | Query management and data events by time range |
| `cloudtrail:GetTrailStatus` | Check if a trail is actively logging |
| `cloudtrail:DescribeTrails` | List configured trails and their regions |

## Credential Resolution

CloudNecromancer uses the standard AWS SDK v2 credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`)
2. Shared credentials file (`~/.aws/credentials`)
3. AWS SSO / IAM Identity Center
4. IAM instance role (EC2, ECS, Lambda)

Use `--profile` to select a named profile from your AWS config.

## Security Notes

- CloudNecromancer never writes to AWS — all operations are read-only
- Fetched events are stored locally in a DuckDB database file
- No credentials are stored by the tool — they are resolved per invocation via the SDK
- The database may contain sensitive resource metadata (instance IDs, role names, bucket names) — protect it accordingly
