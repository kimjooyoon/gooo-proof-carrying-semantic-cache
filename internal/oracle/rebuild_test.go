package oracle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-proof-carrying-semantic-cache/internal/protocol"
)

func TestCommentsDoNotChangeSemanticKeyOrArtifact(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	policy, _, err := protocol.LoadPolicy(filepath.Join(root, ".gooo", "proof-cache.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	base, err := os.ReadFile(filepath.Join(root, "fixtures", "sources", "base.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	commented, err := os.ReadFile(filepath.Join(root, "fixtures", "sources", "commented.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	baseIR, baseArtifact, err := Rebuild("base.gooo", base, policy)
	if err != nil {
		t.Fatal(err)
	}
	commentedIR, commentedArtifact, err := Rebuild("commented.gooo", commented, policy)
	if err != nil {
		t.Fatal(err)
	}
	if baseIR.SemanticKey != commentedIR.SemanticKey || baseArtifact.Digest != commentedArtifact.Digest {
		t.Fatalf("comment-only source did not converge: base=%s/%s commented=%s/%s", baseIR.SemanticKey, baseArtifact.Digest, commentedIR.SemanticKey, commentedArtifact.Digest)
	}
}
