package figani

import (
	"os"
	"reflect"
	"testing"
)

func TestSeparatedResourceMatchesOriginalIndexedAnimation(t *testing.T) {
	const (
		archive = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
		root    = "../../generated-assets/fd2-original-b97caf22/animations"
	)
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if _, err := os.Stat(root + "/FIGANI_004/animation.json"); err != nil {
		t.Skip("separated FIGANI pack is absent")
	}
	want, err := DecodeResource(archive, 4)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedResource(root, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("separated FIGANI#4 differs from archive decoder")
	}
}

func TestSeparatedResourceFailsClosed(t *testing.T) {
	if _, err := LoadSeparatedResource(t.TempDir(), 4); err == nil {
		t.Fatal("missing separated animation was accepted")
	}
}
