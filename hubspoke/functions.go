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

// GetSpokeCredentials accepts an AWS Config object, a hub and spoke account ID,
// a hub and spoke role name, and returns a set of STS session credentials for
// the spoke account.
func GetSpokeCredentials(
	ctx context.Context,
	config *aws.Config,
	hubAccountID,
	spokeAccountID,
	hubRoleName,
	spokeRoleName,
	externalID string,
) (*types.Credentials, aws.Config, error) {
	emptyCredentials := types.Credentials{}
	emptyConfig := aws.Config{}

	rand.Seed(time.Now().UnixNano())

	sessionName := randomString(32)
	hubRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", hubAccountID, hubRoleName)
	spokeRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", spokeAccountID, spokeRoleName)

	// Assume the HUB role.
	stsHubClient := sts.NewFromConfig(*config)
	config.Credentials = aws.NewCredentialsCache(
		stscreds.NewAssumeRoleProvider(stsHubClient, hubRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.ExternalID = aws.String(externalID)
		}),
	)

	// Assume the SPOKE role.
	stsSpokeClient := sts.NewFromConfig(*config)
	response, err := stsSpokeClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(spokeRoleARN),
		RoleSessionName: aws.String(fmt.Sprintf("%s-%s", spokeAccountID, sessionName)),
		ExternalId:      aws.String(externalID),
	})
	if err != nil { // lint:allow_cuddling
		return &emptyCredentials, emptyConfig, errors.Wrap(err, fmt.Sprintf(
			"error assuming '%s' role in account %s",
			spokeRoleARN,
			spokeAccountID,
		))
	}

	config.Credentials = aws.NewCredentialsCache(
		stscreds.NewAssumeRoleProvider(stsSpokeClient, spokeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = fmt.Sprintf("%s-%s", spokeAccountID, sessionName)
			o.ExternalID = aws.String(externalID)
		}),
	)

	return response.Credentials, *config, nil
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
