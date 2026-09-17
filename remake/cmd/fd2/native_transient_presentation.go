package main

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// nativeTransientDamageTextIndex 是 0x1A915 推給 0x15F84 的 FDTXT_000 字串（FFFA＝扣血量
// [0x53AE1]）；nativeTransientDamageCloseFrames 是 0x1A92D 的 0x1E5C0(10)：10 個 BIOS tick
// 換成 60 Hz 約 33 幀。
const (
	nativeTransientDamageTextIndex   = 0x1e7
	nativeTransientDamageCloseFrames = 33
)

// beginNativeTransientPhases 依 sub_1A866 的順序演出一次掃描：先逐筆 +0x25 扣血回覆
// （0x1A8D4 [0x51A83]=0 → 0x12D7B 聚焦該記錄 → [0x51A83]=1 → 0x1956B 開框 → 0x1E7 →
// 0x1E5C0(10) → 0x196CB 關框），再逐筆到期提示（0x1A9AA 同一套聚焦後開框，FDTXT
// 0x1E1..0x1E6）。所有到期幀的素材與肖像在發布交易前先預檢；缺素材時 HP／狀態交易與
// 玩家可見回饋都不變。sub_1DB65 的狀態致死動畫仍未由這裡承擔。
func (g *Game) beginNativeTransientPhases(selectors []byte, then func()) error {
	if g == nil || g.nativeClassUIJob != nil || g.transientUI || g.nativeLevelUpDialogue != nil {
		return errors.New("native transient presentation: another indexed owner is active")
	}
	candidate, expired, damage, err := g.buildNativeTransientPhases(selectors...)
	if err != nil {
		return err
	}
	if len(expired) == 0 && len(damage) == 0 {
		g.adoptNativeStateCandidate(candidate)
		if then != nil {
			then()
		}
		return nil
	}
	if len(expired) != 0 && (g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200) {
		return errors.New("native transient presentation: indexed source assets are unavailable")
	}
	type expiryPresentation struct {
		unit         *battle.Unit
		counterIndex int
		portrait     dato.Frame
	}
	expiries := make([]expiryPresentation, 0, len(expired))
	for _, event := range expired {
		unit, counterIndex, err := nativeTransientPresentationEvent(event)
		if err != nil {
			return err
		}
		portraits, err := loadNativeSeparatedPortrait(unit.BattleFig)
		if err != nil || len(portraits) == 0 {
			return errors.New("native transient presentation: DATO portrait is unavailable")
		}
		// 先合成一次確認字串與框素材齊全；正式幀在聚焦重繪後重合成。
		if _, err := campaign.ComposeNativeTransientExpiryFrame(
			g.nativeMapVGA, g.nativeClassUI.dialogue, portraits[0],
			g.nativeClassUI.strings, g.nativeClassUI.font, counterIndex,
		); err != nil {
			return err
		}
		expiries = append(expiries, expiryPresentation{unit: unit, counterIndex: counterIndex, portrait: portraits[0]})
	}
	damagePages := make([][][][]uint16, len(damage))
	damageWaits := make([]bool, len(damage))
	for index, event := range damage {
		if event.Unit == nil || !event.Unit.HasBattleFig || !event.Unit.HasNativeMapPresentation {
			return errors.New("native transient presentation: damaged record has no DATO selector or map presentation")
		}
		if g.nativePreparationUI == nil || g.nativePreparationUI.status.Strings == nil {
			return errors.New("native transient presentation: FDTXT_000 strings are unavailable")
		}
		pages, wait, err := nativeMessagePages(
			g.nativePreparationUI.status.Strings, nativeTransientDamageTextIndex, event.Amount,
		)
		if err != nil {
			return err
		}
		damagePages[index], damageWaits[index] = pages, wait
	}
	remap := g.adoptNativeStateCandidate(candidate)
	resolve := func(unit *battle.Unit) *battle.Unit {
		if current, ok := remap[unit]; ok {
			return current
		}
		return unit
	}
	g.msg = ""
	var runExpiry func(int)
	runExpiry = func(index int) {
		if index >= len(expiries) {
			g.transientUI = false
			if then != nil {
				then()
			}
			return
		}
		presentation := expiries[index]
		if err := g.focusNativeTransientRecord(resolve(presentation.unit)); err != nil {
			g.transientUI = false
			g.loadErr = "native transient presentation: " + err.Error()
			return
		}
		source := append([]byte(nil), g.nativeMapVGA...)
		final, err := campaign.ComposeNativeTransientExpiryFrame(
			source, g.nativeClassUI.dialogue, presentation.portrait,
			g.nativeClassUI.strings, g.nativeClassUI.font, presentation.counterIndex,
		)
		var opening, closing [][]byte
		if err == nil {
			opening, err = campaign.NativeClassListOpeningFrames(source, final)
		}
		if err == nil {
			closing, err = campaign.NativeClassListClosingFrames(source, final)
		}
		if err != nil {
			g.transientUI = false
			g.loadErr = "native transient presentation: " + err.Error()
			return
		}
		frames := append(opening, closing...)
		g.transientUI = true
		g.nativeClassUIJob = &nativeClassUIJob{
			frames:  frames,
			restore: source,
			after:   func() { runExpiry(index + 1) },
		}
	}
	var runDamage func(int)
	runDamage = func(index int) {
		if index >= len(damage) {
			runExpiry(0)
			return
		}
		unit := resolve(damage[index].Unit)
		if err := g.focusNativeTransientRecord(unit); err != nil {
			g.loadErr = "native transient presentation: " + err.Error()
			return
		}
		if err := g.beginNativeTimedMessageDialogue(
			unit, nativeTransientDamageTextIndex, damagePages[index], damageWaits[index],
			nativeTransientDamageCloseFrames, func() { runDamage(index + 1) },
		); err != nil {
			g.loadErr = "native transient presentation: " + err.Error()
		}
	}
	if len(damage) == 0 {
		// 到期演出是同步的：呼叫端看 transientUI 等它結束。
		g.transientUI = true
	}
	runDamage(0)
	return nil
}

// nativeTransientPresentationActive 回報 sub_1A866 的扣血訊息或到期提示還沒播完。
func (g *Game) nativeTransientPresentationActive() bool {
	return g != nil && (g.transientUI || g.nativeLevelUpDialogue != nil)
}

// focusNativeTransientRecord 是 0x1A8D4／0x1A9AA 的聚焦：[0x51A83]=0 → 0x12D7B(record) →
// [0x51A83]=1。聚焦改到視圖（游標、鏡頭或 HUD anchor）時整幀重繪，讓之後的框貼在新畫面上。
func (g *Game) focusNativeTransientRecord(unit *battle.Unit) error {
	if g == nil || g.st == nil || unit == nil {
		return errors.New("focus record is unavailable")
	}
	if !g.st.HasNativeMapViewState {
		return nil
	}
	viewBefore, hudBefore := g.st.NativeMapViewState, g.st.NativeMapHUDState
	g.st.MaterializeNativeMapRangeMode(0)
	g.aiFocusCursor(unit.X, unit.Y)
	g.st.MaterializeNativeMapRangeMode(1)
	if g.st.NativeMapViewState == viewBefore && g.st.NativeMapHUDState == hudBefore &&
		len(g.nativeMapVGA) == 320*200 {
		return nil
	}
	return g.composeNativeMapFrame()
}

func nativeTransientPresentationEvent(event battle.NativeTransientExpiry) (*battle.Unit, int, error) {
	if event.Unit == nil || !event.Unit.HasBattleFig {
		return nil, 0, errors.New("native transient presentation: raw DATO selector is unavailable")
	}
	counterIndex := event.Offset - battle.NativeTransientOffset
	if counterIndex < 0 || counterIndex >= battle.NativeTransientCount {
		return nil, 0, errors.New("native transient presentation: raw counter offset is invalid")
	}
	return event.Unit, counterIndex, nil
}
