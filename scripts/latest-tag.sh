#!/usr/bin/env bash
# Exit on any error
set -e

# Get the most recent tag sorted by commit date (most recent first)
git describe --tags --abbrev=0 "$(git rev-list --tags --max-count=1)"
