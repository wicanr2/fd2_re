package main

import (
	"fmt"
	"log"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// nativeBattleDialogueAdvanceInput 回報這次鍵盤事件是否推進戰鬥對白。
// 原版戰鬥事件與回合起手對白的 dosgolem 三臂收據都證實 ESC 與 Enter 同義；
// Space 沿用既有桌面便利鍵，呼叫端把 Enter／Space 合併成 enterOrSpace。
func nativeBattleDialogueAdvanceInput(enterOrSpace, escape bool) bool {
	return enterOrSpace || escape
}

func (g *Game) startNativeEventDialogue(action battle.Action) error {
	ref := action.NativeDialogueRef
	if ref == nil || action.NativeSource == "" || action.NativeTextIndex == nil ||
		*action.NativeTextIndex != ref.StringIndex || ref.SceneIndex < 0 || ref.Line < 0 {
		return fmt.Errorf("戰鬥對話缺少一致的來源")
	}
	layout := &campaign.NativeDialogueLayout{
		SourceDAT: ref.SourceDAT, StringIndex: ref.StringIndex, Utterance: ref.Utterance,
		Control: ref.Control, Operand: ref.Operand, Pages: ref.Pages, GlyphPages: ref.GlyphPages,
	}
	if err := layout.Validate(); err != nil {
		return err
	}
	lines, err := loadStoryScriptWithIdentityAt(ref.Script, "", &ref.SceneIndex)
	if err != nil {
		return err
	}
	if ref.Line >= len(lines) {
		return fmt.Errorf("戰鬥對話來源句 %d 不存在", ref.Line)
	}
	upper := ref.Control == "FFED" || ref.Control == "FFEF"
	if g.cutsceneLog { // FD2_CUTSCENE_LOG：印回合／死亡事件的每一句，對原版 0x15F84 呼叫序列比對
		event := -1
		if action.NativeEventID != nil {
			event = *action.NativeEventID
		}
		log.Printf("[cutscene] event dialogue source=%s event=%d text=%d scene=%d line=%d",
			action.NativeSource, event, ref.StringIndex, ref.SceneIndex, ref.Line)
	}
	if !g.nativeBattleDialogueAvailable() {
		// 這一章的戰場還沒有原生視圖與 HUD 狀態，它自己的回合事件也用一般對白框；
		// 死亡台詞照同一種呈現，文字與說話者取自同一行故事腳本，不另造內容。
		line, err := g.resolveCampaignDialogLine(lines[ref.Line], &upper, nil)
		if err != nil {
			return err
		}
		g.dialog = []battle.DialogLine{line}
		g.dlgPage, g.dlgScrollT, g.dlgShown, g.dlgPhase, g.dlgT = 0, 0, dlgNone, 0, 0
		return nil
	}
	line, err := g.resolveCampaignDialogLine(lines[ref.Line], &upper, layout)
	if err != nil {
		return err
	}
	g.dialog = []battle.DialogLine{line}
	g.dlgPage, g.dlgScrollT, g.dlgShown, g.dlgPhase, g.dlgT = 0, 0, dlgNone, 0, 0
	return g.startNativeDialogueFrames()
}

// handleBattleEventDialogueInput 與故事對話共用逐字／收框閘門，續行仍歸戰鬥事件。
func (g *Game) handleBattleEventDialogueInput(advance bool) {
	if !advance || len(g.dialog) == 0 || g.nativeDialogueClosingLive {
		return
	}
	current := g.dialog[len(g.dialog)-1]
	if current.NativeDialogue != nil && g.dlgPage+1 >= dlgPageCount(current) {
		if !g.nativeStoryDialogueAtInputWait() {
			return
		}
		if !g.beginNativeStoryDialogueClosing() {
			g.finishBattleEventWithError("原生對話收框不可用")
		}
		return
	}
	if g.dlgAdvance() && len(g.dialog) == 0 {
		g.advanceBattleEvent()
	}
}

// nativeBattleDialogueAvailable 是原生戰場對白（開框、聚焦說話者、逐字、收框）需要的
// 狀態：原生視圖、HUD 輸入與分離素材都在。
func (g *Game) nativeBattleDialogueAvailable() bool {
	if g.st == nil || !g.st.HasNativeMapViewState || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return false
	}
	_, ok := g.nativeMapHUDInput()
	return ok
}
