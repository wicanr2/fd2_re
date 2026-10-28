package main

import (
	"bytes"
	"image"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestIndependentInterfaceFourModesKeepArtworkAndBattle(t *testing.T) {
	for _, art := range []bool{false, true} {
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			state := &battle.State{Turn: 7}
			g := &Game{st: state, font: &Font{}, interfaceMode: mode, curX: 12, curY: 8}
			if art {
				g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
			}
			selected := g.modernStoryPortraits
			finish := g.beginModernHDText(nil)
			if (g.modernHDText != nil) != (mode == playerInterfaceHD) {
				t.Fatalf("art=%v UI=%s: text followed artwork", art, mode)
			}
			finish()
			if g.modernStoryPortraits != selected || g.st != state || state.Turn != 7 || g.curX != 12 || g.curY != 8 {
				t.Fatal("interface changed artwork, state or cursor")
			}
			if g.hdPresentationEnabled() != (art || mode == playerInterfaceHD) {
				t.Fatal("mixed display mode lost HD resolution")
			}
		}
	}
}

func TestInterfaceRejectsBeforeChangingModeOrArtwork(t *testing.T) {
	art := &modernStoryPortraitSet{hasHD: true}
	g := &Game{interfaceMode: playerInterfaceOriginal, modernStoryPortraits: art}
	for _, mode := range []string{"unknown", playerInterfaceHD} {
		if err := g.setInterfaceMode(mode); err == nil {
			t.Fatal("bad mode or missing font accepted")
		}
		if g.interfaceMode != playerInterfaceOriginal || g.modernStoryPortraits != art || g.modernThemeCache != nil {
			t.Fatal("rejected interface switch published state")
		}
	}
}

func TestOriginalInterfaceKeepsHDPanelPortrait(t *testing.T) {
	picture := &modernStoryPortraitFrame{hdImage: image.NewRGBA(image.Rect(0, 0, 480, 480))}
	g := &Game{interfaceMode: playerInterfaceOriginal, modernStoryPortraits: &modernStoryPortraitSet{
		hasHD: true, portraits: map[int]*modernStoryPortraitFrame{9: picture}}}
	record := make([]byte, 80)
	record[7] = 9
	pass, _ := battle.NativeItemPanelFrameFor(0)
	g.addModernHDPanelPortrait(record, []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom})
	if len(g.modernHDPanelPortraits) != 1 || g.modernHDPanelPortraits[0].portrait != picture || g.modernHDText != nil {
		t.Fatal("original UI reverted or suppressed the HD character portrait")
	}
}

func TestInterfaceSettingsSurviveLocaleAndAudio(t *testing.T) {
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = oldCache })
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSettings(settings{BGMSource: "fm", LocaleID: "zh-Hant", FontScale: 1.25,
		InterfaceMode: playerInterfaceOriginal, ThemeID: modernHandpaintedThemeID})
	g := &Game{bgmSource: "fm", localeID: "zh-Hant", fontScale: 1.25}
	attachOfficialLocale(t, g)
	g.cycleBGMSource()
	g.cycleLocale()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	s := loadSettings()
	if s.InterfaceMode != playerInterfaceOriginal || s.ThemeID != modernHandpaintedThemeID || s.FontScale != 1.25 || s.BGMSource != "mt32" || s.LocaleID != "zh-Hans" {
		t.Fatalf("another preference erased display controls: %+v", s)
	}
	if err := os.WriteFile(settingsPath(), []byte(`{"bgm_source":"fm","locale_id":"en"}`), 0644); err != nil {
		t.Fatal(err)
	}
	s = loadSettings()
	if s.InterfaceMode != "" || s.ThemeID != "" || s.FontScale != 1 {
		t.Fatal("legacy defaults changed", s)
	}
}

func TestInterfaceBattlePanelSelectionKeepsNativePixels(t *testing.T) {
	g, actor, _ := physicalSceneTestGame(t)
	attachOfficialLocale(t, g)
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto font required")
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	scene := &nativePhysicalScene{panelAssets: assets}
	record, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 64000)
	if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil {
		t.Fatal(err)
	}
	sources := &modernHDBattleSources{Base: base}
	bindModernPhysicalPanelText(scene, sources)
	before := append([]byte(nil), base...)
	beforeRecord := append([]byte(nil), record...)
	for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD, playerInterfaceOriginal} {
		g.interfaceMode = mode
		plan, err := g.modernHDBattlePlan(sources, g.nativeUIPalette, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Layers) != 1 || (plan.Layers[0].HD != nil) != (mode == playerInterfaceHD) {
			t.Fatal("panel style ignored interface mode")
		}
		if !bytes.Equal(base, before) || !bytes.Equal(record, beforeRecord) || g.modernStoryPortraits != nil {
			t.Fatal("UI style changed native source or artwork")
		}
	}
}

func TestInterfaceSystemInfoWithOriginalArtwork(t *testing.T) {
	real, _ := systemInfoHDTestSource(t)
	for _, art := range []bool{false, true} {
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			g := nativeSystemInfoRuntimeTestGame(t)
			g.nativeSystemInfoAssets, g.nativeBattleGlyphs, g.font = real.nativeSystemInfoAssets, real.nativeBattleGlyphs, real.font
			attachOfficialLocale(t, g)
			g.interfaceMode = mode
			if art {
				g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
			}
			before := append([]byte(nil), g.nativeMapVGA...)
			s, err := g.prepareNativeSystemInfoUI()
			if err != nil {
				t.Fatal(art, mode, err)
			}
			if (s.hd != nil) != (mode == playerInterfaceHD) || !bytes.Equal(before, g.nativeMapVGA) || s.phase != "opening" {
				t.Fatal("information interface followed artwork or changed native state", art, mode)
			}
		}
	}
}

func TestInterfaceStatusWithOriginalArtwork(t *testing.T) {
	g, screen, unit := newModernHDStatusGame(t)
	defer prepareHDChurchText(t, g, screen)()
	g.modernStoryPortraits, g.interfaceMode = nil, playerInterfaceHD
	data, status, commands, ok := g.prepareNativeUnitStatusData(unit)
	if !ok {
		t.Fatal("status source unavailable")
	}
	s := &nativePlayerStatusState{data: data, status: status, commands: commands, phase: "status"}
	before := append([]byte(nil), status...)
	if !g.drawModernBattlePlayerStatus(screen, s) || len(g.modernHDText.runs) == 0 || !bytes.Equal(before, status) || g.modernStoryPortraits != nil || s.drawn {
		t.Fatal("HD interface required HD artwork or changed native status")
	}
}
