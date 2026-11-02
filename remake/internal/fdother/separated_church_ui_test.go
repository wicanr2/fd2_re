package fdother

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeparatedChurchUIAssetsMatchOriginalCodecs(t *testing.T) {
	const (
		rawPath = "../../../extracted/raw/FDOTHER/FDOTHER_014.bin"
		uiRoot  = "../../generated-assets/fd2-original-b97caf22/ui"
	)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Skip("player-provided FDOTHER_014 is absent")
	}
	root := uiRoot
	if pack := os.Getenv("FD2_ASSET_PACK"); pack != "" {
		root = filepath.Join(pack, "ui")
	}
	got, err := LoadSeparatedChurchUIAssets(root)
	if err != nil {
		if os.Getenv("FD2_ASSET_PACK") != "" {
			t.Fatal(err)
		}
		t.Skipf("separated FDOTHER_014 church pack is absent: %v", err)
	}
	for _, index := range churchUIOpaqueEntries {
		want, err := ParseLMI1OpaqueEntry(raw, index)
		if err != nil || got.Entries[index].Width != want.Width ||
			got.Entries[index].Height != want.Height ||
			!bytes.Equal(got.Entries[index].Pixels, want.Pixels) {
			t.Fatalf("opaque entry %d differs: %v", index, err)
		}
	}
	for index := 3; index <= 10; index++ {
		want, err := ParseLMI1RawEntry(raw, index)
		if err != nil || got.Entries[index].Width != 24 || got.Entries[index].Height != 20 || !bytes.Equal(got.Entries[index].Pixels, want.Pixels) {
			t.Fatalf("raw service%d differs: %v", index, err)
		}
		wrong, err := ParseLMI1OpaqueEntry(raw, index)
		if err == nil && bytes.Equal(wrong.Pixels, want.Pixels) {
			t.Fatalf("test did not distinguish wrong codec%d", index)
		}
	}
	wantPrice, err := ParseLMI1RawEntry(raw, 15)
	if err != nil || got.PriceCell.Width != wantPrice.Width ||
		got.PriceCell.Height != wantPrice.Height ||
		!bytes.Equal(got.PriceCell.Pixels, wantPrice.Pixels) {
		t.Fatalf("raw entry 15 differs: %v", err)
	}
	for _, index := range churchUIFrameEntries {
		want, err := ParseLMI1FrameEntry(raw, index)
		if err != nil {
			t.Fatalf("frame entry %d: %v", index, err)
		}
		wantIndexed, wantMask, err := want.IndexedLayers()
		var frame Frame
		if index == 0 {
			frame = got.Background
		} else {
			frame = got.ReviveFX[index-23]
		}
		if err != nil || frame.Width != want.Width || frame.Height != want.Height ||
			!bytes.Equal(frame.Indexed, wantIndexed) || !bytes.Equal(frame.Mask, wantMask) {
			t.Fatalf("frame entry %d differs: %v", index, err)
		}
	}
}

func TestSeparatedChurchUIAssetsFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedChurchUIAssets(t.TempDir()); err == nil {
		t.Fatal("incomplete separated church UI pack was accepted")
	}
}

// A previous pack whose service payload was interpreted as high-run must
// fail on that explicit codec, even when every required PNG is present.
func TestSeparatedChurchUIAssetsRejectsHighRunServiceMetadata(t *testing.T) {
	root := os.Getenv("FD2_ASSET_PACK")
	if root == "" {
		t.Skip("explicit pack required")
	}
	source := filepath.Join(root, "ui", "fdother_014_church")
	target := filepath.Join(t.TempDir(), "fdother_014_church")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		out := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(out, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(target, "resource.json")
	data, err := os.ReadFile(metadata)
	if err != nil {
		t.Fatal(err)
	}
	var doc separatedItemPanelDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for i := range doc.Entries {
		if doc.Entries[i].Index == 3 {
			doc.Entries[i].Codec = "opaque_high_run"
		}
	}
	data, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, data, 0644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadSeparatedChurchUIAssets(filepath.Dir(target))
	if err == nil || !strings.Contains(err.Error(), "entry3 metadata mismatch") && !strings.Contains(err.Error(), "entry 3 metadata mismatch") {
		t.Fatal("old codec not explicitly rejected", err)
	}
}
