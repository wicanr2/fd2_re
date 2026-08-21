package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const nativeSystemEndTurnDelayFrames = 12 // 0x17259: delay(0xC8 ms)；60 Hz 約十二幀。

type nativeSystemEndTurnUIState struct {
	source, dialogue, question []byte
	accepted, canceled         []byte
	choice                     int
	acceptedOutcome            bool
}

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
	if g.nativeSystemEndTurnUI != nil {
		g.nativeClassUIJob = nil
	}
	g.nativeSystemEndTurnUI = nil
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

// beginNativeSystemEndTurn 承接共用 0x117E7→0x16F55 的 Down→END，並在
// 關閉命令框前完整建立 0x1956b→0x19953 的原版索引畫面。任何資產缺失都
// 失敗即關閉，避免先收掉命令框後才退回不忠實的泛用文字提示。
func (g *Game) beginNativeSystemEndTurn() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.ring || g.ringSel != 3 ||
		g.st == nil || g.aiBusy || g.result != "" || g.nativePreparationUI == nil ||
		g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200 {
		return false
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, ui.portrait)
	if err != nil {
		return false
	}
	question, err := campaign.ComposeNativeBattleEndTurnQuestion(
		dialogue, ui.portrait, ui.status.Strings, ui.status.Font,
	)
	if err != nil {
		return false
	}
	accepted, err := campaign.ComposeNativeBattleEndTurnResponse(question, ui.status.Strings, ui.status.Font, true)
	if err != nil {
		return false
	}
	canceled, err := campaign.ComposeNativeBattleEndTurnResponse(question, ui.status.Strings, ui.status.Font, false)
	if err != nil {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(source, dialogue, question, ui.choices)
	if err != nil || len(frames) != 10 {
		return false
	}
	state := &nativeSystemEndTurnUIState{
		source: source, dialogue: dialogue, question: question,
		accepted: accepted, canceled: canceled,
	}
	g.beginActionOverlayClose(func() {
		g.nativeSystemCursorOverlay = false
		g.nativeSystemEndTurnConfirm = true
		g.nativeSystemEndTurnUI = state
		g.resetNativeClassUIPulse()
		g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	})
	return true
}

func (g *Game) confirmNativeSystemEndTurn() {
	if g == nil || !g.nativeSystemEndTurnConfirm {
		return
	}
	g.finishNativeSystemEndTurnChoice(true)
}

func (g *Game) cancelNativeSystemEndTurn() {
	if g == nil || !g.nativeSystemEndTurnConfirm {
		return
	}
	g.finishNativeSystemEndTurnChoice(false)
}

func (g *Game) finishNativeSystemEndTurnChoice(accepted bool) {
	if g.nativeSystemEndTurnUI == nil || g.nativePreparationUI == nil {
		g.nativeSystemEndTurnConfirm = false
		return
	}
	frames, err := campaign.NativeClassConfirmationClosingFrames(
		g.nativeSystemEndTurnUI.question, g.nativePreparationUI.choices,
	)
	if err != nil || len(frames) != 4 {
		return
	}
	g.nativeSystemEndTurnConfirm = false
	g.nativeSystemEndTurnUI.acceptedOutcome = accepted
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames, after: func() {
		g.nativeSystemEndTurnDelay = nativeSystemEndTurnDelayFrames
	}}
}

func (g *Game) stepNativeSystemEndTurn() {
	if g == nil || g.nativeSystemEndTurnDelay <= 0 {
		return
	}
	g.nativeSystemEndTurnDelay--
	if g.nativeSystemEndTurnDelay == 0 {
		state := g.nativeSystemEndTurnUI
		if state == nil {
			return
		}
		frames, err := campaign.NativeClassListClosingFrames(state.source, state.dialogue)
		if err != nil || len(frames) != 5 {
			return
		}
		g.nativeClassUIJob = &nativeClassUIJob{frames: frames, restore: state.source, after: func() {
			accepted := state.acceptedOutcome
			g.nativeSystemEndTurnUI = nil
			if accepted {
				g.endTurn()
			}
		}}
	}
}

// drawNativeSystemEndTurn 在 END 對話展開、等待輸入、顯示回覆與收合期間，
// 完整擁有320×200索引畫面，避免底下的重製指令層穿透。
func (g *Game) drawNativeSystemEndTurn(screen *ebiten.Image) bool {
	state := g.nativeSystemEndTurnUI
	if state == nil || g.nativeClassUI == nil || g.nativePreparationUI == nil {
		return false
	}
	if g.drawNativeClassUIJob(screen) {
		return true
	}
	var frame []byte
	if g.nativeSystemEndTurnConfirm {
		var err error
		frame, err = campaign.ComposeNativeConfirmationChoices(
			state.question, g.nativePreparationUI.choices,
			state.choice, g.nativeClassUIPulse/2,
		)
		if err != nil {
			return false
		}
	} else if g.nativeSystemEndTurnDelay > 0 {
		frame = state.canceled
		if state.acceptedOutcome {
			frame = state.accepted
		}
	} else {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}
