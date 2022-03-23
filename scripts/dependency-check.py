#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import re
import shutil
import subprocess
import sys
from semantic_version import SimpleSpec, Version

MIN_PYTHON = (3, 7)
if sys.version_info < MIN_PYTHON:
    sys.exit(
        "[ERROR] This script requires Python {version} or newer.".format(
            version='.'.join([str(n) for n in MIN_PYTHON])
        )
    )

#-------------------------------------------------------------------------------

def run(command):
    return subprocess.check_output(
        command.split(" "), stderr=subprocess.STDOUT
    ).decode("utf-8")

def meets_version_requirements(needs_to_meet, current_version):
    needs_to_meet = version_range_handler(needs_to_meet)
    spec = SimpleSpec(needs_to_meet)

    return spec.match(Version(current_version))

def discover_version(text):
    re_version = re.compile(r"v?(\d+\.\d+\.\d+)")
    matches = re_version.search(text)

    if matches == None:
        raise ValueError(f"[ERROR] Could not discover a version number in `{text}`.")

    return matches.group(1)

def version_range_handler(version):
    # Remove extra spaces
    version = version.replace(" ", "")

    # Terraform allows ranges like `~> 0.13`.
    m = re.match("^~>\s*(\d+\.\d+)", version)

    # Uses `~>`
    if m is not None:
        starting_version = Version(f"{m.group(1)}.0")
        ending_version = starting_version.next_minor()
        version = f">={str(starting_version)},<{str(ending_version)}"

    return version

def check_all(
    binary_name,
    needs_version,
    get_version,
    installation_message="",
    unmet_version_message=""
):
    if installation_message == "":
        installation_message = f"If you have macOS and Homebrew, you can install it with `brew install {binary_name}`."

    if shutil.which(binary_name) == None:
        print(
            f"[ERROR] The `{binary_name}` binary is not installed. {installation_message}"
        )

        print("")
        sys.exit(1)

    version_str = discover_version(run(get_version))

    if unmet_version_message == "":
        unmet_version_message = "[ERROR] Requires {binary_name} {needs_version}. Version {version_str} appears to be installed."

    if meets_version_requirements(needs_version, version_str) == False:
        print(
            unmet_version_message.format(
                binary_name=binary_name,
                needs_version=needs_version,
                version_str=version_str,
            )
        )

        print("")
        sys.exit(1)

#-------------------------------------------------------------------------------

# nproc installed
if shutil.which("nproc") == None:
    print(
        "[ERROR] The `nproc` binary is not installed. If you have macOS and Homebrew, you can install it with `brew install coreutils`."
    )

    print("")
    sys.exit(1)

# go
check_all(
    binary_name="go",
    needs_version=">= 1.16",
    get_version="go version",
)

# golangci-lint
check_all(
    binary_name="golangci-lint",
    needs_version=">=1.38.0",
    get_version="golangci-lint version",
)
