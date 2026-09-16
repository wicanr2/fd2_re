package main

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// pendingNativeDeathRewardMessage 是 0x1AA1D 物品型態 0／1 成功臂要播的訊息：
// 0x1956B(killer+7) 開對話格 → 0x15F84 寫 0x1B0／0x1B3（FFFC 物品名／FFFA 金額）→
// 0x16559(0) 蓋 DATO 第 0 幀 → 0x16C57(0) 等一個鍵 → 0x196CB 五幀關框並復原。
// 第七章 r3 seq 389：原版 attack_result 的 checkpoint 就停在這個等鍵處（ui=dialogue），
// 驅動端接著送 enter（finish-dialogue）才回到游標。
type pendingNativeDeathRewardMessage struct {
	killer      *battle.Unit
	kind, value int
}

type nativeDeathRewardMessageState struct {
	killer      *battle.Unit
	actor       *battle.Unit // 這次行動的單位：0x1AA1D 在 0x13512 設 +5 bit7 之前，底圖上它還沒變灰
	awaitAck    bool
	final       []byte
	then        func()
	portrait    dato.Frame
	portraits   []dato.Frame // 0x16559 等待期間店主式眨眼／嘴型各幀，重播端各出一張變體
	kind, value int
}

// composeNativeDeathRewardMessageOn 把同一則訊息（對話格＋頭像＋文字）疊在另一張
// 戰場整幀上；重播端替原版沒記錄的 idle／LUT 相位出變體時用。
func (g *Game) composeNativeDeathRewardMessageOn(source []byte, portraitFrame int) ([]byte, error) {
	state := g.nativeSystemEndTurnUI
	if state == nil || state.rewardMessage == nil || g.nativePreparationUI == nil {
		return nil, errors.New("native death reward message: no active message")
	}
	message := state.rewardMessage
	ui := g.nativePreparationUI
	portrait := message.portrait
	if portraitFrame > 0 {
		if portraitFrame >= len(message.portraits) {
			return nil, errors.New("native death reward message: portrait frame is unavailable")
		}
		portrait = message.portraits[portraitFrame]
	}
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, portrait)
	if err != nil {
		return nil, err
	}
	return campaign.ComposeNativeDeathRewardMessageFrame(dialogue, ui.status.Strings, ui.status.Font, message.kind, message.value)
}

// runPendingNativeDeathRewardMessages 在行動收尾播第一則待播訊息；資產不齊就失敗即
// 關閉（loadErr），不用重製端的 g.msg 文字條替代。
func (g *Game) runPendingNativeDeathRewardMessages(actor *battle.Unit, then func()) bool {
	if g == nil || len(g.pendingNativeRewardMsgs) == 0 || g.nativeSystemEndTurnUI != nil {
		return false
	}
	pending := g.pendingNativeRewardMsgs[0]
	g.pendingNativeRewardMsgs = g.pendingNativeRewardMsgs[1:]
	if err := g.beginNativeDeathRewardMessage(pending, actor, then); err != nil {
		g.loadErr = "native death reward message: " + err.Error()
	}
	return true
}

// composeNativeMapFrameBeforeActed 以「行動單位的 +5 bit7 還沒設」的狀態組整幀：0x18890 的
// handler 裡 0x1AA1D 先播訊息，0x13512 之後才設 bit7；重製端的結算一開始就設了，組底圖時
// 暫時拿掉（第七章 r6 seq 633：原版訊息底圖上攻擊者沒變灰）。
func (g *Game) composeNativeMapFrameBeforeActed(actor *battle.Unit) error {
	if actor != nil && actor.HasNativeRecordByte5 && actor.NativeRecordByte5&0x80 != 0 {
		actor.NativeRecordByte5 &^= 0x80
		defer func() { actor.NativeRecordByte5 |= 0x80 }()
	}
	return g.composeNativeMapFrame()
}

func (g *Game) beginNativeDeathRewardMessage(pending pendingNativeDeathRewardMessage, actor *battle.Unit, then func()) error {
	if pending.killer == nil || !pending.killer.HasBattleFig {
		return errors.New("killer DATO selector is unavailable")
	}
	if g.nativePreparationUI == nil || g.nativeClassUI == nil {
		return errors.New("dialogue assets are unavailable")
	}
	portraits, err := loadNativeSeparatedPortrait(pending.killer.BattleFig)
	if err != nil || len(portraits) == 0 {
		return errors.New("killer portrait is unavailable")
	}
	if err := g.composeNativeMapFrameBeforeActed(actor); err != nil {
		return err
	}
	ui := g.nativePreparationUI
	source := append([]byte(nil), g.nativeMapVGA...)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, portraits[0])
	if err != nil {
		return err
	}
	final, err := campaign.ComposeNativeDeathRewardMessageFrame(
		dialogue, ui.status.Strings, ui.status.Font, pending.kind, pending.value,
	)
	if err != nil {
		return err
	}
	opening, err := campaign.NativeClassListOpeningFrames(source, final)
	if err != nil {
		return err
	}
	message := &nativeDeathRewardMessageState{
		killer: pending.killer, actor: actor, final: final, then: then,
		portrait: portraits[0], portraits: portraits, kind: pending.kind, value: pending.value,
	}
	g.nativeSystemEndTurnUI = &nativeSystemEndTurnUIState{
		source: source, dialogue: final, rewardMessage: message,
	}
	g.nativeSystemEndTurnConfirm = false
	g.ring = false
	g.msg = ""
	g.resetNativeClassUIPulse()
	g.nativeClassUIJob = &nativeClassUIJob{frames: opening, after: func() {
		message.awaitAck = true
	}}
	return nil
}

// acknowledgeNativeDeathRewardMessage 是 0x16C57(0) 收到鍵之後的 0x196CB 關框；
// 型態 1 的金額在關框之後才加進 [0x53BF3]，關完才回到行動收尾（then）。
func (g *Game) acknowledgeNativeDeathRewardMessage() {
	state := g.nativeSystemEndTurnUI
	if state == nil || state.rewardMessage == nil || !state.rewardMessage.awaitAck {
		return
	}
	message := state.rewardMessage
	message.awaitAck = false
	closing, err := campaign.NativeClassListClosingFrames(state.source, message.final)
	if err != nil || len(closing) != 5 {
		g.loadErr = "native death reward message: dialogue close frames unavailable"
		return
	}
	g.nativeClassUIJob = &nativeClassUIJob{frames: closing, restore: state.source, after: func() {
		g.nativeSystemEndTurnUI = nil
		g.nativeSystemEndTurnDelay = 0
		if message.kind == 1 {
			// 0x1ABF8..0x1ABFD：關框之後才 `add [0x53BF3], [0x53AE1]`。
			g.gold += message.value
		}
		if message.then != nil {
			message.then()
		}
	}}
}

func (g *Game) nativeDeathRewardMessageAwaitingKey() bool {
	state := g.nativeSystemEndTurnUI
	return state != nil && state.rewardMessage != nil && state.rewardMessage.awaitAck
}
