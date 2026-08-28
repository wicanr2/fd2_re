package campaign

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedNativeShopAssetsMatchFixedArchive(t *testing.T) {
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	const pack = "../../generated-assets/fd2-original-b97caf22"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	for _, resource := range []int{12, 29, 63} {
		want, err := DecodeNativeShopAssets(archive, resource)
		if err != nil {
			t.Fatal(err)
		}
		got, err := LoadSeparatedNativeShopAssets(pack, resource)
		if err != nil {
			t.Fatalf("resource %d: %v", resource, err)
		}
		if !bytes.Equal(got.Background, want.Background) || got.Decoration.Width != want.Decoration.Width || got.Decoration.Height != want.Decoration.Height || !bytes.Equal(got.Decoration.Pixels, want.Decoration.Pixels) || !bytes.Equal(got.GoldRollStrip.Pixels, want.GoldRollStrip.Pixels) {
			t.Fatalf("resource %d stable assets differ", resource)
		}
		for option := 0; option < 4; option++ {
			for variant := 0; variant < 2; variant++ {
				if !bytes.Equal(got.ServiceCells[option][variant].Pixels, want.ServiceCells[option][variant].Pixels) {
					t.Fatalf("resource %d service %d/%d differs", resource, option, variant)
				}
			}
		}
		if !bytes.Equal(got.PriceCell.Pixels, want.PriceCell.Pixels) || !bytes.Equal(got.Panel.Pixels, want.Panel.Pixels) {
			t.Fatalf("resource %d panel assets differ", resource)
		}
		for i := range got.CompareCells {
			if !bytes.Equal(got.CompareCells[i].Pixels, want.CompareCells[i].Pixels) {
				t.Fatalf("resource %d compare %d differs", resource, i)
			}
		}
		if len(got.SuccessFrames) != len(want.SuccessFrames) {
			t.Fatalf("resource %d success frames=%d want %d", resource, len(got.SuccessFrames), len(want.SuccessFrames))
		}
		for i := range got.SuccessFrames {
			indexed, mask, err := want.SuccessFrames[i].IndexedLayers()
			if err != nil {
				t.Fatal(err)
			}
			if got.SuccessFrames[i].Width != want.SuccessFrames[i].Width || got.SuccessFrames[i].Height != want.SuccessFrames[i].Height || !bytes.Equal(got.SuccessFrames[i].Indexed, indexed) || !bytes.Equal(got.SuccessFrames[i].Mask, mask) {
				t.Fatalf("resource %d success frame %d differs", resource, i)
			}
		}
	}
}

func TestSeparatedNativeShopAssetsFailClosedWithoutCompletePack(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadSeparatedNativeShopAssets(root, 12); err == nil {
		t.Fatal("missing shop pack was accepted")
	}
	dir := filepath.Join(root, "shop", "FDOTHER_012")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(`{"schema_version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSeparatedNativeShopAssets(root, 12); err == nil {
		t.Fatal("incomplete shop metadata was accepted")
	}
}
