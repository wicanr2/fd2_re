package main

import "github.com/wicanr2/fd2_re/remake/internal/fdother"

const nativeSystemEndTurnDelayFrames = 12 // 0x17259: delay(0xC8 ms)；60 Hz 約十二幀。

const (
	actionOverlayOpening = "opening"
	actionOverlayOpen    = "open"
	actionOverlayClosing = "closing"
)

// beginActionOverlayOpen starts the four presents recovered at 0x1741c.
// There is no delay call between native presents, so the remake assigns one
// presented Ebiten frame to each step without claiming an original duration.
func (g *Game) beginActionOverlayOpen(selection int) {
	g.ring = true
	g.ringSel = selection
	g.actionOverlayPhase = actionOverlayOpening
	g.actionOverlayFrame = 0
	g.actionOverlayAfter = nil
	g.actionOverlayDrawn = false
	g.actionOverlayShotHold = false
}

// beginActionOverlayClose starts the independent four-present sequence from
// 0x176b4. The selected action is deferred until all four close frames have
// been presented; it must not appear beneath an overlay that native code was
// still closing.
func (g *Game) beginActionOverlayClose(after func()) {
	if !g.ring {
		if after != nil {
			after()
		}
		return
	}
	g.actionOverlayPhase = actionOverlayClosing
	g.actionOverlayFrame = 0
	g.actionOverlayAfter = after
	g.actionOverlayDrawn = false
	g.actionOverlayShotHold = false
}

func (g *Game) actionOverlayBlocksInput() bool {
	return g.actionOverlayPhase == actionOverlayOpening ||
		g.actionOverlayPhase == actionOverlayClosing
}

func (g *Game) actionOverlayRenderState() (frame int, closing bool) {
	switch g.actionOverlayPhase {
	case actionOverlayOpening:
		return g.actionOverlayFrame, false
	case actionOverlayClosing:
		return g.actionOverlayFrame, true
	default:
		return 3, false
	}
}

// stepActionOverlayLifecycle runs once near the start of Update. A sequence is
// initialized later in an input Update, so frame zero is drawn before the next
// call advances it. The callback similarly runs only after close frame three
// was available to Draw for a complete update interval.
func (g *Game) stepActionOverlayLifecycle() {
	if g.actionOverlayShotHold {
		return
	}
	if g.actionOverlayBlocksInput() && !g.actionOverlayDrawn {
		return
	}
	switch g.actionOverlayPhase {
	case actionOverlayOpening:
		if g.actionOverlayFrame < 3 {
			g.actionOverlayFrame++
			g.actionOverlayDrawn = false
			return
		}
		g.actionOverlayPhase = actionOverlayOpen
		g.actionOverlayDrawn = false
	case actionOverlayClosing:
		if g.actionOverlayFrame < 3 {
			g.actionOverlayFrame++
			g.actionOverlayDrawn = false
			return
		}
		after := g.actionOverlayAfter
		g.actionOverlayPhase = ""
		g.actionOverlayFrame = 0
		g.actionOverlayAfter = nil
		g.actionOverlayDrawn = false
		g.ring = false
		if after != nil {
			after()
		}
	}
}

func (g *Game) markActionOverlayDrawn() {
	if g.actionOverlayBlocksInput() {
		g.actionOverlayDrawn = true
	}
}

func (g *Game) resetActionOverlayLifecycle() {
	g.ring = false
	g.nativeSystemCursorOverlay = false
	g.nativeSystemEndTurnConfirm = false
	g.nativeSystemEndTurnDelay = 0
	g.actionOverlayPhase = ""
	g.actionOverlayFrame = 0
	g.actionOverlayAfter = nil
	g.actionOverlayDrawn = false
	g.actionOverlayShotHold = false
}

// nativeSystemOverlayReady prevents the shared empty-cursor command from
// becoming an invisible hot zone when FDOTHER #2 is absent or incomplete.
// Only the four exact 0x16F55 cells are required; their action ownership is
// still checked separately at confirm time.
func (g *Game) nativeSystemOverlayReady() bool {
	if g == nil || g.st == nil || g.m == nil || g.aiBusy || g.result != "" {
		return false
	}
	state := fdother.NativeContinueActionOverlayState()
	for direction := 0; direction < 4; direction++ {
		index, err := state.CellIndex(direction)
		if err != nil || index < 0 || index >= len(g.nativeActionCells) || g.nativeActionCells[index] == nil {
			return false
		}
	}
	return true
}

// beginNativeSystemEndTurn 承接共用 0x117E7→0x16F55 的 Down→END。
// 原版直接指令證實 END 會開確認且 YES 會進 0x1A30B；其餘三格 owner
// 與重製端確認提示像素仍未閉合。
func (g *Game) beginNativeSystemEndTurn() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.ring || g.ringSel != 3 ||
		g.st == nil || g.aiBusy || g.result != "" {
		return false
	}
	g.beginActionOverlayClose(func() {
		g.nativeSystemCursorOverlay = false
		g.nativeSystemEndTurnConfirm = true
		g.msg = "要結束本回合的行動嗎？"
	})
	return true
}

func (g *Game) confirmNativeSystemEndTurn() {
	if g == nil || !g.nativeSystemEndTurnConfirm {
		return
	}
	g.nativeSystemEndTurnConfirm = false
	g.nativeSystemEndTurnDelay = nativeSystemEndTurnDelayFrames
	// FDTXT_000[0x1A4] 的 0xFFFE 是換行控制碼。原版在這段文字後等待
	// 0xC8 ms，才呼叫 0x1A30B 進入敵方回合。
	g.msg = "好的，\n就結束本回合的行動吧！"
}

func (g *Game) cancelNativeSystemEndTurn() {
	if g == nil || !g.nativeSystemEndTurnConfirm {
		return
	}
	g.nativeSystemEndTurnConfirm = false
	g.msg = ""
}

func (g *Game) stepNativeSystemEndTurn() {
	if g == nil || g.nativeSystemEndTurnDelay <= 0 {
		return
	}
	g.nativeSystemEndTurnDelay--
	if g.nativeSystemEndTurnDelay == 0 {
		g.msg = ""
		g.endTurn()
	}
}
