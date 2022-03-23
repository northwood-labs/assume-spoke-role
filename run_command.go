package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

func runCommand(creds *types.Credentials, args []string) {
	cmd := exec.Command(args[0], args[1:]...) // lint:allow_possible_insecure

	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", *creds.AccessKeyId),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", *creds.SecretAccessKey),
		fmt.Sprintf("AWS_SECURITY_TOKEN=%s", *creds.SessionToken),
		fmt.Sprintf("AWS_SESSION_TOKEN=%s", *creds.SessionToken),
		fmt.Sprintf("AWS_SESSION_EXPIRATION=%s", creds.Expiration.String()),
	)

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
