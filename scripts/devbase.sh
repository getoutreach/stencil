#!/usr/bin/env bash
# This script ensures we have a copy of devbase
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
libDir="$DIR/../.bootstrap"
lockfile="$DIR/../stencil.lock"
serviceYaml="$DIR/../service.yaml"

# get_absolute_path returns the absolute path of a file
get_absolute_path() {
  python="$(command -v python3 || command -v python)"
  "$python" -c "import os,sys; print(os.path.realpath(sys.argv[1]))" "$1"
}

# get_field_from_yaml reads a field from a yaml file using either go-yq or python-yq
get_field_from_yaml() {
  field="$1"
  file="$2"

  if [[ "$(yq e '.a' '-' <<<'{"a": "true"}' 2>&1)" == "true" ]]; then
    # using golang version
    yq e "$field" "$file"
  else
    # probably using python version
    yq -r "$field" <"$file"
  fi
}

# Use the version of devbase from stencil
version=$(get_field_from_yaml '.modules[] | select(.name == "github.com/getoutreach/devbase") | .version' "$lockfile")
existingVersion=$(cat "$libDir/.version" 2>/dev/null || true)

if [[ ! -e $libDir ]] || [[ $existingVersion != "$version" ]] || [[ ! -e "$libDir/.version" ]]; then
  rm -rf "$libDir"

  if [[ $version == "local" ]]; then
    # If we're using a local version, we should use ln
    # to map the local devbase into the bootstrap dir

    path=$(get_field_from_yaml '.replacements["github.com/getoutreach/devbase"]' "$serviceYaml")
    absolute_path=$(get_absolute_path "$path")

    ln -sf "$absolute_path" "$libDir"
  else
    git clone -q --single-branch --branch "$version" git@github.com:getoutreach/devbase \
      "$libDir" >/dev/null

    # Don't let devbase be confused by the existence of one there :(
    rm "$libDir/service.yaml" || true # ignore errors
  fi

  echo -n "$version" >"$libDir/.version"
fi
