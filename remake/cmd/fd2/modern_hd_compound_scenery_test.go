package main

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var compoundSceneryGPUStats struct {
	Draws, Pixels int
	Phases        map[string]int
	Sources       map[string]int
}

func initCompoundSceneryFixture(t *testing.T, g *Game) {
	t.Helper()
	if os.Getenv("FD2_HD_COMPOUND_SCENERY") != "1" {
		return
	}
	side, err := strconv.Atoi(os.Getenv("FD2_HD_COMPOUND_SCENERY_SIDE"))
	if err != nil || side != 0 && side != 2 || g.st == nil || len(g.st.Units) == 0 {
		t.Fatal("explicit compound scenery fixture side required")
	}
	g.st.Units[0].NativeRecordByte6 = byte(side)
	surface, err := strconv.Atoi(os.Getenv("FD2_HD_COMPOUND_SCENERY_SURFACE"))
	actor := g.st.Units[0]
	if err != nil || surface < 0 || surface >= 56 || g.m == nil || len(g.m.Tiles) != g.m.W*g.m.H {
		t.Fatal("explicit compound scenery surface required")
	}
	tile := -1
	for index := 0; index < len(g.m.NativeTerrainControl)/4 && index < 1024; index++ {
		if g.m.NativeTerrainControl[index*4+2] == byte(surface) {
			tile = index
			break
		}
	}
	if tile < 0 {
		t.Fatal("no existing native tile selects compound scenery fixture", surface)
	}
	g.m.Tiles = append([]int(nil), g.m.Tiles...)
	g.m.Tiles[actor.Y*g.m.W+actor.X] = tile
	compoundSceneryGPUStats.Draws, compoundSceneryGPUStats.Pixels = 0, 0
	compoundSceneryGPUStats.Phases, compoundSceneryGPUStats.Sources = map[string]int{}, map[string]int{}
}

func auditModernCompoundSceneryGPU(g *Game, frame *nativeCompoundPresentedFrame, plan modernHDBattlePlan) error {
	if os.Getenv("FD2_HD_COMPOUND_SCENERY") != "1" {
		return nil
	}
	plan.HDMix, plan.HDTarget = frame.hdMix, frame.hdTarget
	bg := 0
	for i, layer := range plan.Layers {
		key := ""
		for _, source := range append(append([]modernHDBattleSource(nil), frame.hdSources.BaseScenery...), frame.hdSources.Layers...) {
			if source.Archive != "BG.DAT" && source.Archive != "TAI.DAT" {
				continue
			}
			registered := g.modernStoryPortraits.battleSurfaces[source.Archive][source.Resource]
			if registered == nil || layer.HDKey != registered.hash {
				continue
			}
			if layer.Rect == image.Rect((source.Frame.X+source.DX)*6, (source.Frame.Y+source.DY)*6,
				(source.Frame.X+source.DX+source.Frame.Width)*6, (source.Frame.Y+source.DY+source.Frame.Height)*6) {
				key = fmt.Sprintf("%s:%d", source.Archive, source.Resource)
				if source.Archive == "BG.DAT" {
					bg++
				}
				break
			}
		}
		if key == "" {
			continue
		}
		if layer.HD == nil || layer.HDKey == "" {
			return fmt.Errorf("registered compound scenery not HD: %s", key)
		}
		clip := layer.Rect.Intersect(image.Rect(0, 0, 1920, 1200))
		if layer.Clip != nil {
			clip = clip.Intersect(*layer.Clip)
		}
		if clip.Empty() {
			continue
		}
		pixels := make([]byte, clip.Dx()*clip.Dy()*4)
		g.modernHDBattleGPU.World.SubImage(clip).(*ebiten.Image).ReadPixels(pixels)
		checked := 0
		for y := clip.Min.Y; y < clip.Max.Y; y += 13 {
			for x := clip.Min.X; x < clip.Max.X; x += 17 {
				point := image.Pt(x, y)
				if commandPanelOccluded(plan.Layers[i+1:], point) {
					continue
				}
				r, green, b, a := layer.HD.At(x-layer.Rect.Min.X, y-layer.Rect.Min.Y).RGBA()
				if a != 65535 {
					continue
				}
				at := ((y-clip.Min.Y)*clip.Dx() + x - clip.Min.X) * 4
				for c, value := range []uint32{r, green, b} {
					raw := float32(value>>8)/255 - plan.HDSubtract
					if raw < 0 {
						raw = 0
					}
					want := int(math.Round(float64((raw*(1-plan.HDMix) + plan.HDTarget[c]*plan.HDMix) * 255)))
					diff := int(pixels[at+c]) - want
					if diff < -1 || diff > 1 || pixels[at+3] != 255 {
						return fmt.Errorf("compound scenery %s %s at %v channel%d got%d want%d", frame.hdPhase, key, point, c, pixels[at+c], want)
					}
				}
				checked++
			}
		}
		compoundSceneryGPUStats.Pixels += checked
		compoundSceneryGPUStats.Sources[key] += checked
		compoundSceneryGPUStats.Phases[frame.hdPhase] += checked
	}
	if bg != 1 {
		return fmt.Errorf("compound scenery expected one background, got%d", bg)
	}
	compoundSceneryGPUStats.Draws++
	return nil
}

func appendModernCompoundSceneryReport(t *testing.T, report map[string]any) {
	t.Helper()
	if os.Getenv("FD2_HD_COMPOUND_SCENERY") != "1" {
		return
	}
	if compoundSceneryGPUStats.Draws == 0 || compoundSceneryGPUStats.Pixels < 100 {
		t.Fatal("missing actual compound scenery pixels", compoundSceneryGPUStats)
	}
	report["compound_scenery"] = map[string]any{"side": os.Getenv("FD2_HD_COMPOUND_SCENERY_SIDE"), "surface": os.Getenv("FD2_HD_COMPOUND_SCENERY_SURFACE"), "audited_draws": compoundSceneryGPUStats.Draws,
		"matched_pixels": compoundSceneryGPUStats.Pixels, "sampled_phases": compoundSceneryGPUStats.Phases, "sources": compoundSceneryGPUStats.Sources,
		"scope": "registered BG/TAI actual World readback; unoccluded opaque pixels, existing subtract/mix approximation; controlled fixture"}
}

func TestModernCompoundScenerySources(t *testing.T) {
	value := os.Getenv("FD2_HD_COMMAND_PANEL_OWNER")
	if value == "" {
		t.Skip("one explicit compound owner")
	}
	command, err := strconv.Atoi(value)
	if err != nil || command < 32 || command > 35 {
		t.Fatal("compound owner required")
	}
	g, frames := commandPanelTestFrames(t, command, true)
	selector, err := nativeCommand24BGSelector(g.m, g.st.Units[0])
	if err != nil {
		t.Fatal(err)
	}
	count, platforms := 0, 0
	for _, frame := range frames {
		if frame.hdSources == nil {
			continue
		}
		bg, tai := 0, 0
		for _, source := range append(append([]modernHDBattleSource(nil), frame.hdSources.BaseScenery...), frame.hdSources.Layers...) {
			if (source.Archive == "BG.DAT" || source.Archive == "TAI.DAT") && source.Resource != selector {
				t.Fatal("scenery selector differs from actual caller", source.Archive, source.Resource, selector)
			}
			if source.Archive == "BG.DAT" {
				bg++
			}
			if source.Archive == "TAI.DAT" {
				tai++
			}
		}
		if bg != 1 || tai != int(g.st.Units[0].NativeRecordByte6/2) {
			t.Fatal("missing or duplicate native scenery", frame.hdPhase, bg, tai)
		}
		count++
		platforms += tai
	}
	if count == 0 {
		t.Fatal("vacuous compound coverage")
	}
	if out := os.Getenv("FD2_HD_COMMAND_PANEL_REPORT"); out != "" {
		data, _ := json.MarshalIndent(map[string]any{"status": "PASS-COMPOUND-SCENERY-SOURCES", "command": command,
			"side": g.st.Units[0].NativeRecordByte6, "surface": selector, "battle_frames": count, "platform_frames": platforms, "total_frames": len(frames),
			"native_base_reconstruction": "checked before atomic binding for every battle base", "scope": "controlled source fixture; no new original or normal-player claim"}, "", "  ")
		if err := os.WriteFile(out, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestModernCompoundSceneryAtomic(t *testing.T) {
	g, actor, _ := nativeCommand32PresentationFixture(t)
	selector, err := nativeCommand24BGSelector(g.m, actor)
	if err != nil {
		t.Fatal(err)
	}
	bg, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", selector)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	record, _ := battle.NativeBattlePanelRecordForUnit(actor)
	base, err := g.localizedNativeCommand24BackgroundBase(bg, assets, record, 0, g.handlerChapter, nil)
	if err != nil {
		t.Fatal(err)
	}
	source, err := modernSplitStrikeScenery(bg, selector, nil)
	if err != nil || len(source) != 1 {
		t.Fatal("fixture scenery", err)
	}
	valid := &modernHDBattleSources{Base: base}
	bad := &modernHDBattleSources{Base: append([]byte(nil), base...)}
	for i, mask := range source[0].Frame.Mask {
		if mask != 0 {
			bad.Base[(50+i/source[0].Frame.Width)*320+i%source[0].Frame.Width] ^= 1
			break
		}
	}
	frames := []nativeCompoundPresentedFrame{{hdSources: valid, hdPhase: "actor"}, {hdSources: bad, hdPhase: "effect-main"}}
	before := append([]nativeCompoundPresentedFrame(nil), frames...)
	if err := bindModernHDCompoundScenery(frames, bg, selector, nil, nil); err == nil || !reflect.DeepEqual(before, frames) {
		t.Fatal("late native mismatch published partial sources", err)
	}
	set := &modernStoryPortraitSet{battleSurfaces: map[string]map[int]*modernBattleSurfaceHD{"BG.DAT": {selector: {native: source[0].Frame}}}}
	if err := bindModernHDCompoundScenery(frames[:1], bg, selector, nil, set); err == nil || frames[0].hdSources != valid {
		t.Fatal("broken registered HD source published", err)
	}
	if err := bindModernHDCompoundScenery(frames[:1], bg, selector, nil, nil); err != nil || frames[0].hdSources == valid || len(frames[0].hdSources.BaseScenery) != 1 {
		t.Fatal("native fallback did not bind atomically", err)
	}
}
