# CI evidence v1

`ci-metrics.json` is emitted only by the GitHub Actions verification workflow.
Every field is an exact integer observation from that run or an exact count
over the checked-out subject; no improvement value is derived from a score,
percentage, average, or estimate.

The top-level inventory excludes the root `README.md` and `.git`. It counts Go
and Gooo files, physical lines for each language, descendant directories,
regular files, and generated Go files/bytes. Each stage records `wall_ms` and
`peak_rss_kib` for compile, build, test, conformance, and integration.

Each fixed case also carries:

- `pair.cold` — one independent rebuild with `tests_executed=1`;
- `pair.proved_reuse` — either a proved test reuse or a fallback rebuild;
- `pair_identity` — scenario, source, semantic key, contract, toolchain, and
  runner bindings;
- `improvement` — `CLOSED` only when semantic proof, oracle equality, and the
  exact pair identity all hold.

The case report retains refutations and full UNKNOWN tuples. A cache hit is
therefore visible even when it is rejected or lacks enough evidence to reuse.
