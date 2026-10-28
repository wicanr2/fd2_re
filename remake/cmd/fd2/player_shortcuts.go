package main

import (
	"bytes"
	"fmt"
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
		set := g.modernThemeCache
		if set == nil {
			packRoot := os.Getenv("FD2_MODERN_THEME_PACK")
			if packRoot == "" {
				packRoot = assetPath("assets/themes/modern")
			}
			var err error
			set, err = loadModernStoryPortraitSet(assetPath("assets/themes/modern/catalog.json"), packRoot)
			if err != nil {
				g.msg = "現代主題不可用：" + err.Error()
				return
			}
		}
		if err := g.applyTheme(set); err != nil {
			g.msg = "現代主題不可用：" + err.Error()
			return
		}
		g.modernThemeCache = set
	} else {
		if err := g.applyTheme(nil); err != nil {
			g.msg = "主題切換失敗：" + err.Error()
			return
		}
	}
	name := "忠實原版"
	if g.modernStoryPortraits != nil {
		name = "現代手繪"
	}
	g.msg = "主題：" + name
	g.saveDisplaySettings()
}

// applyTheme validates the current typed speaker before publishing any resource.
// Rebinding only the portrait preserves the active dialogue's pages and progress.
func (g *Game) applyTheme(set *modernStoryPortraitSet) error {
	style := g.interfaceThemeForArt(set)
	publishSystemInfo, err := g.stageModernHDSystemInfo(style)
	if err != nil {
		return err
	}
	publishItemPanels, err := g.stageModernHDItemPanels(style)
	if err != nil {
		return err
	}
	publishExpiry, err := g.stageModernHDTransientExpiry(set)
	if err != nil {
		return err
	}
	publishSkyKey, err := g.stageModernHDSkyKey(set)
	if err != nil {
		return err
	}
	publishCh23, err := g.stageModernHDCh23(set)
	if err != nil {
		return err
	}
	publishRowShake, err := g.stageModernHDRowShake(set)
	if err != nil {
		return err
	}
	publishIndexedTransition, err := g.stageModernHDIndexedTransition(set)
	if err != nil {
		return err
	}
	publishUnitPresent, err := g.stageModernHDUnitPresent(set)
	if err != nil {
		return err
	}
	storyWorld, err := g.prepareModernHDStoryMapFrame(set)
	if err != nil {
		return err
	}
	var rewardWorld *modernHDMapFrame
	var fullRewardWorld *modernHDMapFrame
	if p := g.nativeDeathRewardUI; set != nil && p != nil {
		fullRewardWorld, err = g.prepareModernHDFullRewardWorld(p, set)
		if err != nil {
			return err
		}
	}
	if state := g.nativeSystemEndTurnUI; set != nil && state != nil && state.rewardMessage != nil {
		var err error
		rewardWorld, err = g.prepareModernHDRewardWorld(state.rewardMessage, state.source, set)
		if err != nil {
			return err
		}
	}
	var healFrames *modernHDHealFrames
	publishCompoundMap, err := g.stageModernHDCompoundMap(g.modernCompoundMapFrames(), set)
	if err != nil {
		return err
	}
	if set != nil && g.nativeHealPresentation != nil {
		var err error
		healFrames, err = g.prepareModernHDHeal(g.nativeHealPresentation, set)
		if err != nil {
			return err
		}
	}
	var command1012Frames *modernHDCommand1012Frames
	var commandPreludeFrames []*modernHDMapFrame
	if set != nil && g.nativeCmd1012 != nil {
		var err error
		command1012Frames, err = g.prepareModernHDCommand1012(g.nativeCmd1012, set)
		if err != nil {
			return err
		}
		if command1012Frames != nil && g.native2189A != nil {
			commandPreludeFrames, err = g.prepareModernHDCommandPrelude(g.native2189A, set)
			if err != nil {
				return err
			}
		}
	}
	var player9Frames *modernHDPlayer9Frames
	if set != nil && g.nativeCmd9Player != nil {
		var err error
		player9Frames, err = g.prepareModernHDPlayer9Frames(g.nativeCmd9Player, set)
		if err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd24Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd24Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd29Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd29Presentation.hd); err != nil {
			return err
		}
	}
	var deathFrames []*modernHDMapFrame
	if hasModernHDDeathBank(set) && g.nativeCh28PostPresent != nil {
		var err error
		deathFrames, err = g.prepareModernHDDeathFrames(g.nativeCh28PostPresent, set)
		if err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd5Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd5Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd0Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd0Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd1Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd1Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd2Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd2Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd3Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd3Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd6Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd6Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd7Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd7Presentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd8Presentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd8Presentation.hd); err != nil {
			return err
		}
	}
	if resource := g.modernCompoundEffectResource(); set != nil && resource >= 0 {
		// 共用複合指令與指令9的owner各自預備，不共用資源索引。
		if err := set.prepareBattleHDBanks(g.modernCompoundFigureResources(), []int{resource}); err != nil {
			return err
		}
	}
	if set != nil && g.nativeCmd9AIPresentation != nil {
		if err := prepareModernHDCommandBanks(set, g.nativeCmd9AIPresentation.hd); err != nil {
			return err
		}
	}
	if set != nil && g.atk != nil && g.atk.nativeScene != nil {
		if err := set.validatePhysicalScenery(g.atk.nativeScene); err != nil {
			return err
		}
		if err := set.prepareBattleHDResources(g.atk.nativeScene.modernResources()...); err != nil {
			return err
		}
	}
	var portrait *modernStoryPortraitFrame
	if set != nil && len(g.dialog) != 0 {
		dl := g.dialog[len(g.dialog)-1]
		if dl.Speaker >= 0 || dl.NativeDialogue != nil {
			var ok bool
			portrait, ok = set.portraits[dl.Speaker]
			if !ok || portrait == nil {
				return fmt.Errorf("modern portrait for speaker %d is unavailable", dl.Speaker)
			}
		}
		if dl.NativeDialogue == nil {
			portrait = nil
		}
	}
	if g.modernStoryPortraits != nil {
		g.modernThemeCache = g.modernStoryPortraits
	}
	if set != nil {
		g.modernThemeCache = set
	}
	g.modernStoryPortraits = set
	publishItemPanels()
	publishSystemInfo()
	if g.hasModernHDStoryMapSource() {
		g.modernHDMapFrame = storyWorld
	}
	publishSkyKey()
	publishCh23()
	publishRowShake()
	publishUnitPresent()
	publishIndexedTransition()
	publishCompoundMap()
	publishExpiry()
	if p := g.nativeDeathRewardUI; p != nil && p.hd != nil {
		p.hd.world, p.hd.theme = fullRewardWorld, set
	}
	if state := g.nativeSystemEndTurnUI; state != nil && state.rewardMessage != nil {
		state.rewardMessage.hdWorld, state.rewardMessage.hdTheme = rewardWorld, set
	}
	if job := g.nativeHealPresentation; job != nil {
		job.hdFrames, job.hdTheme = healFrames, set
	}
	if job := g.nativeCmd1012; job != nil {
		job.hdFrames, job.hdTheme = command1012Frames, set
		if prelude := g.native2189A; prelude != nil {
			prelude.hdFrames, prelude.hdTheme = commandPreludeFrames, set
		}
	}
	if job := g.nativeCmd9Player; job != nil {
		job.hdFrames, job.hdTheme = player9Frames, set
	}
	if job := g.nativeCh28PostPresent; job != nil {
		job.hdFrames, job.hdTheme = deathFrames, set
	}
	g.nativeDialogueModernPortrait = portrait
	g.modernHDOverlay = nil
	g.refreshThemeVisuals()
	return nil
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
	}
	if style := g.interfaceAssetTheme(); style != nil && len(style.battleActionIcons) == 4 {
		for i, icon := range style.battleActionIcons {
			g.ringIcons[i] = ebiten.NewImageFromImage(icon)
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
		"F7：字級大小　F8：遊戲加速　F9：快速讀檔",
		"F11：原版／HD介面　Alt+Enter：全螢幕",
		"F12：開發除錯資訊",
		"戰場系統選單也可控制音樂、音效、速度與狀態欄。",
		"秘密商店：章節指定的修飾鍵＋F1..F10。",
	}
	for i, line := range lines {
		scale := 0.9
		y := float64(94 + (i-1)*25)
		if i == 0 {
			scale = 1.7
			y = 54
		}
		for g.font.Width(line, scale) > 524 {
			scale -= 0.025
		}
		g.font.Draw(screen, line, 58, y, scale, color.RGBA{238, 244, 255, 255})
	}
}
