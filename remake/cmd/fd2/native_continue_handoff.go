package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// publishNativeContinueBattle is the application-owned half of the native
// CONTINUE boundary.  It consumes a state that has already passed all typed
// save/resource adapters and publishes it to Game in one shallow transaction;
// it never calls resetBattle or Scenario.Setup, so saved turn/runtime order is
// not replaced by an authored battle opening.
//
// The title caller is intentionally separate: it must provide the signed BIOS
// timer seed and the exact campaign node.  Until that caller is evidence
// complete, title CONTINUE remains fail-closed.
func (g *Game) publishNativeContinueBattle(
	input fdsave.ContinueRuntimeInput,
	state *battle.State,
	scenario *battle.Scenario,
	nodeID, unitsPath, scenarioPath string,
) error {
	if g == nil {
		return fmt.Errorf("native CONTINUE battle handoff: nil game")
	}
	if err := campaign.ValidateNativeContinueBattleHandoff(
		input, state, scenario,
	); err != nil {
		return err
	}
	if g.m == nil || g.m.W != state.W || g.m.H != state.H {
		return fmt.Errorf("native CONTINUE battle handoff: map asset is unavailable")
	}
	if g.camp == nil || g.camp.C == nil {
		return fmt.Errorf("native CONTINUE battle handoff: campaign graph is unavailable")
	}
	node := g.camp.C.Nodes[nodeID]
	if node == nil || node.Type != "battle" {
		return fmt.Errorf("native CONTINUE battle handoff: node %q is not a battle", nodeID)
	}
	if unitsPath == "" || scenarioPath == "" ||
		node.Units != unitsPath || node.Scenario != scenarioPath {
		return fmt.Errorf("native CONTINUE battle handoff: campaign asset identity mismatch")
	}

	runner := *g.camp
	runner.Flags = make(map[string]bool, len(g.camp.Flags))
	for key, value := range g.camp.Flags {
		runner.Flags[key] = value
	}
	runner.Cur = nodeID

	candidate := *g
	candidate.camp = &runner
	candidate.st = state
	candidate.sc = scenario
	candidate.titlePhase = ""
	candidate.storyBG = false
	candidate.storyActors = nil
	candidate.storyRoster = nil
	candidate.storyCompositionEventBytes = nil
	candidate.storySpawned = nil
	candidate.storyRosterPath = ""
	candidate.storyPartyScenario = ""
	candidate.storyNativeMapView = battle.NativeMapViewState{}
	candidate.hasStoryNativeMapView = false
	candidate.dialog = nil
	candidate.dlgShown, candidate.dlgPhase, candidate.dlgT = dlgNone, 0, 0
	candidate.dlgUpper = nil
	candidate.dlgPage, candidate.dlgScrollT, candidate.dlgScrollFrom = 0, 0, 0
	candidate.beats, candidate.beatIdx, candidate.beatDelay = nil, -1, 0
	candidate.camPan, candidate.focusJob, candidate.actJob = nil, nil, nil
	candidate.fade, candidate.transitionReveal = nil, nil
	candidate.indexedTransition, candidate.spawnIntroTransition = nil, nil
	candidate.nativeTurnStaging = nil
	candidate.nativeFieldEvent61 = nil
	candidate.nativeAIIdleRecovery = nil
	candidate.battleEvent, candidate.battleEventDelay = nil, 0
	candidate.atk, candidate.walk = nil, nil
	candidate.sel, candidate.reach = nil, nil
	candidate.moved, candidate.result = false, ""
	candidate.msg, candidate.loadErr = "", ""
	candidate.nativeChapterRestore = nil
	candidate.nativeMapWork, candidate.nativeMapVGA = nil, nil
	candidate.nativeMapClock.Reset()
	candidate.resetActionOverlayLifecycle()
	if !candidate.syncNativeMapView() {
		return fmt.Errorf("native CONTINUE battle handoff: map view publication failed")
	}
	candidate.prevCurX, candidate.prevCurY = candidate.curX, candidate.curY
	*g = candidate
	return nil
}
