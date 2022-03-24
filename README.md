# Assume IAM Roles using the Hub-Spoke model

When you have _several_ AWS accounts to manage, you can keep things secure and locked-down by adopting the hub-spoke model of assuming IAM roles across accounts.

1. You have a user (or a bot if you're automating) with permission to assume the "Hub" role in an account (doesn't need to be the same as the user).

1. From the "Hub", the user can then traverse to a "Spoke" account to perform the actions that are granted to an assumer of that "Hub" role.

This model is recommended by AWS (read below). You will need to provision the roles (via [Service Control Policies](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_scps.html) or perhaps [Terraform](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)), but there's only one user (per workload) that you need to grant permissions to.

## What is the Hub-Spoke model?

In a multi-account setup, optionally managed with AWS Organizations and AWS SSO, think of it like a bicycle wheel: One hub, many spokes.

For example, if an automated process (e.g., a robot) needs to perform the same kinds of actions in every account you own (e.g., security analysis, reporting, account inventory), you would set up:

1. A "service account" for the robot/job/process/whatever.
1. Designate one account as the "hub". You need to connect here before connecting to anything else. It is conceptually similar to a "jump box".
1. All accounts where actions need to be performed are the "spokes". They would all have the same policy that can be assumed by the user connecting through the hub account and out to spoke accounts.

In larger setups that use AWS Control Tower, these policies can be provisioned with _Service Control Policies_ (SCPs). In smaller setups, you can use tools like Terraform or the AWS CLI for automation.

## Install as a CLI tool

1. You must have the Golang toolchain installed first.

    ```bash
    brew install go
    ```

1. Add `$GOPATH/bin` to your `$PATH` environment variable. By default (i.e., without configuration), `$GOPATH` is defined as `$HOME/go`.

    ```bash
    export PATH="$PATH:$GOPATH/bin"
    ```

1. Once you've done everything above, you can use `go get`.

    ```bash
    go get github.com/northwood-labs/assume-spoke-role
    ```

## Usage as CLI Tool

```bash
# Learn how it works.
assume-spoke-role --help
```

Run a command in another account (assuming you have permissions to assume a role). The ` -- ` marker signifies the end of passing options, and to begin treating subsequent text as the command to run with those credentials.

Assuming you're using [AWS Vault](https://github.com/99designs/aws-vault) to manage your credentials, and want to manage common configurations via environment variables:

```bash
export ASSUME_ROLE_EXTERNAL_ID=this-is-my-robot
export ASSUME_ROLE_HUB_ACCOUNT=999999999999
export ASSUME_ROLE_HUB_ROLE=robot-hub-role
export ASSUME_ROLE_SPOKE_ROLE=robot-spoke-role

# Using your local credentials (sys-robot), assume a role in the "HUB" account, before
# pivoting to a "SPOKE" account, then executing a command with those "SPOKE"
# credentials.
aws-vault exec sys-robot --no-session -- \
    assume-spoke-role --spoke-account 888888888888 -- \
        aws sts get-caller-identity
```

Or, if you want to more explicitly rely on CLI parameters rather than environment variables:

```bash
aws-vault exec sys-robot --no-session -- \
    assume-spoke-role \
        --hub-account 999999999999 \
        --spoke-account 888888888888 \
        --hub-role robot-hub-role \
        --spoke-role robot-spoke-role \
        --external-id this-is-my-robot \
        -- \
            aws sts get-caller-identity
```

## Usage as Library

This can also be used as a library in your own applications for generating a set of STS credentials.

```go
import "github.com/northwood-labs/assume-spoke-role/hubspoke"
```

See `main.go`, which implements this library to produce this very same CLI tool.

## Setting up the Hub/Spoke configuration

Following the [Principle of Least Privilege](https://www.cisecurity.org/spotlight/ei-isac-cybersecurity-spotlight-principle-of-least-privilege/), we're going to scope-down the permissions to as few as necessary.

### The User

In one of your AWS accounts, create an IAM user/instance-profile/whatever dedicated to this task. Since this user represents a _process_ and not a _person_, I recommend prefixing the user name with `sys-`. If we wanted to do things on behalf of the "robot" process, then perhaps we'd call this user `sys-robot`.

Just like a [Meeseeks](https://rickandmorty.fandom.com/wiki/Mr._Meeseeks), this user is intended for only a single task. It's better to have more users (with corresponding spoke roles) with fewer permissions, than it is to have fewer users (with corresponding spoke roles) with more permissions. Please don't re-use the same user for many tasks, as you increase your cybersecurity "blast radius" that way.

This user — as itself — can only do one thing: assume an IAM role in the "hub" account. (Replace `{hub-account-id}` with the AWS Account ID where your "hub" role is located.)

**Policy:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": [
        "arn:aws:iam::{hub-account-id}:role/robot-hub-role"
      ]
    }
  ]
}
```

#### AWS Vault

As an enlightened user of AWS, you use [AWS Vault](https://github.com/99designs/aws-vault) (or maybe [AWS Okta](https://github.com/fiveai/aws-okta)) to manage your credentials. This stores them in the system keychain instead of as plain text on-disk, it automatically generates STS session credentials on your behalf, and it's easy to pass the credentials to things that are built with the AWS SDKs _besides_ the AWS CLI. Oh — and it also supports AWS SSO out-of-the-box.

### The Hub

Using the "robot" process example, let's follow-through with creating an IAM role to assume, and call it `robot-hub-role`.

This is an IAM role which will grant access to your user for assuming the "spoke" role in every account which has that identically-named role.

(If you're not using AWS Organizations, you can remove the entire `Condition` block. You should also specify the AWS Account IDs of the accounts you want to access.)

**Policy:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "sts:AssumeRole"
      ],
      "Resource": "arn:aws:iam::*:role/robot-spoke-role",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": "o-ZZZZZZZZZZ"
        }
      }
    }
  ]
}

```

You'll also need to configure the _trust relationship_ for the "hub" role so that only our user can assume it.

(If you're not using AWS Organizations, you can remove the entire `Condition` block.)

**Trust Relationship:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::{user-creation-account-id}:user/sys-robot"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": "o-ZZZZZZZZZZ"
        }
      }
    }
  ]
}
```

This creates a bi-directional symbiosis where the user can only assume the hub role, and the hub role can only be assumed by the user.

### The Spoke

Using the "robot" process example, let's follow-through with creating an IAM role to assume, and call it `robot-spoke-role`.

This is an IAM role which will grant access to your user (via the hub role) and grants the permissions for what can be done in this account. In our case, we want to grant `ReadOnlyAccess` (the built-in, AWS managed policy). Your needs may be different, so adapt accordingly.

You will need to configure the _trust relationship_ for the "spoke" role so that only our "hub" role can access it.

For an extra bit of entropy in our security, we can require an _External ID_ which is known only to the IAM role and the user accessing it.

**Policy:**

This should be a policy which lists the things that the assuming user is permitted to do in the spoke account.

**Trust Relationship:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::{hub-account-id}:role/robot-hub-role"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "{your-external-id}"
        }
      }
    }
  ]
}
```

This creates a bi-directional symbiosis where the hub role can only assume the spoke role, and the spoke role can only be assumed by the hub role.

## Development

### Requirements

TBD

### Testing

TBD
