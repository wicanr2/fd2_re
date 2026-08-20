package main

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
	g.nativeContinueCursorOverlay = false
	g.nativeContinueEndTurnConfirm = false
	g.actionOverlayPhase = ""
	g.actionOverlayFrame = 0
	g.actionOverlayAfter = nil
	g.actionOverlayDrawn = false
	g.actionOverlayShotHold = false
}

// beginNativeContinueEndTurn 只承接 chapter0 current-runtime 已觀測的
// Down→END。原版 trace 證實 END 會開確認且 YES 會進 ENEMY PHASE；它沒有證實
// 其餘三個 0x16f55 cell owner，也沒有證實重製端確認提示的像素。
func (g *Game) beginNativeContinueEndTurn() bool {
	if g == nil || !g.nativeContinueCursorOverlay || !g.ring || g.ringSel != 3 ||
		g.st == nil || g.aiBusy || g.result != "" {
		return false
	}
	g.beginActionOverlayClose(func() {
		g.nativeContinueCursorOverlay = false
		g.nativeContinueEndTurnConfirm = true
		g.msg = "結束本回合的行動嗎？　Enter＝是，Esc＝否"
	})
	return true
}

func (g *Game) confirmNativeContinueEndTurn() {
	if g == nil || !g.nativeContinueEndTurnConfirm {
		return
	}
	g.nativeContinueEndTurnConfirm = false
	g.msg = ""
	g.endTurn()
}

func (g *Game) cancelNativeContinueEndTurn() {
	if g == nil || !g.nativeContinueEndTurnConfirm {
		return
	}
	g.nativeContinueEndTurnConfirm = false
	g.msg = ""
}
