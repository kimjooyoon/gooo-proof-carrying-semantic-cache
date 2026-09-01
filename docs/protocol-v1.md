# Proof-carrying semantic cache protocol v1

The contract source is `.gooo/proof-cache.gooo`; JSON contracts are projections
for CI shape checks and do not add semantic rules. The pipeline is:

`source .gooo -> independent cold rebuild oracle -> generated Go artifact`  
`cache artifact + proof receipt -> binding/proof/witness/oracle verification -> decision`

The semantic key is the digest of canonical source records. The declared
normalization ignores comments, collapses whitespace, and preserves string
literals. Cache identity additionally binds the transitive dependency digest,
immutable origin digest, Go toolchain/runner digest, and contract digest.

The proof receipt must bind every declared obligation to its required marker,
carry the generated artifact digest, and carry the terminal reason/effect
witness. A missing receipt is `UNKNOWN`; a known contradiction is `REFUTED`.
Toolchain mismatch is deliberately `UNKNOWN` because the available evidence is
insufficient to refute the artifact's semantics across toolchains.

The independent rebuild oracle is never populated from the cache entry. A
`CLOSED` reuse result requires oracle equality for semantic key, generated
artifact bytes/digest, proof obligations, and terminal witness. Replay repeats
the oracle and checks semantic key, artifact bytes, and witness identity.

The conformance report preserves the full fixed vector and per-case evidence.
The CI evidence report adds exact stage measurements and inventory. All output
paths are outside the input repository.
