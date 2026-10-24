package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

var ordinarySceneryGPUStats struct {
	NativeBlanks           map[string]int
	Draws, Pixels, Changed int
	Phases, Sources, LUTs  map[string]int
	OpaquePoints           map[string][]image.Point
}

func initOrdinarySceneryFixture(t *testing.T, g *Game, actor, target *battle.Unit) {
	t.Helper()
	if os.Getenv("FD2_HD_ORDINARY_SCENERY") != "1" {
		return
	}
	side, err := strconv.Atoi(os.Getenv("FD2_HD_ORDINARY_SCENERY_SIDE"))
	if err != nil || side != 0 && side != 2 {
		t.Fatal("explicit ordinary scenery side0/2 required")
	}
	surface, err := strconv.Atoi(os.Getenv("FD2_HD_ORDINARY_SCENERY_SURFACE"))
	if err != nil || surface < 0 || surface >= 56 || g.m == nil || len(g.m.Tiles) != g.m.W*g.m.H {
		t.Fatal("explicit ordinary scenery control surface required")
	}
	tile := -1
	for index := 0; index < len(g.m.NativeTerrainControl)/4 && index < 1024; index++ {
		if g.m.NativeTerrainControl[index*4+2] == byte(surface) {
			tile = index
			break
		}
	}
	if tile < 0 {
		t.Fatal("no fixed native tile selects requested ordinary scenery", surface)
	}
	g.m.Tiles = append([]int(nil), g.m.Tiles...)
	for _, unit := range g.st.Units {
		if !unit.OnField || unit.X < 0 || unit.Y < 0 || unit.X >= g.m.W || unit.Y >= g.m.H {
			continue
		}
		g.m.Tiles[unit.Y*g.m.W+unit.X] = tile
		unit.NativeRecordByte6, unit.HasNativeRecordByte6 = byte(2-side), true
	}
	actor.NativeRecordByte6, actor.HasNativeRecordByte6 = byte(side), true
	target.NativeRecordByte6, target.HasNativeRecordByte6 = byte(2-side), true
	ordinarySceneryGPUStats.Draws, ordinarySceneryGPUStats.Pixels, ordinarySceneryGPUStats.Changed = 0, 0, 0
	ordinarySceneryGPUStats.Phases, ordinarySceneryGPUStats.Sources, ordinarySceneryGPUStats.LUTs = map[string]int{}, map[string]int{}, map[string]int{}
	ordinarySceneryGPUStats.OpaquePoints = make(map[string][]image.Point)
	ordinarySceneryGPUStats.NativeBlanks = make(map[string]int)
}

func ordinarySceneryOpaquePoints(raw image.Image, key string) []image.Point {
	if points, exists := ordinarySceneryGPUStats.OpaquePoints[key]; exists {
		return points
	}
	bounds := raw.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := raw.At(x, y).RGBA()
			if a == 65535 {
				count++
			}
		}
	}
	stride := (count + 1023) / 1024
	if stride < 1 {
		stride = 1
	}
	points := make([]image.Point, 0, 1024)
	ordinal := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := raw.At(x, y).RGBA()
			if a == 65535 {
				if ordinal%stride == 0 {
					points = append(points, image.Pt(x-bounds.Min.X, y-bounds.Min.Y))
				}
				ordinal++
			}
		}
	}
	ordinarySceneryGPUStats.OpaquePoints[key] = points
	return points
}

func auditModernOrdinarySceneryGPU(g *Game, frame *nativeCompoundPresentedFrame, plan modernHDBattlePlan) error {
	if os.Getenv("FD2_HD_ORDINARY_SCENERY") != "1" {
		return nil
	}
	ordered := append(append([]modernHDBattleSource(nil), frame.hdSources.BaseScenery...), frame.hdSources.Layers...)
	if len(ordered) != len(plan.Layers) {
		return fmt.Errorf("ordinary scenery layer/source count differs")
	}
	bg, tai := 0, 0
	for i, source := range ordered {
		if source.Archive != "BG.DAT" && source.Archive != "TAI.DAT" {
			continue
		}
		layer := plan.Layers[i]
		raw, key, err := g.modernStoryPortraits.battleSurfaceHD(source)
		if err != nil {
			return fmt.Errorf("ordinary scenery actual registered source unavailable: %s:%d", source.Archive, source.Resource)
		}
		if raw == nil {
			for _, mask := range source.Frame.Mask {
				if mask != 0 {
					return fmt.Errorf("ordinary visible scenery not HD: %s:%d", source.Archive, source.Resource)
				}
			}
			if layer.HD != nil {
				return fmt.Errorf("native blank source acquired fabricated HD")
			}
			ordinarySceneryGPUStats.NativeBlanks[fmt.Sprintf("%s:%d", source.Archive, source.Resource)]++
			if source.Archive == "BG.DAT" {
				bg++
			} else {
				tai++
			}
			continue
		}
		rawKey := key
		if source.SurfaceLUT != nil {
			_, key, err = source.SurfaceLUT.prepared(g.modernStoryPortraits, raw, key, source.Frame)
			if err != nil {
				return err
			}
		}
		if layer.HD == nil || layer.HDKey != key {
			return fmt.Errorf("ordinary scenery actual HD variant differs")
		}
		if source.Archive == "BG.DAT" {
			bg++
		} else {
			tai++
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
		checked, changed := 0, 0
		for _, sample := range ordinarySceneryOpaquePoints(raw, rawKey) {
			point := sample.Add(layer.Rect.Min)
			x, y := point.X, point.Y
			if !point.In(clip) {
				continue
			}
			if commandPanelOccluded(plan.Layers[i+1:], point) {
				continue
			}
			sx, sy := x-layer.Rect.Min.X, y-layer.Rect.Min.Y
			r, green, b, a := layer.HD.At(sx, sy).RGBA()
			if a != 65535 {
				continue
			}
			at := ((y-clip.Min.Y)*clip.Dx() + x - clip.Min.X) * 4
			for c, value := range []uint32{r, green, b} {
				v := float32(value>>8)/255 - plan.HDSubtract
				if v < 0 {
					v = 0
				}
				want := int(math.Round(float64((v*(1-frame.hdMix) + frame.hdTarget[c]*frame.hdMix) * 255)))
				diff := int(pixels[at+c]) - want
				if diff < -1 || diff > 1 || pixels[at+3] != 255 {
					return fmt.Errorf("ordinary scenery %s %s:%d at%v channel%d got%d want%d", frame.hdPhase, source.Archive, source.Resource, point, c, pixels[at+c], want)
				}
			}
			r0, g0, b0, _ := raw.At(sx, sy).RGBA()
			if source.SurfaceLUT != nil && (r != r0 || green != g0 || b != b0) {
				changed++
			}
			checked++
		}
		ordinarySceneryGPUStats.Pixels += checked
		ordinarySceneryGPUStats.Changed += changed
		ordinarySceneryGPUStats.Phases[frame.hdPhase] += checked
		ordinarySceneryGPUStats.Sources[fmt.Sprintf("%s:%d", source.Archive, source.Resource)] += checked
		if source.SurfaceLUT != nil {
			ordinarySceneryGPUStats.LUTs[strconv.Itoa(source.SurfaceLUT.index)] += checked
		}
	}
	if bg != 1 || tai != 1 {
		return fmt.Errorf("ordinary scenery expected one BG and one TAI, got%d/%d", bg, tai)
	}
	ordinarySceneryGPUStats.Draws++
	return nil
}

func appendModernOrdinarySceneryReport(t *testing.T, report map[string]any) {
	t.Helper()
	if os.Getenv("FD2_HD_ORDINARY_SCENERY") != "1" {
		return
	}
	if ordinarySceneryGPUStats.Draws == 0 || ordinarySceneryGPUStats.Pixels < 100 || ordinarySceneryGPUStats.Changed < 100 {
		t.Fatal("missing actual ordinary scenery or changed LUT pixels", ordinarySceneryGPUStats)
	}
	report["ordinary_scenery"] = map[string]any{"side": os.Getenv("FD2_HD_ORDINARY_SCENERY_SIDE"), "control_surface": os.Getenv("FD2_HD_ORDINARY_SCENERY_SURFACE"),
		"audited_draws": ordinarySceneryGPUStats.Draws, "matched_pixels": ordinarySceneryGPUStats.Pixels, "changed_lut_pixels": ordinarySceneryGPUStats.Changed,
		"sampled_phases": ordinarySceneryGPUStats.Phases, "sources": ordinarySceneryGPUStats.Sources, "luts": ordinarySceneryGPUStats.LUTs,
		"native_blank_sources": ordinarySceneryGPUStats.NativeBlanks,
		"scope":                "actual World readback, unoccluded opaque BG/TAI pixels; fixed original controls with explicit tile and side fixture; nearest-four RGB approximation"}
}

func ordinaryScenerySourceReport(t *testing.T, g *Game, frames []nativeCompoundPresentedFrame) map[string]any {
	t.Helper()
	counts, luts := map[string]int{}, map[string]int{}
	for _, frame := range frames {
		if frame.hdSources == nil {
			continue
		}
		bg, tai := 0, 0
		for _, source := range append(append([]modernHDBattleSource(nil), frame.hdSources.BaseScenery...), frame.hdSources.Layers...) {
			if source.Archive != "BG.DAT" && source.Archive != "TAI.DAT" {
				continue
			}
			if source.Archive == "BG.DAT" {
				bg++
			} else {
				tai++
			}
			counts[fmt.Sprintf("%s:%d", source.Archive, source.Resource)]++
			if source.SurfaceLUT != nil {
				luts[strconv.Itoa(source.SurfaceLUT.index)]++
			}
		}
		if bg != 1 || tai != 1 {
			t.Fatal("missing ordinary native scenery", frame.hdPhase, bg, tai)
		}
	}
	return map[string]any{"side": os.Getenv("FD2_HD_ORDINARY_SCENERY_SIDE"), "control_surface": os.Getenv("FD2_HD_ORDINARY_SCENERY_SURFACE"), "sources": counts, "luts": luts}
}

func surfaceLUTTestBasis(t *testing.T) (*modernBattleSurfaceLUTBasis, figani.Frame, *image.NRGBA, []byte) {
	t.Helper()
	frame := figani.Frame{Width: 2, Height: 2, Pixels: []byte{10, 20, 30, 40}, Mask: []byte{255, 255, 255, 0}}
	palette := make(color.Palette, 256)
	lookup := make([]byte, 256)
	for i := range palette {
		palette[i] = color.RGBA{byte(i), byte(i), byte(i), 255}
		lookup[i] = byte(i)
	}
	basis, err := newModernBattleSurfaceLUTBasis(frame, palette)
	if err != nil {
		t.Fatal(err)
	}
	picture := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			picture.SetNRGBA(x, y, color.NRGBA{40, 60, 80, []byte{0, 1, 128, 255}[x%4]})
		}
	}
	return basis, frame, picture, lookup
}

func TestModernBattleSurfaceLUTColourAndAlpha(t *testing.T) {
	basis, frame, raw, lookup := surfaceLUTTestBasis(t)
	for _, index := range basis.colours {
		lookup[index] = index + 10
	}
	p, err := basis.profile(11, lookup)
	if err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	if _, _, err := p.prepared(set, raw, "unit", frame); err == nil {
		t.Fatal("Draw accepted unprepared colour image")
	}
	if err := p.prepare(set, raw, "unit", frame); err != nil {
		t.Fatal(err)
	}
	picture, key, err := p.prepared(set, raw, "unit", frame)
	if err != nil || key == "unit" {
		t.Fatal("missing colour variant", err)
	}
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			want := raw.NRGBAAt(x, y)
			if want.A != 0 {
				want.R, want.G, want.B = want.R+10, want.G+10, want.B+10
			}
			got := picture.(*image.NRGBA).NRGBAAt(x, y)
			if got != want {
				t.Fatalf("straight-alpha mapping at%d,%d got%v want%v", x, y, got, want)
			}
		}
	}
	mapped, err := p.nativeFrame(frame)
	if err != nil || !bytes.Equal(mapped.Pixels, []byte{20, 30, 40, 40}) || !bytes.Equal(frame.Pixels, []byte{10, 20, 30, 40}) {
		t.Fatal("native lookup did not preserve mask and raw source", err)
	}
	if _, _, err := p.prepared(&modernStoryPortraitSet{}, raw, "different-content", frame); err == nil {
		t.Fatal("another source content reused the prepared image")
	}
	for _, candidate := range []string{"candidate-one", "candidate-two"} {
		if err := p.prepare(&modernStoryPortraitSet{}, raw, candidate, frame); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := p.prepared(set, raw, "unit", frame); err != nil || len(p.cache) != 2 {
		t.Fatal("F2 candidates evicted active content", err)
	}
	lookup[10] = 200
	if p.lookup[10] != 20 {
		t.Fatal("frozen lookup aliases caller")
	}
	p.lookup[10] = 200
	if _, err := p.nativeFrame(frame); err == nil {
		t.Fatal("mutated frozen lookup accepted")
	}
}

func TestModernBattleSurfaceLUTEndpoints(t *testing.T) {
	for _, target := range []byte{0, 255} {
		basis, frame, raw, lookup := surfaceLUTTestBasis(t)
		for _, index := range basis.colours {
			lookup[index] = target
		}
		p, err := basis.profile(12, lookup)
		if err != nil || p.prepare(nil, raw, "endpoint", frame) != nil {
			t.Fatal("endpoint preparation", err)
		}
		for y := 0; y < 12; y++ {
			for x := 0; x < 12; x++ {
				v := p.cache["endpoint"].picture.(*image.NRGBA).NRGBAAt(x, y)
				if v.A != raw.NRGBAAt(x, y).A || v.A != 0 && (v.R != target || v.G != target || v.B != target) {
					t.Fatal("endpoint or alpha differs", target, v)
				}
			}
		}
	}
}

func TestModernBattleSurfaceLUTIdentityAndSourceGuard(t *testing.T) {
	basis, frame, raw, lookup := surfaceLUTTestBasis(t)
	p, err := basis.profile(13, lookup)
	if err != nil || p.prepare(nil, raw, "identity", frame) != nil || p.cache["identity"].picture != raw || p.cache["identity"].key != "identity" {
		t.Fatal("identity changed original HD", err)
	}
	bad := frame
	bad.Pixels = append([]byte(nil), frame.Pixels...)
	bad.Pixels[0]++
	if _, err := p.nativeFrame(bad); err == nil {
		t.Fatal("wrong raw source accepted")
	}
	if _, err := basis.profile(16, lookup); err == nil {
		t.Fatal("unproven lookup accepted")
	}
	if _, err := basis.profile(14, lookup[:255]); err == nil {
		t.Fatal("partial lookup accepted")
	}
	basis.palette[10] = color.RGBA{255, 0, 0, 255}
	if _, err := p.nativeFrame(frame); err == nil {
		t.Fatal("changed frozen palette accepted")
	}
}

type ordinarySceneryTestBank []nativeCompoundPresentedFrame

func (b ordinarySceneryTestBank) sourceFrames() []nativeCompoundPresentedFrame { return b }

func TestModernOrdinarySceneryAtomic(t *testing.T) {
	g, actor, target := ordinaryCommand0HDTestGame(t)
	bg, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", 4)
	if err != nil {
		t.Fatal(err)
	}
	tai, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", 4)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	ar, _ := battle.NativeBattlePanelRecordForUnit(actor)
	tr, _ := battle.NativeBattlePanelRecordForUnit(target)
	base, err := g.localizedNativeCommand0Base(bg, assets, ar, tr, 0, 4, g.handlerChapter, &tai)
	if err != nil {
		t.Fatal(err)
	}
	shaded, err := modernHDCommandShade(base, bg, tai, g.nativeMapAssets.LUTs[11])
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]byte(nil), shaded...)
	source, err := modernHDBattleSurfaceFrame(bg)
	if err != nil {
		t.Fatal(err)
	}
	for i, mask := range source.Mask {
		if mask == 0 {
			continue
		}
		used := map[byte]bool{source.Pixels[i]: true}
		for index := 11; index <= 15; index++ {
			used[g.nativeMapAssets.LUTs[index][source.Pixels[i]]] = true
		}
		for v := 0; v < 256; v++ {
			if !used[byte(v)] {
				bad[(50+i/source.Width)*320+i%source.Width] = byte(v)
				break
			}
		}
		break
	}
	first := &modernHDBattleSources{Base: shaded}
	bank := ordinarySceneryTestBank{{hdSources: first, hdPhase: "actor"}, {hdSources: &modernHDBattleSources{Base: bad}, hdPhase: "tail"}}
	before := append([]byte(nil), first.Base...)
	if err := bindModernHDOrdinaryScenery(bank, bg, 4, tai, 4, g.nativeUIPalette, g.nativeMapAssets.LUTs); err == nil ||
		!bytes.Equal(first.Base, before) || len(first.BaseScenery) != 0 {
		t.Fatal("late mismatch published a partial bank", err)
	}
	if err := bindModernHDOrdinaryScenery(bank[:1], bg, 4, tai, 4, g.nativeUIPalette, g.nativeMapAssets.LUTs); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(surfaceIndexedReference(first), shaded) || len(first.BaseScenery) != 2 || reflect.DeepEqual(first.Base, shaded) {
		t.Fatal("successful binding did not reconstruct native shaded base")
	}
}

func TestModernOrdinaryAI9OppositeSideRejected(t *testing.T) {
	g, actor, target := ordinaryRemainingHDTestGame(t, 9)
	actor.NativeRecordByte6 = 2
	g.rng = rand.New(rand.NewSource(902))
	beforeActor, beforeTarget, seed := *actor, *target, g.nativeRNGState
	err := g.startNativeCommand9AIPresentation(actor, target, nil)
	if err == nil || err.Error() != "native command9 AI raw-side-zero actor provenance unavailable" || g.nativeCmd9AIPresentation != nil ||
		!reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTarget, *target) || seed != g.nativeRNGState ||
		g.rng.Int63() != rand.New(rand.NewSource(902)).Int63() {
		t.Fatal("illegal AI9 side did not reject before owner/state/RNG", err)
	}
}

func TestModernBattleSurfaceLUTFallbackGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ORDINARY_LUT_FALLBACK_GPU") != "1" {
		t.Skip("dedicated native LUT GPU process")
	}
	g, _, _ := ordinaryCommand0HDTestGame(t)
	var cases []modernHDBattleGPUCase
	for _, archive := range []string{"BG.DAT", "TAI.DAT"} {
		native, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), archive, 4)
		if err != nil {
			t.Fatal(err)
		}
		frame, err := modernHDBattleSurfaceFrame(native)
		if err != nil {
			t.Fatal(err)
		}
		frame.X, frame.Y = 0, 50
		if archive == "TAI.DAT" {
			frame.X, frame.Y = 164, 157
		}
		basis, err := newModernBattleSurfaceLUTBasis(frame, g.nativeUIPalette)
		if err != nil {
			t.Fatal(err)
		}
		for index := 11; index <= 15; index++ {
			profile, err := basis.profile(index, g.nativeMapAssets.LUTs[index])
			if err != nil {
				t.Fatal(err)
			}
			base := bytes.Repeat([]byte{7}, 64000)
			sources := &modernHDBattleSources{Base: base, BaseScenery: []modernHDBattleSource{{Archive: archive, Resource: 4, Frame: frame, Fill: -1, SurfaceLUT: profile}}}
			plan, err := (&Game{}).modernHDBattlePlan(sources, g.nativeUIPalette, 0.4)
			if err != nil || len(plan.Layers) != 1 || plan.Layers[0].HD != nil {
				t.Fatal("missing exact native fallback", err)
			}
			want := append([]byte(nil), base...)
			for i, mask := range frame.Mask {
				if mask != 0 {
					want[(frame.Y+i/frame.Width)*320+frame.X+i%frame.Width] = g.nativeMapAssets.LUTs[index][frame.Pixels[i]]
				}
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("surface-native/%s/lut%d", archive, index), plan: plan, indexed: want})
		}
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != 10 {
		t.Fatalf("native LUT fallback GPU %d/10: %v", probe.index, probe.err)
	}
	t.Log("10 complete native BG/TAI LUT11-15 GPU worlds exactly match independent indexed lookup; HD subtract does not affect native fallback")
}
