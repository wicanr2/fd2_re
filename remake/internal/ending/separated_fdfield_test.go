package ending

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func selector30SeparatedRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", "fields", "fdfield")
	if _, err := os.Stat(filepath.Join(root, "selector_30", "field.json")); os.IsNotExist(err) {
		t.Skip("separated FDFIELD selector 30 is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSeparatedFDFIELDSelector30MatchesFixedArchiveResources(t *testing.T) {
	root := selector30SeparatedRoot(t)
	archive := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2", "FDFIELD.DAT")
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		t.Skip("player-provided FDFIELD.DAT is absent")
	} else if err != nil {
		t.Fatal(err)
	}
	gotMap, gotControl, gotPositions, err := loadSeparatedFDFIELDSelector30(root)
	if err != nil {
		t.Fatal(err)
	}
	for offset, got := range [][]byte{gotMap, gotControl, gotPositions} {
		want, err := fdother.ReadResource(archive, 90+offset)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("separated FDFIELD #%d differs from fixed archive", 90+offset)
		}
	}
}

func TestSeparatedFDFIELDSelector30FailsClosedOnEditedContentWithoutUpdatedProvenance(t *testing.T) {
	root := selector30SeparatedRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "selector_30", "field.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc separatedFDFIELDDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Tiles[0] ^= 1
	tempRoot := t.TempDir()
	directory := filepath.Join(tempRoot, "selector_30")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	corrupt, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "field.json"), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := loadSeparatedFDFIELDSelector30(tempRoot); err == nil {
		t.Fatal("edited selector 30 content unexpectedly bypassed source hash gate")
	}
}
