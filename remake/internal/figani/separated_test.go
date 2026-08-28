package figani

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestSeparatedZeroHeaderFallbackMatchesOriginalPreviousResource(t *testing.T) {
	const (
		archive = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
		root    = "../../generated-assets/fd2-original-b97caf22/animations"
	)
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if _, err := os.Stat(root + "/FIGANI_002/resource.json"); err != nil {
		t.Skip("separated FIGANI status pack is absent")
	}
	want, err := DecodeResource(archive, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedResourceWithZeroHeaderFallback(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("separated zero-header fallback differs from original previous resource")
	}
}

func TestSeparatedFDOTHERAnimationMatchesOriginalResource(t *testing.T) {
	const (
		archive = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
		root    = "../../generated-assets/fd2-original-b97caf22/animations"
	)
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	if _, err := os.Stat(root + "/FDOTHER_018/animation.json"); err != nil {
		t.Skip("separated FDOTHER animation pack is absent")
	}
	want, err := DecodeResource(archive, 18)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedArchiveResource(root, "FDOTHER.DAT", 18)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("separated FDOTHER#18 differs from archive decoder")
	}
}

func TestSeparatedResourceFailsClosed(t *testing.T) {
	if _, err := LoadSeparatedResource(t.TempDir(), 4); err == nil {
		t.Fatal("missing separated animation was accepted")
	}
}

func TestSeparatedZeroHeaderFallbackRequiresConfirmedStatus(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "FIGANI_002")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	document := map[string]any{
		"schema_version": 1,
		"kind":           "animation_resource_status",
		"resource_id":    "animation-resource/figani_002",
		"source":         map[string]any{"file": "FIGANI.DAT", "resource": 2},
		"raw_size":       3,
		"header_word_le": 0,
		"status":         "empty_header_zero",
		"reason_code":    "zero_header_word",
		"evidence":       "confirmed",
		"extensions":     map[string]any{},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "resource.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSeparatedResourceWithZeroHeaderFallback(root, 2); err == nil {
		t.Fatal("confirmed zero header accepted without the fallback animation")
	}
	document["evidence"] = "hypothesis"
	raw, _ = json.Marshal(document)
	if err := os.WriteFile(filepath.Join(directory, "resource.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if zero, err := separatedResourceHasZeroHeader(root, 2); err != nil || zero {
		t.Fatalf("unconfirmed zero header accepted: zero=%v err=%v", zero, err)
	}
}
