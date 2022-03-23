#! /usr/bin/env python3
# -*- coding: utf-8 -*-

import pathlib
import re
import subprocess
import sys

MIN_PYTHON = (3, 7)
if sys.version_info < MIN_PYTHON:
    sys.exit(
        "[ERROR] This script requires Python {version} or newer.".format(
            version='.'.join([str(n) for n in MIN_PYTHON])
        )
    )

#-------------------------------------------------------------------------------

reqs = subprocess.check_output([sys.executable, '-m', 'pip', 'freeze'])
installed_packages = [r.decode().split('==')[0] for r in reqs.split()]
script_dir = pathlib.Path(__file__).parent.absolute()

# Python packages
with open(f"{script_dir}/requirements.txt") as f:
    for dep in [line for line in f]:
        if re.match(r"^([^(>|<|=)]+)", dep).group(0) not in installed_packages:
            print(
                "Cannot find dependencies. Running `pip install -r scripts/requirements.txt` automatically."
            )
            reqs = subprocess.check_output([
                sys.executable, '-m', 'pip', 'install', '-r', 'scripts/requirements.txt'
            ])

            print("")
            break
