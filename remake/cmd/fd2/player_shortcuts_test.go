package main

import (
	"image"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestRefreshThemeVisualsKeepsLiveBattleState(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 48, 24))
	modern := image.NewRGBA(image.Rect(0, 0, 48, 24))
	state := &battle.State{}
	scenario := &battle.Scenario{}
	g := &Game{
		m:              &MapData{TileW: 24, TileH: 24, Cols: 2},
		baseMapTileset: base,
		currentMapID:   0,
		st:             state,
		sc:             scenario,
		curX:           7,
		curY:           9,
		modernStoryPortraits: &modernStoryPortraitSet{
			portraits:   map[int]*modernStoryPortraitFrame{},
			mapSprites:  map[int][]image.Image{},
			mapTilesets: map[int]image.Image{0: modern},
		},
	}

	g.refreshThemeVisuals()

	if g.st != state || g.sc != scenario || g.curX != 7 || g.curY != 9 {
		t.Fatalf("主題切換改動戰況：st=%p sc=%p cursor=(%d,%d)", g.st, g.sc, g.curX, g.curY)
	}
	if !g.modernMapTilesetLoaded || len(g.tiles) != 2 {
		t.Fatalf("現代地圖未原子套用：loaded=%v tiles=%d", g.modernMapTilesetLoaded, len(g.tiles))
	}

	g.modernStoryPortraits = nil
	g.refreshThemeVisuals()
	if g.modernMapTilesetLoaded || len(g.tiles) != 2 || g.st != state || g.sc != scenario {
		t.Fatalf("切回忠實主題破壞戰況：loaded=%v tiles=%d", g.modernMapTilesetLoaded, len(g.tiles))
	}
}

func TestApplyThemeRebindsActiveDialogueWithoutReset(t *testing.T) {
	portrait := &modernStoryPortraitFrame{image: image.NewRGBA(image.Rect(0, 0, 80, 80))}
	state := &battle.State{Turn: 7}
	dialogue := []battle.DialogLine{{Speaker: 48, NativeDialogue: &battle.NativeDialogueLayout{Pages: [][]string{{"first"}, {"second"}}}}}
	g := &Game{m: &MapData{TileW: 24, TileH: 24, Cols: 2},
		baseMapTileset: image.NewRGBA(image.Rect(0, 0, 48, 24)), st: state,
		dialog: dialogue, nativeDialogueProgress: 19, dlgT: 11,
		modernHDOverlay: &modernHDOverlay{}}
	set := &modernStoryPortraitSet{portraits: map[int]*modernStoryPortraitFrame{48: portrait}}
	for _, next := range []*modernStoryPortraitSet{set, nil, set} {
		if err := g.applyTheme(next); err != nil {
			t.Fatal(err)
		}
		if g.st != state || state.Turn != 7 || !reflect.DeepEqual(g.dialog, dialogue) ||
			g.nativeDialogueProgress != 19 || g.dlgT != 11 || g.modernHDOverlay != nil {
			t.Fatal("theme change reset live gameplay or dialogue")
		}
		if (g.nativeDialogueModernPortrait == portrait) != (next != nil) {
			t.Fatal("active portrait did not follow theme")
		}
	}
}

func TestApplyThemeRejectsMissingActiveSpeakerBeforePublication(t *testing.T) {
	old := &modernStoryPortraitSet{}
	portrait := &modernStoryPortraitFrame{}
	overlay := &modernHDOverlay{portrait: portrait}
	g := &Game{modernStoryPortraits: old, nativeDialogueModernPortrait: portrait,
		modernHDOverlay: overlay, dialog: []battle.DialogLine{{Speaker: 48}}}
	if err := g.applyTheme(&modernStoryPortraitSet{}); err == nil {
		t.Fatal("missing current speaker accepted")
	}
	if g.modernStoryPortraits != old || g.nativeDialogueModernPortrait != portrait || g.modernHDOverlay != overlay {
		t.Fatal("rejected theme partially published")
	}
}

func TestCycleThemeReusesValidatedPackAfterReturningToFaithful(t *testing.T) {
	t.Setenv("FD2_MODERN_THEME_PACK", t.TempDir()) // No files to decode a second time.
	set := &modernStoryPortraitSet{hasHD: true}
	g := &Game{m: &MapData{TileW: 24, TileH: 24, Cols: 2},
		baseMapTileset: image.NewRGBA(image.Rect(0, 0, 48, 24)), modernStoryPortraits: set}
	g.cycleTheme()
	if g.modernStoryPortraits != nil || g.modernThemeCache != set {
		t.Fatal("validated resources were discarded")
	}
	g.cycleTheme()
	if g.modernStoryPortraits != set {
		t.Fatal("return to HD attempted to decode the unavailable source again")
	}
}
