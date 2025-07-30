#!/usr/bin/env bash
#
# Custom script to set up the environment before running tests.

# This is needed to ensure that the French locale is available for tests.
sudo apt-get update
sudo apt-get install --yes language-pack-fr
