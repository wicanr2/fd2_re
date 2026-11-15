package main

// player_input_pointer.go — 滑鼠指向換算成游標格或選單列。
//
// 只在 pointerTick.active（滑鼠這一刻移動或按下左鍵）時作用；換算後走的是方向鍵同一條路
// （moveMapCursor、選單的上下事件、指令環的方向閘門），所以原版的游標規則、鏡頭安全帶與
// 不可用方向的閘門都照舊。左鍵另外由 playerInput 轉成 Enter：同一刻先移到指向的列，再確認。
// 點到不能選的指令環方向時作廢這一次 Enter，避免確認到別的指令。

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// 原生地圖窗格：0x11EB0 把 312×192 搬到 VGA (4,4)，每格 24 原生像素；畫布是兩倍。
const (
	nativeViewportCanvasX = indexedmap.NativeMapViewportX0 * 2
	nativeViewportCanvasY = indexedmap.NativeMapViewportY0 * 2
	nativeCellCanvas      = 24 * 2
)

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// pointerMapCell 回傳指向的地圖格。原生地圖依窗格幾何；舊版地圖沿觸控點格的換算。
func (g *Game) pointerMapCell() (int, int, bool) {
	if !g.pointer.active || g.m == nil || g.m.W <= 0 || g.m.H <= 0 {
		return 0, 0, false
	}
	x, y := 0, 0
	if g.st != nil && g.st.HasNativeMapViewState {
		col := floorDiv(g.pointer.x-nativeViewportCanvasX, nativeCellCanvas)
		row := floorDiv(g.pointer.y-nativeViewportCanvasY, nativeCellCanvas)
		if col < 0 || row < 0 || col >= fdother.NativeMapViewportColumns || row >= fdother.NativeMapViewportRows {
			return 0, 0, false
		}
		view := g.st.NativeMapViewState
		x, y = view.CameraX+col, view.CameraY+row
	} else {
		if g.m.TileW <= 0 || g.m.TileH <= 0 {
			return 0, 0, false
		}
		x, y = (int(g.camX)+g.pointer.x)/g.m.TileW, (int(g.camY)+g.pointer.y)/g.m.TileH
	}
	if x < 0 || y < 0 || x >= g.m.W || y >= g.m.H {
		return 0, 0, false
	}
	return x, y, true
}

// followPointerOnMap 把地圖游標逐格移到指向的格子（先 X 後 Y，同觸控點格），每一步都走
// 鍵盤游標處理器。某一步被拒絕就停下，不會卡在迴圈裡。
func (g *Game) followPointerOnMap() {
	targetX, targetY, ok := g.pointerMapCell()
	if !ok {
		return
	}
	for steps := 0; steps < g.m.W+g.m.H+4 && (g.curX != targetX || g.curY != targetY); steps++ {
		dx, dy := 0, 0
		switch {
		case g.curX < targetX:
			dx = 1
		case g.curX > targetX:
			dx = -1
		case g.curY < targetY:
			dy = 1
		default:
			dy = -1
		}
		beforeX, beforeY := g.curX, g.curY
		g.moveMapCursor(dx, dy)
		if g.curX == beforeX && g.curY == beforeY {
			return
		}
	}
}

// pointerOverlayDirection 依指向點相對可見游標格中心的主軸，回傳指令環的方向
// （0 上、1 左、2 右、3 下，同 applyNativeOverlayDirectionInput）。四個圖格開在游標格的上下左右
// 一格（0x1741c 的末幀位移），所以指在游標格本身或兩格以外都不算。
func (g *Game) pointerOverlayDirection() (int, bool) {
	if !g.pointer.active || g.st == nil || !g.st.HasNativeMapViewState {
		return 0, false
	}
	view := g.st.NativeMapViewState
	centerX := nativeViewportCanvasX + view.VisibleCursorX*nativeCellCanvas + nativeCellCanvas/2
	centerY := nativeViewportCanvasY + view.VisibleCursorY*nativeCellCanvas + nativeCellCanvas/2
	dx, dy := g.pointer.x-centerX, g.pointer.y-centerY
	ax, ay := dx, dy
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	if max(ax, ay) < nativeCellCanvas/2 || max(ax, ay) > nativeCellCanvas*2 {
		return 0, false
	}
	if ay >= ax {
		if dy < 0 {
			return 0, true
		}
		return 3, true
	}
	if dx < 0 {
		return 1, true
	}
	return 2, true
}

// applyPointerToOverlay 在方向鍵之後套用指向的方向；不可用的方向照原版閘門不改選擇。
func (g *Game) applyPointerToOverlay(availability [4]int) {
	if direction, ok := g.pointerOverlayDirection(); ok && fdother.ActionOverlayAcceptsDirection(availability, direction) {
		g.ringSel = direction
	}
}

// pointerOverlayRejectsClick 在指令環讀 Enter 之前判斷：點在不可用的方向上，就作廢這一下。
func (g *Game) pointerOverlayRejectsClick() {
	// 只在指令環本身等方向輸入時判斷；物品欄、指令格、法術表與 END 確認各自畫在別處。
	if !g.pointer.clicked || !g.ring || g.itemOpen || g.nativeCommandOpen || g.spellOpen ||
		g.nativeSystemEndTurnConfirm || g.nativeSystemInfoUI != nil {
		return
	}
	direction, ok := g.pointerOverlayDirection()
	if !ok {
		return
	}
	if !fdother.ActionOverlayAcceptsDirection(g.nativeActionOverlayState().Availability, direction) && g.input != nil {
		g.input.suppressVirtual(ebiten.KeyEnter)
	}
}

// titleMenuRects 是標題選單三項的畫布矩形（drawTitle 的 menu 分支同一組座標）。
func (g *Game) titleMenuRects() []image.Rectangle {
	ta := g.titleAssets
	if ta == nil {
		return nil
	}
	rects := make([]image.Rectangle, 3)
	for i := range rects {
		it := ta.items[i][0]
		if it == nil {
			return nil
		}
		b := it.Bounds()
		x, y := (320-b.Dx())/2*2, titleMenuY(i)*2
		rects[i] = image.Rect(x, y, x+b.Dx()*2, y+titleMenuGapY*2)
	}
	return rects
}

// applyPointerToTitleMenu 走 TitleMenuUp／Down 到指向的項目；確認閃爍中不動。
func (g *Game) applyPointerToTitleMenu() {
	if !g.pointer.active || g.titleFlash != 0 {
		return
	}
	target := pointerIn(g.titleMenuRects(), g.pointer.x, g.pointer.y)
	for steps := 0; target >= 0 && g.titleSel != target && steps < 3; steps++ {
		event := TitleMenuDown
		if (g.titleSel+2)%3 == target {
			event = TitleMenuUp
		}
		g.applyTitleMenuEvent(event)
	}
}

// 讀檔四格的列位置沿 campaign.ComposeNativeLoadSlotsFrame：文字列 y=119+19×列、框寬 5..315。
func titleSlotRects() []image.Rectangle {
	rects := make([]image.Rectangle, 4)
	for row := range rects {
		y := 119 + row*19
		rects[row] = image.Rect(5*2, (y-2)*2, 315*2, (y+17)*2)
	}
	return rects
}

func (g *Game) applyPointerToTitleSlots() {
	if !g.pointer.active {
		return
	}
	target := pointerIn(titleSlotRects(), g.pointer.x, g.pointer.y)
	for steps := 0; target >= 0 && g.titleSlotSel != target && steps < 4; steps++ {
		event := TitleSlotDown
		if target < g.titleSlotSel {
			event = TitleSlotUp
		}
		g.applyTitleSlotEvent(event)
	}
}

// nativeTownSelectionRects 是五個設施游標人物的位置（每格 24 原生像素，四周各放寬 4）。
func nativeTownSelectionRects(variant int) []image.Rectangle {
	rects := make([]image.Rectangle, 5)
	for selection := range rects {
		geometry, err := campaign.NativeTownCursorGeometryFor(variant, selection, 0)
		if err != nil {
			return nil
		}
		p := geometry.Position
		rects[selection] = image.Rect((p.X-4)*2, (p.Y-4)*2, (p.X+28)*2, (p.Y+28)*2)
	}
	return rects
}

// applyPointerToNativeTown 沿 0x2cf01..0x2cf65 的左右循環走到指向的設施，取步數少的方向。
func (g *Game) applyPointerToNativeTown(variant int) {
	if !g.pointer.active || g.campSel < 0 || g.campSel > 4 {
		return
	}
	target := pointerIn(nativeTownSelectionRects(variant), g.pointer.x, g.pointer.y)
	if target < 0 || target == g.campSel {
		return
	}
	left := (target - g.campSel + 5) % 5 // moveNativeTownSelection(1) 的步數
	delta := 1
	if 5-left < left {
		delta = -1
	}
	for steps := 0; g.campSel != target && steps < 5; steps++ {
		if !g.moveNativeTownSelection(delta) {
			return
		}
	}
}

// choiceMenuRects 是重製端選項框的列（drawCampaign 的 choice 分支：x 160..480、第 i 列 y=162+28i）。
func choiceMenuRects(count int) []image.Rectangle {
	rects := make([]image.Rectangle, count)
	for i := range rects {
		rects[i] = image.Rect(160, 158+i*28, 480, 186+i*28)
	}
	return rects
}

func (g *Game) applyPointerToChoiceMenu() {
	if !g.pointer.active || g.camp == nil {
		return
	}
	count := len(g.camp.Visible())
	target := pointerIn(choiceMenuRects(count), g.pointer.x, g.pointer.y)
	for steps := 0; target >= 0 && g.campSel != target && steps < count; steps++ {
		event := campaign.MenuDown
		if target < g.campSel {
			event = campaign.MenuUp
		}
		g.stepCampaignMenu(event)
	}
}
