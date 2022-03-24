package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	errors "github.com/go-faster/errors"
	"github.com/gookit/color"
	cli "github.com/jawher/mow.cli"
	"github.com/northwood-labs/golang-utils/exiterrorf"
)

var (
	// Make referencable throughout.
	app *cli.Cli
	err error

	ctx = context.Background()

	// Color text.
	colorHeader = color.New(color.FgWhite, color.BgBlue, color.OpBold)

	// Buildtime variables.
	commit  string
	date    string
	version string
)

func main() {
	desc := `Assumes a 'spoke' role, from a 'hub' role, from a resource.

Example #1:

    export ASSUME_ROLE_EXTERNAL_ID=this-is-my-robot
    export ASSUME_ROLE_HUB_ACCOUNT=999999999999
    export ASSUME_ROLE_HUB_ROLE=robot-hub-role
    export ASSUME_ROLE_SPOKE_ROLE=robot-spoke-role

    aws-vault exec sys-robot --no-session -- \
        assume-spoke-role --spoke-account 888888888888 -- \
            aws sts get-caller-identity

Example #2:

    aws-vault exec sys-robot --no-session -- \
        assume-spoke-role \
            --hub-account 999999999999 \
            --spoke-account 888888888888 \
            --hub-role robot-hub-role \
            --spoke-role robot-spoke-role \
            --external-id this-is-my-robot \
            -- \
                aws sts get-caller-identity`

	app = cli.App("assume-spoke-role", desc)
	app.Version("version", fmt.Sprintf(
		"assume-spoke-role %s (%s/%s)",
		version,
		runtime.GOOS,
		runtime.GOARCH,
	))

	app.Command("run", "Perform the action of assuming roles and running an action.", cmdRun)
	app.Command("version", "Verbose information about the build.", cmdVersion)

	err = app.Run(os.Args)
	if err != nil {
		exiterrorf.ExitErrorf(errors.Wrap(err, "failed to execute application"))
	}
}
