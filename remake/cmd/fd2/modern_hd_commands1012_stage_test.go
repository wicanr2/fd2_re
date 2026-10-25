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

func commands1012StageGame(t *testing.T, id, variant int, hd bool) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, a, b := player9StageGame(t, hd, variant != 1)
	record := g.st.NativeCommandBook[9]
	record.ID = id
	a.SetMapPlacement(26, 16, 3)
	b.SetMapPlacement(27, 16, 0)
	units := []*battle.Unit{a, b}
	if variant == 2 {
		u := g.st.Units[5]
		u.Camp, u.OnField, u.HP, u.MaxHP = b.Camp, true, 1000, 1000
		u.ClassID, u.NativeRecordClass, u.HasNativeRecordClass = b.ClassID, b.NativeRecordClass, b.HasNativeRecordClass
		u.SetMapPlacement(27, 15, 0)
		units = append(units, u)
		record.EffectMode = 1
	}
	g.st.NativeCommandBook[id] = record
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
	return g, a, b
}

func TestModernHDCommand1012StageIsolationAndReject(t *testing.T) {
	for _, id := range []int{10, 11, 12} {
		g, a, b := commands1012StageGame(t, id, 0, true)
		if err := g.startNativeCommand1012Presentation(a, b, id, nil); err != nil {
			t.Fatal(id, err)
		}
		j, set := g.nativeCmd1012, g.modernStoryPortraits
		if j.hdFrames == nil || !j.hdFrames.Hold.TransientTerrain || len(j.hdFrames.Results) != 22 {
			t.Fatal("missing full stage")
		}
		want := healHDCPUWorld(j.hdFrames.Hold, set)
		g.nativeCh23State.staging[0] ^= 1
		fresh, err := g.prepareModernHDCommand1012(j, set)
		if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(fresh.Hold, set).Pix) {
			t.Fatal("live stage leaked", err)
		}
		if p := g.native2189A; p != nil {
			want := healHDCPUWorld(p.hdFrames[9], set)
			fresh, err := g.prepareModernHDCommandPrelude(p, set)
			if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(fresh[9], set).Pix) {
				t.Fatal("live stage leaked into prelude", err)
			}
		}
		g.nativeCh23State.staging[0] ^= 1
		active := j.hdFrames
		check := func() {
			t.Helper()
			before := stage42NativeState(g)
			if err := g.applyTheme(set); err == nil || j.hdFrames != active || g.modernStoryPortraits != set || !reflect.DeepEqual(before, stage42NativeState(g)) {
				t.Fatal("changed source published")
			}
		}
		j.hdScene.stage.staging[0] ^= 1
		check()
		j.hdScene.stage.staging[0] ^= 1
		j.mainFrames[59][0] ^= 1
		check()
		j.mainFrames[59][0] ^= 1
		if p := g.native2189A; p != nil {
			p.hdSteady.work[0] ^= 1
			check()
			p.hdSteady.work[0] ^= 1
			p.mapStage.staging[0] ^= 1
			check()
			p.mapStage.staging[0] ^= 1
			p.workFrames[9][0] ^= 1
			check()
			p.workFrames[9][0] ^= 1
		}
		if err := g.applyTheme(nil); err != nil {
			t.Fatal(err)
		}
		if err := g.applyTheme(set); err != nil {
			t.Fatal(err)
		}
	}
}

func TestModernHDCommand1012StageColdReject(t *testing.T) {
	for _, id := range []int{10, 11, 12} {
		g, a, b := commands1012StageGame(t, id, 0, true)
		g.nativeCh23State = nil
		before := stage42NativeState(g)
		if err := g.startNativeCommand1012Presentation(a, b, id, nil); err == nil || g.nativeCmd1012 != nil || g.native2189A != nil || !reflect.DeepEqual(before, stage42NativeState(g)) {
			t.Fatal("missing stage was not atomic", id, err)
		}
	}
}

func TestModernHDCommandPreludeStageMarker(t *testing.T) {
	g, a, b := commands1012StageGame(t, 12, 0, true)
	set := *g.modernStoryPortraits
	stage := *set.mapStages[42]
	stage.native.Pixels = bytes.Repeat([]byte{243}, 312*192)
	base := color.RGBAModel.Convert(g.nativeMapAssets.Palette[243]).(color.RGBA)
	picture := image.NewRGBA(image.Rect(0, 0, 1872, 1152))
	for y := 0; y < 1152; y++ {
		for x := 0; x < 1872; x++ {
			picture.SetRGBA(x, y, base)
		}
	}
	stage.art = modernTerrainHDAsset{Image: picture, Colours: []byte{243}}
	set.mapStages = map[int]*modernMapStageHD{42: &stage}
	assets := *g.nativeMapAssets
	bank := *assets.Terrain
	bank.Sprites = append(bank.Sprites[:0:0], bank.Sprites...)
	for i := range bank.Sprites {
		bank.Sprites[i].Mask = make([]byte, 576)
		bank.Sprites[i].RemapMask = make([]byte, 576)
	}
	assets.Terrain = &bank
	g.nativeMapAssets = &assets
	g.nativeCh23State.staging = append([]byte(nil), stage.native.Pixels...)
	g.modernStoryPortraits = &set
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if err := g.startNativeCommand1012Presentation(a, b, 12, nil); err != nil {
		t.Fatal(err)
	}
	j := g.native2189A
	inside, outside := 0, 0
	for i, spec := range j.hdRadial {
		work := append([]byte(nil), j.hdTerrain...)
		if err := fdother.ApplyRadialLUTRemap(work[0x8088:], 456, j.hdLUTs[i], spec); err != nil {
			t.Fatal(err)
		}
		vga := make([]byte, 64000)
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			t.Fatal(err)
		}
		plane := j.hdFrames[i].Terrain[0].Image
		for y := 0; y < 192; y += 3 {
			for x := 0; x < 312; x += 3 {
				got := plane.RGBAAt(x*6, y*6)
				want := color.RGBAModel.Convert(j.palette[vga[y*320+x]]).(color.RGBA)
				if got != want {
					t.Fatalf("prelude marker pass%d %d,%d got%v want%v", i, x, y, got, want)
				}
				if vga[y*320+x] == 243 {
					outside++
				} else {
					inside++
				}
			}
		}
	}
	if inside == 0 || outside == 0 {
		t.Fatal("vacuous radial marker", inside, outside)
	}
}

func TestNativeCommandPreludeStageUsesSingleSnapshot(t *testing.T) {
	for _, id := range []int{11, 12} {
		g, a, b := commands1012StageGame(t, id, 0, false)
		// A controlled non-identity destination remap exposes rebuilding from
		// an already composed frame. Original banks and call geometry remain.
		assets := *g.nativeMapAssets
		assets.LUTs = append([][]byte(nil), assets.LUTs...)
		roster, err := g.st.NativeMapFrameRoster()
		if err != nil {
			t.Fatal(err)
		}
		phase, err := fdother.NativeTerrainLUTIndex(roster.TerrainPhase)
		if err != nil {
			t.Fatal(err)
		}
		assets.LUTs[phase] = make([]byte, 256)
		for i := range assets.LUTs[phase] {
			assets.LUTs[phase][i] = byte(i + 1)
		}
		g.nativeMapAssets = &assets
		for i := range g.m.NativeTileBlitModes {
			g.m.NativeTileBlitModes[i] = 0
		}
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		before := stage42NativeState(g)
		if err := g.startNativeCommand1012Presentation(a, b, id, nil); err != nil {
			t.Fatal(err)
		}
		j := g.native2189A
		input := nativeTransitionInput(j.hdInput)
		want := append([]byte(nil), g.nativeCmd1012.baselineWork...)
		if err := indexedmap.SeedNativeCh23Staging(want, g.nativeCh23State.staging); err != nil {
			t.Fatal(err)
		}
		if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(want, input); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, j.hdTerrain) {
			t.Fatalf("command%d prelude did not restore the same stage seed", id)
		}
		for i, spec := range j.hdRadial {
			work := append([]byte(nil), want...)
			if err := fdother.ApplyRadialLUTRemap(work[0x8088:], 456, j.hdLUTs[i], spec); err != nil {
				t.Fatal(err)
			}
			if err := indexedmap.RedrawNativeUnitPresentObjects(work, input); err != nil {
				t.Fatal(err)
			}
			vga := append([]byte(nil), g.nativeCmd1012.baselineVGA...)
			if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(work, j.workFrames[i]) || !bytes.Equal(vga, j.vgaFrames[i]) {
				t.Fatalf("command%d native pass%d source changed", id, i)
			}
		}
		// Starting the prelude publishes pixels, but no rule transaction.
		if a.MP != 100 || a.Acted || g.nativeRNGState != 7 || b.HP != 1000 || reflect.DeepEqual(before, stage42NativeState(g)) {
			t.Fatal("prelude publication or transaction boundary changed")
		}
	}
}
