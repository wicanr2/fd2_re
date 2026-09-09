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
