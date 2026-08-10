package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// startNativeAIMode11 owns the direct two-stage dispatcher after the battle
// package has selected raw producers.  It never asks NextAIPlan between
// stages, so 0x15311 → 0x1548E/0x14121 → 0x13FD4 remains one ordered owner.
func (g *Game) startNativeAIMode11(plan *battle.AIPlan) {
	if g == nil {
		return
	}
	if plan == nil || plan.U == nil || len(plan.NativeMode11Stages) == 0 {
		g.loadErr = "native AI mode 11: executable stages are unavailable"
		g.aiBusy = false
		return
	}
	g.runNativeAIMode11Stage(plan, 0)
}

func (g *Game) runNativeAIMode11Stage(plan *battle.AIPlan, index int) {
	if g == nil || plan == nil || plan.U == nil {
		return
	}
	if g.result != "" {
		g.finishSuccessfulUnitAction(plan.U, nil)
		return
	}
	if index >= len(plan.NativeMode11Stages) {
		g.finishSuccessfulUnitAction(plan.U, nil)
		return
	}
	stage := plan.NativeMode11Stages[index]
	next := func() { g.runNativeAIMode11Stage(plan, index+1) }
	if stage.Recovery != nil {
		if err := g.beginNativeAIIdleRecovery(plan.U, *stage.Recovery, next); err != nil {
			g.loadErr = "native AI mode 11 0x13fd4: " + err.Error()
			g.aiBusy = false
		}
		return
	}
	action := stage.Action
	if action == nil {
		// 0x14121 returned zero but 0x13FD4's HP/gate preflight rejected;
		// the original common tail performs no action.
		next()
		return
	}
	if action.NativeModeWriteRangeZero {
		g.st.NativeMapRangeMode = 0
		g.st.HasNativeMapRangeModeState = true
	}
	act := func() {
		if action.NativeActionKind == battle.NativeAIActionCommand ||
			action.NativeActionKind == battle.NativeAIActionItem {
			if err := g.executeNativeAIActionWithContinuation(action, next); err != nil {
				g.loadErr = "native AI mode 11 action: " + err.Error()
				g.aiBusy = false
			}
			return
		}
		if action.NativeActionKind != battle.NativeAIActionPhysical {
			g.finishSuccessfulUnitAction(action.U, next)
			return
		}
		g.executeNativeAIMode11Physical(action, next)
	}
	if len(action.Path) >= 2 {
		g.walk = &walkAnim{u: action.U, path: action.Path, then: act}
		return
	}
	act()
}

// executeNativeAIMode11Physical is the 0x1548E selected-result presentation
// owner.  It deliberately reuses the same typed damage and FIGANI schedule as
// the already closed 0x14EF0 physical route; the route label remains raw.
func (g *Game) executeNativeAIMode11Physical(plan *battle.AIPlan, after func()) {
	if g == nil || plan == nil || plan.U == nil || plan.Target == nil || !plan.Target.Alive() {
		g.loadErr = "native AI mode 11 0x1548e: raw target is unavailable"
		g.aiBusy = false
		return
	}
	actor, target := plan.U, plan.Target
	actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
	defHP0 := target.HP
	attackResult, err := g.resolvePhysicalAttack(actor, target)
	if err != nil {
		g.loadErr = "native AI mode 11 0x1548e: " + err.Error()
		g.aiBusy = false
		return
	}
	g.awardDeathReward(target, actor)
	g.msg = playerPhysicalAttackMessage(actor, target, attackResult)
	attackerName, defenderName := actor.Name, target.Name
	if attackerName == "" {
		attackerName = actor.ClsName
	}
	if defenderName == "" {
		defenderName = target.ClsName
	}
	g.atk = g.newAtkAnim(actor.BattleFig, target.BattleFig,
		attackerName, defenderName, actor.HP, actor.MaxHP, actor.Lv, actor.MP,
		target.Lv, target.MP, defHP0, target.HP, target.MaxHP,
		g.terrainAt(target.X, target.Y), actor.Camp == battle.Own)
	if g.atk == nil {
		g.loadErr = fmt.Sprintf("native AI mode 11 0x1548e FIGANI unavailable: %d -> %d", actor.BattleFig, target.BattleFig)
		g.aiBusy = false
		return
	}
	g.atk.after = func() {
		g.finishSuccessfulUnitAction(actor, after)
	}
	g.checkResult()
}
