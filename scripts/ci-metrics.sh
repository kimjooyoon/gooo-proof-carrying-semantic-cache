#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 8 ]]; then
  echo "usage: ci-metrics.sh REPOSITORY OUTPUT CONFORMANCE COMPILE BUILD TEST CONFORMANCE_STAGE INTEGRATION" >&2
  exit 64
fi

repo_root=$1
output=$2
conformance=$3
compile=$4
build=$5
test_stage=$6
conformance_stage=$7
integration=$8

sum_lines() {
  local extension=$1
  local total=0
  while IFS= read -r -d '' file; do
    total=$((total + $(wc -l < "$file")))
  done < <(find "$repo_root" -type f -name "$extension" ! -path "$repo_root/.git/*" ! -name README.md -print0)
  echo "$total"
}

count_files() {
  local extension=$1
  find "$repo_root" -type f -name "$extension" ! -path "$repo_root/.git/*" ! -name README.md -print | wc -l | tr -d ' '
}

read_stage() {
  jq -c '{wall_ms:(.wall_ms|tonumber),peak_rss_kib:(.peak_rss_kib|tonumber)}' "$1"
}

go_files=$(count_files '*.go')
gooo_files=$(count_files '*.gooo')
go_lines=$(sum_lines '*.go')
gooo_lines=$(sum_lines '*.gooo')
regular_files=$(find "$repo_root" -type f ! -path "$repo_root/.git/*" ! -name README.md -print | wc -l | tr -d ' ')
descendant_dirs=$(find "$repo_root" -mindepth 1 -type d ! -path "$repo_root/.git" ! -path "$repo_root/.git/*" -print | wc -l | tr -d ' ')
generated_files=$(find "$conformance" -type f -path '*/generated/*.go' -print | wc -l | tr -d ' ')
generated_bytes=$(find "$conformance" -type f -path '*/generated/*.go' -exec wc -c {} + | awk 'END {print $1 + 0}')
repository_writes=$(git -C "$repo_root" status --porcelain=v1 | wc -l | tr -d ' ')
toolchain=$(go env GOVERSION)
runner_material="${RUNNER_OS:-unknown}|${RUNNER_ARCH:-unknown}|${ImageOS:-unknown}|${ImageVersion:-unknown}"
runner_digest="sha256:$(printf '%s' "$runner_material" | sha256sum | awk '{print $1}')"

jq -n \
  --arg schema "gooo-proof-carrying-semantic-cache/ci-evidence/v1" \
  --arg go_version "$toolchain" --arg runner_digest "$runner_digest" \
  --argjson go_files "$go_files" --argjson gooo_files "$gooo_files" \
  --argjson go_lines "$go_lines" --argjson gooo_lines "$gooo_lines" \
  --argjson regular_files "$regular_files" --argjson descendant_dirs "$descendant_dirs" \
  --argjson generated_files "$generated_files" --argjson generated_bytes "$generated_bytes" \
  --argjson repository_writes "$repository_writes" \
  --argjson compile "$(read_stage "$compile")" --argjson build "$(read_stage "$build")" \
  --argjson test_stage "$(read_stage "$test_stage")" --argjson conformance_stage "$(read_stage "$conformance_stage")" \
  --argjson integration "$(read_stage "$integration")" --slurpfile report "$conformance" \
  '{schema:$schema,verification_authority:"GITHUB_ACTIONS",go_version:$go_version,runner_digest:$runner_digest,
    root_readme_excluded:true,repository_writes:$repository_writes,
    inventory:{go_files:$go_files,gooo_files:$gooo_files,go_physical_lines:$go_lines,gooo_physical_lines:$gooo_lines,descendant_dirs:$descendant_dirs,regular_files:$regular_files},
    generated:{files:$generated_files,bytes:$generated_bytes},
    stages:{compile:$compile,build:$build,test:$test_stage,conformance:$conformance_stage,integration:$integration},
    tests:($report[0].summary|{total:.tests_total,selected:.tests_selected,executed:.tests_executed,reused:.tests_reused,failed:.tests_failed,unknown:.tests_unknown}),
    corpus:($report[0].summary|{total:.total_cases,closed:.closed,unknown:.unknown,refuted:.refuted}),
    exact_pair_vectors:($report[0].cases|map({id,expected,decision,pair,pair_identity,improvement,rebuild_performed})),
    authority:($report[0].authority|. + {local_test_executions:0,cross_project_required_gates:0,automatic_commit:0,automatic_push:0,automatic_merge:0,automatic_release:0})
  }' > "$output"
