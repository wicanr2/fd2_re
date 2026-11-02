package main

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
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
	var collected []*modernBattlePanelText
	var collector *[]*modernBattlePanelText
	if g.font != nil && g.font.sf != nil && g.localeEntities != nil {
		collector = &collected
	}
	if err := g.renderBattlePanelText(assets, record, pixels, 0, 0, collector); err != nil {
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
	sourceImage := ebiten.NewImageFromImage(panel)
	panelX := 9
	if g.st.NativeMapViewState.VisibleCursorX < 7 {
		panelX = 160
	}
	hd, err := g.freezeModernHDMovementPanel(u, plan, sourceImage, record, panel.Pix, panelX, collected)
	if err != nil {
		return err
	}
	if g.modernHDInterfaceEnabled() && hd == nil {
		return errors.New("modern HD movement panel: verified name/font unavailable")
	}
	g.nativeMovePanel, g.nativeMovePanelX, g.movementPanelHD = sourceImage, panelX, hd
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
	g.movementPanelHD, g.movementPanelDraw = nil, nil
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
	g.movementPanelDraw = nil
	if g.sel == nil || g.nativeMovePanel == nil {
		return
	}
	rect := image.Rect(g.nativeMovePanelX*2, 18, g.nativeMovePanelX*2+298, 102)
	if g.modernHDInterfaceEnabled() {
		if hd, err := g.prepareModernHDMovementPanel(); err == nil {
			g.movementPanelDraw = &modernHDMovementPanelDraw{image: hd.gpu, rect: rect, hd: true, artHD: g.modernStoryPortraits != nil, key: hd.pictureKey}
			return
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(g.nativeMovePanelX*2), 18)
	screen.DrawImage(g.nativeMovePanel, op)
	g.movementPanelDraw = &modernHDMovementPanelDraw{rect: rect, artHD: g.modernStoryPortraits != nil}

}

// beginPlayerAttackTargeting 是指令環攻擊收合後的共同入口（0x18f22 分支）：
// 標記射程、提示選目標。重播測試也走這裡，不另抄一份。
func (g *Game) beginPlayerAttackTargeting() {
	g.markNativePlayerAttackField()
	if message, ok := g.localeMessage("battle.attack.choose_target"); ok {
		g.msg = message
	}
}

// 0x18F76以targetCode0進0x115B6。未通過確認仍留在目標選擇，不能提交待機。
func (g *Game) nativePhysicalAttackConfirmationAllowed() bool {
	if g == nil || g.st == nil || g.sel == nil ||
		g.curX < 0 || g.curX >= g.st.W || g.curY < 0 || g.curY >= g.st.H ||
		len(g.st.NativeTileBlitModes) != g.st.W*g.st.H {
		return false
	}
	allowed, err := battle.NativeCursorConfirmationAllowed(
		battle.Cell{X: g.curX, Y: g.curY},
		g.st.NativeTileBlitModes[g.curY*g.st.W+g.curX],
		g.st.NativeMapRangeMode, 0, g.st.Units,
	)
	if err != nil {
		g.loadErr = err.Error()
	}
	return err == nil && allowed
}

// Update與重播共用既有Escape owner；0x115B6回-1後，0x18F86清掉射程。
func (g *Game) returnPlayerAttackTargetToRing() bool {
	if g == nil || g.st == nil || g.sel == nil || !g.moved {
		return false
	}
	g.resetNativeTargetField()
	if fdother.ActionOverlayAcceptsDirection(g.actionOverlayAvailability(), g.ringSel) {
		g.beginActionOverlayOpen(g.ringSel)
	} else {
		g.beginBattleActionOverlay()
	}
	g.msg = ""
	return true
}

// markNativePlayerAttackField 在指令環選攻擊收合後，把 0x18f6a 那次 0x14818 的
// 標記寫進 NativeTileBlitModes；選目標期間整幀就會照原版染出武器射程。舊版
// JSON 戰場沒有原版物品表時維持不標記。
func (g *Game) markNativePlayerAttackField() {
	if g == nil || g.st == nil || g.sel == nil ||
		len(g.st.NativeTileBlitModes) != g.st.W*g.st.H {
		return
	}
	field, err := g.st.NativePlayerAttackTargetField(g.sel)
	if err != nil {
		return
	}
	copy(g.st.NativeTileBlitModes, field)
}
