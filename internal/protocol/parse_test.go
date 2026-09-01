package protocol

import (
	"os"
	"path/filepath"
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
