package main

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"image"
)

func (g *Game) prepareNativePlayerMovement(u *battle.Unit) error {
	if g == nil || g.st == nil || len(g.nativeUIPalette) != 256 ||
		len(g.st.NativeTileBlitModes) != g.st.W*g.st.H {
		return errors.New("native player movement: renderer provenance unavailable")
	}
	plan, err := g.st.PlanNativePlayerMovement(u)
	if err != nil {
		return err
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	record, err := battle.NativeBattlePanelRecordForUnit(u)
	if err != nil {
		return err
	}
	pixels := make([]byte, 320*200)
	if err := g.renderLocalizedNativeBattlePanel(assets, record, pixels, 0, 0); err != nil {
		return err
	}
	x, y, err := battle.NativeBattlePanelOrigin(record, 0, 0)
	if err != nil {
		return err
	}
	panel := image.NewPaletted(image.Rect(0, 0, 149, 42), g.nativeUIPalette)
	for row := 0; row < 42; row++ {
		copy(panel.Pix[row*panel.Stride:row*panel.Stride+149], pixels[(y+row)*320+x:(y+row)*320+x+149])
	}
	g.nativeMovePanel = ebiten.NewImageFromImage(panel)
	g.nativeMovePanelX = 9
	if g.st.NativeMapViewState.VisibleCursorX < 7 {
		g.nativeMovePanelX = 160
	}
	g.nativeMovePlan = plan
	g.reach = plan.Reach
	copy(g.st.NativeTileBlitModes, plan.Field)
	return nil
}

func (g *Game) clearNativePlayerMovement() {
	if g.nativeMovePlan != nil {
		g.resetNativeTargetField()
	}
	g.nativeMovePlan, g.nativeMovePanel = nil, nil
}

func (g *Game) restorePlayerMovementCursor() {
	if g.st != nil && g.st.HasNativeMapViewState {
		for g.st.NativeMapViewState.CursorX != g.selOrigX {
			delta := 1
			if g.st.NativeMapViewState.CursorX > g.selOrigX {
				delta = -1
			}
			moved, ok := g.st.MoveNativeMapCursor(delta, 0)
			if !ok || !moved {
				break
			}
		}
		for g.st.NativeMapViewState.CursorY != g.selOrigY {
			delta := 1
			if g.st.NativeMapViewState.CursorY > g.selOrigY {
				delta = -1
			}
			moved, ok := g.st.MoveNativeMapCursor(0, delta)
			if !ok || !moved {
				break
			}
		}
		g.syncNativeMapView()
		return
	}
	g.curX, g.curY = g.selOrigX, g.selOrigY
}

func (g *Game) drawNativeMovementPanel(screen *ebiten.Image) {
	if g.sel == nil || g.nativeMovePanel == nil || g.modernStoryPortraits != nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(g.nativeMovePanelX*2), 18)
	screen.DrawImage(g.nativeMovePanel, op)
}
