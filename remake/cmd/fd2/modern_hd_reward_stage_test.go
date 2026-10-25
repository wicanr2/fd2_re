package main

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func rewardStageGame(t *testing.T, set *modernStoryPortraitSet) (*Game, *battle.Unit) {
	t.Helper()
	g, a, _ := player9StageGame(t, false, true)
	// The shared command fixture sets BattleFig9 for its cast animation.
	// This reward fixture keeps Sol's original map selector and DATO0 together.
	a.BattleFig = 0
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto font required")
	}
	_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.sel, g.ring, g.nativeCommand0Targeting = nil, false, false
	g.dialog, g.dlgShown = nil, dlgNone
	g.bannerT, g.banner = 0, ""
	g.gold = 123
	g.camX, g.camY = 16*24, 14*24
	a.Acted = true
	a.NativeRecordByte5 |= 0x80
	a.HasNativeRecordByte5 = true
	g.modernStoryPortraits = set
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	key, err := g.nativeMapFrameSource.Frame.SelectorCache.KeyForSlot(0)
	if err != nil || key != 0 || a.BattleFig != 0 {
		t.Fatal("controlled Sol map/DATO identity mismatch", key, a.BattleFig, err)
	}
	return g, a
}

func TestModernHDRewardStageSource(t *testing.T) {
	set := itemHDSet(t)
	g, a := rewardStageGame(t, set)
	if err := g.beginNativeDeathRewardMessage(pendingNativeDeathRewardMessage{killer: a, kind: 1, value: 123456}, a, nil); err != nil {
		t.Fatal(err)
	}
	m := g.nativeSystemEndTurnUI.rewardMessage
	if m.hdWorld == nil || len(m.hdWorld.Terrain) == 0 || !m.hdWorld.Terrain[0].Stage || m.hdWorld.Terrain[0].Image.Bounds() != image.Rect(0, 0, 1872, 1152) {
		t.Fatal("reward lost frozen map23 stage")
	}
	if a.NativeRecordByte5&0x80 == 0 || !a.Acted || g.gold != 123 {
		t.Fatal("reward preflight changed transaction")
	}
	want := healHDCPUWorld(m.hdWorld, set)
	g.nativeCh23State.staging[0] ^= 1
	fresh, err := g.prepareModernHDRewardWorld(m, g.nativeSystemEndTurnUI.source, set)
	if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(fresh, set).Pix) {
		t.Fatal("live stage leaked into reward", err)
	}
	g.nativeCh23State.staging[0] ^= 1
	active, job := m.hdWorld, g.nativeClassUIJob
	check := func(theme *modernStoryPortraitSet) {
		t.Helper()
		before := rewardStageNativeState(g)
		if err := g.applyTheme(theme); err == nil || m.hdWorld != active || g.nativeClassUIJob != job || g.modernStoryPortraits != set || !reflect.DeepEqual(before, rewardStageNativeState(g)) {
			t.Fatal("invalid reward source partially published", err)
		}
	}
	for _, kind := range []string{"stage", "work", "vga", "typed", "palette", "panel-source", "missing-stage", "missing-scene"} {
		t.Run(kind, func(t *testing.T) {
			s := m.hdScene
			switch kind {
			case "stage":
				s.stage.staging[0] ^= 1
				defer func() { s.stage.staging[0] ^= 1 }()
			case "work":
				s.work[0] ^= 1
				defer func() { s.work[0] ^= 1 }()
			case "vga":
				s.vga[0] ^= 1
				defer func() { s.vga[0] ^= 1 }()
			case "typed":
				s.input.Frame.Cells[0].Tile ^= 1
				defer func() { s.input.Frame.Cells[0].Tile ^= 1 }()
			case "palette":
				old := s.palette[0]
				s.palette[0] = color.RGBA{1, 2, 3, 255}
				defer func() { s.palette[0] = old }()
			case "panel-source":
				g.nativeSystemEndTurnUI.source[0] ^= 1
				defer func() { g.nativeSystemEndTurnUI.source[0] ^= 1 }()
			case "missing-stage":
				stage := s.stage
				s.stage = nil
				defer func() { s.stage = stage }()
			case "missing-scene":
				m.hdScene = nil
				defer func() { m.hdScene = s }()
			}
			check(set)
		})
	}
	bad := *set
	stage := *set.mapStages[42]
	stage.art.Image = image.NewRGBA(image.Rect(0, 0, 2, 2))
	bad.mapStages = map[int]*modernMapStageHD{42: &stage}
	check(&bad)
	before := rewardStageNativeState(g)
	if err := g.applyTheme(nil); err != nil || m.hdWorld != nil {
		t.Fatal("pixel theme failed", err)
	}
	if err := g.applyTheme(set); err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(m.hdWorld, set).Pix) || !reflect.DeepEqual(before, rewardStageNativeState(g)) {
		t.Fatal("valid reward F2 changed frozen state", err)
	}
	cold, actor := rewardStageGame(t, &bad)
	units := unitPresentGPUState(cold)
	if err := cold.beginNativeDeathRewardMessage(pendingNativeDeathRewardMessage{killer: actor, kind: 0, value: 192}, actor, nil); err == nil || cold.nativeSystemEndTurnUI != nil || cold.nativeClassUIJob != nil || cold.gold != 123 || cold.nativeRNGState != 7 || !reflect.DeepEqual(units, unitPresentGPUState(cold)) {
		t.Fatal("cold invalid stage published reward", err)
	}
}

func rewardStageNativeState(g *Game) any {
	frame, drawn := -1, false
	if j := g.nativeClassUIJob; j != nil {
		frame, drawn = j.frame, j.drawn
	}
	return []any{unitPresentGPUState(g), g.gold, g.nativeRNGState, g.frame, frame, drawn,
		g.nativeSystemEndTurnUI != nil, g.nativeDeathRewardMessageAwaitingKey(), len(g.pendingNativeRewardMsgs),
		append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)}
}
