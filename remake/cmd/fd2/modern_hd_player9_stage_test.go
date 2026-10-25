package main

import (
	"bytes"
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func player9StageGame(t *testing.T, hd, hit bool) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	t.Setenv("FD2_MUTE", "1")
	now := time.Unix(1000, 0)
	g := stage42ActualGame(t, &now)
	a, b := g.st.Units[0], g.st.Units[4]
	a.MP, a.Acted, b.HP, b.MaxHP = 100, false, 1000, 1000
	chance := 0
	if hit {
		chance = 100
	}
	g.st.NativeCommandBook[9] = battle.NativeCommandRecord{ID: 9, Damage: 999, Hit: chance, SelectionMode: 3, EffectMode: 0, MPCost: 30, TargetCode: 0}
	g.nativeRNGState = 7
	if hd {
		g.modernStoryPortraits = itemHDSet(t)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, a, b
}

func TestModernHDPlayer9StageFreezeUsesSeed(t *testing.T) {
	g := stage42TestGame(t, true)
	in, err := freezeCompoundMapInput(*g.nativeMapFrameSource)
	if err != nil {
		t.Fatal(err)
	}
	in.Frame.LUT = make([]byte, 256)
	for i := range in.Frame.LUT {
		in.Frame.LUT[i] = byte(i + 1)
	}
	for i := range in.Frame.Cells {
		in.Frame.Cells[i].BlitMode = 0
	}
	w, v := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := indexedmap.SeedNativeCh23Staging(w, g.nativeCh23State.staging); err != nil {
		t.Fatal(err)
	}
	if err := indexedmap.ComposeNativeFrame(w, v, in); err != nil {
		t.Fatal(err)
	}
	againW, againV := append([]byte(nil), w...), append([]byte(nil), v...)
	if err := indexedmap.ComposeNativeFrame(againW, againV, in); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(v, againV) {
		t.Fatal("fixture did not exercise repeated destination mapping")
	}
	g.nativeMapFrameSource, g.nativeMapWork, g.nativeMapVGA = &in, w, v
	before := stage42NativeState(g)
	if _, err := g.freezeNativeCommand9MapInput(); err != nil {
		t.Fatal("freeze must seed the same composition", err)
	}
	if !reflect.DeepEqual(before, stage42NativeState(g)) {
		t.Fatal("freeze mutated native state")
	}
}

func TestModernHDPlayer9ActualStage(t *testing.T) {
	for _, hit := range []bool{true, false} {
		g, a, b := player9StageGame(t, true, hit)
		before := stage42NativeState(g)
		if err := g.startNativeCommand9PlayerPresentation(a, b, nil); err != nil {
			t.Fatal(err)
		}
		j := g.nativeCmd9Player
		if j == nil || j.hdFrames == nil || len(j.hdFrames.Palette) != 8 || len(j.hdFrames.Effects) != 27 || len(j.hdFrames.Results) != 22 {
			t.Fatal("incomplete player9 owner")
		}
		frames := append(append(append([]*modernHDMapFrame{j.hdFrames.Hold}, j.hdFrames.Palette...), j.hdFrames.Effects...), j.hdFrames.Results...)
		for _, f := range frames {
			if !f.TransientTerrain || len(f.Terrain) == 0 || f.Terrain[0].Image.Bounds() != image.Rect(0, 0, 1872, 1152) {
				t.Fatal("map23 background lost from player9 phase")
			}
		}
		if !reflect.DeepEqual(before, stage42NativeState(g)) {
			t.Fatal("HD preflight changed native state")
		}
		want := healHDCPUWorld(j.hdFrames.Hold, g.modernStoryPortraits)
		g.nativeCh23State.staging[0] ^= 1
		fresh, err := g.prepareModernHDPlayer9Frames(j, g.modernStoryPortraits)
		if err != nil {
			t.Fatal("live stage leaked into frozen source", err)
		}
		if !bytes.Equal(want.Pix, healHDCPUWorld(fresh.Hold, g.modernStoryPortraits).Pix) {
			t.Fatal("live stage changed frozen background")
		}
	}
}

func TestModernHDPlayer9StageRejectsEditedSource(t *testing.T) {
	g, a, b := player9StageGame(t, true, true)
	if err := g.startNativeCommand9PlayerPresentation(a, b, nil); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCmd9Player
	active, theme := j.hdFrames, g.modernStoryPortraits
	check := func() {
		t.Helper()
		if err := g.applyTheme(theme); err == nil || j.hdFrames != active || g.modernStoryPortraits != theme {
			t.Fatal("edited frozen producer replaced active theme")
		}
	}
	j.baselineVGA[0] ^= 1
	check()
	j.baselineVGA[0] ^= 1
	j.hdInput.Frame.Cells[0].Tile ^= 1
	check()
	j.hdInput.Frame.Cells[0].Tile ^= 1
	j.hdScene.stage.staging[0] ^= 1
	check()
	j.hdScene.stage.staging[0] ^= 1
	bad := *theme
	stage := *theme.mapStages[42]
	stage.art.Image = image.NewRGBA(image.Rect(0, 0, 10, 10))
	bad.mapStages = map[int]*modernMapStageHD{42: &stage}
	if err := g.applyTheme(&bad); err == nil || j.hdFrames != active || g.modernStoryPortraits != theme {
		t.Fatal("wrong-size stage replaced active theme")
	}
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	if err := g.applyTheme(theme); err != nil || j.hdFrames == nil || j.hdTheme != theme {
		t.Fatal("valid F2 recovery failed", err)
	}
}
