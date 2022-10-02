#!/usr/bin/env bash
# Quick wrapper for cmd testing of validators
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
srcFile="$DIR/$1"
dstFile="$2"

echo "\"$(cat "$srcFile")\""
echo "\"$(cat "$dstFile")\""

# Run the validator
exec cmp "$DIR/$1" "$2"
