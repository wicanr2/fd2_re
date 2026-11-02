package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	xdraw "golang.org/x/image/draw"
)

func TestModernHDMovementPanelSourcePrototype(t *testing.T) {
	out := os.Getenv("FD2_HD_MOVEMENT_PANEL_PROTOTYPE_OUT")
	if out == "" {
		t.Skip("explicit movement panel prototype output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1000, 0)
	g := stage42ActualGame(t, &now)
	attachOfficialLocale(t, g)
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("HD font unavailable")
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal([]any{unitPresentGPUState(g), g.gold, g.nativeRNGState, g.st.NativeTileBlitModes})
	rows := []map[string]any{}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		g.localeID = locale
		g.localeEntities, err = loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		for _, slot := range []int{0, 4} {
			actor := g.st.Units[slot]
			record, err := battle.NativeBattlePanelRecordForUnit(actor)
			if err != nil {
				t.Fatal(err)
			}
			native := make([]byte, 64000)
			if err := g.renderLocalizedNativeBattlePanel(assets, record, native, 0, 0); err != nil {
				t.Fatal(err)
			}
			replay := make([]byte, 64000)
			collected := []*modernBattlePanelText{}
			if err := g.renderBattlePanelText(assets, record, replay, 0, 0, &collected); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(native, replay) || len(collected) != 1 {
				t.Fatal("HD collector changed original producer", locale, slot)
			}
			p := collected[0]
			x, y, err := battle.NativeBattlePanelOrigin(record, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			panel := image.NewPaletted(image.Rect(0, 0, 149, 42), g.nativeUIPalette)
			for row := 0; row < 42; row++ {
				copy(panel.Pix[row*149:(row+1)*149], native[(y+row)*320+x:(y+row)*320+x+149])
			}
			if !bytes.Equal(panel.Pix, p.native.Pixels) {
				t.Fatal("source crop differs")
			}
			normal := image.NewRGBA(image.Rect(0, 0, 894, 252))
			xdraw.NearestNeighbor.Scale(normal, normal.Bounds(), panel, panel.Bounds(), xdraw.Src, nil)
			hd, key, err := p.raster(g.nativeUIPalette)
			if err != nil {
				t.Fatal(err)
			}
			if hd.Bounds() != normal.Bounds() {
				t.Fatal("HD geometry differs")
			}
			files := []map[string]any{}
			for state, pic := range map[string]image.Image{"original": normal, "hd": hd} {
				name := fmt.Sprintf("slot%d-%s-%s.png", slot, locale, state)
				path := filepath.Join(out, name)
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, pic)
				file.Close()
				if err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				files = append(files, map[string]any{"file": name, "sha256": fmt.Sprintf("%x", sha256.Sum256(raw))})
			}
			rows = append(rows, map[string]any{"locale": locale, "actor_slot": slot, "record_sha256": fmt.Sprintf("%x", sha256.Sum256(record)), "native_panel_sha256": fmt.Sprintf("%x", sha256.Sum256(panel.Pix)), "native_origin": []int{x, y}, "hd_key": key, "name": p.name, "values": p.values, "files": files})
		}
	}
	after, _ := json.Marshal([]any{unitPresentGPUState(g), g.gold, g.nativeRNGState, g.st.NativeTileBlitModes})
	if !bytes.Equal(before, after) {
		t.Fatal("display prototype changed unit, RNG or movement field")
	}
	report := map[string]any{"status": "PASS-SOURCE-PROTOTYPE", "states": len(rows), "pngs": len(rows) * 2, "source_raw_bytes_equal": true, "gameplay_unchanged": true, "rows": rows, "scope": "controlled actors on existing map23 geometry; same NativeBattlePanelRecordForUnit and renderLocalizedNativeBattlePanel used by movement constructor; existing HD panel collector only, no production hook or normal-player/original claim"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "prototype.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("movement source prototype states=%d pngs=%d; original bytes and gameplay kept", len(rows), len(rows)*2)
}
