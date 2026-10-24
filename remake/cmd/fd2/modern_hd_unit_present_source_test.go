package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func unitPresentHDTestAsset(t *testing.T) (modernThemeAsset, string) {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit unit-present catalog/pack required")
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var found []modernThemeAsset
	for _, a := range doc.Assets {
		if a.Role == "map_overlay_hd_set" && a.SourceResource == 6 && a.FrameCount == 11 {
			found = append(found, a)
		}
	}
	if len(found) != 1 {
		t.Fatal("exactly one declared unit-present group required")
	}
	return found[0], pack
}

func TestModernHDUnitPresentActualSources(t *testing.T) {
	if os.Getenv("FD2_HD_UNIT_PRESENT_SOURCE") != "1" {
		t.Skip("dedicated real-source process required")
	}
	t.Setenv("FD2_MUTE", "1")
	asset, pack := unitPresentHDTestAsset(t)
	set := modernHDMapRuntimeFixture(t).modernStoryPortraits
	if err := loadModernMapOverlayHDAsset(set, asset, pack); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	for index := 114; index <= 124; index++ {
		pic := set.mapEffects.frames[index]
		desc := set.mapEffects.native[index]
		if pic == nil || pic.Bounds() != image.Rect(0, 0, desc.Width*6, desc.Height*6) {
			t.Fatal("actual image absent or wrong bounds")
		}
		varying := 0
		for y := 0; y < pic.Rect.Dy(); y += 6 {
			for x := 0; x < pic.Rect.Dx(); x += 6 {
				first := pic.RGBAAt(x, y)
				different := false
				for yy := 0; yy < 6; yy++ {
					for xx := 0; xx < 6; xx++ {
						if pic.RGBAAt(x+xx, y+yy) != first {
							different = true
						}
					}
				}
				if different {
					varying++
				}
			}
		}
		if varying == 0 {
			t.Fatal("HD effect is only nearest-neighbour enlargement", index)
		}
		rows = append(rows, map[string]any{"index": index, "width": pic.Rect.Dx(), "height": pic.Rect.Dy(), "varying_native_blocks": varying})
	}
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	g.modernStoryPortraits = set
	if err := g.startNativeUnitPresent(campaign.NativeUnitPresent{Slot: 0, NewX: 255, NewY: 255, VisualX: 0, VisualY: 0}, nil); err != nil {
		t.Fatal(err)
	}
	j := g.nativeUnitPresent
	if len(j.hdFrames) != 45 {
		t.Fatal("actual 18-row producer incomplete")
	}
	for i := 0; i < 11; i++ {
		f := j.hdFrames[i]
		if len(f.Terrain) == 0 || f.Terrain[len(f.Terrain)-1].Image != set.mapEffects.frames[114+i] {
			t.Fatal("intro not bound to actual source", i)
		}
	}
	rejections := 0
	for _, kind := range []string{"metadata", "hash", "shape", "partial", "descriptor"} {
		raw, _ := json.Marshal(asset)
		var bad modernThemeAsset
		if err := json.Unmarshal(raw, &bad); err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "metadata":
			bad.SourceMetadataSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		case "hash":
			bad.FrameSHA256[0] = "0000000000000000000000000000000000000000000000000000000000000000"
		case "shape":
			bad.Width = 1919
		case "partial":
			bad.Files = bad.Files[:10]
		case "descriptor":
			bad.NativeLMIFrames[0].Width++
		}
		dst := &modernStoryPortraitSet{}
		if err := loadModernMapOverlayHDAsset(dst, bad, pack); err == nil || dst.mapEffects != nil {
			t.Fatal("invalid actual bank published", kind)
		}
		rejections++
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"status": "PASS-UNIT-PRESENT-HD-SOURCES", "frames": 11, "native_frames": 45, "rejections": rejections, "sources": rows, "scope": "actual LMI/source metadata with synthetic world images; complete Game.Draw is a separate gate"}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
