package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// executeNativeAIAction consumes only the route selected by the raw
// 0x14ef0 bridge.  Recovered indexed owners are mandatory where the original
// caller consumes them; no normalized spell or guessed item effect is
// substituted when a command ID is outside the recovered families.
func (g *Game) executeNativeAIAction(plan *battle.AIPlan) error {
	return g.executeNativeAIActionWithContinuation(plan, nil)
}

// executeNativeAIActionWithContinuation reuses the recovered command/item
// owner for mode 11 without prematurely handing control back to NextAIPlan.
// The continuation runs only after the existing successful-action boundary
// (including selector-1 field-event ownership) has completed.
func (g *Game) executeNativeAIActionWithContinuation(plan *battle.AIPlan, after func()) error {
	if g == nil || g.st == nil || plan == nil || plan.U == nil {
		return fmt.Errorf("native AI action context unavailable")
	}
	actor := plan.U
	target := plan.Target
	switch plan.NativeActionKind {
	case battle.NativeAIActionCommand:
		if target == nil {
			return fmt.Errorf("native command %d has no raw target", plan.NativeCommandID)
		}
		id := plan.NativeCommandID
		var message string
		damageTargets := make([]*battle.Unit, 0)
		switch {
		case id == 0:
			return g.startNativeCommand0Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 0：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 1:
			return g.startNativeCommand1Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 1：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 2:
			return g.startNativeCommand2Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 2：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 3:
			return g.startNativeCommand3Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 3：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 4:
			return g.startNativeCommand4AIPresentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 4：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 5:
			return g.startNativeCommand5Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 5：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 7:
			return g.startNativeCommand7Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 7：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 8:
			return g.startNativeCommand8Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 8：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 9:
			return g.startNativeCommand9AIPresentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 9：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id >= 10 && id <= 12:
			return g.startNativeCommand1012Presentation(actor, target, id, func(results []battle.NativeCommandDamageResult) {
				for _, result := range results {
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 %d：完成敵方 indexed 演出", id)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 6:
			return g.startNativeCommand6Presentation(actor, target, func(results []battle.NativeCommandDamageResult) {
				hit, total := 0, 0
				for _, result := range results {
					if result.Hit {
						hit++
						total += result.Damage
					}
					g.awardDeathReward(result.Target, actor)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 6：命中 %d，傷害 %d", hit, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id >= 13 && id <= 16:
			targets, err := g.st.NativeAICommandHealTargets(actor, id)
			if err != nil {
				return err
			}
			return g.startNativeCommandHealPresentation(id, targets, func() ([]battle.NativeCommandHealResult, error) {
				return g.st.ExecuteNativeAICommandHeal(actor, id, g.rng)
			}, func(results []battle.NativeCommandHealResult) {
				total := 0
				for _, result := range results {
					total += result.Restore.Actual
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 %d：回復 %d", id, total)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id >= 17 && id <= 19:
			return g.startNativeAICommandModifierPresentation(actor, id, func(result battle.NativeCommandModifierResult) {
				count := len(result.WordSteps)
				if id == 19 {
					count = len(result.PairSteps)
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 %d：完成敵方 indexed modifier (%d targets)", id, count)
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 20 || id == 21:
			return g.startNativeAICommand2022Presentation(actor, id, func(result *battle.NativeAICommand2022Plan) {
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 %d：完成敵方 indexed 尾段 (%d targets)", id, len(result.Results))
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 22:
			return g.startNativeAICommand2022Presentation(actor, id, func(result *battle.NativeAICommand2022Plan) {
				for _, entry := range result.Results {
					if entry.Apply != nil && entry.Apply.Damage.Actual > 0 {
						g.awardDeathReward(entry.Target, actor)
					}
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 22：完成敵方 indexed 尾段 (%d targets)", len(result.Results))
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 26 || id == 27:
			return g.startNativeAICommand2022Presentation(actor, id, func(result *battle.NativeAICommand2022Plan) {
				for _, entry := range result.Results {
					if entry.Apply != nil && entry.Apply.Damage.Actual > 0 {
						g.awardDeathReward(entry.Target, actor)
					}
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
				g.msg = fmt.Sprintf("原始指令 %d：完成敵方 indexed 尾段 (%d targets)", id, len(result.Results))
				g.finishSuccessfulUnitAction(actor, after)
				g.checkResult()
			})
		case id == 24 || id == 28 || id == 29 || id == 31:
			results, err := g.st.ExecuteNativeCommandDerivedStrike(actor, target, id, g.rng)
			if err != nil {
				return err
			}
			total := 0
			for _, result := range results {
				total += result.Damage
				damageTargets = append(damageTargets, result.Target)
			}
			message = fmt.Sprintf("原始指令 %d：傷害 %d", id, total)
		case id == 25:
			return fmt.Errorf("native AI command 25 has no confirmed normal producer")
		default:
			return fmt.Errorf("native AI command executor unavailable id=%d", id)
		}
		for _, dead := range damageTargets {
			g.awardDeathReward(dead, actor)
		}
		actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))
		g.msg = message
		g.finishSuccessfulUnitAction(actor, after)
		g.checkResult()
		return nil

	case battle.NativeAIActionItem:
		if target == nil {
			return fmt.Errorf("native item %d has no raw target", plan.NativeItemID)
		}
		if len(g.nativeItemEffectRows) == 0 {
			rows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
			if err != nil {
				return fmt.Errorf("native AI item rows: %w", err)
			}
			g.nativeItemEffectRows = rows
		}
		started, err := g.startNativeTargetItemPresentation(plan, func() {
			g.msg = fmt.Sprintf("原始物品 %02Xh：完成完整目標清單效果", plan.NativeItemID)
			g.finishSuccessfulUnitAction(actor, after)
			g.checkResult()
		})
		if err != nil {
			return err
		}
		if started {
			return nil
		}
		if err := g.applyNativeAITargetItem(plan); err != nil {
			return err
		}
		g.msg = fmt.Sprintf("原始物品 %02Xh：完成完整目標清單效果", plan.NativeItemID)
		g.finishSuccessfulUnitAction(actor, after)
		g.checkResult()
		return nil
	default:
		return fmt.Errorf("native AI action kind %d is not executable", plan.NativeActionKind)
	}
}
