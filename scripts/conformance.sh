#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 3 ]]; then
  echo "usage: conformance.sh REPOSITORY BINARY OUTPUT" >&2
  exit 64
fi

repository=$1
binary=$2
output=$3
before=$(git -C "$repository" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
mkdir -p "$output"
toolchain="$(go env GOVERSION)/$(go env GOOS)/$(go env GOARCH)"
runner_material="${RUNNER_OS:-unknown}|${RUNNER_ARCH:-unknown}|${ImageOS:-unknown}|${ImageVersion:-unknown}"
runner_digest="sha256:$(printf '%s' "$runner_material" | sha256sum | awk '{print $1}')"

"$binary" conformance \
  --meta .gooo/proof-cache.gooo \
  --corpus fixtures/corpus.json \
  --root "$repository" \
  --out "$output/cases" \
  --toolchain "$toolchain" \
  --runner-digest "$runner_digest"

if ! jq -e '
  .schema == "gooo-proof-carrying-semantic-cache/conformance/v1" and
  .decision == "CLOSED" and
  .summary == {total_cases:7,closed:3,unknown:2,refuted:2,tests_total:7,tests_selected:7,tests_executed:4,tests_reused:3,tests_failed:2,tests_unknown:2} and
  .authority.repository_writes == 0 and
  .authority.output_scope == "CALLER_OWNED_TEMP_OUTPUT_ONLY" and
  ([.cases[] | select(.id == "exact-proof-hit") | .decision] == ["CLOSED"]) and
  ([.cases[] | select(.id == "comment-only-semantic-hit") | .decision] == ["CLOSED"]) and
  ([.cases[] | select(.id == "transitive-dependency-change") | .decision] == ["REFUTED"]) and
  ([.cases[] | select(.id == "stale-terminal-witness") | .decision] == ["REFUTED"]) and
  ([.cases[] | select(.id == "missing-proof") | .decision] == ["UNKNOWN"]) and
  ([.cases[] | select(.id == "cross-toolchain") | .decision] == ["UNKNOWN"]) and
  ([.cases[] | select(.id == "replay") | .decision] == ["CLOSED"]) and
  ([.cases[] | select(.decision == "UNKNOWN") | .unknowns[] | (.stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by|length) > 0)] | all) and
  ([.cases[] | select(.improvement.status == "CLOSED") | .improvement.exact_pair] | all) and
  ([.. | objects | keys[]? | select(test("score|percentage|average|estimate"; "i"))] | length) == 0
' "$output/cases/conformance.json" >/dev/null; then
  jq -c '{schema_ok:(.schema == "gooo-proof-carrying-semantic-cache/conformance/v1"),decision_ok:(.decision == "CLOSED"),summary,authority,cases:(.cases|map({id,expected,decision,rebuild_performed,unknowns,refutations,improvement,replay})),forbidden_keys:([.. | objects | keys[]? | select(test("score|percentage|average|estimate"; "i"))])}' "$output/cases/conformance.json"
  exit 1
fi

after=$(git -C "$repository" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"
