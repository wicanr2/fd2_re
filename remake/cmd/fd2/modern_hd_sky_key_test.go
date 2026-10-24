package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// Each dedicated process uses one immutable source pack. Mutating tests clone
// the bank description; they never edit shared decoded pixels or source files.
var skyKeyTestTemplates = map[string]*modernStoryPortraitSet{}

func skyKeyHDTestSet(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("dedicated sky key verifier supplies catalog and pack")
	}
	key := catalog + "\x00" + pack
	if set := skyKeyTestTemplates[key]; set != nil {
		return set
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	if set.skyKeyFD34 == nil || set.skyKeyANI == nil {
		t.Fatal("complete sky key pair required")
	}
	skyKeyTestTemplates[key] = set
	return set
}

// Real map20 geometry and native banks with two controlled actors. This fixture
// proves the renderer boundary, not a normal campaign or original-EXE path.
func skyKeyHDTestGame(t *testing.T, now *time.Time) *Game {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t, func() time.Time { return *now })
	if err := g.loadMap("assets/maps/map20"); err != nil {
		t.Fatal(err)
	}
	state, err := battle.Load(assetPath("assets/maps/map20/map20_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.st.W, g.st.H, g.st.Cost = state.W, state.H, state.Cost
	g.st.NativeCompositionEventBytes = state.NativeCompositionEventBytes
	g.st.NativeMapEventGrid, g.st.HasNativeMapEventGrid = state.NativeMapEventGrid, state.HasNativeMapEventGrid
	g.st.NativeTileBlitModes = state.NativeTileBlitModes
	g.st.NativeTerrainControl, g.st.NativeTerrainMoveCodes = state.NativeTerrainControl, state.NativeTerrainMoveCodes
	for _, u := range g.st.Units {
		u.OnField = u == actor || u == target
	}
	actor.SetMapPlacement(18, 12, 0)
	target.SetMapPlacement(22, 12, 0)
	for _, u := range []*battle.Unit{actor, target} {
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 12, CameraY: 6, CursorX: 16, CursorY: 10, VisibleCursorX: 4, VisibleCursorY: 4}); err != nil {
		t.Fatal(err)
	}
	g.st.MaterializeNativeMapHUDState(1, 1, 1)
	g.st.MaterializeNativeMapRangeMode(1)
	g.curX, g.curY = 16, 10
	g.nativeMapWork, g.nativeMapVGA = nil, nil
	g.nativeMapClock.Reset()
	g.nativeFDOTHERPaletteTick, g.nativeFDOTHERPalettePhase = 0, 0
	if err := g.composeNativeMapFrameAt(*now); err != nil {
		t.Fatal(err)
	}
	return g
}

func skyKeyNativeState(g *Game) any {
	var job any
	if j := g.nativeCh20SkyKey; j != nil {
		job = []any{j.spec, j.phase, j.frame, j.ticks, append([]byte(nil), j.base...), append([]byte(nil), j.vga...), append([]byte(nil), j.dac...), append(color.Palette(nil), j.palette...)}
	}
	return []any{rowShakeNativeState(g), g.nativeFDOTHERPaletteTick, job}
}

func TestModernHDSkyKeySourceBanks(t *testing.T) {
	set := skyKeyHDTestSet(t)
	g := &Game{}
	frames, err := fdother.LoadSeparatedCh20SkyKeyFrames(separatedAssetPath("animations/fdother_034_ch20_sky_key"))
	if err != nil {
		t.Fatal(err)
	}
	clip, err := loadNativeCh20SkyKeyANI(recoveredCh20SkyKeySpec(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = g.preflightModernSkyKey(set, frames, clip); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"duplicate", "geometry", "hash", "coverage", "plan", "partial"} {
		t.Run(kind, func(t *testing.T) {
			var asset modernThemeAsset
			raw, _ := json.Marshal(set.skyKeyFD34.asset)
			if err := json.Unmarshal(raw, &asset); err != nil {
				t.Fatal(err)
			}
			root := set.skyKeyFD34.root
			target := &modernStoryPortraitSet{}
			switch kind {
			case "duplicate":
				target.skyKeyFD34 = set.skyKeyFD34
			case "geometry":
				asset.NativeFrames[0].Width++
			case "hash":
				asset.FrameSHA256[0] = skyKeySHA(nil)
			case "coverage":
				asset.AlphaContract = "binary"
			case "plan":
				asset.SourceSHA256[len(asset.SourceSHA256)-1] = skyKeySHA(nil)
			case "partial":
				bad := *set
				bad.skyKeyANI = nil
				if _, _, err := g.modernSkyKeyPair(&bad); err == nil {
					t.Fatal("partial pair admitted")
				}
				return
			}
			if err := loadModernSkyKeyHDAsset(target, asset, root); err == nil {
				t.Fatal("invalid bank admitted", kind)
			}
		})
	}
	mutated := append([]byte(nil), clip.IndexedFrames[95]...)
	mutated[0] ^= 1
	if err = set.skyKeyANI.validateANIFrame(95, mutated, clip.Palettes[95]); err == nil {
		t.Fatal("late ANI source edit admitted")
	}
	frames[100].Width++
	if err = set.skyKeyFD34.validateFDFrame(100, frames[100]); err == nil {
		t.Fatal("late FDOTHER edit admitted")
	}
	var absent *modernSkyKeyHDBank
	if _, err = absent.picture(0); err == nil {
		t.Fatal("nil bank admitted")
	}
}

func TestModernHDSkyKeyFullSequenceNativeControl(t *testing.T) {
	now := time.Unix(1000, 0)
	set := skyKeyHDTestSet(t)
	g, control := skyKeyHDTestGame(t, &now), skyKeyHDTestGame(t, &now)
	g.modernStoryPortraits = set
	a, b := 0, 0
	spec := recoveredCh20SkyKeySpec(t)
	if err := g.startNativeCh20SkyKeySequence(spec, func() { a++ }); err != nil {
		t.Fatal(err)
	}
	if err := control.startNativeCh20SkyKeySequence(spec, func() { b++ }); err != nil {
		t.Fatal(err)
	}
	counts := map[nativeCh20SkyKeyPhase]int{}
	switches := 0
	for n := 0; g.nativeCh20SkyKey != nil && n < 250; n++ {
		j := g.nativeCh20SkyKey
		if j.hdFrame == nil || !reflect.DeepEqual(skyKeyNativeState(g), skyKeyNativeState(control)) {
			t.Fatal("native producer diverged", n, g.loadErr, control.loadErr)
		}
		before := skyKeyNativeState(g)
		g.stepNativeCh20SkyKey()
		if !reflect.DeepEqual(before, skyKeyNativeState(g)) {
			t.Fatal("advanced without Draw")
		}
		if counts[j.phase] == 0 {
			t.Logf("phase %d begins at frame %d", j.phase, j.frame)
			if err := g.applyTheme(nil); err != nil {
				t.Fatal(err)
			}
			if j.hdFrame != nil {
				t.Fatal("native switch retained HD")
			}
			if err := g.applyTheme(set); err != nil {
				t.Fatal(err)
			}
			if j.hdFrame == nil || !reflect.DeepEqual(before, skyKeyNativeState(g)) {
				t.Fatal("F2 changed native state")
			}
			switches++
		}
		counts[j.phase]++
		j.drawn = true
		control.nativeCh20SkyKey.drawn = true
		ticks := j.ticks
		if ticks < 1 {
			ticks = 1
		}
		for tick := 0; tick < ticks; tick++ {
			g.stepNativeCh20SkyKey()
			control.stepNativeCh20SkyKey()
		}
	}
	if a != 1 || b != 1 || g.nativeCh20SkyKey != nil || control.nativeCh20SkyKey != nil || switches != 6 || !reflect.DeepEqual(skyKeyNativeState(g), skyKeyNativeState(control)) {
		t.Fatal("sequence did not return exactly once", a, b, counts, g.loadErr)
	}
	want := map[nativeCh20SkyKeyPhase]int{nativeCh20SkyKeyPan: 4, nativeCh20SkyKeyFirstFrames: 68, nativeCh20SkyKeyANI: 96, nativeCh20SkyKeyFlash: 1, nativeCh20SkyKeyRestore: 1, nativeCh20SkyKeyTailFrames: 32}
	if !reflect.DeepEqual(counts, want) {
		t.Fatal("native cadence changed", counts)
	}
}

func TestModernHDSkyKeyThemeTransactionRejects(t *testing.T) {
	set := skyKeyHDTestSet(t)
	for _, kind := range []string{"world", "native", "future", "cached-pixels", "partial", "later-owner"} {
		t.Run(kind, func(t *testing.T) {
			now := time.Unix(1000, 0)
			g := skyKeyHDTestGame(t, &now)
			if err := g.startNativeCh20SkyKeySequence(recoveredCh20SkyKeySpec(t), nil); err != nil {
				t.Fatal(err)
			}
			j := g.nativeCh20SkyKey
			bad := *set
			switch kind {
			case "world":
				j.hdWorld.work[0]++
			case "native":
				j.fdFrames[100].Width++
			case "future":
				bank := *set.skyKeyANI
				bank.asset.Files = append([]string(nil), bank.asset.Files...)
				bank.asset.Files[95] = "missing-future.png"
				bad.skyKeyANI = &bank
			case "cached-pixels":
				bank := *set.skyKeyANI
				bank.pictures = append([]*image.RGBA(nil), bank.pictures...)
				original := bank.pictures[95]
				picture := image.NewRGBA(original.Bounds())
				copy(picture.Pix, original.Pix)
				picture.Pix[0] ^= 1
				bank.pictures[95] = picture
				bad.skyKeyANI = &bank
			case "partial":
				bad.skyKeyANI = nil
			case "later-owner":
				g.nativeUnitPresent = &nativeUnitPresentJob{}
			}
			before := skyKeyNativeState(g)
			if err := g.applyTheme(&bad); err == nil || g.modernStoryPortraits != nil || j.hdFrame != nil || j.hdTheme != nil || !reflect.DeepEqual(before, skyKeyNativeState(g)) {
				t.Fatal("failed F2 published partial theme", kind, err)
			}
		})
	}
}

func TestModernHDSkyKeyFailureAndLegacy(t *testing.T) {
	set := skyKeyHDTestSet(t)
	now := time.Unix(1000, 0)
	for _, late := range []bool{false, true} {
		g := skyKeyHDTestGame(t, &now)
		g.modernStoryPortraits = set
		before := skyKeyNativeState(g)
		source := g.nativeMapFrameSource
		tick := g.nativeFDOTHERPaletteTick
		bad := *set
		bank := *set.skyKeyANI
		bank.asset.Files = append([]string(nil), bank.asset.Files...)
		bank.asset.Files[95] = "missing-future.png"
		bad.skyKeyANI = &bank
		if !late {
			g.modernStoryPortraits = &bad
		}
		continued := 0
		err := g.startNativeCh20SkyKeySequence(recoveredCh20SkyKeySpec(t), func() { continued++ })
		if late {
			if err != nil {
				t.Fatal(err)
			}
			for g.nativeCh20SkyKey.phase == nativeCh20SkyKeyPan {
				g.nativeCh20SkyKey.drawn = true
				g.stepNativeCh20SkyKey()
			}
			j := g.nativeCh20SkyKey
			j.hdWorld.work[0]++
			j.drawn = true
			for n := 0; n < 10 && g.nativeCh20SkyKey != nil; n++ {
				g.stepNativeCh20SkyKey()
			}

			if g.loadErr == "" {
				t.Fatal("late failure was not rejected")
			}
		} else if err == nil {
			t.Fatal("cold future-frame failure admitted")
		}
		if g.nativeCh20SkyKey != nil || continued != 0 || !reflect.DeepEqual(before, skyKeyNativeState(g)) || g.nativeMapFrameSource != source || g.nativeFDOTHERPaletteTick != tick {
			t.Fatal("failure leaked map transaction", late)
		}
	}
	g := skyKeyHDTestGame(t, &now)
	legacy := *set
	legacy.skyKeyFD34, legacy.skyKeyANI = nil, nil
	g.modernStoryPortraits = &legacy
	if err := g.startNativeCh20SkyKeySequence(recoveredCh20SkyKeySpec(t), nil); err != nil || g.nativeCh20SkyKey.hdFrame != nil {
		t.Fatal("legacy theme lost native fallback", err)
	}
	// The generic palette helper must preserve authored alpha at both endpoints.
	im := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	im.SetNRGBA(0, 0, color.NRGBA{100, 50, 20, 128})
	before := append([]byte(nil), im.Pix...)
	ctx := modernTerrainColourContext{}
	ctx.Base[1] = [3]float64{100, 50, 20}
	ctx.Raw[1] = [3]float64{255, 255, 255}
	out, err := shadeModernHDImage(im, []byte{1}, ctx)
	if err != nil || out.RGBAAt(0, 0) != (color.RGBA{128, 128, 128, 128}) || !bytes.Equal(before, im.Pix) {
		t.Fatal("flash lost coverage", err)
	}
}
