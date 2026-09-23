package protocol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetacodeOwnsFixedDenominatorAndPrecedence(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	policy, digest, err := LoadPolicy(filepath.Join(root, ".gooo", "proof-cache.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	if digest == "" || policy.Authority != "metacode" || policy.Denominator.Count != 7 {
		t.Fatalf("unexpected metacode identity: %+v", policy)
	}
	if len(policy.Scenarios) != 7 || policy.Precedence[0] != Refuted || policy.Precedence[1] != Unknown || policy.Precedence[2] != Closed {
		t.Fatalf("metacode denominator or precedence was not preserved")
	}
}

func TestCorpusIsFixedAndExplicit(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	corpus, _, err := LoadCorpus(filepath.Join(root, "fixtures", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) != 7 {
		t.Fatalf("got %d cases, want 7", len(corpus.Cases))
	}
	for _, fixture := range corpus.Cases {
		if fixture.ID == "" || fixture.TestID == "" || fixture.Origin == "" || len(fixture.Dependencies) == 0 {
			t.Fatalf("incomplete fixture: %+v", fixture)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "fixtures", "sources", "commented.gooo")); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyRejectsDuplicateContractRecords(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".gooo", "proof-cache.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		duplicate string
	}{
		{name: "authority", duplicate: `  authority "metacode"`},
		{name: "precedence", duplicate: `  precedence "REFUTED" "UNKNOWN" "CLOSED"`},
		{name: "unknown_fields", duplicate: `  unknown_fields "stage" "step" "reason" "unknown_class" "next_operation" "blocked_by"`},
		{name: "denominator", duplicate: `  denominator "proof-carrying-semantic-cache-v1" count "7"`},
		{name: "normalization", duplicate: `  normalization comments "ignored" whitespace "collapsed" string_literals "preserved"`},
		{name: "cache_key", duplicate: `  cache_key semantic_key "source-semantic-digest" dependency_binding "transitive-dependency-digest" origin_binding "immutable-origin-digest" toolchain_binding "go-toolchain-runner-digest" contract_binding "metacode-contract-digest"`},
		{name: "binding", duplicate: `  binding semantic_key stage "KEY" step "compare-semantic-cache-key" mismatch "REFUTED" unknown_class "SEMANTIC_KEY_UNAVAILABLE"`},
		{name: "obligation", duplicate: `  obligation semantic_key stage "PROOF" step "verify-semantic-key" proof "SEMANTIC_KEY_MATCHED" missing "PROOF_RECEIPT_MISSING"`},
		{name: "witness", duplicate: `  witness terminal stage "WITNESS" step "compare-terminal-reason-effect" reason "terminal-reason" effect "terminal-effect" mismatch "REFUTED"`},
		{name: "reuse", duplicate: `  reuse rule "proof-reuse" status "CLOSED" requires "all-bindings-proof-witness-oracle"`},
		{name: "fallback", duplicate: `  fallback status "REFUTED" operation "independent-rebuild" unknown_class "KNOWN_CONTRADICTION"`},
		{name: "replay", duplicate: `  replay rule "deterministic-replay" status "CLOSED" requires "same-semantic-key-artifact-proof-witness"`},
		{name: "metrics", duplicate: `  metrics vector "tests_executed,tests_reused,wall_ms,peak_rss_kib" pair "same-scenario-source-contract-toolchain-runner" improvement "exact-pair-only" forbidden "score,percentage,average,estimate"`},
		{name: "authority_rule", duplicate: `  authority_rule repository_writes "0" output_scope "CALLER_OWNED_TEMP_OUTPUT_ONLY" automatic_commit "0" automatic_push "0" automatic_merge "0" automatic_release "0"`},
		{name: "generation", duplicate: `  generation "go" package "main" entrypoint "main"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			insert := strings.LastIndex(string(raw), "\n}")
			if insert < 0 {
				t.Fatal("contract fixture has no closing brace")
			}
			candidate := string(raw[:insert]) + "\n" + tc.duplicate + string(raw[insert:])
			if _, err := ParsePolicy(candidate); err == nil {
				t.Fatalf("ParsePolicy accepted duplicate %s record", tc.name)
			}
		})
	}
}
