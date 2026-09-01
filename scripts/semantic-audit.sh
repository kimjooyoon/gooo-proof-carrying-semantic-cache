#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
meta="$repo_root/.gooo/proof-cache.gooo"

test "$(grep -c '^  scenario ' "$meta")" -eq 7
grep -q '^  precedence "REFUTED" "UNKNOWN" "CLOSED"$' "$meta"
grep -q '^  denominator "proof-carrying-semantic-cache-v1" count "7"$' "$meta"
grep -q '^  cache_key .*semantic_key.*dependency_binding.*origin_binding.*toolchain_binding.*contract_binding' "$meta"
grep -q '^  binding semantic_key .*mismatch "REFUTED"' "$meta"
grep -q '^  binding dependency .*mismatch "REFUTED"' "$meta"
grep -q '^  binding toolchain .*mismatch "UNKNOWN"' "$meta"
grep -q '^  witness terminal .*reason "terminal-reason" effect "terminal-effect" mismatch "REFUTED"' "$meta"
grep -q '^  metrics vector "tests_executed,tests_reused,wall_ms,peak_rss_kib" pair ' "$meta"
grep -q '^  authority_rule .*repository_writes "0" .*automatic_release "0"' "$meta"
grep -q '^  unknown_fields "stage" "step" "reason" "unknown_class" "next_operation" "blocked_by"$' "$meta"
jq -e '.authority == "metacode" and .required_gate == 0 and .precedence == ["REFUTED","UNKNOWN","CLOSED"]' "$repo_root/contracts/denominator-v1.json" >/dev/null
jq -e '.repository_writes == 0 and .local_test_executions == 0 and .cross_project_required_gates == 0 and (.pair_vectors|length) == 4' "$repo_root/contracts/metrics-v1.json" >/dev/null
if rg -nE 'git (commit|merge|push|reset|checkout)|gh (pr merge|release delete|release edit)' "$repo_root/cmd" "$repo_root/internal" "$repo_root/scripts"; then
  echo 'automatic repository integration authority is forbidden' >&2
  exit 1
fi
echo 'semantic_audit=CLOSED'
