#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "usage: enable-immutable-releases.sh OWNER/REPOSITORY" >&2
  exit 64
fi

repository=$1
gh api -X PUT "repos/$repository/immutable-releases" >/dev/null
settings=$(gh api "repos/$repository/immutable-releases")
jq -e '.enabled == true' <<<"$settings" >/dev/null
echo "$settings"
