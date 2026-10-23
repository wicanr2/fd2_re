package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestModernBattleSceneryLegacyFallbackGPU(t *testing.T) {
	requireAllSceneryProcess(t)
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Assets) != 684 {
		t.Fatal("explicit previous684 catalog required", len(doc.Assets))
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "cd4b81d9c3b9d1d6cd2a97c10eca6e27d8841698f47da4290816c99d86f96309" {
		t.Fatal("previous684 catalog SHA differs")
	}
	set := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), nil)
	registered := 0
	for _, bank := range set.battleSurfaces {
		registered += len(bank)
	}
	if registered != 9 {
		t.Fatal("previous registered surface count differs", registered)
	}
	_, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{modernStoryPortraits: set}
	fixtures := []struct {
		archive  string
		resource int
	}{
		{"BG.DAT", 3}, {"BG.DAT", 34}, {"BG.DAT", 36}, {"BG.DAT", 37},
		{"TAI.DAT", 7}, {"TAI.DAT", 32}, {"TAI.DAT", 49}, {"TAI.DAT", 55},
	}
	var cases []modernHDBattleGPUCase
	var ids []string
	for _, f := range fixtures {
		if set.battleSurfaces[f.archive][f.resource] != nil {
			t.Fatal("new source unexpectedly registered", f)
		}
		native, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), f.archive, f.resource)
		if err != nil {
			t.Fatal(err)
		}
		frame, err := modernHDBattleSurfaceFrame(native)
		if err != nil {
			t.Fatal(err)
		}
		frame.X, frame.Y = 0, 50
		if f.archive == "TAI.DAT" {
			frame.X, frame.Y = 164, 157
		}
		identity := fmt.Sprintf("%s:%03d", f.archive, f.resource)
		ids = append(ids, identity)
		for variant := 0; variant < 2; variant++ {
			source := modernHDBattleSource{Archive: f.archive, Resource: f.resource, Frame: frame, Fill: -1}
			subtract := float32(0)
			if variant != 0 {
				subtract = float32(12) / 63
				clip := image.Rect(9, 40, 303, 160)
				source.DX = -21
				if f.archive == "TAI.DAT" {
					source.DX = 35
					clip = image.Rect(160, 150, 312, 200)
				}
				source.Clip = &clip
			}
			sources := &modernHDBattleSources{Base: make([]byte, 64000), BaseScenery: []modernHDBattleSource{source}}
			plan, err := g.modernHDBattlePlan(sources, palette, subtract)
			if err != nil || len(plan.Layers) != 1 || plan.Layers[0].HD != nil || plan.Layers[0].HDKey != "" {
				t.Fatal("legacy source did not use native fallback", identity, err)
			}
			if variant == 0 && plan.Layers[0].Rect != image.Rect(frame.X*6, frame.Y*6, (frame.X+frame.Width)*6, (frame.Y+frame.Height)*6) {
				t.Fatal("native fallback rectangle differs", identity, plan.Layers[0].Rect)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("legacy684/%s/%d", identity, variant), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
		}
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) || probe.index != 16 {
		t.Fatalf("legacy fallback GPU %d/%d: %v", probe.index, len(cases), probe.err)
	}
	out := os.Getenv("FD2_TEST_HD_SCENERY_OUT")
	if out == "" {
		t.Fatal("legacy fallback receipt path required")
	}
	report := map[string]any{"status": "PASS-OLD684-NEW-SCENERY-FALLBACK-GPU", "legacy_catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "legacy_registered_surfaces": registered, "source_ids": ids, "completed_gpu_cases": probe.index, "tolerance": 1, "scope": "old684 catalog/native fallback, three BG geometries and short/narrow TAI; isolated actual GPU, not normal-player or original RGB parity"}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("old684 catalog leaves all8 new sources undeclared;16 native fallback GPU worlds with clipped translation/shade passed")
}
