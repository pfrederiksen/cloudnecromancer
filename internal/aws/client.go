package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
)

// CloudTrailAPI is the interface wrapping the AWS CloudTrail SDK calls we use.
// All production code goes through this interface so tests can mock it.
type CloudTrailAPI interface {
	LookupEvents(ctx context.Context, params *cloudtrail.LookupEventsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.LookupEventsOutput, error)
}
