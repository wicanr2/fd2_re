package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

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
func (g *Game) handleBattleEventDialogueInput(enter bool) {
	if !enter || len(g.dialog) == 0 || g.nativeDialogueClosingLive {
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
