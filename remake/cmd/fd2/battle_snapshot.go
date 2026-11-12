// battle_snapshot.go — 戰鬥中 F5 的 JSON 戰場快照（56「2026-11-13 #303 戰場JSON快照」）。
//
// 讀檔先走正常的戰鬥節點進場，讓所有由資產與劇本綁定的表與一般路徑相同；再以
// 快照覆寫可變狀態，清掉開場劇本產生的暫態。快照不是原版 FD2.SAV ABI，戰鬥中也
// 不寫原版章節槽。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

const battleSnapshotVersion = 1

type battleSnapshot struct {
	Version  int    `json:"version"`
	Node     string `json:"node"`
	Units    string `json:"units"`
	Scenario string `json:"scenario"`
	// MapSHA256 綁定讀檔時重建的地圖；第 22 章這類中途換圖不符就拒絕。
	MapSHA256 string `json:"map_sha256"`

	State       battle.StateSnapshot   `json:"state"`
	FiredEvents []battle.FiredEventRef `json:"fired_events,omitempty"`

	NativeRNGState uint16 `json:"native_rng_state"`
	CursorX        int    `json:"cursor_x"`
	CursorY        int    `json:"cursor_y"`
	// NativeMapClockTick 是地圖動畫計時的 BIOS low word；nil 代表時鐘還沒取樣。
	NativeMapClockTick *int `json:"native_map_clock_tick"`
}

// battleSnapshotBlocker 回傳不能存快照的原因；空字串代表我方可操作的地圖待機。
func (g *Game) battleSnapshotBlocker() string { return g.battleSnapshotBlockerFor(false) }

// battleSnapshotBlockerFor 的 fromSystemMenu 只放行戰鬥系統選單本身（游標空格開啟
// 的 sub_16F55／巢狀 sub_19DF7）。選單不屬於戰場狀態，還原後回到地圖待機。
func (g *Game) battleSnapshotBlockerFor(fromSystemMenu bool) string {
	systemMenu := fromSystemMenu && g != nil && g.nativeSystemCursorOverlay && g.ring && g.sel == nil
	switch {
	case g == nil || g.camp == nil || g.st == nil:
		return "no battle"
	case g.camp.Node() == nil || g.camp.Node().Type != "battle":
		return "not a battle node"
	case g.result != "":
		return "battle result pending"
	case g.aiBusy || g.nativeAIIdleRecovery != nil:
		return "enemy phase"
	case g.battleEvent != nil || g.battleEventDelay != 0 || g.nativeTurnStaging != nil ||
		g.nativeFieldEvent61 != nil || len(g.beats) != 0:
		return "battle event"
	case len(g.dialog) != 0 || g.nativeLevelUpDialogue != nil || len(g.pendingNativeLevelUps) != 0:
		return "dialogue"
	case g.walk != nil || g.atk != nil || g.nativeMovePlan != nil || g.sel != nil || g.moved:
		return "unit action"
	case !systemMenu && (g.ring || g.nativeSystemCursorOverlay):
		return "menu open"
	case g.nativeCommandOpen || g.itemOpen || g.nativeItemTargeting || g.nativeCommand0Targeting ||
		g.nativeSystemEndTurnUI != nil || g.nativeClassUIJob != nil ||
		g.transientUI || g.nativePlayerStatus != nil:
		return "menu open"
	case g.spawnIntroTransition != nil || g.indexedTransition != nil || g.transitionReveal != nil ||
		g.nativeUnitPresent != nil || g.actJob != nil || g.camPan != nil || g.focusJob != nil ||
		g.nativePlayerFocus != nil ||
		g.fade != nil || g.bannerT > 0 || g.nativeCh22Reload != nil || g.nativeCh23Loop != nil:
		return "presentation"
	case g.nativeEndTurnRecovery != nil:
		return "END recovery"
	case g.sc != nil && g.sc.PendingJoinCount() != 0:
		return "pending JOIN"
	}
	if err := g.st.SnapshotReady(); err != nil {
		return err.Error()
	}
	return ""
}

func (g *Game) captureBattleSnapshot(fromSystemMenu bool) (*battleSnapshot, error) {
	if reason := g.battleSnapshotBlockerFor(fromSystemMenu); reason != "" {
		return nil, fmt.Errorf("battle snapshot: %s", reason)
	}
	state, err := g.st.Snapshot()
	if err != nil {
		return nil, err
	}
	node := g.camp.Node()
	var clock *int
	if tick, ok := g.nativeMapClock.Current(); ok {
		clock = &tick
	}
	return &battleSnapshot{
		NativeMapClockTick: clock,
		Version: battleSnapshotVersion,
		Node:    g.camp.Cur, Units: node.Units, Scenario: node.Scenario,
		MapSHA256:      g.battleSnapshotMapDigest(),
		State:          state,
		FiredEvents:    g.sc.FiredEvents(),
		NativeRNGState: g.nativeRNGState,
		CursorX:        g.curX, CursorY: g.curY,
	}, nil
}

func (g *Game) battleSnapshotMapDigest() string {
	if g.m == nil {
		return ""
	}
	h := sha256.New()
	var word [8]byte
	binary.LittleEndian.PutUint32(word[0:4], uint32(g.m.W))
	binary.LittleEndian.PutUint32(word[4:8], uint32(g.m.H))
	h.Write(word[:])
	for _, tile := range g.m.Tiles {
		binary.LittleEndian.PutUint32(word[0:4], uint32(tile))
		h.Write(word[0:4])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// validateBattleSnapshotNode 在動到任何 Game 狀態前檢查快照與戰役圖相符。
func (g *Game) validateBattleSnapshotNode(snap *battleSnapshot, node string) error {
	if snap.Version != battleSnapshotVersion {
		return fmt.Errorf("戰場快照版本 %d 不支援", snap.Version)
	}
	n := g.camp.C.Nodes[node]
	if snap.Node != node || n == nil || n.Type != "battle" {
		return errors.New("戰場快照的節點不是存檔節點的戰鬥")
	}
	if n.Units != snap.Units || n.Scenario != snap.Scenario {
		return errors.New("戰場快照的單位或劇本資產與戰役圖不符")
	}
	return nil
}

// applyBattleSnapshot 在 enterNode 已正常建好同一戰鬥節點後覆寫可變狀態。
// 全部檢查在私有複本上完成，失敗時 g 保持剛進場的狀態。
func (g *Game) applyBattleSnapshot(snap *battleSnapshot) error {
	if g.st == nil || g.camp == nil || g.camp.Cur != snap.Node {
		return errors.New("戰場快照：戰鬥節點沒有建立")
	}
	if digest := g.battleSnapshotMapDigest(); digest != snap.MapSHA256 {
		return errors.New("戰場快照：地圖與存檔時不同")
	}
	if snap.State.W != g.m.W || snap.State.H != g.m.H {
		return errors.New("戰場快照：戰場尺寸與地圖不符")
	}
	state := *g.st
	if err := state.RestoreSnapshot(snap.State); err != nil {
		return fmt.Errorf("戰場快照：%w", err)
	}
	var scenario *battle.Scenario
	if g.sc != nil {
		copySC := *g.sc
		copySC.Events = append([]battle.Event(nil), g.sc.Events...)
		scenario = &copySC
	}
	if err := scenario.RestoreFiredEvents(snap.FiredEvents); err != nil {
		return fmt.Errorf("戰場快照：%w", err)
	}

	candidate := *g
	candidate.st = &state
	candidate.sc = scenario
	candidate.bindNativeDeathPrograms()
	candidate.clearBattleSnapshotTransients()
	candidate.nativeRNGState = snap.NativeRNGState
	candidate.curX, candidate.curY = snap.CursorX, snap.CursorY
	candidate.nativeMapClock.Reset()
	if snap.NativeMapClockTick != nil && !candidate.nativeMapClock.RestoreTick(*snap.NativeMapClockTick) {
		return errors.New("戰場快照：地圖計時超出範圍")
	}
	if candidate.st.HasNativeMapViewState {
		if !candidate.syncNativeMapView() {
			return errors.New("戰場快照：地圖視角無法發布")
		}
	}
	candidate.prevCurX, candidate.prevCurY = candidate.curX, candidate.curY
	if reason := candidate.battleSnapshotBlocker(); reason != "" {
		return fmt.Errorf("戰場快照：還原後不是可操作狀態（%s）", reason)
	}
	*g = candidate
	return nil
}

// clearBattleSnapshotTransients 丟棄進場劇本與開場演出產生的暫態；沿 CONTINUE
// 發布清單（publishNativeContinueBattle）。
func (g *Game) clearBattleSnapshotTransients() {
	g.dialog = nil
	g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
	g.dlgUpper = nil
	g.dlgPage, g.dlgScrollT, g.dlgScrollFrom = 0, 0, 0
	g.beats, g.beatIdx, g.beatDelay = nil, -1, 0
	g.camPan, g.focusJob, g.actJob = nil, nil, nil
	g.fade, g.transitionReveal = nil, nil
	g.indexedTransition, g.spawnIntroTransition = nil, nil
	g.nativeTurnStaging = nil
	g.nativeFieldEvent61 = nil
	g.nativeAIIdleRecovery = nil
	g.nativeUnitPresent = nil
	g.battleEvent, g.battleEventDelay = nil, 0
	g.atk, g.walk = nil, nil
	g.sel, g.reach = nil, nil
	g.moved, g.result = false, ""
	g.aiBusy = false
	g.banner, g.bannerT = "", 0
	g.nativeLevelUpDialogue = nil
	g.pendingNativeLevelUps = nil
	g.nativePlayerFocus, g.nativePlayerFocusStarted = nil, false
	g.nativeMapWork, g.nativeMapVGA = nil, nil
	g.resetActionOverlayLifecycle()
}
