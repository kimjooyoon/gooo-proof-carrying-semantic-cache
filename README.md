# gooo-proof-carrying-semantic-cache

`gooo-proof-carrying-semantic-cache` reduces repeated build/test work without
mistaking a cache hit for semantic proof. The authoritative contract is
[`/.gooo/proof-cache.gooo`](.gooo/proof-cache.gooo). It declares the semantic
cache key, dependency/origin/toolchain/contract bindings, proof obligations,
terminal reason/effect witness, precedence, fallback, replay, fixed denominator,
and exact metric vector. Go only parses, lowers, executes, and verifies that
contract.

Every cache entry carries a generated Go artifact and a proof receipt together.
The verifier always constructs an independent cold rebuild oracle from the
current `.gooo` source and compares its semantic key, artifact digest, proof
obligations, and terminal witness before authorizing reuse. `REFUTED > UNKNOWN >
CLOSED`; every UNKNOWN record has `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`.

The fixed seven-case vector is:

| Case | Expected | Meaning |
| --- | --- | --- |
| `exact-proof-hit` | `CLOSED` | Matching key, bindings, receipt, witness, artifact, and oracle. |
| `comment-only-semantic-hit` | `CLOSED` | Raw comments/spacing differ; the semantic key and artifact converge. |
| `transitive-dependency-change` | `REFUTED` | A stale transitive dependency binding forces rebuild. |
| `stale-terminal-witness` | `REFUTED` | A cached reason/effect witness contradicts the oracle. |
| `missing-proof` | `UNKNOWN` | Missing proof receipt forces rebuild with a six-field frontier. |
| `cross-toolchain` | `UNKNOWN` | A cross-toolchain binding is unavailable for reuse and rebuilds. |
| `replay` | `CLOSED` | Repeated independent rebuild produces the same semantic/artifact/witness identity. |

CI measures cold rebuild and proved-reuse vectors on the same run and runner.
The separate integer fields are `tests_executed`, `tests_reused`, `wall_ms`, and
`peak_rss_kib`; inventory also records Go/Gooo files and physical lines,
descendant directories, regular files, and generated files/bytes. Improvement
is `CLOSED` only for an exact scenario/source/contract/toolchain/runner pair
that has semantic proof and independent-oracle closure. No score, weighted
average, percentage, or estimate is emitted.

GitHub Actions is the verification authority. It runs Go 1.27, generated
artifact builds, Go tests, vet, fixed conformance, and integration. Generated
outputs are caller-owned temporary outputs; the input repository reports
`repository_writes=0`. Local build, test, vet, and conformance execution is
intentionally outside the development contract.

See [`docs/protocol-v1.md`](docs/protocol-v1.md) for the evidence contract and
[`docs/release-policy.md`](docs/release-policy.md) for the human-authorized
merge/tag/release boundary.
