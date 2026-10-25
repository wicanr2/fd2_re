package main

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestModernHDUnitStageRemapMatchesNativeMarker(t *testing.T) {
	g, _, _ := player9StageGame(t, false, true)
	set := *itemHDSet(t)
	stage := *set.mapStages[42]
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
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(w, nativeTransitionInput(in)); err != nil {
		t.Fatal(err)
	}
	if err := indexedmap.CopyNativeUnitPresentViewport(v, w); err != nil {
		t.Fatal(err)
	}
	normal, err := g.prepareModernHDCommandSteadyStage(in, w, v, g.nativeMapAssets.Palette, &set, g.nativeCh23State)
	if err != nil {
		t.Fatal(err)
	}
	passes, err := fdother.NativeUnitPresentLUTFrames(11, 2)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[byte]bool{}
	for _, index := range []int{0, 3, 5, 6, 10, 15} {
		p := passes[index]
		lut := g.nativeMapAssets.LUTs[p.LUTIndex]
		work, vga := make([]byte, len(w)), make([]byte, 64000)
		if err := indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, w, lut, nativeTransitionInput(in), p); err != nil {
			t.Fatal(err)
		}
		f, err := g.prepareModernHDRadialMap(modernHDRadialMapSpec{Input: in, Palette: g.nativeMapAssets.Palette, Pass: p.Pass, LUT: lut, WorkOffset: unitPresentHDBase, Background: vga, Stage: g.nativeCh23State}, &set, normal, nil)
		if err != nil {
			t.Fatal(err)
		}
		counts, _, err := modernHDRadialCounts(p.Pass, unitPresentHDBase)
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
						t.Fatalf("unit stage marker pass%d at%d,%d got%v want%v", index, x, y, got, want)
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
	// The final rectangle ends at centerY, where the second radial starts.
	// These native regions do not overlap, so this caller has counts 0..2.
	for count := byte(0); count < 3; count++ {
		if !seen[count] {
			t.Fatal("missing native remap count", count, seen)
		}
	}
}

func TestNativeUnitPresentStageUsesSnapshotSeed(t *testing.T) {
	for _, y := range []int{14, 19} {
		g, _, _ := player9StageGame(t, false, true)
		g.camX, g.camY = 16*24, 14*24
		spec := campaign.NativeUnitPresent{Slot: 0, NewX: 255, NewY: 255, VisualX: 27, VisualY: y}
		j, err := g.buildNativeUnitPresentJob(spec, g.st, g.nativeMapWork, g.nativeMapVGA, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		if err := indexedmap.SeedNativeCh23Staging(want, g.nativeCh23State.staging); err != nil {
			t.Fatal(err)
		}
		if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(want, nativeTransitionInput(j.hdSource.Before)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, j.hdSource.Terrain) {
			t.Fatalf("visualY%d lost the original222aa stage seed", y)
		}
	}
}

func TestModernHDUnitStageRelocationSourcesAndReject(t *testing.T) {
	set := itemHDSet(t)
	for _, top := range []bool{true, false} {
		g, a, b := unitStageRelocationGame(t, top, set)
		root := g.nativeModifierPresentation.unitPresent
		if root == nil || root.hdNext == nil || a.MP != 30 || a.Acted {
			t.Fatal("two legs were not staged atomically")
		}
		for _, j := range []*nativeUnitPresentJob{root, root.hdNext} {
			if j.hdSource.Stage == nil || len(j.hdFrames) != len(j.vgaFrames) {
				t.Fatal("stage source absent")
			}
			for _, i := range []int{0, 11, len(j.hdFrames) - 1} {
				f := j.hdFrames[i]
				if !f.TransientTerrain || len(f.Terrain) == 0 || !f.Terrain[0].Stage || f.Terrain[0].Image.Bounds() != image.Rect(0, 0, 1872, 1152) {
					t.Fatal("full HD stage not bound", i)
				}
			}
		}
		before := unitPresentGPUState(g)
		want := healHDCPUWorld(root.hdFrames[11], set)
		g.nativeCh23State.staging[0] ^= 1
		fresh, err := g.prepareModernHDUnitPresent(root, set)
		if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(fresh[11], set).Pix) {
			t.Fatal("live stage leaked", err)
		}
		g.nativeCh23State.staging[0] ^= 1
		active := root.hdFrames
		check := func(theme *modernStoryPortraitSet) {
			t.Helper()
			if err := g.applyTheme(theme); err == nil || root.hdFrames[0] != active[0] || g.modernStoryPortraits != set || !reflect.DeepEqual(before, unitPresentGPUState(g)) {
				t.Fatal("invalid stage source partially published", err)
			}
		}
		root.hdNext.hdSource.Stage.staging[0] ^= 1
		check(set)
		root.hdNext.hdSource.Stage.staging[0] ^= 1
		bad := *set
		stage := *set.mapStages[42]
		stage.art.Image = image.NewRGBA(image.Rect(0, 0, 10, 10))
		bad.mapStages = map[int]*modernMapStageHD{42: &stage}
		check(&bad)
		if err := g.applyTheme(nil); err != nil {
			t.Fatal(err)
		}
		if err := g.applyTheme(set); err != nil {
			t.Fatal(err)
		}
		if b.X != 27 || a.MP != 30 || a.Acted {
			t.Fatal("preflight changed transaction")
		}
	}
}

func TestNativeUnitStageMissingSourceIsAtomic(t *testing.T) {
	g, _, _ := player9StageGame(t, true, true)
	g.camX, g.camY = 16*24, 14*24
	g.nativeCh23State = nil
	before := stage42NativeState(g)
	err := g.startNativeUnitPresent(campaign.NativeUnitPresent{Slot: 0, NewX: 255, NewY: 255, VisualX: 20, VisualY: 18}, nil)
	if err == nil || g.nativeUnitPresent != nil || !reflect.DeepEqual(before, stage42NativeState(g)) {
		t.Fatal("missing stage published", err)
	}
}
