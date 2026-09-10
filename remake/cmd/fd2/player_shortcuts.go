package main

import (
	"bytes"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) cycleTheme() {
	if g == nil || g.baseMapTileset == nil || g.m == nil {
		return
	}
	previous := g.modernStoryPortraits
	if previous == nil {
		packRoot := os.Getenv("FD2_MODERN_THEME_PACK")
		if packRoot == "" {
			packRoot = assetPath("assets/themes/modern")
		}
		set, err := loadModernStoryPortraitSet(assetPath("assets/themes/modern/catalog.json"), packRoot)
		if err != nil {
			g.msg = "現代主題不可用：" + err.Error()
			return
		}
		g.modernStoryPortraits = set
	} else {
		g.modernStoryPortraits = nil
	}
	g.refreshThemeVisuals()
	name := "忠實原版"
	if g.modernStoryPortraits != nil {
		name = "現代手繪"
	}
	g.msg = "主題：" + name
}

// refreshThemeVisuals 只替換繪圖資源；戰場單位、回合、事件、鏡頭與存檔狀態均不重設。
func (g *Game) refreshThemeVisuals() {
	img := g.baseMapTileset
	g.modernMapTilesetLoaded = false
	if g.modernStoryPortraits != nil {
		if modern, ok := g.modernStoryPortraits.mapTilesets[g.currentMapID]; ok {
			img = modern
			g.modernMapTilesetLoaded = true
		}
	}
	g.tileset = ebiten.NewImageFromImage(img)
	g.tiles = nil
	cols := g.m.Cols
	if cols == 0 {
		cols = g.tileset.Bounds().Dx() / g.m.TileW
	}
	for i, n := 0, (g.tileset.Bounds().Dy()/g.m.TileH)*cols; i < n; i++ {
		sx, sy := (i%cols)*g.m.TileW, (i/cols)*g.m.TileH
		g.tiles = append(g.tiles, g.tileset.SubImage(image.Rect(sx, sy, sx+g.m.TileW, sy+g.m.TileH)).(*ebiten.Image))
	}
	g.sprites = loadSprites()
	g.portraits = loadPortraits()
	for i, name := range []string{"attack", "status", "item", "wait"} {
		g.ringIcons[i] = nil
		if raw, err := os.ReadFile(assetPath("assets/ui/ring_" + name + ".png")); err == nil {
			if decoded, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
				g.ringIcons[i] = ebiten.NewImageFromImage(decoded)
			}
		}
	}
	if g.modernStoryPortraits != nil {
		for group, sourceFrames := range g.modernStoryPortraits.mapSprites {
			frames := make([]*ebiten.Image, len(sourceFrames))
			for i, source := range sourceFrames {
				frames[i] = ebiten.NewImageFromImage(source)
			}
			g.sprites[group] = frames
		}
		if len(g.modernStoryPortraits.battleActionIcons) == 4 {
			for i, icon := range g.modernStoryPortraits.battleActionIcons {
				g.ringIcons[i] = ebiten.NewImageFromImage(icon)
			}
		}
	}
}

func (g *Game) drawPlayerHelp(screen *ebiten.Image) {
	panel := ebiten.NewImage(560, 344)
	panel.Fill(color.RGBA{8, 18, 44, 242})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(40, 36)
	screen.DrawImage(panel, op)
	if g.font == nil {
		return
	}
	lines := []string{
		"操作說明",
		"方向鍵／WASD：移動游標",
		"Enter／Space：確認　Esc：取消",
		"F1：說明　F2：忠實原版／現代手繪主題",
		"F3：Sound Blaster／MT-32 音源",
		"F4：切換官方語言",
		"F5：快速存檔　F6：開啟／關閉音樂",
		"F7：字級大小　F9：快速讀檔　F12：開發除錯資訊",
		"戰場系統選單也可控制音樂、音效、速度與狀態欄。",
		"秘密商店：章節指定的修飾鍵＋F1..F10。",
	}
	for i, line := range lines {
		scale := 0.9
		if i == 0 {
			scale = 1.7
		}
		g.font.Draw(screen, line, 58, float64(54+i*30), scale, color.RGBA{238, 244, 255, 255})
	}
}
