package campaign

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedNativeTownAssetsMatchFixedArchive(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	const pack = "../../generated-assets/fd2-original-b97caf22"
	if _, err := os.Stat(base + "/FDOTHER.DAT"); err != nil {
		t.Skip("player-provided originals are absent")
	}
	want, err := DecodeNativeTownAssetsArchive(base+"/FDOTHER.DAT", base+"/FDICON.B24")
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadNativeTownAssets(pack)
	if err != nil {
		t.Fatal(err)
	}
	for variant := range got.Backgrounds {
		if !bytes.Equal(got.Backgrounds[variant], want.Backgrounds[variant]) {
			t.Fatalf("variant %d background differs", variant)
		}
	}
	if got.Label.Width != want.Label.Width || got.Label.Height != want.Label.Height || !bytes.Equal(got.Label.Pixels, want.Label.Pixels) {
		t.Fatal("town label differs")
	}
	strings, font := loadNativeTownTextAssets(t, base)
	for variant := 0; variant < 3; variant++ {
		for selection := 0; selection < 6; selection++ {
			for pulse := 0; pulse < 4; pulse++ {
				a, err := ComposeNativeTownFrame(want, strings, font, variant, selection, pulse)
				if err != nil {
					t.Fatal(err)
				}
				b, err := ComposeNativeTownFrame(got, strings, font, variant, selection, pulse)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(a, b) {
					t.Fatalf("composite differs variant=%d selection=%d pulse=%d", variant, selection, pulse)
				}
			}
		}
	}
}

func TestSeparatedNativeTownAssetsFailClosedWithoutCompletePack(t *testing.T) {
	if _, err := LoadNativeTownAssets(t.TempDir()); err == nil {
		t.Fatal("incomplete town pack was accepted")
	}
}
