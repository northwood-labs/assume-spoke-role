package hubspoke

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	errors "github.com/go-faster/errors"
)

// SpokeCredentialsInput is an input object for the GetSpokeCredentials
// function.
type SpokeCredentialsInput struct {
	// (Required) A context.Context object.
	Context context.Context

	// (Required) An AWS SDK v2 configuration object.
	Config *aws.Config

	// (Required) The AWS Account ID of the "hub" account.
	HubAccountID string

	// (Required) The AWS Account ID of the "spoke" account.
	SpokeAccountID string

	// (Required) The name of the role to assume inside the "hub" account.
	HubRoleName string

	// (Required) The name of the role to assume inside the "spoke" account.
	SpokeRoleName string

	// (Optional) A string identifier that should match what the IAM policies
	// require, if anything. If empty, an empty string will be passed along.
	ExternalID string

	// (Optional) A string identifier to use to represent the
	// user/software/role, and will show up under the `UserId` result of `aws
	// sts get-caller-identity`. If empty, a random string will be generated.
	SessionString string
}

// GetSpokeCredentials accepts a GetSpokeCredentialsInput object, and returns a
// set of STS session credentials for the spoke account.
func GetSpokeCredentials(input *SpokeCredentialsInput) (*types.Credentials, aws.Config, error) {
	emptyCredentials := types.Credentials{}
	emptyConfig := aws.Config{}

	sessionName := input.SessionString

	hubRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", input.HubAccountID, input.HubRoleName)
	spokeRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", input.SpokeAccountID, input.SpokeRoleName)

	// Assume the HUB role.
	stsHubClient := sts.NewFromConfig(*input.Config)
	input.Config.Credentials = aws.NewCredentialsCache(
		stscreds.NewAssumeRoleProvider(stsHubClient, hubRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = sessionName
            o.ExternalID = aws.String(input.ExternalID)
		}),
	)

	// Assume the SPOKE role.
	stsSpokeClient := sts.NewFromConfig(*input.Config)
	response, err := stsSpokeClient.AssumeRole(input.Context, &sts.AssumeRoleInput{
		RoleArn:         aws.String(spokeRoleARN),
		RoleSessionName: aws.String(fmt.Sprintf("%s-%s", input.SpokeAccountID, sessionName)),
		ExternalId:      aws.String(input.ExternalID),
	})
	if err != nil { // lint:allow_cuddling
		return &emptyCredentials, emptyConfig, errors.Wrap(err, fmt.Sprintf(
			"error assuming '%s' role in account %s",
			spokeRoleARN,
			input.SpokeAccountID,
		))
	}

	input.Config.Credentials = aws.NewCredentialsCache(
		stscreds.NewAssumeRoleProvider(stsSpokeClient, spokeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = fmt.Sprintf("%s-%s", input.SpokeAccountID, sessionName)
			o.ExternalID = aws.String(input.ExternalID)
		}),
	)

	return response.Credentials, *input.Config, nil
}
