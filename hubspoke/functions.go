package hubspoke

import (
	"context"
	"fmt"
	"math/rand"
	"time"

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
func GetSpokeCredentials(o SpokeCredentialsInput) (*types.Credentials, aws.Config, error) {
	emptyCredentials := types.Credentials{}
	emptyConfig := aws.Config{}

	sessionName := o.SessionString
	if sessionName == "" {
		rand.Seed(time.Now().UnixNano())
		sessionName = randomString(32)
	}

	hubRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", o.HubAccountID, o.HubRoleName)
	spokeRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", o.SpokeAccountID, o.SpokeRoleName)

	// Assume the HUB role.
	stsHubClient := sts.NewFromConfig(*o.Config)
	o.Config.Credentials = aws.NewCredentialsCache(
		stscreds.NewAssumeRoleProvider(stsHubClient, hubRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.ExternalID = aws.String(*o.ExternalID)
		}),
	)

	// Assume the SPOKE role.
	stsSpokeClient := sts.NewFromConfig(*o.Config)
	response, err := stsSpokeClient.AssumeRole(o.Context, &sts.AssumeRoleInput{
		RoleArn:         aws.String(spokeRoleARN),
		RoleSessionName: aws.String(fmt.Sprintf("%s-%s", o.SpokeAccountID, sessionName)),
		ExternalId:      aws.String(o.ExternalID),
	})
	if err != nil { // lint:allow_cuddling
		return &emptyCredentials, emptyConfig, errors.Wrap(err, fmt.Sprintf(
			"error assuming '%s' role in account %s",
			spokeRoleARN,
			o.SpokeAccountID,
		))
	}

	o.Config.Credentials = aws.NewCredentialsCache(
		stscreds.NewAssumeRoleProvider(stsSpokeClient, spokeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = fmt.Sprintf("%s-%s", o.SpokeAccountID, sessionName)
			o.ExternalID = aws.String(*o.ExternalID)
		}),
	)

	return response.Credentials, *o.Config, nil
}

func randomString(l int) string {
	const (
		// Pool of characters to choose from when generating a random session ID.
		pool = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	)

	bytes := make([]byte, l)

	for i := 0; i < l; i++ {
		bytes[i] = pool[rand.Intn(len(pool))]
	}

	return string(bytes)
}
