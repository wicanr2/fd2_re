package main

import (
	"image"
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
