package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

const nativeSystemEndTurnDelayFrames = 12 // 0x17259: delay(0xC8 ms)；60 Hz 約十二幀。

type nativeSystemEndTurnUIState struct {
	source, dialogue, question []byte
	accepted, canceled         [][]byte
	choice                     int
	acceptedOutcome            bool
	exitProgram                bool
	groupMarch                 bool
	groupMarchPlan             battle.NativeSystemGroupMarchPlan
	saveCurrent                bool
	savePath                   string
	saveStored                 []byte
	loadCurrent                bool
	loadCandidate              *Game
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
	g.cancelNativeAICommandModifierPresentation()
	g.cancelNativeCommand32Presentation()
	g.cancelNativeCommand33Presentation()
	g.cancelNativeCommand34Presentation()
	g.cancelNativeCommand35Presentation()
	g.cancelNativeAIItemPresentation()
	g.ring = false
	g.nativeSystemCursorOverlay = false
	g.nativeSystemNestedOpen = false
	g.nativeSystemOptionsOpen = false
	g.nativeSystemInfoUI = nil
	g.nativeSystemEndTurnConfirm = false
	g.nativeSystemEndTurnDelay = 0
	g.nativeSystemExitRequested = false
	g.nativeSystemGroupMarch = nil
	g.nativeSystemGroupMarchStep = 0
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

func (g *Game) currentNativeSystemOptions() fdother.NativeSystemOptions {
	if g != nil && g.nativeSystemOptions != nil {
		return *g.nativeSystemOptions
	}
	return fdother.DefaultNativeSystemOptions()
}

func (g *Game) nativeFullPresentationEnabled() bool {
	return g == nil || g.currentNativeSystemOptions().FullPresentationEnabled()
}

func (g *Game) nativeSystemOptionsReady() bool {
	if _, err := g.currentNativeSystemOptions().ActionOverlayState(); err != nil {
		return false
	}
	for _, index := range [...]int{54, 57, 60, 63, 66, 69, 72, 75} {
		if index >= len(g.nativeActionCells) || g.nativeActionCells[index] == nil {
			return false
		}
	}
	return true
}

func (g *Game) toggleNativeSystemOption(selector int) bool {
	current := g.currentNativeSystemOptions()
	next, err := current.Toggle(selector)
	if err != nil {
		return false
	}
	g.nativeSystemOptions = &next
	switch selector {
	case 0:
		if next.MusicEnabled() {
			if g.nativeSystemBGMTrack != "" {
				track := g.nativeSystemBGMTrack
				g.nativeSystemBGMTrack = ""
				g.playBGM(track)
			}
		} else {
			g.muteNativeSystemBGM()
		}
	case 3:
		g.restoreNativeMapHUDGateA(next.Raw51AAB)
		if g.st != nil && g.st.HasNativeMapHUDState {
			g.st.NativeMapHUDState.DisplayGateA = next.Raw51AAB
		}
	}
	return true
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
	accepted, err := campaign.NativeBattleEndTurnResponseFrames(question, ui.status.Strings, ui.status.Font, true)
	if err != nil {
		return false
	}
	canceled, err := campaign.NativeBattleEndTurnResponseFrames(question, ui.status.Strings, ui.status.Font, false)
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

// beginNativeNestedSystemExit 承接 sub_19DF7 selector3。原版只在 YES
// 回覆、停止音樂、0xC8 延遲與對話框完整收合後，才把 -1 傳回 main 的清理出口。
// 因此重製端也延後發布終止要求；資產或來源不完整時保持巢狀選單不變。
func (g *Game) beginNativeNestedSystemExit() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.nativeSystemNestedOpen ||
		!g.ring || g.ringSel != 3 || g.st == nil || g.aiBusy || g.result != "" ||
		g.nativePreparationUI == nil || g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200 {
		return false
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, ui.portrait)
	if err != nil {
		return false
	}
	question, err := campaign.ComposeNativeBattleExitQuestion(
		dialogue, ui.portrait, ui.status.Strings, ui.status.Font,
	)
	if err != nil {
		return false
	}
	accepted, err := campaign.NativeBattleExitResponseFrames(question, ui.status.Strings, ui.status.Font, true)
	if err != nil {
		return false
	}
	canceled, err := campaign.NativeBattleExitResponseFrames(question, ui.status.Strings, ui.status.Font, false)
	if err != nil {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(source, dialogue, question, ui.choices)
	if err != nil || len(frames) != 10 {
		return false
	}
	state := &nativeSystemEndTurnUIState{
		source: source, dialogue: dialogue, question: question,
		accepted: accepted, canceled: canceled, exitProgram: true,
	}
	g.beginActionOverlayClose(func() {
		g.nativeSystemNestedOpen = false
		g.nativeSystemCursorOverlay = false
		g.nativeSystemEndTurnConfirm = true
		g.nativeSystemEndTurnUI = state
		g.resetNativeClassUIPulse()
		g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	})
	return true
}

// beginNativeNestedCurrentSave owns sub_19DF7 selector1. It prebuilds both
// the exact encoded replacement and every indexed confirmation frame before
// closing the nested overlay. Only the later YES publication touches disk.
func (g *Game) beginNativeNestedCurrentSave() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.nativeSystemNestedOpen ||
		!g.ring || g.ringSel != 1 || g.st == nil || g.aiBusy || g.result != "" ||
		g.nativePreparationUI == nil || g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200 {
		return false
	}
	path, stored, err := g.buildNativeCurrentSaveStored()
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, ui.portrait)
	if err != nil {
		return false
	}
	question, err := campaign.ComposeNativeBattleCurrentSaveQuestion(
		dialogue, ui.portrait, ui.status.Strings, ui.status.Font,
	)
	if err != nil {
		return false
	}
	accepted, err := campaign.NativeBattleCurrentSaveResponseFrames(
		question, ui.status.Strings, ui.status.Font, true,
	)
	if err != nil {
		return false
	}
	canceled, err := campaign.NativeBattleCurrentSaveResponseFrames(
		question, ui.status.Strings, ui.status.Font, false,
	)
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
		saveCurrent: true, savePath: path, saveStored: append([]byte(nil), stored...),
	}
	g.beginActionOverlayClose(func() {
		g.nativeSystemNestedOpen = false
		g.nativeSystemCursorOverlay = false
		g.nativeSystemEndTurnConfirm = true
		g.nativeSystemEndTurnUI = state
		g.resetNativeClassUIPulse()
		g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	})
	return true
}

// beginNativeNestedCurrentLoad owns sub_19DF7 selector2. It builds the whole
// restore candidate before closing the nested overlay. The live battle is not
// replaced until the later YES response and dialogue lifecycle complete.
func (g *Game) beginNativeNestedCurrentLoad() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.nativeSystemNestedOpen ||
		!g.ring || g.ringSel != 2 || g.st == nil || g.aiBusy || g.result != "" ||
		g.nativePreparationUI == nil || g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200 {
		return false
	}
	timer, ok := g.nativeMapClock.Current()
	if !ok {
		g.loadErr = "原版目前戰況讀檔：目前戰場時鐘尚未物化"
		return false
	}
	candidate, err := g.prepareNativeContinueFromCurrentSnapshot(nativeCurrentSavePath(), timer)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, ui.portrait)
	if err != nil {
		return false
	}
	question, err := campaign.ComposeNativeBattleCurrentLoadQuestion(
		dialogue, ui.portrait, ui.status.Strings, ui.status.Font,
	)
	if err != nil {
		return false
	}
	accepted, err := campaign.NativeBattleCurrentLoadResponseFrames(
		question, ui.status.Strings, ui.status.Font, true,
	)
	if err != nil {
		return false
	}
	canceled, err := campaign.NativeBattleCurrentLoadResponseFrames(
		question, ui.status.Strings, ui.status.Font, false,
	)
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
		loadCurrent: true, loadCandidate: &candidate,
	}
	g.beginActionOverlayClose(func() {
		g.nativeSystemNestedOpen = false
		g.nativeSystemCursorOverlay = false
		g.nativeSystemEndTurnConfirm = true
		g.nativeSystemEndTurnUI = state
		g.resetNativeClassUIPulse()
		g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	})
	return true
}

// beginNativeSystemGroupMarch 承接 sub_16F55 selector1。它在收掉外層命令框前
// 完整預演逐單位尋路與事件；任一未知事件使整批保持原狀。
func (g *Game) beginNativeSystemGroupMarch() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.ring || g.ringSel != 1 ||
		g.st == nil || g.m == nil || g.aiBusy || g.result != "" ||
		g.nativePreparationUI == nil || g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200 ||
		len(g.st.Units) == 0 || g.st.Units[0] == nil || !g.st.Units[0].HasBattleFig {
		return false
	}
	plan, err := g.st.PlanNativeSystemGroupMarch(battle.Cell{X: g.curX, Y: g.curY})
	if err != nil {
		return false
	}
	if !g.preflightNativeSystemGroupMarchEvents(plan) {
		return false
	}
	portraits, err := loadNativeSeparatedPortrait(g.st.Units[0].BattleFig)
	if err != nil || len(portraits) == 0 {
		return false
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, portraits[0])
	if err != nil {
		return false
	}
	question, err := campaign.ComposeNativeBattleGroupMarchQuestion(
		dialogue, portraits[0], ui.status.Strings, ui.status.Font,
	)
	if err != nil {
		return false
	}
	accepted, err := campaign.NativeBattleGroupMarchResponseFrames(question, ui.status.Strings, ui.status.Font, true)
	if err != nil {
		return false
	}
	canceled, err := campaign.NativeBattleGroupMarchResponseFrames(question, ui.status.Strings, ui.status.Font, false)
	if err != nil {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(source, dialogue, question, ui.choices)
	if err != nil || len(frames) != 10 {
		return false
	}
	state := &nativeSystemEndTurnUIState{
		source: source, dialogue: dialogue, question: question,
		accepted: accepted, canceled: canceled, groupMarch: true, groupMarchPlan: plan,
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

// activateNativeSystemDirectionOne 固定兩層同一方向的不同 owner：外層
// sub_16F55 selector1 是全軍移動；巢狀 sub_19DF7 selector1 是 current-runtime
// FD2.SAV 保存。後者尚未接線時只能失敗即關閉，絕不能落入全軍移動。
func (g *Game) activateNativeSystemDirectionOne() bool {
	if g == nil || !g.nativeSystemCursorOverlay || !g.ring || g.ringSel != 1 {
		return false
	}
	if g.nativeSystemNestedOpen {
		if !g.beginNativeNestedCurrentSave() {
			g.msg = "原版目前戰況存檔來源或資產不完整，未寫入檔案"
		}
		return true
	}
	if !g.beginNativeSystemGroupMarch() {
		g.msg = "原版全軍移動來源或事件不完整，未執行"
	}
	return true
}

// preflightNativeSystemGroupMarchEvents 驗證既有 event61／75 UI owner 的可編輯
// 對話及原始演出資產。battle planner 已在私有 State 投影驗證 mutation；這裡
// 補齊只有 Game 層能取得的 presentation admission，避免先移動才發現缺素材。
func (g *Game) preflightNativeSystemGroupMarchEvents(plan battle.NativeSystemGroupMarchPlan) bool {
	for _, step := range plan.Steps {
		for _, event := range step.Events {
			switch event.EventID {
			case 61:
				if _, ok := event61DialogueActions(g, 0, 10, 2); !ok {
					return false
				}
				if event.Presentation {
					if _, ok := event61DialogueActions(g, 1, 0, 10); !ok {
						return false
					}
					frames, err := fdother.LoadSeparatedEvent61Frames(
						separatedAssetPath("animations/fdother_045_event61"),
					)
					if err != nil || len(frames) != 59 {
						return false
					}
				}
			case 75:
				probe := &battle.Unit{BattleFig: 0, HasBattleFig: true}
				if _, ok := event75DialogueActions(g, event.TextIndex, probe); !ok {
					return false
				}
			default:
				return false
			}
		}
	}
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
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames, after: func() {
		state := g.nativeSystemEndTurnUI
		if state == nil {
			return
		}
		if accepted && state.saveCurrent {
			if err := replaceNativeCurrentSaveAtomic(state.savePath, state.saveStored); err != nil {
				g.loadErr = err.Error()
				accepted = false
			} else if plain, err := fdsave.Decode(state.saveStored); err != nil {
				g.loadErr = "原版目前戰況存檔：寫入後 baseline 解碼失敗：" + err.Error()
				accepted = false
			} else {
				g.nativeCurrentSavePlain = plain
			}
		}
		state.acceptedOutcome = accepted
		response := g.nativeSystemEndTurnUI.canceled
		if accepted {
			response = g.nativeSystemEndTurnUI.accepted
		}
		g.nativeClassUIJob = &nativeClassUIJob{frames: response, after: func() {
			if accepted && g.nativeSystemEndTurnUI != nil && g.nativeSystemEndTurnUI.exitProgram {
				g.stopBGM()
			}
			g.nativeSystemEndTurnDelay = nativeSystemEndTurnDelayFrames
		}}
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
				if state.loadCurrent {
					if state.loadCandidate == nil {
						g.loadErr = "原版目前戰況讀檔：私有候選狀態遺失"
						return
					}
					candidate := *state.loadCandidate
					*g = candidate
				} else if state.exitProgram {
					g.nativeSystemExitRequested = true
				} else if state.groupMarch {
					plan := state.groupMarchPlan
					g.nativeSystemGroupMarch = &plan
					g.nativeSystemGroupMarchStep = 0
					g.startNextNativeSystemGroupMarchStep()
				} else if !state.saveCurrent {
					g.endTurn()
				}
			}
		}}
	}
}

func (g *Game) startNextNativeSystemGroupMarchStep() {
	for g != nil && g.nativeSystemGroupMarch != nil && g.walk == nil {
		if g.nativeSystemGroupMarchStep >= len(g.nativeSystemGroupMarch.Steps) {
			g.nativeSystemGroupMarch = nil
			g.nativeSystemGroupMarchStep = 0
			g.endTurn()
			return
		}
		step := g.nativeSystemGroupMarch.Steps[g.nativeSystemGroupMarchStep]
		if step.UnitIndex < 0 || step.UnitIndex >= len(g.st.Units) || g.st.Units[step.UnitIndex] == nil {
			g.loadErr = "native system group-march runtime unit disappeared"
			g.nativeSystemGroupMarch = nil
			return
		}
		finish := func() {
			if err := g.st.CommitNativeSystemGroupMarchStep(step); err != nil {
				g.loadErr = "native system group-march commit: " + err.Error()
				g.nativeSystemGroupMarch = nil
				return
			}
			g.nativeSystemGroupMarchStep++
			g.startNextNativeSystemGroupMarchStep()
		}
		if len(step.Path) < 2 {
			finish()
			continue
		}
		selector := byte(1)
		g.walk = &walkAnim{
			u: g.st.Units[step.UnitIndex], path: append([]battle.Cell(nil), step.Path...),
			then: finish, nativeEventSelector: &selector,
			nativeGroupMarchEvents: append([]battle.NativeSystemGroupMarchEvent(nil), step.Events...),
		}
	}
}

func (g *Game) beginNativeSystemGroupMarchFieldEvent(
	walk *walkAnim,
	event battle.NativeSystemGroupMarchEvent,
) bool {
	if g == nil || walk == nil || g.walk != walk || walk.u == nil {
		return false
	}
	beforeErr := g.loadErr
	resume := func() {
		if g.walk != walk {
			return
		}
		if g.loadErr != beforeErr {
			g.walk = nil
			g.nativeSystemGroupMarch = nil
			return
		}
		walk.nativeGroupMarchEvent++
		walk.nativeGroupMarchPaused = false
	}
	switch event.EventID {
	case 61:
		return g.beginNativeFieldEvent61(walk.u, resume)
	case 75:
		return g.beginNativeFieldEvent75(walk.u, resume)
	default:
		return false
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
		frames := state.canceled
		if state.acceptedOutcome {
			frames = state.accepted
		}
		if len(frames) == 0 {
			return false
		}
		frame = frames[len(frames)-1]
	} else {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}
