package editorcanonical

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInBundleValidatesAllCanonicalDocuments(t *testing.T) {
	root := filepath.Join("..", "..", "assets", "editor-canonical")
	summary, err := ValidateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Documents) != 66 || summary.IdentityDiagnostics != 4 {
		t.Fatalf("summary documents=%d identity diagnostics=%d", len(summary.Documents), summary.IdentityDiagnostics)
	}
}

func TestBundleHashMismatchFailsClosed(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "editor-canonical")
	root := t.TempDir()
	if err := os.Symlink(filepath.Join(source, "bundle-summary.json"), filepath.Join(root, "bundle-summary.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateBundle(root); err == nil {
		t.Fatal("incomplete bundle was accepted")
	}
}
