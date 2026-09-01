#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 2 ]]; then
  echo "usage: integration.sh CONFORMANCE_OUTPUT OUTPUT_ROOT" >&2
  exit 64
fi

conformance=$1
output_root=$2
mkdir -p "$output_root"
expected='program=proof-cache reason=PASS effect=ARTIFACT:proof-cache'
count=0
for source in "$conformance"/*/generated/main.go; do
  [[ -f "$source" ]] || continue
  case_id=$(basename "$(dirname "$(dirname "$source")")")
  build_dir="$output_root/$case_id"
  mkdir -p "$build_dir"
  cp "$source" "$build_dir/main.go"
  printf 'module generated-%s\n\ngo 1.27.0\n' "$case_id" > "$build_dir/go.mod"
  go build -trimpath -o "$build_dir/program" "$build_dir"
  actual=$("$build_dir/program")
  test "$actual" = "$expected"
  jq -n --arg id "$case_id" --arg output "$actual" '{case_id:$id,status:"CLOSED",output:$output}' > "$build_dir/integration.json"
  count=$((count + 1))
done
test "$count" -eq 7
jq -n --argjson cases "$count" '{schema:"gooo-proof-carrying-semantic-cache/integration/v1",generated_cases:$cases,decision:"CLOSED"}' > "$output_root/integration.json"
