package main

import (
	"context"
	"os"

	errors "github.com/go-faster/errors"
	"github.com/northwood-labs/awsutils"
	"github.com/northwood-labs/golang-utils/exiterrorf"
	flag "github.com/spf13/pflag"

	"github.com/northwood-labs/assume-spoke-role/hubspoke"
)

func main() {
	ctx := context.Background()
	// logger := log.GetStdTextLogger()
	externalID := os.Getenv("ASSUME_ROLE_EXTERNAL_ID")
	hubAccount := os.Getenv("ASSUME_ROLE_HUB_ACCOUNT")
	hubRole := os.Getenv("ASSUME_ROLE_HUB_ROLE")
	spokeAccount := os.Getenv("ASSUME_ROLE_SPOKE_ACCOUNT")
	spokeRole := os.Getenv("ASSUME_ROLE_SPOKE_ROLE")

	// Flags
	externalIDFlag := flag.StringP("external-id", "e", "", "The external ID value that is required by your hub and spoke policies, if any. Takes precedence over the `ASSUME_ROLE_EXTERNAL_ID` environment variable.") // lint:ignore-length
	hubAccountFlag := flag.StringP("hub-account", "h", "", "The 12-digit numeric ID of the AWS account containing the HUB policy. Takes precedence over the `ASSUME_ROLE_HUB_ACCOUNT` environment variable.")          // lint:ignore-length
	spokeAccountFlag := flag.StringP("spoke-account", "s", "", "The 12-digit numeric ID of the AWS account containing the SPOKE policy. Takes precedence over the `ASSUME_ROLE_SPOKE_ACCOUNT` environment variable.")  // lint:ignore-length
	hubRoleFlag := flag.String("hub-role", "", "The name of the IAM role to assume in the HUB account. Takes precedence over the `ASSUME_ROLE_HUB_ROLE` environment variable.")                                        // lint:ignore-length
	spokeRoleFlag := flag.String("spoke-role", "", "The name of the IAM role to assume in the HUB account. Takes precedence over the `ASSUME_ROLE_SPOKE_ROLE` environment variable.")                                  // lint:ignore-length
	retriesFlag := flag.IntP("retries", "r", 3, "The maximum number of retries that the underlying AWS SDK should perform.")                                                                                           // lint:ignore-length
	verboseFlag := flag.CountP("verbose", "v", "Enable verbose logging. Can be stacked up to `-vvv`.")                                                                                                                 // lint:ignore-length

	flag.Parse()

	// Do we have a hub account?
	if *hubAccountFlag == "" {
		// if *verboseFlag == 2 {
		// 	logger.Info().Str("ASSUME_ROLE_HUB_ACCOUNT", hubAccount).Msg("Flag --hub-account not set. Reading from `ASSUME_ROLE_HUB_ACCOUNT`.")
		// }

		*hubAccountFlag = hubAccount
	}

	// Do we have a spoke account?
	if *spokeAccountFlag == "" {
		// if *verboseFlag == 2 {
		// 	logger.Info().Str("ASSUME_ROLE_SPOKE_ACCOUNT", hubAccount).Msg("Flag --spoke-account not set. Reading from `ASSUME_ROLE_SPOKE_ACCOUNT`.")
		// }

		*spokeAccountFlag = spokeAccount
	}

	// Do we have a hub role?
	if *hubRoleFlag == "" {
		*hubRoleFlag = hubRole
	}

	// Do we have a spoke role?
	if *spokeRoleFlag == "" {
		*spokeRoleFlag = spokeRole
	}

	// Do we have an external ID?
	if *externalIDFlag == "" {
		*externalIDFlag = externalID
	}

	// Get AWS credentials from environment.
	config, err := awsutils.GetAWSConfig(ctx, "", "", *retriesFlag, *verboseFlag == 3)
	if err != nil {
		exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate a valid AWS configuration object"))
	}

	// Assume appropriate roles and return session credentials for the "Spoke" account.
	roleCredentials, _, err := hubspoke.GetSpokeCredentials(
		ctx,
		&config,
		*hubAccountFlag,
		*spokeAccountFlag,
		hubRoleFlag,
		spokeRoleFlag,
		externalIDFlag,
	)
	if err != nil {
		exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate valid AWS credentials for the 'spoke' account"))
	}

	// Pass the spoke credentials to a CLI task.
	runCommand(roleCredentials, flag.CommandLine.Args())
}
