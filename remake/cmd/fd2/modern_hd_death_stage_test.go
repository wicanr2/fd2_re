package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func deathStageTestGame(t *testing.T) *Game {
	t.Helper()
	g, _, _ := player9StageGame(t, false, true)
	g.sel, g.ring, g.nativeCommand0Targeting = nil, false, false
	g.dialog, g.dlgShown, g.bannerT, g.banner = nil, dlgNone, 0, ""
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g
}

func deathStageNativeState(g *Game) string {
	frame, wait, drawn := -1, -1, false
	if j := g.nativeCh28PostPresent; j != nil {
		frame, wait, drawn = j.frame, j.wait, j.drawn
	}
	raw, err := json.Marshal([]any{rewardStageNativeState(g), g.st.Units, frame, wait, drawn})
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func TestModernHDDeathStageSourceAndReject(t *testing.T) {
	set := itemHDSet(t)
	g := deathStageTestGame(t)
	g.modernStoryPortraits = set
	startDeathStageScript(t, g)
	j := g.nativeCh28PostPresent
	if j.hdStage == nil || !j.hdStage.sealed || len(j.hdFrames) != 26 {
		t.Fatal("sealed stage absent")
	}
	for i, f := range j.hdFrames {
		if !f.TransientTerrain || len(f.Terrain) == 0 || !f.Terrain[0].Stage || f.Terrain[0].Image.Bounds() != image.Rect(0, 0, 1872, 1152) {
			t.Fatal("HD stage missing", i)
		}
		want := image.Rect(0, 0, 1872, 1152)
		if i == 25 {
			want = want.Add(image.Pt(24, 24))
		}
		if f.Terrain[0].Rect != want {
			t.Fatal("stage displacement", i, f.Terrain[0].Rect, want)
		}
	}
	want := healHDCPUWorld(j.hdFrames[19], set)
	g.nativeCh23State.staging[0] ^= 1
	fresh, err := g.prepareModernHDDeathFrames(j, set)
	g.nativeCh23State.staging[0] ^= 1
	if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(fresh[19], set).Pix) {
		t.Fatal("live stage leaked", err)
	}
	active := j.hdFrames[0]
	check := func(t *testing.T) {
		t.Helper()
		before := deathStageNativeState(g)
		if err := g.applyTheme(set); err == nil || j.hdFrames[0] != active || g.modernStoryPortraits != set || before != deathStageNativeState(g) {
			t.Fatal("source mutation partially published", err)
		}
	}
	for _, kind := range []string{"stage", "baseline", "work", "vga", "typed", "palette", "target", "wait", "missing"} {
		t.Run(kind, func(t *testing.T) {
			switch kind {
			case "stage":
				j.hdStage.stage.staging[0] ^= 1
				defer func() { j.hdStage.stage.staging[0] ^= 1 }()
			case "baseline":
				j.hdStage.baselineWork[0] ^= 1
				defer func() { j.hdStage.baselineWork[0] ^= 1 }()
			case "work":
				j.workFrames[19][0] ^= 1
				defer func() { j.workFrames[19][0] ^= 1 }()
			case "vga":
				j.vgaFrames[13][0] ^= 1
				defer func() { j.vgaFrames[13][0] ^= 1 }()
			case "typed":
				j.hdInputs[12].Frame.Cells[0].Tile ^= 1
				defer func() { j.hdInputs[12].Frame.Cells[0].Tile ^= 1 }()
			case "palette":
				old := j.palette[0]
				j.palette[0] = color.RGBA{1, 2, 3, 255}
				defer func() { j.palette[0] = old }()
			case "target":
				j.hdTargets[0].Origin++
				defer func() { j.hdTargets[0].Origin-- }()
			case "wait":
				j.waits[0]++
				defer func() { j.waits[0]-- }()
			case "missing":
				source := j.hdStage
				j.hdStage = nil
				defer func() { j.hdStage = source }()
			}
			check(t)
		})
	}
	bad := *set
	stage := *set.mapStages[42]
	stage.art.Image = image.NewRGBA(image.Rect(0, 0, 2, 2))
	bad.mapStages = map[int]*modernMapStageHD{42: &stage}
	if err := g.applyTheme(&bad); err == nil || j.hdFrames[0] != active {
		t.Fatal("bad stage published", err)
	}
	before := deathStageNativeState(g)
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	if err := g.applyTheme(set); err != nil || before != deathStageNativeState(g) || !bytes.Equal(want.Pix, healHDCPUWorld(j.hdFrames[19], set).Pix) {
		t.Fatal("valid F2 changed source", err)
	}
}

func TestNativeDeathStageColdFailureIsAtomic(t *testing.T) {
	g := deathStageTestGame(t)
	g.nativeCh23State = nil
	before := deathStageNativeState(g)
	err := g.startNativeUnitDeathPresent(func(s *battle.State) error { s.Units[4].HP = 0; return nil }, nil)
	if err == nil || g.nativeCh28PostPresent != nil || before != deathStageNativeState(g) {
		t.Fatal("missing stage published a death", err)
	}
}

func TestModernHDDeathStageOmitsSteadyHUDUnit(t *testing.T) {
	g := deathStageTestGame(t)
	g.modernStoryPortraits = deathHDTestSet(g)
	startDeathStageScript(t, g)
	j := g.nativeCh28PostPresent
	if j.hdFrames[25].HUDUnit == nil {
		t.Fatal("fixture lacks steady HUD unit")
	}
	for i, frame := range j.hdFrames[:25] {
		if frame.HUDUnit != nil {
			t.Fatal("death event added steady HUD unit before final redraw", i)
		}
	}
}

func startDeathStageScript(t *testing.T, g *Game) {
	t.Helper()
	if err := g.startNativeUnitDeathPresent(func(s *battle.State) error {
		s.Units[4].HP = 0
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestNativeDeathStageUsesSnapshotSeed(t *testing.T) {
	g := deathStageTestGame(t)
	startDeathStageScript(t, g)
	j := g.nativeCh28PostPresent
	if len(j.hdTargets) == 0 {
		t.Fatal("controlled target not visible")
	}
	in := nativeTransitionInput(j.hdInputs[19])
	want := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.SeedNativeCh23Staging(want, g.nativeCh23State.staging); err != nil {
		t.Fatal(err)
	}
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(want, in); err != nil {
		t.Fatal(err)
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(want, in); err != nil {
		t.Fatal(err)
	}
	for _, target := range j.hdTargets {
		if err := fdother.BlitNativeCh28PostOverlay(g.nativeMapAssets.CommandHealDigits[74], want, target.Origin); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(want, j.workFrames[19]) {
		t.Fatal("death overlay B lost same-compose map23 stage seed")
	}
}
