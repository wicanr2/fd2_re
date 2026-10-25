package main

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func healStageGame(t *testing.T, id int, hd bool) (*Game, *battle.Unit, *battle.Unit, []*battle.Unit) {
	t.Helper()
	g, a, b := player9StageGame(t, hd, false)
	b.Camp, b.NativeRecordByte6, b.HasNativeRecordByte6 = battle.Own, 0, true
	b.HP, b.MaxHP, b.NativeRecordWord42, b.HasNativeRecordWord42 = 20, 100, 100, true
	g.st.NativeCommandBook[id] = battle.NativeCommandRecord{ID: id, Damage: 70, SelectionMode: 1, EffectMode: 0, MPCost: 3, TargetCode: 1}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	targets, err := g.st.NativeCommandHealTargets(a, b, id)
	if err != nil || len(targets) != 1 {
		t.Fatal("heal stage targets", err, len(targets))
	}
	return g, a, b, targets
}

// Controlled actors near a visible stage edge make the radial masks cross
// the actual HD underlay. All geometry and native banks remain map23.
func healStageScenario(t *testing.T, id int, hd, multi bool) (*Game, *battle.Unit, *battle.Unit, []*battle.Unit) {
	t.Helper()
	g, a, b, _ := healStageGame(t, id, hd)
	a.SetMapPlacement(26, 16, 3)
	b.SetMapPlacement(27, 16, 0)
	units := []*battle.Unit{a, b}
	if multi {
		u := g.st.Units[5]
		u.Camp, u.OnField, u.NativeRecordByte6, u.HasNativeRecordByte6 = battle.Own, true, 0, true
		u.HP, u.MaxHP, u.NativeRecordWord42, u.HasNativeRecordWord42 = 20, 100, 100, true
		u.SetMapPlacement(27, 15, 0)
		units = append(units, u)
		g.st.NativeCommandBook[id].EffectMode = 1
	}
	for _, u := range units {
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 16, CameraY: 14, CursorX: 27, CursorY: 16, VisibleCursorX: 11, VisibleCursorY: 2}); err != nil {
		t.Fatal(err)
	}
	g.curX, g.curY = 27, 16
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	targets, err := g.st.NativeCommandHealTargets(a, b, id)
	want := 1
	if multi {
		want = 3
	}
	if err != nil || len(targets) != want {
		t.Fatal("stage edge targets", err, len(targets), want)
	}
	return g, a, b, targets
}

func TestModernHDHealStageRemapMatchesNativeMarker(t *testing.T) {
	g, _, _, _ := healStageScenario(t, 16, true, false)
	set := *g.modernStoryPortraits
	stage := *set.mapStages[42]
	// A single exact native colour gives an independent RGB oracle: the
	// native indexed compositor must equal every visible opaque HD pixel.
	stage.native.Pixels = bytes.Repeat([]byte{243}, 312*192)
	picture := image.NewRGBA(image.Rect(0, 0, 1872, 1152))
	base := color.RGBAModel.Convert(g.nativeMapAssets.Palette[243]).(color.RGBA)
	for y := 0; y < 1152; y++ {
		for x := 0; x < 1872; x++ {
			picture.SetRGBA(x, y, base)
		}
	}
	stage.art = modernTerrainHDAsset{Image: picture, Colours: []byte{243}}
	set.mapStages = map[int]*modernMapStageHD{42: &stage}
	g.modernStoryPortraits = &set
	// Isolate every remap count from the real map's occluding castle tiles.
	// The other tests and Game.Draw matrix keep the actual map23 banks.
	assets := *g.nativeMapAssets
	bank := *assets.Terrain
	bank.Sprites = append(bank.Sprites[:0:0], bank.Sprites...)
	for i := range bank.Sprites {
		bank.Sprites[i].Mask = make([]byte, 576)
		bank.Sprites[i].RemapMask = make([]byte, 576)
		for at := range bank.Sprites[i].RemapMask {
			bank.Sprites[i].RemapMask[at] = byte(at % 2)
		}
	}
	assets.Terrain = &bank
	g.nativeMapAssets = &assets
	g.nativeCh23State.staging = append([]byte(nil), stage.native.Pixels...)
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	in, err := freezeCompoundMapInput(*g.nativeMapFrameSource)
	if err != nil {
		t.Fatal(err)
	}
	in.Frame.Units, in.Frame.ForegroundUnits = nil, nil
	in.Frame.RangeMode, in.Frame.SelectionOverlay = 0, nil
	for i := range in.Frame.Cells {
		in.Frame.Cells[i].BlitMode = 0
	}
	w, v := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, 64000)
	if err := indexedmap.SeedNativeCh23Staging(w, g.nativeCh23State.staging); err != nil {
		t.Fatal(err)
	}
	if err := indexedmap.ComposeNativeFrame(w, v, in); err != nil {
		t.Fatal(err)
	}
	normal, err := g.prepareModernHDCommandSteadyStage(in, w, v, g.nativeMapAssets.Palette, &set, g.nativeCh23State)
	if err != nil {
		t.Fatal(err)
	}
	pass := fdother.IndexedTransitionPass{
		FirstRadial:  fdother.RadialLUTRemap{CenterX: 352, CenterY: 102, Radius: 30, Scale: 10, StartY: 0, EndY: 292, ClipWidth: 456},
		SecondRadial: fdother.RadialLUTRemap{CenterX: 352, CenterY: 102, Radius: 20, Scale: 10, StartY: 0, EndY: 292, ClipWidth: 456},
		FinalRect:    fdother.CenteredRectLUTRemap{CenterX: 352, HorizontalRadius: 5, StartY: 99, EndY: 105, ClipWidth: 456},
	}
	seen := map[byte]bool{}
	for _, index := range []int{1, 3, 9} {
		lut, err := fdother.NativeIndexedTransitionLUT(g.nativeMapAssets.LUTs, index)
		if err != nil {
			t.Fatal(err)
		}
		work, vga := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, 64000)
		if err := indexedmap.SeedNativeCh23Staging(work, g.nativeCh23State.staging); err != nil {
			t.Fatal(err)
		}
		if err := indexedmap.ComposeNativeTransitionFrame(work, vga, nativeTransitionInput(in), pass, lut); err != nil {
			t.Fatal(err)
		}
		f, err := g.prepareModernHDRadialMap(modernHDRadialMapSpec{Input: in, Palette: g.nativeMapAssets.Palette, Pass: pass, LUT: lut, Background: vga, Stage: g.nativeCh23State}, &set, normal, nil)
		if err != nil {
			t.Fatal(err)
		}
		counts, _, err := healHDCounts(pass)
		if err != nil {
			t.Fatal(err)
		}
		checked := 0
		for _, layer := range f.Terrain {
			if !layer.Stage {
				continue
			}
			for y := 0; y < 192; y += 2 {
				for x := 0; x < 312; x++ {
					got := layer.Image.RGBAAt(x*6, y*6)
					if got.A == 0 {
						continue
					}
					want := color.RGBAModel.Convert(g.nativeMapAssets.Palette[vga[y*320+x]]).(color.RGBA)
					if got != want {
						t.Fatalf("stage marker LUT%d %d,%d got%v want%v", index, x, y, got, want)
					}
					seen[healHDCountAt(counts, x, y)] = true
					checked++
				}
			}
		}
		if checked == 0 {
			t.Fatal("no visible stage pixels checked")
		}
	}
	for count := byte(0); count < 4; count++ {
		if !seen[count] {
			t.Fatal("missing native remap count", count, seen)
		}
	}
}

func TestModernHDHealStageSourceIsolationAndReject(t *testing.T) {
	g, a, b, targets := healStageScenario(t, 13, true, false)
	calls, done := 0, 0
	startHealHDTest(t, g, a, b, targets, 13, &calls, &done)
	j := g.nativeHealPresentation
	want := healHDCPUWorld(j.hdFrames.Front[15], g.modernStoryPortraits)
	g.nativeCh23State.staging[0] ^= 1
	fresh, err := g.prepareModernHDHeal(j, g.modernStoryPortraits)
	if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(fresh.Front[15], g.modernStoryPortraits).Pix) {
		t.Fatal("live stage leaked into heal snapshot", err)
	}
	active, theme := j.hdFrames, g.modernStoryPortraits
	j.mapStage.staging[0] ^= 1
	if err := g.applyTheme(theme); err == nil || j.hdFrames != active || g.modernStoryPortraits != theme {
		t.Fatal("modified stage replaced active theme")
	}
	j.mapStage.staging[0] ^= 1
	j.frontFrames[0][0] ^= 1
	if err := g.applyTheme(theme); err == nil || j.hdFrames != active {
		t.Fatal("modified native frame replaced active theme")
	}
	j.frontFrames[0][0] ^= 1
	if calls != 0 || done != 0 {
		t.Fatal("source checks committed transaction")
	}
}

func TestNativeHealStageLateFailureRestoresTransaction(t *testing.T) {
	g, a, b, targets := healStageScenario(t, 16, true, true)
	before := stage42NativeState(g)
	calls, done := 0, 0
	startHealHDTest(t, g, a, b, targets, 16, &calls, &done)
	j := g.nativeHealPresentation
	transaction := j.transaction
	j.transaction = func() ([]battle.NativeCommandHealResult, error) {
		results, err := transaction()
		j.digits = nil
		return results, err
	}
	for steps := 0; g.nativeHealPresentation != nil && steps < 500; steps++ {
		j.drawn = true
		g.stepNativeCommandHealPresentation()
	}
	if g.nativeHealPresentation != nil || g.loadErr == "" || calls != 1 || done != 0 || !reflect.DeepEqual(before, stage42NativeState(g)) {
		t.Fatal("late redraw failure left a partial heal", g.loadErr, calls, done)
	}
}

func TestNativeHealStageFrontUsesOriginalSnapshot(t *testing.T) {
	for _, id := range []int{13, 14, 15, 16} {
		g, a, b, targets := healStageGame(t, id, false)
		calls, done := 0, 0
		startHealHDTest(t, g, a, b, targets, id, &calls, &done)
		j := g.nativeHealPresentation
		for i, pass := range j.hdPasses {
			w, v := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, 64000)
			if err := indexedmap.SeedNativeCh23Staging(w, g.nativeCh23State.staging); err != nil {
				t.Fatal(err)
			}
			if err := indexedmap.ComposeNativeTransitionFrame(w, v, nativeTransitionInput(j.hdInput), pass, j.hdLUTs[i]); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(v, j.frontFrames[i]) {
				t.Fatalf("command%d frame%d lacks the original 0x11EEE stage snapshot", id, i)
			}
		}
		if calls != 0 || done != 0 {
			t.Fatal("preflight committed transaction")
		}
	}
}
