package main

import (
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	playerInterfaceOriginal = "original"
	playerInterfaceHD       = "hd"
)

// Empty mode retains compatibility with isolated legacy render fixtures. A
// normal game resolves it once at startup, then F2 and F11 stay independent.
func (g *Game) modernHDInterfaceEnabled() bool {
	if g == nil {
		return false
	}
	switch g.interfaceMode {
	case playerInterfaceOriginal:
		return false
	case playerInterfaceHD:
		return true
	default:
		return g.modernStoryPortraits != nil && g.modernStoryPortraits.hasHD
	}
}

func (g *Game) hdPresentationEnabled() bool {
	return g.modernHDInterfaceEnabled() || (g.modernStoryPortraits != nil && g.modernStoryPortraits.hasHD)
}

func (g *Game) interfaceThemeForArt(art *modernStoryPortraitSet) *modernStoryPortraitSet {
	if g.interfaceMode == playerInterfaceOriginal {
		return nil
	}
	if art != nil {
		return art
	}
	if g.interfaceMode == playerInterfaceHD {
		if g.modernThemeCache != nil {
			return g.modernThemeCache
		}
		return g.modernStoryPortraits
	}
	return nil
}

func (g *Game) interfaceAssetTheme() *modernStoryPortraitSet {
	return g.interfaceThemeForArt(g.modernStoryPortraits)
}

func loadPlayerModernTheme() (*modernStoryPortraitSet, error) {
	root := os.Getenv("FD2_MODERN_THEME_PACK")
	if root == "" {
		root = assetPath("assets/themes/modern")
	}
	return loadModernStoryPortraitSet(assetPath("assets/themes/modern/catalog.json"), root)
}

func (g *Game) setInterfaceMode(mode string) error {
	if mode != playerInterfaceOriginal && mode != playerInterfaceHD {
		return fmt.Errorf("interface: unsupported mode %q", mode)
	}
	candidate := *g
	candidate.interfaceMode = mode
	candidate.modernHDMapHUDArt = nil
	if mode == playerInterfaceHD {
		if g.font == nil || g.font.sf == nil {
			return fmt.Errorf("interface: HD font unavailable")
		}
		if candidate.interfaceAssetTheme() == nil {
			set, err := loadPlayerModernTheme()
			if err != nil {
				return err
			}
			candidate.modernThemeCache = set
		}
	}
	style := candidate.interfaceAssetTheme()
	var world *modernHDMapFrame
	if g.nativeMapFrameSource != nil && len(g.nativeMapDAC) == 256*3 {
		palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			return err
		}
		if g.modernStoryPortraits != nil && g.modernHDMapFrame != nil {
			world, err = candidate.prepareModernHDMapFrame(*g.nativeMapFrameSource, g.modernHDMapFrame.TerrainSeed, palette)
		} else if mode == playerInterfaceHD && g.modernStoryPortraits == nil {
			candidate.nativeInterfaceHUD, err = candidate.prepareModernHDMapHUDPanel(*g.nativeMapFrameSource, palette)
		}
		if err != nil {
			return err
		}
	}
	if g.baseMapTileset != nil && g.m != nil {
		// Reuse F2's complete admission for frozen owners, without changing
		// the selected artwork. No clock or battle transaction is advanced.
		if err := candidate.applyTheme(g.modernStoryPortraits); err != nil {
			return err
		}
		if world != nil {
			candidate.modernHDMapFrame = world
		}
		candidate.modernHDText = nil
		*g = candidate
		return nil
	}
	confirmation, err := candidate.stageModernHDSystemConfirmation(g.modernStoryPortraits)
	if err != nil {
		return err
	}
	panels, err := candidate.stageModernHDItemPanels(style)
	if err != nil {
		return err
	}
	info, err := candidate.stageModernHDSystemInfo(style)
	if err != nil {
		return err
	}
	panels()
	confirmation()
	info()
	candidate.modernHDText = nil
	*g = candidate
	return nil
}

func (g *Game) cycleInterfaceMode() {
	mode := playerInterfaceHD
	if g.modernHDInterfaceEnabled() {
		mode = playerInterfaceOriginal
	}
	if err := g.setInterfaceMode(mode); err != nil {
		g.msg = "介面切換失敗：" + err.Error()
		return
	}
	g.saveDisplaySettings()
	g.msg = "介面：" + map[string]string{playerInterfaceOriginal: "原版", playerInterfaceHD: "HD"}[mode]
}

func (g *Game) saveDisplaySettings() {
	s := loadSettings()
	s.InterfaceMode = g.interfaceMode
	s.ThemeID = "faithful"
	if g.modernStoryPortraits != nil {
		s.ThemeID = modernHandpaintedThemeID
	}
	s.FontScale, s.LocaleID, s.BGMSource = g.fontScale, g.localeID, g.bgmSource
	saveSettings(s)
}
