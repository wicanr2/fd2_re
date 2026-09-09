package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type nativeTreasurePrompt struct {
	actor    *battle.Unit
	reward   battle.Treasure
	x, y     int
	awaitAck bool
}

// beginNativeTreasureItemPrompt 只接 0x190AC 有空欄位的普通物品分支。
// 所有資產先驗證，之後才關閉行動環；取消不改寶箱與庫存。
func (g *Game) beginNativeTreasureItemPrompt(u *battle.Unit, reward battle.Treasure) error {
	if g.nativePreparationUI == nil || g.nativeClassUI == nil || !u.HasBattleFig {
		return fmt.Errorf("native treasure: dialogue assets or actor provenance unavailable")
	}
	portraits, err := loadNativeSeparatedPortrait(u.BattleFig)
	if err != nil || len(portraits) == 0 {
		return fmt.Errorf("native treasure: actor portrait unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		return err
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, portraits[0])
	if err != nil {
		return err
	}
	question, err := campaign.ComposeNativeTreasureItemQuestion(dialogue, portraits[0], ui.status.Strings, ui.status.Font, reward.Hidden)
	if err != nil {
		return err
	}
	accepted, err := campaign.NativeTreasureItemResponseFrames(question, ui.status.Strings, ui.status.Font, reward.Value, reward.Hidden)
	if err != nil {
		return err
	}
	canceled, err := campaign.NativeBattleEndTurnResponseFrames(question, ui.status.Strings, ui.status.Font, false)
	if err != nil {
		return err
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(source, dialogue, question, ui.choices)
	if err != nil {
		return err
	}
	g.nativeSystemEndTurnUI = &nativeSystemEndTurnUIState{
		source: source, dialogue: dialogue, question: question, accepted: accepted, canceled: canceled,
		treasure: &nativeTreasurePrompt{actor: u, reward: reward, x: u.X, y: u.Y},
	}
	g.nativeSystemEndTurnConfirm = true
	g.ring = false
	g.msg = ""
	g.resetNativeClassUIPulse()
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return nil
}

func (g *Game) commitNativeTreasurePrompt(p *nativeTreasurePrompt) bool {
	current, ok := g.st.TreasureAt(p.x, p.y)
	if !ok || current != p.reward || p.actor.X != p.x || p.actor.Y != p.y {
		g.loadErr = "native treasure: source changed before confirmation"
		return false
	}
	if _, ok := g.st.ClaimTreasure(p.actor, p.x, p.y); !ok {
		g.loadErr = "native treasure: item transaction rejected"
		return false
	}
	return true
}

func (g *Game) finishNativeTreasurePrompt(p *nativeTreasurePrompt) {
	g.finishSuccessfulUnitAction(p.actor, func() { g.sel, g.reach, g.moved = nil, nil, false })
}
