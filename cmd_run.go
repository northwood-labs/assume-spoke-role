package main

import (
	"os"

	errors "github.com/go-faster/errors"
	cli "github.com/jawher/mow.cli"
	"github.com/northwood-labs/awsutils"
	"github.com/northwood-labs/golang-utils/exiterrorf"

	"github.com/northwood-labs/assume-spoke-role/hubspoke"
)

func cmdRun(cmd *cli.Cmd) {
	cmd.LongDesc = `Perform the action of assuming roles and running an action.

Use environment variables to store parameter values consistently. CLI options
take precedence over environment variables.`

	const (
		defaultAWSRetries = 3
	)

	var (
		externalID   = os.Getenv("ASSUME_ROLE_EXTERNAL_ID")
		hubAccount   = os.Getenv("ASSUME_ROLE_HUB_ACCOUNT")
		hubRole      = os.Getenv("ASSUME_ROLE_HUB_ROLE")
		spokeAccount = os.Getenv("ASSUME_ROLE_SPOKE_ACCOUNT")
		spokeRole    = os.Getenv("ASSUME_ROLE_SPOKE_ROLE")

		externalIDFlag   = cmd.StringOpt("e external-id", externalID, "(ASSUME_ROLE_EXTERNAL_ID) The external ID value that is required by your hub and spoke policies, if any.") // lint:ignore-length
		hubAccountFlag   = cmd.StringOpt("h hub-account", hubAccount, "(ASSUME_ROLE_HUB_ACCOUNT) The 12-digit AWS account ID containing the HUB policy.")                         // lint:ignore-length
		spokeAccountFlag = cmd.StringOpt("s spoke-account", spokeAccount, "(ASSUME_ROLE_SPOKE_ACCOUNT) The 12-digit AWS account ID containing the SPOKE policy.")                 // lint:ignore-length
		hubRoleFlag      = cmd.StringOpt("H hub-role", hubRole, "(ASSUME_ROLE_HUB_ROLE) The name of the IAM role to assume in the HUB account.")                                  // lint:ignore-length
		spokeRoleFlag    = cmd.StringOpt("S spoke-role", spokeRole, "(ASSUME_ROLE_SPOKE_ROLE) The name of the IAM role to assume in the HUB account.")                            // lint:ignore-length
		retriesFlag      = cmd.IntOpt("r retries", defaultAWSRetries, "The maximum number of retries that the underlying AWS SDK should perform.")                                // lint:ignore-length
		verboseFlag      = cmd.BoolOpt("v verbose", false, "Enable verbose logging.")                                                                                             // lint:ignore-length
		cmdd             = cmd.StringsArg("COMMAND", []string{""}, "The command to run using the spoke policy.")
	)

	cmd.Spec = `[-e=<external-id>] [-h=<hub-account>] [-s=<spoke-account>] [-H=<hub-role>] ` +
		`[-S=<spoke-role>] [-r=<retries>] [--verbose] -- COMMAND...`

	cmd.Action = func() {
		// Get AWS credentials from environment.
		config, err := awsutils.GetAWSConfig(ctx, "", "", *retriesFlag, *verboseFlag)
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate a valid AWS configuration object"))
		}

		// Assume appropriate roles and return session credentials for the "Spoke" account.
		roleCredentials, _, err := hubspoke.GetSpokeCredentials(
			ctx,
			&config,
			*hubAccountFlag,
			*spokeAccountFlag,
			*hubRoleFlag,
			*spokeRoleFlag,
			*externalIDFlag,
		)
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate valid AWS credentials for the 'spoke' account"))
		}

		// Pass the spoke credentials to a CLI task.
		runCommand(roleCredentials, *cmdd)
	}
}
