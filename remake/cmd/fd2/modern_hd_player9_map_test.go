package main

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// 草案幾何探針：先確認凍結輸入可重生真正snapshot，不把相近背景接入HD。
func TestModernHDPlayer9FrozenInputMatchesSnapshot(t *testing.T) {
	g, _, _ := ordinaryCommand0HDTestGame(t)
	in, err := g.nativeCh28PostFrameInput(g.st)
	if err != nil {
		t.Fatal(err)
	}
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := indexedmap.ComposeNativeFrame(work, vga, in); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(vga, g.nativeMapVGA) {
		count := 0
		for i, b := range vga {
			if b != g.nativeMapVGA[i] {
				count++
			}
		}
		t.Fatalf("frozen native input differs from snapshot by %d bytes", count)
	}
}

func player9MapTestGame(t *testing.T, hit bool) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t)
	chance := 0
	if hit {
		chance = 100
	}
	g.st.NativeCommandBook[9] = battle.NativeCommandRecord{ID: 9, Damage: 999, Hit: chance, SelectionMode: 3, EffectMode: 0, MPCost: 30, TargetCode: 0}
	return g, actor, target
}

// Independent byte rasterizer: no native compositor, BlitAt, or HD placement
// helper is used to reproduce the effect/result snapshots.
func independentPlayer9VGA(t *testing.T, g *Game, j *nativeCommand9PlayerJob, layers []indexedmap.NativeCommandTailLayer) []byte {
	t.Helper()
	work := append([]byte(nil), j.baselineWork...)
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	for _, layer := range layers {
		var entries []fdother.LMI1Entry
		if layer.Resource == 6 {
			entries = g.nativeMapAssets.FDOTHER6
		} else if layer.Resource == 5 {
			entries = g.nativeMapAssets.CommandHealDigits
		} else {
			t.Fatal("unexpected source")
		}
		entry := entries[layer.Entry]
		for y := 0; y < entry.Height; y++ {
			for x := 0; x < entry.Width; x++ {
				value := entry.Pixels[y*entry.Width+x]
				if value == 0 {
					continue
				}
				at := base + (layer.Y+y)*fdicon.NativeMapStride + layer.X + x
				if at < 0 || at >= len(work) {
					t.Fatal("source exceeded native work")
				}
				work[at] = value
			}
		}
	}
	want := append([]byte(nil), j.baselineVGA...)
	for y := 0; y < 192; y++ {
		for x := 0; x < 312; x++ {
			want[y*320+x] = work[base+y*fdicon.NativeMapStride+x]
		}
	}
	return want
}

func TestModernHDPlayer9NativeSourcesMatchFullVGA(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	for _, hit := range []bool{false, true} {
		g, actor, target := player9MapTestGame(t, hit)
		if err := g.startNativeCommand9PlayerPresentation(actor, target, nil); err != nil {
			t.Fatal(err)
		}
		j := g.nativeCmd9Player
		if j.hdInputErr != nil {
			t.Fatal(j.hdInputErr)
		}
		if len(j.effectSources) != 27 || len(j.resultSources) != 22 || j.plan.Results[0].Hit != hit {
			t.Fatal("incomplete native player9 producer")
		}
		for i, sources := range j.effectSources {
			if len(sources) != 1 || sources[0].Resource != 6 || sources[0].Entry != 87+i || sources[0].X != 24 || sources[0].Y != -6 {
				t.Fatalf("wrong original effect origin %d %+v", i, sources)
			}
			if !bytes.Equal(independentPlayer9VGA(t, g, j, sources), j.effectFrames[i]) {
				t.Fatalf("independent effect %d differs", i)
			}
		}
		for i, sources := range j.resultSources {
			if !hit {
				if len(sources) != 4 {
					t.Fatal("MISS must preserve four ordered glyph writes")
				}
				for k, index := range []int{116, 117, 118, 118} {
					if sources[k].Entry != index {
						t.Fatal("wrong MISS source")
					}
				}
			}
			if !bytes.Equal(independentPlayer9VGA(t, g, j, sources), j.resultFrames[i]) {
				t.Fatalf("independent result %d differs", i)
			}
		}
		// Frozen records survive the numerical commit; reconstruction cannot
		// consult the now-dead target when F2 is pressed during the result.
		before := append([]fdicon.NativeUnitLayerEntry(nil), j.hdInput.Frame.Units...)
		target.HP = 0
		if !reflect.DeepEqual(before, j.hdInput.Frame.Units) {
			t.Fatal("input followed mutable target state")
		}
	}
}

func TestModernHDPlayer9FrozenInputRejectsDifferentSnapshot(t *testing.T) {
	g, _, _ := player9MapTestGame(t, true)
	if _, err := g.freezeNativeCommand9MapInput(); err != nil {
		t.Fatal("positive snapshot", err)
	}
	g.nativeMapVGA[4*320+4] ^= 1
	if _, err := g.freezeNativeCommand9MapInput(); err == nil {
		t.Fatal("different steady snapshot accepted")
	}
}

func clonePlayer9MapTheme(set *modernStoryPortraitSet) *modernStoryPortraitSet {
	clone := *set
	copyBank := func(bank *modernMapOverlayHDResource) *modernMapOverlayHDResource {
		if bank == nil {
			return nil
		}
		out := &modernMapOverlayHDResource{frames: make(map[int]*image.RGBA), native: make(map[int]modernLMIHDDescriptor)}
		for i, p := range bank.frames {
			out.frames[i] = p
		}
		for i, d := range bank.native {
			out.native[i] = d
		}
		return out
	}
	clone.mapOverlays, clone.mapEffects = copyBank(set.mapOverlays), copyBank(set.mapEffects)
	return &clone
}

func TestModernHDPlayer9FullThemePreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_PLAYER9_MAP_CATALOG"), os.Getenv("FD2_TEST_PLAYER9_MAP_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit complete player9 candidate required")
	}
	t.Setenv("FD2_MUTE", "1")
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("complete candidate theme loaded; checking five start/F2 refusals")
	for _, fault := range []string{"late-effect", "unused-digit", "changed-native", "partial-group", "missing-world"} {
		t.Run(fault, func(t *testing.T) {
			bad := clonePlayer9MapTheme(set)
			switch fault {
			case "late-effect":
				delete(bad.mapEffects.frames, 113)
			case "unused-digit":
				delete(bad.mapOverlays.frames, 103)
			case "changed-native":
				d := bad.mapOverlays.native[94]
				d.PixelsSHA256 = "changed"
				bad.mapOverlays.native[94] = d
			case "partial-group":
				bad.mapEffects = nil
			case "missing-world":
				bad.mapSpritesHD = nil
			}
			g, a, b := player9MapTestGame(t, true)
			g.modernStoryPortraits = bad
			beforeA, beforeB, seed := *a, *b, g.nativeRNGState
			if err := g.startNativeCommand9PlayerPresentation(a, b, nil); err == nil || g.nativeCmd9Player != nil || !reflect.DeepEqual(beforeA, *a) || !reflect.DeepEqual(beforeB, *b) || g.nativeRNGState != seed {
				t.Fatal("failed start published owner/state", err)
			}
			g.modernStoryPortraits = nil
			if err := g.startNativeCommand9PlayerPresentation(a, b, nil); err != nil {
				t.Fatal(err)
			}
			job := g.nativeCmd9Player
			phase, frame, hold, drawn := job.phase, job.frame, job.hold, job.drawn
			if err := g.applyTheme(bad); err == nil || g.modernStoryPortraits != nil || g.nativeCmd9Player != job || job.phase != phase || job.frame != frame || job.hold != hold || job.drawn != drawn || !reflect.DeepEqual(beforeA, *a) || !reflect.DeepEqual(beforeB, *b) || g.nativeRNGState != seed {
				t.Fatal("failed F2 changed transaction", err)
			}
			if err := g.applyTheme(set); err != nil || job.hdFrames == nil || !job.hdFrames.UsesHDEffects {
				t.Fatal("positive F2 failed", err)
			}
		})
	}
	// A missing PNG is refused by the actual complete-theme loader, including
	// a late effect and an unused result digit, before a set can be published.
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []int{5, 6} {
		var edited map[string]any
		if err := json.Unmarshal(raw, &edited); err != nil {
			t.Fatal(err)
		}
		for _, value := range edited["assets"].([]any) {
			a := value.(map[string]any)
			if a["role"] == "map_overlay_hd_set" && int(a["source_resource"].(float64)) == resource && int(a["frame_count"].(float64)) != 12 {
				files := a["files"].([]any)
				files[len(files)-1] = "missing-player9.png"
			}
		}
		// A complete valid theme was admitted above. Keep every asset in the
		// negative catalog but put the broken bank first so this tests its
		// startup refusal without decoding 1.89 billion unrelated pixels again.
		all := edited["assets"].([]any)
		for i, value := range all {
			a := value.(map[string]any)
			if a["role"] == "map_overlay_hd_set" && int(a["source_resource"].(float64)) == resource && int(a["frame_count"].(float64)) != 12 {
				edited["assets"] = append([]any{value}, append(append([]any(nil), all[:i]...), all[i+1:]...)...)
				break
			}
		}
		data, _ := json.Marshal(edited)
		path := filepath.Join(t.TempDir(), "catalog.json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if loaded, err := loadModernStoryPortraitSet(path, pack); err == nil || loaded != nil {
			t.Fatal("missing registered source published a theme")
		}
	}
}
