package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var compoundMapGPUStats struct {
	Draws, Pixels, HDLayers, NativeLayers int
	Phases                                map[string]int
	Worlds                                map[*modernHDMapFrame]*image.RGBA
	Captures                              map[string]bool
	Readback                              []byte
	OverlayPixels, ChangedOverlayPixels   int
}

func initCompoundMapFixture(t *testing.T, g *Game) {
	t.Helper()
	if os.Getenv("FD2_HD_COMPOUND_MAP") != "1" {
		return
	}
	// Earlier fixture changes to raw side/terrain must be in the native snapshot.
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	compoundMapGPUStats.Draws, compoundMapGPUStats.Pixels = 0, 0
	compoundMapGPUStats.HDLayers, compoundMapGPUStats.NativeLayers = 0, 0
	compoundMapGPUStats.OverlayPixels, compoundMapGPUStats.ChangedOverlayPixels = 0, 0
	compoundMapGPUStats.Phases = make(map[string]int)
	compoundMapGPUStats.Worlds = make(map[*modernHDMapFrame]*image.RGBA)
	compoundMapGPUStats.Captures = make(map[string]bool)
}

func TestModernCompoundMapBindingAtomic(t *testing.T) {
	g := &Game{}
	if _, err := g.stageModernHDCompoundMap(nil, &modernStoryPortraitSet{mapSpritesHD: map[int][]image.Image{1: nil}}); err != nil {
		t.Fatal("inactive compound owner changed theme admission", err)
	}
	scene := &modernHDCompoundMapScene{}
	frames := make([]nativeCompoundPresentedFrame, 2)
	err := bindModernHDCompoundMap(frames, scene, [][]byte{make([]byte, 64000), nil}, nil, nil, 0, false, "bad")
	if err == nil || frames[0].hdMap != nil || frames[1].hdMap != nil {
		t.Fatal("partial map source publication")
	}
}

func TestModernCompoundMapAllSources(t *testing.T) {
	if os.Getenv("FD2_HD_COMPOUND_MAP") != "1" {
		t.Skip("dedicated compound map source gate")
	}
	command, err := strconv.Atoi(os.Getenv("FD2_HD_COMMAND_PANEL_OWNER"))
	if err != nil || command < 32 || command > 35 {
		t.Fatal("explicit compound owner32-35 required")
	}
	g, frames := commandPanelTestFrames(t, command, false)
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	beforeUnits := compoundMapUnitSnapshot(g)
	beforeRNG := g.nativeRNGState
	publish, err := g.stageModernHDCompoundMap(frames, set)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range frames {
		if f.hdMap != nil && f.hdMap.prepared != nil {
			t.Fatal("candidate published before closure")
		}
	}
	publish()
	count := 0
	phases := map[string]int{}
	sources := map[string]int{}
	hd, native := 0, 0
	var last *modernHDCompoundMapSource
	for _, f := range frames {
		if f.hdMap == nil {
			if f.hdSources == nil {
				t.Fatal("unbound presentation", f.hdPhase)
			}
			continue
		}
		s := f.hdMap
		count++
		phases[f.hdPhase]++
		last = s
		if s.prepared == nil || s.theme != set {
			t.Fatal("map frame not prepared")
		}
		for _, layer := range s.layers {
			sources[fmt.Sprintf("%d:%d", layer.Resource, layer.Entry)]++
			bank := set.mapEffects
			if layer.Resource == 5 {
				bank = set.mapOverlays
			}
			if bank != nil && bank.frames[layer.Entry] != nil {
				hd++
			} else {
				native++
			}
		}
	}
	if count < 80 || phases["map-ramp"] != 41 || last == nil {
		t.Fatal("missing complete map phases", phases)
	}
	if !reflect.DeepEqual(beforeUnits, compoundMapUnitSnapshot(g)) || beforeRNG != g.nativeRNGState {
		t.Fatal("map preparation changed live state")
	}
	active := last.prepared
	g.modernStoryPortraits = set
	bad := *set
	old := last.pixels
	last.pixels = append([]byte(nil), old...)
	last.pixels[0] ^= 1
	if _, err := g.stageModernHDCompoundMap(frames, &bad); err == nil || last.prepared != active || last.theme != set {
		t.Fatal("bad last native source replaced active theme")
	}
	last.pixels = old
	// The green/red numeric bank is already declared. Missing an entry may not fall back.
	bad = *set
	bank := *set.mapOverlays
	bank.frames = make(map[int]*image.RGBA)
	for k, v := range set.mapOverlays.frames {
		bank.frames[k] = v
	}
	bad.mapOverlays = &bank
	removed := false
	for _, layer := range last.layers {
		if layer.Resource == 5 && bank.frames[layer.Entry] != nil {
			delete(bank.frames, layer.Entry)
			removed = true
			break
		}
	}
	if !removed {
		t.Fatal("fixture has no registered final digit")
	}
	if err := g.applyTheme(&bad); err == nil || g.modernStoryPortraits != set || last.prepared != active || last.theme != set {
		t.Fatal("missing declared digit replaced active theme")
	}
	report := map[string]any{"status": "PASS-COMPOUND-MAP-SOURCES", "command": command, "frames": len(frames), "map_frames": count, "phases": phases, "sources": sources, "registered_layers": hd, "native_fallback_layers": native, "native_reconstruction": true, "atomic_rejections": 2, "rng_unchanged": true, "unit_state_unchanged": true}
	if out := os.Getenv("FD2_HD_COMPOUND_MAP_REPORT"); out != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(out, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func (p *modernHDCompoundProbe) drawCompoundMapTail(f *nativeCompoundPresentedFrame, index int, screen *ebiten.Image) {
	if p.g.modernStoryPortraits == nil {
		if p.g.modernHDWorldVisible || p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("faithful compound map retained HD")
		}
		p.faithful++
		return
	}
	s := f.hdMap
	if !p.g.modernHDWorldVisible || p.g.modernHDBattleVisible || s.prepared == nil || p.g.modernHDMapPresented != s.prepared {
		p.err = fmt.Errorf("compound map did not draw its frozen HD world")
		return
	}
	p.hd++
	p.phaseDraws[f.hdPhase]++
	stats := &compoundMapGPUStats
	want := stats.Worlds[s.prepared]
	if want == nil {
		want = healHDCPUWorld(s.prepared, p.g.modernStoryPortraits)
		stats.Worlds = map[*modernHDMapFrame]*image.RGBA{s.prepared: want}
	}
	if len(stats.Readback) != 1920*1200*4 {
		stats.Readback = make([]byte, 1920*1200*4)
	}
	got := stats.Readback
	p.g.modernHDMapGPU.World.ReadPixels(got)
	// Check a deterministic whole-canvas lattice, including borders, UI and overlays.
	for y := 0; y < 1200; y += 11 {
		for x := 0; x < 1920; x += 13 {
			at := y*want.Stride + x*4
			for c := 0; c < 4; c++ {
				value := int(want.Pix[at+c])
				if c < 3 {
					a := float32(want.Pix[at+3]) / 255
					value = int(math.Round(float64(float32(value)*(1-s.mix) + s.target[c]*255*a*s.mix)))
				}
				diff := int(got[at+c]) - value
				if diff < -2 || diff > 2 {
					p.err = fmt.Errorf("compound map %s (%d,%d) channel%d got%d want%d", f.hdPhase, x, y, c, got[at+c], value)
					return
				}
			}
			stats.Pixels++
		}
	}
	stats.Draws++
	stats.Phases[f.hdPhase]++
	for i, source := range s.layers {
		bank := p.g.modernStoryPortraits.mapEffects
		if source.Resource == 5 {
			bank = p.g.modernStoryPortraits.mapOverlays
		}
		if bank != nil && bank.frames[source.Entry] != nil {
			stats.HDLayers++
			layerAt := len(s.prepared.Overlays) - len(s.layers) + i
			layer := s.prepared.Overlays[layerAt]
			if layer.Image != bank.frames[source.Entry] {
				p.err = fmt.Errorf("registered compound map source was substituted")
				return
			}
			for y := 0; y < layer.Image.Bounds().Dy(); y += 3 {
				for x := 0; x < layer.Image.Bounds().Dx(); x += 3 {
					point := layer.Rect.Min.Add(image.Pt(x, y))
					if !point.In(s.prepared.Viewport) || layer.Image.RGBAAt(x, y).A != 255 {
						continue
					}
					covered := false
					for _, later := range s.prepared.Overlays[layerAt+1:] {
						if point.In(later.Rect) {
							_, _, _, a := later.Image.At(point.X-later.Rect.Min.X, point.Y-later.Rect.Min.Y).RGBA()
							covered = covered || a != 0
						}
					}
					if covered {
						continue
					}
					at := (point.Y*1920 + point.X) * 4
					colour := layer.Image.RGBAAt(x, y)
					for c, v := range []byte{colour.R, colour.G, colour.B, colour.A} {
						diff := int(got[at+c]) - int(v)
						if diff < -2 || diff > 2 {
							p.err = fmt.Errorf("registered compound map pixel differs %d:%d at%v", source.Resource, source.Entry, point)
							return
						}
					}
					stats.OverlayPixels++
					nativeAt := point.Y/6*320 + point.X/6
					r, green, b, _ := s.scene.palette[s.pixels[nativeAt]].RGBA()
					if colour.R != byte(r>>8) || colour.G != byte(green>>8) || colour.B != byte(b>>8) {
						stats.ChangedOverlayPixels++
					}
				}
			}
		} else {
			stats.NativeLayers++
		}
	}
	// Retain the first and last frame of each phase plus each original LMI index.
	key := f.hdPhase
	if f.hdPhase == "map-ramp" {
		if s.mix != 0 && s.mix != 0.5 && s.mix != 1 {
			return
		}
		key += fmt.Sprint(s.mix)
	}
	for _, layer := range s.layers {
		key += fmt.Sprintf("/%d:%d", layer.Resource, layer.Entry)
	}
	if s.event && len(s.maskTargets) > 0 {
		key += "/mask"
	}
	if stats.Captures[key] || p.out == "" {
		return
	}
	stats.Captures[key] = true
	if p.readback == nil {
		p.readback = ebiten.NewImage(1920, 1200)
		p.readbackPixels = make([]byte, 1920*1200*4)
	}
	p.readback.Clear()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(p.readback, screen, geometry)
	p.readback.ReadPixels(p.readbackPixels)
	if !bytes.Equal(p.readbackPixels, got) {
		p.err = fmt.Errorf("final compositor differs from compound HD world")
		return
	}
	label := fmt.Sprintf("%s-present-%03d.png", f.hdPhase, index)
	file, err := os.Create(filepath.Join(p.out, label))
	if err != nil {
		p.err = err
		return
	}
	err = png.Encode(file, &image.RGBA{Pix: p.readbackPixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
	closeErr := file.Close()
	if err != nil {
		p.err = err
		return
	}
	if closeErr != nil {
		p.err = closeErr
		return
	}
	if p.sourceShots == nil {
		p.sourceShots = make(map[string]any)
	}
	p.sourceShots[label] = map[string]any{"phase": f.hdPhase, "map": true, "mix": s.mix, "layers": s.layers, "event_viewport": s.event}
	p.shots[label] = true
}

func appendCompoundMapGPUReport(t *testing.T, report map[string]any) {
	t.Helper()
	if os.Getenv("FD2_HD_COMPOUND_MAP") != "1" {
		return
	}
	s := compoundMapGPUStats
	if s.Draws < 80 || s.Pixels < 10000 || s.Phases["map-ramp"] == 0 || s.HDLayers == 0 || s.OverlayPixels == 0 || s.ChangedOverlayPixels == 0 {
		t.Fatal("missing compound map actual GPU coverage", s.Draws, s.Phases)
	}
	phases := report["phase_f2_switches"].(map[string]bool)
	for phase := range s.Phases {
		if !phases[phase] {
			t.Fatal("map phase lacks F2 roundtrip", phase)
		}
	}
	report["compound_map"] = map[string]any{"audited_draws": s.Draws, "matched_pixels": s.Pixels, "phases": s.Phases, "registered_layers": s.HDLayers, "native_fallback_layers": s.NativeLayers, "registered_overlay_pixels": s.OverlayPixels, "changed_overlay_pixels": s.ChangedOverlayPixels, "cpu_lattice_tolerance": 2}
}

// Avoid confusing pointer-list formatting with a unit-state comparison.
func compoundMapUnitSnapshot(g *Game) any {
	var units []any
	for _, u := range g.st.Units {
		if u == nil {
			units = append(units, nil)
		} else {
			units = append(units, *u)
		}
	}
	return units
}
