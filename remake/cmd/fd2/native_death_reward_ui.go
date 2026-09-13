package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

type pendingNativeDeathReward struct {
	killer *battle.Unit
	item   int
}

type nativeDeathRewardUIState struct {
	killer            *battle.Unit
	previousSelection *battle.Unit
	portrait          dato.Frame
	item, selected    int
	source, dialogue  []byte
}

type nativeDeathRewardInput struct {
	up, down, left, right bool
	confirm, cancel       bool
}

// runPendingNativeDeathRewards starts the first 0x1AA56 modal only at action
// completion, after the attack presentation. It preflights both the indexed
// question and the later item panel before exposing any interactive state.
func (g *Game) runPendingNativeDeathRewards(then func()) bool {
	if g == nil || g.nativeDeathRewardUI != nil || len(g.pendingNativeDeathRewards) == 0 {
		return false
	}
	pending := g.pendingNativeDeathRewards[0]
	if pending.killer == nil || pending.item < 0 || pending.item > 0xff ||
		len(pending.killer.Inventory) != 8 ||
		battle.ValidateNativeInventoryProjection(pending.killer) != nil ||
		g.nativePreparationUI == nil || g.nativeClassUI == nil ||
		!pending.killer.HasBattleFig {
		g.loadErr = "native death reward: full inventory or dialogue provenance unavailable"
		return true
	}
	for _, flag := range pending.killer.NativeInventoryFlags {
		if flag&0x80 != 0 {
			g.loadErr = "native death reward: full raw inventory contains an empty slot"
			return true
		}
	}
	portraits, err := loadNativeSeparatedPortrait(pending.killer.BattleFig)
	if err != nil || len(portraits) == 0 {
		g.loadErr = "native death reward: killer portrait unavailable"
		return true
	}
	if err := g.composeNativeMapFrame(); err != nil {
		g.loadErr = "native death reward: " + err.Error()
		return true
	}
	if !g.prepareNativeItemPanel(pending.killer) {
		g.loadErr = "native death reward: eight-slot item panel unavailable"
		return true
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, portraits[0])
	if err != nil {
		g.clearNativeItemPanel()
		g.loadErr = "native death reward: " + err.Error()
		return true
	}
	question, err := campaign.ComposeNativeDeathRewardDiscardQuestion(
		dialogue, portraits[0], ui.status.Strings, ui.status.Font,
	)
	if err != nil {
		g.clearNativeItemPanel()
		g.loadErr = "native death reward: " + err.Error()
		return true
	}
	canceled, err := campaign.NativeDeathRewardAbandonResponseFrames(
		question, ui.status.Strings, ui.status.Font, pending.item,
	)
	if err != nil {
		g.clearNativeItemPanel()
		g.loadErr = "native death reward: " + err.Error()
		return true
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(
		source, dialogue, question, ui.choices,
	)
	if err != nil || len(frames) != 10 {
		g.clearNativeItemPanel()
		g.loadErr = "native death reward: prompt opening frames unavailable"
		return true
	}
	prompt := &nativeDeathRewardUIState{
		killer: pending.killer, previousSelection: g.sel, portrait: portraits[0],
		item: pending.item, selected: -1, source: source, dialogue: dialogue,
	}
	g.pendingNativeDeathRewards = g.pendingNativeDeathRewards[1:]
	g.nativeDeathRewardUI = prompt
	g.nativeDeathRewardThen = then
	g.nativeSystemEndTurnUI = &nativeSystemEndTurnUIState{
		source: source, dialogue: dialogue, question: question,
		canceled: canceled, deathReward: prompt,
	}
	g.nativeSystemEndTurnConfirm = true
	g.nativeSystemEndTurnDelay = 0
	g.ring = false
	g.msg = ""
	g.resetNativeClassUIPulse()
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) handleNativeDeathRewardItemInput(input nativeDeathRewardInput) bool {
	prompt := g.nativeDeathRewardUI
	if prompt == nil || !g.itemOpen || g.sel != prompt.killer || g.nativeItemPanel == nil {
		return false
	}
	scanCode := 0
	switch {
	case input.up:
		scanCode = 72
	case input.down:
		scanCode = 80
	case input.left:
		scanCode = 75
	case input.right:
		scanCode = 77
	case input.confirm:
		scanCode = 28
	case input.cancel:
		scanCode = 1
	}
	if scanCode == 0 {
		return true
	}
	selected, result, err := battle.AdvanceNativeItemSelector(
		g.itemSel, 8, scanCode, false, 0,
	)
	if err != nil {
		g.loadErr = "native death reward: " + err.Error()
		return true
	}
	if selected != g.itemSel {
		g.itemSel = selected
		if !g.refreshNativeItemPanel(prompt.killer) {
			g.loadErr = "native death reward: item panel refresh failed"
		}
	}
	switch result {
	case battle.NativeItemSelectorConfirm:
		probe := *prompt.killer
		probe.Inventory = append([]int(nil), prompt.killer.Inventory...)
		probe.Equipped = append([]bool(nil), prompt.killer.Equipped...)
		probe.InventorySlots = append([]int(nil), prompt.killer.InventorySlots...)
		probe.NativeInventoryFlags = append([]int(nil), prompt.killer.NativeInventoryFlags...)
		if err := battle.ReplaceNativeFullInventoryReward(&probe, g.itemSel, prompt.item); err != nil {
			g.loadErr = "native death reward: " + err.Error()
			return true
		}
		prompt.selected = g.itemSel
		g.itemClosing = true
		g.itemAnimStep = 0
	case battle.NativeItemSelectorCancel:
		prompt.selected = -1
		g.itemClosing = true
		g.itemAnimStep = 0
	}
	return true
}

func (g *Game) finishNativeDeathRewardItemClose() {
	prompt := g.nativeDeathRewardUI
	if prompt == nil {
		return
	}
	if prompt.selected >= 0 {
		if err := battle.ReplaceNativeFullInventoryReward(prompt.killer, prompt.selected, prompt.item); err != nil {
			g.loadErr = "native death reward: " + err.Error()
			return
		}
		g.finishNativeDeathRewardUI()
		return
	}
	g.beginNativeDeathRewardSelectorCancel(prompt)
}

func (g *Game) beginNativeDeathRewardSelectorCancel(prompt *nativeDeathRewardUIState) {
	ui := g.nativePreparationUI
	if prompt == nil || ui == nil {
		g.loadErr = "native death reward: selector-cancel dialogue unavailable"
		return
	}
	abandon, err := campaign.ComposeNativeDeathRewardSelectorCancel(
		prompt.dialogue, prompt.portrait, ui.status.Strings, ui.status.Font, prompt.item,
	)
	if err != nil {
		g.loadErr = "native death reward: " + err.Error()
		return
	}
	frames, err := campaign.NativeClassListOpeningFrames(prompt.source, abandon)
	if err != nil || len(frames) != 6 {
		g.loadErr = "native death reward: selector-cancel opening frames unavailable"
		return
	}
	g.nativeSystemEndTurnUI = &nativeSystemEndTurnUIState{
		source: prompt.source, dialogue: abandon, question: abandon,
		canceled: [][]byte{abandon}, deathReward: prompt,
	}
	g.nativeSystemEndTurnConfirm = false
	g.nativeSystemEndTurnDelay = 0
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames, after: func() {
		g.nativeSystemEndTurnDelay = nativeSystemEndTurnDelayFrames
	}}
}

func (g *Game) finishNativeDeathRewardUI() {
	prompt := g.nativeDeathRewardUI
	if prompt == nil {
		return
	}
	then := g.nativeDeathRewardThen
	g.sel = prompt.previousSelection
	g.nativeDeathRewardUI = nil
	g.nativeDeathRewardThen = nil
	g.nativeSystemEndTurnUI = nil
	g.nativeSystemEndTurnConfirm = false
	g.nativeSystemEndTurnDelay = 0
	g.nativeClassUIJob = nil
	g.clearNativeItemPanel()
	if then != nil {
		then()
	}
}
