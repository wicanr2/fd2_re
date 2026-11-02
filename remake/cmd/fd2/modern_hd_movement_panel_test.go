package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func movementPanelTestGame(t *testing.T, x int) (*Game, *battle.Unit) {
	t.Helper()
	source := os.Getenv("FD2_HD_MOVEMENT_PANEL_SAVE")
	if source == "" {
		t.Skip("complete normal movement save required")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("FD2_NATIVE_SAVE", "")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_THEME", "faithful")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	if err := os.MkdirAll(filepath.Dir(saveSlotPath(0)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(saveSlotPath(0), raw, 0644); err != nil {
		t.Fatal(err)
	}
	g := loadGame()
	if g.startupBlocked || g.loadErr != "" || !g.loadJSONGameFromSlot(0, false) || g.st == nil {
		t.Fatal("normal save admission", g.loadErr, g.msg)
	}
	if g.camp.NodeID() != "battle_ch01" {
		t.Fatal("movement fixture requires complete first-chapter source")
	}
	fixedNow := time.Unix(1000, 0)
	g.nativeMapFrozenNow = func() time.Time { return fixedNow }
	g.nativeMapClock.Reset()
	g.interfaceMode = playerInterfaceOriginal
	actor := g.st.Units[0]
	g.sel = actor
	view := g.st.NativeMapViewState
	visible := 4
	if x == 9 {
		visible = 7
	}
	view.CameraX = view.CursorX - visible
	view.VisibleCursorX = visible
	if err := g.st.MaterializeNativeMapViewState(view); err != nil {
		t.Fatal(err)
	}
	if err := g.prepareNativePlayerMovement(actor); err != nil {
		t.Fatal(err)
	}
	if g.nativeMovePanelX != x || g.movementPanelHD == nil {
		t.Fatal("actual movement constructor source missing", g.nativeMovePanelX, x)
	}
	return g, actor
}
func movementPanelGameplay(g *Game) []byte {
	raw, err := json.Marshal([]any{unitPresentGPUState(g), g.gold, g.nativeRNGState, g.st.NativeTileBlitModes})
	if err != nil {
		panic(err)
	}
	return raw
}
func TestModernHDMovementPanelSources(t *testing.T) {
	count := 0
	for _, x := range []int{9, 160} {
		g, _ := movementPanelTestGame(t, x)
		h := g.movementPanelHD
		before := movementPanelGameplay(g)
		plan := g.nativeMovePlan
		source := g.nativeMovePanel
		seal := h.seal
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				g.modernStoryPortraits = nil
				if art {
					g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
				}
				g.interfaceMode = playerInterfaceOriginal
				if ui {
					g.interfaceMode = playerInterfaceHD
				}
				g.drawNativeMovementPanel(ebiten.NewImage(640, 400))
				d := g.movementPanelDraw
				if d == nil || d.hd != ui || d.artHD != art || d.rect != image.Rect(x*2, 18, x*2+298, 102) {
					t.Fatal("position or independent mode", x, art, ui, d)
				}
				if ui && (d.image == nil || d.image.Bounds() != image.Rect(0, 0, 894, 252) || d.key == "") {
					t.Fatal("missing high-resolution panel")
				}
				if g.nativeMovePlan != plan || g.nativeMovePanel != source || h.seal != seal || !bytes.Equal(before, movementPanelGameplay(g)) {
					t.Fatal("display changed native movement")
				}
				count++
			}
		}
		g.applyFontScale(1)
		first, err := g.prepareModernHDMovementPanel()
		if err != nil {
			t.Fatal(err)
		}
		oldKey := first.pictureKey
		g.applyFontScale(1.25)
		next, err := g.prepareModernHDMovementPanel()
		if err != nil {
			t.Fatal(err)
		}
		if next.pictureKey == oldKey || next.plan != plan || !bytes.Equal(before, movementPanelGameplay(g)) {
			t.Fatal("font setting failed to rebuild only image")
		}
		g.nativeMovePanel = nil
		g.drawNativeMovementPanel(ebiten.NewImage(640, 400))
		if g.movementPanelDraw != nil {
			t.Fatal("hidden panel retained last draw")
		}
	}
	t.Logf("actual movement admission; %d position/style states; font rebuild and hidden cleanup kept gameplay", count)
}
func TestModernHDMovementPanelRejectsChangedSource(t *testing.T) {
	base, actor := movementPanelTestGame(t, 160)
	if _, err := base.prepareModernHDMovementPanel(); err != nil {
		t.Fatal(err)
	}
	original := base.movementPanelHD
	faults := []struct {
		name string
		edit func(*Game, *modernHDMovementPanel)
	}{
		{"missing", func(g *Game, h *modernHDMovementPanel) { g.movementPanelHD = nil }},
		{"actor", func(g *Game, h *modernHDMovementPanel) { h.actor = &battle.Unit{} }},
		{"plan", func(g *Game, h *modernHDMovementPanel) { h.plan = &battle.NativePlayerMovement{} }},
		{"source_image", func(g *Game, h *modernHDMovementPanel) { g.nativeMovePanel = ebiten.NewImage(149, 42) }},
		{"anchor", func(g *Game, h *modernHDMovementPanel) { g.nativeMovePanelX = 9 }},
		{"record", func(g *Game, h *modernHDMovementPanel) { h.record[8] ^= 1 }},
		{"raw", func(g *Game, h *modernHDMovementPanel) { h.raw[0] ^= 1 }},
		{"native_pixels", func(g *Game, h *modernHDMovementPanel) { h.recipe.native.Pixels[0] ^= 1 }},
		{"native_mask", func(g *Game, h *modernHDMovementPanel) { h.recipe.native.Mask[0] ^= 255 }},
		{"name", func(g *Game, h *modernHDMovementPanel) { h.recipe.name = "changed" }},
		{"values", func(g *Game, h *modernHDMovementPanel) { h.recipe.values.HP++ }},
		{"chrome", func(g *Game, h *modernHDMovementPanel) { h.recipe.chrome.border ^= 1 }},
		{"font", func(g *Game, h *modernHDMovementPanel) { h.primary = loadFont() }},
		{"font_source", func(g *Game, h *modernHDMovementPanel) { h.fontSF = nil }},
		{"palette", func(g *Game, h *modernHDMovementPanel) {
			g.nativeUIPalette = append(color.Palette(nil), g.nativeUIPalette...)
			g.nativeUIPalette[0] = color.RGBA{R: 1, A: 255}
		}},
		{"raster", func(g *Game, h *modernHDMovementPanel) { h.recipe.picture.Pix[0] ^= 1 }},
		{"raster_key", func(g *Game, h *modernHDMovementPanel) { h.recipe.key = "changed" }},
		{"gpu", func(g *Game, h *modernHDMovementPanel) { h.gpu = ebiten.NewImage(894, 252) }},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			g := *base
			h := *original
			recipe := *h.recipe
			h.record = append([]byte(nil), h.record...)
			h.raw = append([]byte(nil), h.raw...)
			h.palette = append([]byte(nil), h.palette...)
			recipe.native.Pixels = append([]byte(nil), recipe.native.Pixels...)
			recipe.native.Mask = append([]byte(nil), recipe.native.Mask...)
			picture := image.NewRGBA(recipe.picture.Bounds())
			copy(picture.Pix, recipe.picture.Pix)
			recipe.picture = picture
			h.recipe = &recipe
			g.movementPanelHD = &h
			fault.edit(&g, &h)
			g.interfaceMode = playerInterfaceOriginal
			before := movementPanelGameplay(&g)
			plan := g.nativeMovePlan
			source := g.nativeMovePanel
			if g.setInterfaceMode(playerInterfaceHD) == nil || g.interfaceMode != playerInterfaceOriginal {
				t.Fatal("changed source switched interface")
			}
			if g.sel != actor || g.nativeMovePlan != plan || g.nativeMovePanel != source || !bytes.Equal(before, movementPanelGameplay(&g)) {
				t.Fatal("rejection changed movement")
			}
			g.interfaceMode = playerInterfaceHD
			g.drawNativeMovementPanel(ebiten.NewImage(640, 400))
			if g.movementPanelDraw == nil || g.movementPanelDraw.hd || g.movementPanelDraw.image != nil {
				t.Fatal("changed source retained HD instead of native fallback")
			}
		})
	}
	t.Log("18 source/owner/cache faults rejected before F11 publication; native fallback kept")
}
func TestModernHDMovementPanelLegacyNativeAndClear(t *testing.T) {
	g, actor := movementPanelTestGame(t, 9)
	g.clearNativePlayerMovement()
	g.sel = actor
	g.font = nil
	g.localeEntities = nil
	if err := g.prepareNativePlayerMovement(actor); err != nil {
		t.Fatal("legacy native source blocked", err)
	}
	if g.movementPanelHD != nil || g.nativeMovePanel == nil || g.nativeMovePlan == nil {
		t.Fatal("legacy native owner differs")
	}
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
	g.drawNativeMovementPanel(ebiten.NewImage(640, 400))
	if g.movementPanelDraw == nil || g.movementPanelDraw.hd {
		t.Fatal("original UI did not draw over modern art")
	}
	g.clearNativePlayerMovement()
	if g.nativeMovePlan != nil || g.nativeMovePanel != nil || g.movementPanelHD != nil || g.movementPanelDraw != nil {
		t.Fatal("clear retained source or last draw")
	}
}
