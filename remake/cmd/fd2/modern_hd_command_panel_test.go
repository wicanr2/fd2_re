package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var commandPanelGPUStats struct {
	Draws, Samples int
	Phases         map[string]int
	Captures       map[string]bool
}

func maybeAttachCommandPanelHDTest(t *testing.T, g *Game) {
	if os.Getenv("FD2_HD_COMMAND_PANEL_TEXT") != "1" {
		return
	}
	attachCommandPanelHDTest(t, g)
	commandPanelGPUStats.Draws, commandPanelGPUStats.Samples = 0, 0
	commandPanelGPUStats.Phases = make(map[string]int)
	commandPanelGPUStats.Captures = make(map[string]bool)
	initCompoundSceneryFixture(t, g)
}

// All presents are audited. PNGs retain each typed state and scroll half,
// instead of repeating the same panel for every effect/idle frame.
func wantCommandPanelCapture(frame *nativeCompoundPresentedFrame) bool {
	if os.Getenv("FD2_HD_COMMAND_PANEL_TEXT") != "1" {
		return true
	}
	key := frame.hdPhase
	if frame.hdSources != nil {
		for _, s := range append(append([]modernHDBattleSource(nil), frame.hdSources.BaseScenery...), frame.hdSources.Layers...) {
			if p := s.PanelText; p != nil {
				key += fmt.Sprintf("/%s/%v/%d/%d", p.name, p.values, s.DX/160, s.DY/100)
			}
		}
	}
	if commandPanelGPUStats.Captures[key] {
		return false
	}
	commandPanelGPUStats.Captures[key] = true
	return true
}

func appendCommandPanelGPUReport(t *testing.T, report map[string]any) {
	t.Helper()
	if os.Getenv("FD2_HD_COMMAND_PANEL_TEXT") != "1" {
		return
	}
	if commandPanelGPUStats.Draws == 0 || commandPanelGPUStats.Samples < 100 {
		t.Fatal("missing actual GPU panel glyph samples", commandPanelGPUStats)
	}
	report["panel_text"] = map[string]any{"locale": os.Getenv("FD2_HD_PANEL_LOCALE"), "audited_draws": commandPanelGPUStats.Draws, "matched_rgba_pixels": commandPanelGPUStats.Samples, "sampled_phases": commandPanelGPUStats.Phases, "scope": "actual battle World readback; only unoccluded glyph pixels; fixture only"}
	appendModernCompoundSceneryReport(t, report)
	appendModernOrdinarySceneryReport(t, report)
}

func commandPanelOccluded(layers []modernHDBattleLayer, point image.Point) bool {
	for _, layer := range layers {
		if !point.In(layer.Rect) || layer.Clip != nil && !point.In(*layer.Clip) {
			continue
		}
		x, y := point.X-layer.Rect.Min.X, point.Y-layer.Rect.Min.Y
		if layer.HD != nil {
			bounds := layer.HD.Bounds()
			x = x * bounds.Dx() / layer.Rect.Dx()
			y = y * bounds.Dy() / layer.Rect.Dy()
			_, _, _, a := layer.HD.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			if a != 0 {
				return true
			}
		} else {
			x = x * layer.Native.Width / layer.Rect.Dx()
			y = y * layer.Native.Height / layer.Rect.Dy()
			at := y*layer.Native.Width + x
			if at < 0 || at >= len(layer.Native.Mask) || layer.Native.Mask[at] != 0 {
				return true
			}
		}
	}
	return false
}

func auditCommandPanelGPU(g *Game, frame *nativeCompoundPresentedFrame) error {
	if os.Getenv("FD2_HD_COMMAND_PANEL_TEXT") != "1" || g.modernStoryPortraits == nil || frame == nil || frame.hdSources == nil {
		return nil
	}
	if !g.modernHDBattleVisible || g.modernHDBattleGPU == nil || g.modernHDBattleGPU.World == nil {
		return fmt.Errorf("panel actual battle World unavailable")
	}
	plan, err := g.modernHDBattlePlan(frame.hdSources, frame.hdPalette, frame.hdSubtract)
	if err != nil {
		return err
	}
	for i, layer := range plan.Layers {
		if !layer.PreservePalette || layer.HD == nil {
			continue
		}
		clip := layer.Rect.Intersect(image.Rect(0, 0, 1920, 1200))
		if layer.Clip != nil {
			clip = clip.Intersect(*layer.Clip)
		}
		if clip.Empty() {
			continue
		}
		var points []image.Point
		for _, rect := range []image.Rectangle{image.Rect(5, 4, 114, 20), image.Rect(115, 4, 131, 13), image.Rect(132, 4, 147, 13), image.Rect(3, 21, 20, 30), image.Rect(126, 21, 147, 30), image.Rect(3, 30, 20, 39), image.Rect(126, 30, 147, 39)} {
			var glyph []image.Point
			background := layer.HD.At(100*6, 20*6)
			for y := rect.Min.Y * 6; y < rect.Max.Y*6; y += 2 {
				for x := rect.Min.X * 6; x < rect.Max.X*6; x += 2 {
					p := image.Pt(x+layer.Rect.Min.X, y+layer.Rect.Min.Y)
					if p.In(clip) && layer.HD.At(x, y) != background && !commandPanelOccluded(plan.Layers[i+1:], p) {
						glyph = append(glyph, p)
					}
				}
			}
			stride := len(glyph) / 32
			if stride < 1 {
				stride = 1
			}
			for k := 0; k < len(glyph); k += stride {
				points = append(points, glyph[k])
			}
		}
		if len(points) == 0 {
			continue
		}
		pixels := make([]byte, clip.Dx()*clip.Dy()*4)
		g.modernHDBattleGPU.World.SubImage(clip).(*ebiten.Image).ReadPixels(pixels)
		for _, point := range points {
			r, green, b, a := layer.HD.At(point.X-layer.Rect.Min.X, point.Y-layer.Rect.Min.Y).RGBA()
			want := []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), byte(a >> 8)}
			at := ((point.Y-clip.Min.Y)*clip.Dx() + point.X - clip.Min.X) * 4
			for c, v := range want {
				diff := int(pixels[at+c]) - int(v)
				if diff < -1 || diff > 1 {
					return fmt.Errorf("panel glyph %s at %v channel%d got%d want%d", frame.hdPhase, point, c, pixels[at+c], v)
				}
			}
		}
		commandPanelGPUStats.Samples += len(points)
		commandPanelGPUStats.Phases[frame.hdPhase] += len(points)
	}
	commandPanelGPUStats.Draws++
	if err := auditModernCompoundSceneryGPU(g, frame, plan); err != nil {
		return err
	}
	return auditModernOrdinarySceneryGPU(g, frame, plan)
}

func attachCommandPanelHDTest(t *testing.T, g *Game) {
	t.Helper()
	g.font = loadFont()
	locale := os.Getenv("FD2_HD_PANEL_LOCALE")
	if locale == "" {
		locale = "zh-Hant"
	}
	g.localeID = locale
	var err error
	g.localeEntities, err = loadOfficialLocaleEntities(locale)
	if err != nil || g.font == nil {
		t.Fatal("panel font or entity catalog unavailable", err)
	}
}

func commandPanelTestFrames(t *testing.T, command int, hd bool) (*Game, []nativeCompoundPresentedFrame) {
	t.Helper()
	var g *Game
	var actor, target *battle.Unit
	switch command {
	case 0:
		g, actor, target = ordinaryCommand0HDTestGame(t)
	case 1:
		g, actor, target = ordinaryCommand1HDTestGame(t)
	case 4, 5:
		g, actor, target = ordinaryCommand5HDTestGame(t)
		if command == 4 {
			record := g.st.NativeCommandBook[5]
			record.ID = 4
			g.st.NativeCommandBook[4] = record
		}
	case 2, 3, 6, 7, 8, 9:
		g, actor, target = ordinaryRemainingHDTestGame(t, command)
	case 24:
		g, actor, target = newNativeCommand24PresentationTestGame(t)
	case 29:
		var targets []*battle.Unit
		g, actor, targets = newNativeCommand29PresentationTestGame(t)
		target = targets[0]
	case 32:
		g, actor, target = nativeCommand32PresentationFixture(t)
	case 33:
		g, actor, target = nativeCommand33PresentationFixture(t)
	case 34:
		g, actor, target = nativeCommand34PresentationFixture(t)
	case 35:
		g, actor, target = nativeCommand35PresentationFixture(t)
	default:
		t.Fatal("unsupported command", command)
	}
	initOrdinarySceneryFixture(t, g, actor, target)
	if hd {
		attachCommandPanelHDTest(t, g)
	}
	g.nativeRNGState = 7
	if command == 9 {
		g.nativeRNGState = 3
	}
	var err error
	var frames []nativeCompoundPresentedFrame
	switch command {
	case 0:
		err = g.startNativeCommand0Presentation(actor, target, nil)
		if err == nil {
			frames = g.nativeCmd0Presentation.hd.sourceFrames()
		}
	case 1:
		err = g.startNativeCommand1Presentation(actor, target, nil)
		if err == nil {
			frames = g.nativeCmd1Presentation.hd.sourceFrames()
		}
	case 4, 5:
		err = g.startNativeCommand45Presentation(actor, target, command, nil)
		if err == nil {
			frames = g.nativeCmd5Presentation.hd.sourceFrames()
		}
	case 2, 3, 6, 7, 8, 9:
		err = startOrdinaryRemainingHD(g, command, actor, target, nil)
		if err == nil {
			frames = ordinaryRemainingOwner(g, command).hd.sourceFrames()
		}
	case 24:
		err = g.startNativeCommand24Presentation(actor, target, nil)
		if err == nil {
			frames = commandPanelSplitTestFrames(g.nativeCmd24Presentation.hd)
		}
	case 29:
		err = g.startNativeCommand29Presentation(actor, target, nil)
		if err == nil {
			frames = commandPanelSplitTestFrames(g.nativeCmd29Presentation.hd)
		}
	case 32:
		err = g.startNativeCommand32Presentation(actor, target, nil)
		if err == nil {
			frames = g.nativeCmd32Presentation.frames
		}
	case 33:
		err = g.startNativeCommand33Presentation(actor, target, nil)
		if err == nil {
			frames = g.nativeCmd33Presentation.frames
		}
	case 34:
		err = g.startNativeCommand34Presentation(actor, target, nil)
		if err == nil {
			frames = g.nativeCmd34Presentation.frames
		}
	case 35:
		err = g.startNativeCommand35Presentation(actor, target, nil)
		if err == nil {
			frames = g.nativeCmd35Presentation.frames
		}
	}
	if err != nil {
		t.Fatal("command preparation", command, err)
	}
	return g, frames
}

func commandPanelSplitTestFrames(h *modernHDSplitStrike) []nativeCompoundPresentedFrame {
	frames := h.sourceFrames()
	for target := range h.targets {
		for _, effect := range []int{0, h.actorImpact, h.targetStart, h.impact} {
			for _, damaged := range []bool{false, true} {
				f := h.frame(-1, -1, target, effect, 0, damaged, 0)
				if f != nil {
					frames = append(frames, *f)
				}
			}
		}
		frames = append(frames, nativeCompoundPresentedFrame{hdSources: h.targetEndSources(target, 0), hdPhase: "target-end"})
	}
	return frames
}

func TestModernCommandPanelAllOwners(t *testing.T) {
	value := os.Getenv("FD2_HD_COMMAND_PANEL_OWNER")
	if value == "" {
		t.Skip("one explicit owner per bounded process")
	}
	command, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_MUTE", "1")
	control, native := commandPanelTestFrames(t, command, false)
	g, frames := commandPanelTestFrames(t, command, true)
	if len(native) != len(frames) || len(g.st.Units) != len(control.st.Units) {
		t.Fatal("preparation shape changed")
	}
	for i := range g.st.Units {
		if !reflect.DeepEqual(g.st.Units[i], control.st.Units[i]) {
			t.Fatal("panel altered unit", command, i)
		}
	}
	if g.nativeRNGState != control.nativeRNGState || g.rng.Int63() != control.rng.Int63() {
		t.Fatal("panel altered RNG")
	}
	bound := 0
	phases := map[string]int{}
	snapshots := map[string]bool{}
	for i, f := range frames {
		if f.hdSources == nil {
			continue
		}
		if native[i].hdSources == nil || !bytes.Equal(surfaceIndexedReference(f.hdSources), surfaceIndexedReference(native[i].hdSources)) {
			t.Fatal("native source reconstruction changed", command, i, f.hdPhase)
		}
		count := 0
		for _, s := range append(append([]modernHDBattleSource(nil), f.hdSources.BaseScenery...), f.hdSources.Layers...) {
			if p := s.PanelText; p != nil {
				if !reflect.DeepEqual(s.Frame, p.native) {
					t.Fatal("panel source descriptor changed")
				}
				count++
				snapshots[fmt.Sprintf("%s@%d,%d LV%d HP%d/%d MP%d/%d", p.name, p.native.X, p.native.Y, p.values.Level, p.values.HP, p.values.MaxHP, p.values.MP, p.values.MaxMP)] = true
			}
		}
		if count == 0 {
			t.Fatal("unbound battle panel", command, i, f.hdPhase)
		}
		bound++
		phases[f.hdPhase]++
	}
	if bound == 0 || len(snapshots) == 0 {
		t.Fatal("vacuous owner coverage")
	}
	values := make([]string, 0, len(snapshots))
	for s := range snapshots {
		values = append(values, s)
	}
	sort.Strings(values)
	if out := os.Getenv("FD2_HD_COMMAND_PANEL_REPORT"); out != "" {
		seed := 7
		if command == 9 {
			seed = 3
		}
		report := map[string]any{"status": "PASS-TYPED-COMMAND-PANEL", "command_id": command, "frames": len(frames), "bound_battle_frames": bound, "phases": phases, "snapshots": values, "native_sources_equal": true, "unit_state_equal": true, "rng_equal": true, "native_seed_before": seed, "native_seed_control_before": seed, "native_rng_after": g.nativeRNGState, "seed_setup": "assigned before either preparation; fixed fixtures only"}
		if os.Getenv("FD2_HD_ORDINARY_SCENERY") == "1" {
			report["ordinary_scenery"] = ordinaryScenerySourceReport(t, g, frames)
		}
		data, _ := json.MarshalIndent(report, "", "  ")
		f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(append(data, '\n')); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("command %d: %d/%d battle frames, %d frozen panels", command, bound, len(frames), len(values))
}

func TestModernCommandPanelCollectorAtomic(t *testing.T) {
	g, actor, target := physicalSceneTestGame(t)
	attachCommandPanelHDTest(t, g)
	a, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	ar, _ := battle.NativeBattlePanelRecordForUnit(actor)
	tr, _ := battle.NativeBattlePanelRecordForUnit(target)
	var panels []*modernBattlePanelText
	bg := fdother.Frame{Width: 320, Height: 100}
	if _, err := g.localizedNativeCommand0Base(bg, a, ar, tr, 0, 1, 7, nil, &panels); err == nil || len(panels) != 0 {
		t.Fatal("bad background leaked collector")
	}
	if _, err := g.localizedNativeCommand24CapturedBase(bg, a, ar, 0, 7, nil, &panels); err == nil || len(panels) != 0 {
		t.Fatal("bad split background leaked collector")
	}
	if err := g.renderBattlePanelText(a, ar, make([]byte, 64000), 0, 7, &panels); err != nil {
		t.Fatal(err)
	}
	first := panels[0]
	if err := g.renderBattlePanelText(a, ar, make([]byte, 64000), 0, 7, &panels); err != nil || len(panels) != 1 || panels[0] != first {
		t.Fatal("duplicate source created new panel")
	}
}

func TestModernCommandPanelPalette(t *testing.T) {
	g, actor, _ := physicalSceneTestGame(t)
	attachCommandPanelHDTest(t, g)
	a, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	record, _ := battle.NativeBattlePanelRecordForUnit(actor)
	var panels []*modernBattlePanelText
	if err := g.renderBattlePanelText(a, record, make([]byte, 64000), 0, 7, &panels); err != nil {
		t.Fatal(err)
	}
	p := panels[0]
	base, baseKey, err := p.raster(g.nativeUIPalette)
	if err != nil {
		t.Fatal(err)
	}
	palette := append(color.Palette(nil), g.nativeUIPalette...)
	ink := color.RGBA{10, 200, 90, 255}
	palette[0xcd] = ink
	picture, key, err := p.raster(palette)
	if err != nil || key == baseKey {
		t.Fatal("palette did not change panel", err)
	}
	changed := 0
	oldWhite := color.RGBA{240, 244, 255, 255}
	for y := 4 * 6; y < 20*6; y++ {
		for x := 5 * 6; x < 114*6; x++ {
			if picture.At(x, y) == ink {
				changed++
			}
			if base.At(x, y) != picture.At(x, y) && picture.At(x, y) == oldWhite {
				t.Fatal("fixed HD white survived native palette")
			}
		}
	}
	if changed < 100 {
		t.Fatal("missing real palette-coloured glyph ink", changed)
	}
}
