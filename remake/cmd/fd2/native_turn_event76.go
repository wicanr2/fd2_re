package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func event76DialogueActions(g *Game, textIndex int) ([]battle.Action, bool) {
	sceneIndex, line, count := 0, 0, 0
	switch textIndex {
	case 2:
		line, count = 6, 3
	case 3:
		line, count = 9, 1
	case 4:
		line, count = 10, 1
	case 5:
		line, count = 11, 1
	case 6:
		sceneIndex, line, count = 1, 0, 5
	default:
		return nil, false
	}
	lines := loadStoryScriptAt("assets/story/ch29.json", "", &sceneIndex)
	if line < 0 || count <= 0 || line+count > len(lines) {
		return nil, false
	}
	actions := make([]battle.Action, 0, count)
	for _, source := range lines[line : line+count] {
		if source.SpeakerSlot != nil {
			return nil, false
		}
		text, err := g.localizedStoryText(source)
		if err != nil {
			g.loadErr = "event76 locale: " + err.Error()
			return nil, false
		}
		actions = append(actions, battle.Action{Type: "dialogue", Speaker: source.Speaker, Text: text})
	}
	return actions, true
}

func nativeEvent76PulseSpec() campaign.NativePalettePulse {
	return campaign.NativePalettePulse{
		RiseStart: 0, RiseEnd: 63, RiseDelayMs: 8,
		HoldMs:    400,
		FallStart: 62, FallEnd: 0, FallDelayMs: 8,
	}
}

func (g *Game) preflightNativeEvent76FinalState(event battle.NativeTurnEvent) (*battle.State, error) {
	plan, err := battle.PlanNativeTurnEvent76(g.st, event)
	if err != nil || !plan.Final {
		return nil, fmt.Errorf("event76 final plan unavailable: %v", err)
	}
	candidate, err := cloneNativeTurnStagingState(g.st)
	if err != nil {
		return nil, err
	}
	p := event.Progression
	base := len(candidate.Units)
	n, err := candidate.AppendGroupWithNativePlacement(p.SpawnGroup, byte(p.RawPlacementGate))
	if err != nil || n != p.SpawnCount || base > 0xff {
		return nil, fmt.Errorf("event76 group%d append=%d base=%d err=%v", p.SpawnGroup, n, base, err)
	}
	activation := p.FinalActivation
	row := candidate.NativeTurnEventControls[activation.Slot]
	if row != (battle.NativeTurnEventControl{Turn: 0xff, EventID: byte(activation.EventID), RawCamp: activation.RawCamp}) {
		return nil, fmt.Errorf("event76 dormant event79 row identity mismatch")
	}
	candidate.NativeEventState[p.BaseStateIndex] = byte(base)
	candidate.NativeTurnEventControls[activation.Slot].Turn = byte(candidate.NativeRoundCounter)
	if candidate.HasNativeFieldControlState {
		offset := 3 + activation.Slot*3
		if len(candidate.NativeFieldControlRaw) <= offset+2 || candidate.NativeFieldControlRaw[offset] != 0xff ||
			candidate.NativeFieldControlRaw[offset+1] != byte(activation.EventID) ||
			candidate.NativeFieldControlRaw[offset+2] != activation.RawCamp {
			return nil, fmt.Errorf("event76 raw event79 row disagrees with typed controls")
		}
		candidate.NativeFieldControlRaw[offset] = byte(candidate.NativeRoundCounter)
	}
	return candidate, nil
}

func (g *Game) preflightNativeEvent76Final(event battle.NativeTurnEvent) (*battle.State, []byte, error) {
	candidate, err := g.preflightNativeEvent76FinalState(event)
	if err != nil {
		return nil, nil, err
	}
	if g.nativePalettePulse != nil || g.nativePaletteRamp != nil || g.nativeUnitPresent != nil ||
		g.transitionReveal != nil || g.indexedTransition != nil ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		return nil, nil, fmt.Errorf("event76 indexed palette presenter unavailable")
	}
	probe := *g
	probe.st = candidate
	probe.nativeMapWork, probe.nativeMapVGA = nil, nil
	if err := probe.composeNativeMapFrame(); err != nil {
		return nil, nil, fmt.Errorf("event76 group1 indexed frame: %w", err)
	}
	if len(probe.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return nil, nil, fmt.Errorf("event76 indexed frame size=%d", len(probe.nativeMapVGA))
	}
	return candidate, append([]byte(nil), probe.nativeMapVGA...), nil
}

// startNativeRawCamp2TurnEvents 是 round increment 後、玩家重新取得輸入前的
// sub_1A813(2) owner。缺少唯一 typed consumer 時維持失敗即關閉。
func (g *Game) startNativeRawCamp2TurnEvents(then func()) (bool, error) {
	if g.sc == nil || len(g.sc.NativeTurnEvents) == 0 {
		return false, nil
	}
	events, err := g.sc.NativeTurnEventsAt(g.st, 2)
	if err != nil {
		return false, err
	}
	if len(events) == 0 {
		return false, nil
	}
	if len(events) != 1 || events[0].EventID != 76 {
		return false, fmt.Errorf("raw camp2 has %d handlers without a unique event76 adapter", len(events))
	}
	event := events[0]
	plan, err := battle.PlanNativeTurnEvent76(g.st, event)
	if err != nil {
		return false, err
	}
	if !plan.Final {
		if err := battle.CommitNativeTurnEvent76Repeat(g.st, plan); err != nil {
			return false, err
		}
		if then != nil {
			then()
		}
		return true, nil
	}
	if _, _, err := g.preflightNativeEvent76Final(event); err != nil {
		return false, err
	}
	actions, ok := event76DialogueActions(g, event.Progression.FinalTextIndex)
	if !ok {
		return false, fmt.Errorf("event76 FDTXT_029 index2 unavailable")
	}
	g.startBattleEvent(actions, func() {
		candidate, frame, err := g.preflightNativeEvent76Final(event)
		if err != nil {
			g.loadErr = err.Error()
			return
		}
		*g.st = *candidate
		g.nativeMapVGA = frame
		g.runNativeEvent76Presentation(event, 0, then)
	})
	return true, nil
}

func (g *Game) runNativeEvent76Presentation(event battle.NativeTurnEvent, pulse int, then func()) {
	p := event.Progression
	if p == nil || pulse < 0 || pulse > p.PulseCount {
		g.loadErr = "event76 presentation state unavailable"
		return
	}
	if pulse == p.PulseCount {
		if then != nil {
			then()
		}
		return
	}
	err := g.startNativePalettePulse(nativeEvent76PulseSpec(), func() {
		next := func() { g.runNativeEvent76Presentation(event, pulse+1, then) }
		if pulse < 2 {
			g.startBattleEvent([]battle.Action{{Type: "delay", Ms: p.ExtraDelayMS}}, next)
			return
		}
		actions, ok := event76DialogueActions(g, p.TailTextIndices[pulse-2])
		if !ok {
			g.loadErr = fmt.Sprintf("event76 tail dialogue %d unavailable", pulse-2)
			return
		}
		g.startBattleEvent(actions, next)
	})
	if err != nil {
		g.loadErr = "event76 palette pulse: " + err.Error()
	}
}
