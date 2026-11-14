package main

import "github.com/wicanr2/fd2_re/remake/internal/battle"

// 玩家指令的經驗發放。原版玩家控制器 0x117E7 在 0x18890（玩家行動）返回後才
// 0x11959 壓 99、0x1196D 呼叫 0x1E292 發經驗；物理攻擊在 resolvePhysicalAttackFull
// 由 battle 結算直接發放，這裡處理其餘指令家族。各家族的累加量與寫入端位址見
// internal/battle/native_command_experience.go。
//
// 時機和物理攻擊相同：在指令結果回呼裡、finishSuccessfulUnitAction 之前發放並排入
// 升級對話，對話在死亡獎勵訊息與死亡程式之後由收尾流程播出。

// awardNativePlayerCommandExperience 把 delta 交給 State.AwardNativePlayerActionExperience，
// 寫回全域亂數並排入經驗／升級對話。只處理有原版 +6 == 2 的玩家單位；累加量算不出來
// （高階職業表缺件）時失敗即關閉，寫 loadErr 且不發經驗。
func (g *Game) awardNativePlayerCommandExperience(actor *battle.Unit, delta int, err error, targets []*battle.Unit) {
	if g == nil || g.st == nil || actor == nil || !actor.HasNativeRecordByte6 || actor.NativeRecordByte6 != 2 {
		return
	}
	if err != nil {
		g.loadErr = err.Error()
		return
	}
	got, ups, next := g.st.AwardNativePlayerActionExperience(actor, delta, g.nativeRNGState, targets...)
	g.nativeRNGState = next
	g.queueNativeLevelUpDialogue(actor, got, ups)
}

// 指令 0～12（0x2A6BD → 0x1C75E → 0x1C81F）。
func (g *Game) awardNativeCommandDamageExperience(actor *battle.Unit, results []battle.NativeCommandDamageResult) {
	targets := make([]*battle.Unit, 0, len(results))
	for _, result := range results {
		targets = append(targets, result.Target)
	}
	exp, err := battle.NativeCommandDamageExperience(results)
	g.awardNativePlayerCommandExperience(actor, exp, err, targets)
}

// 指令 13～16（0x1C916）。
func (g *Game) awardNativeCommandHealExperience(actor *battle.Unit, results []battle.NativeCommandHealResult) {
	targets := make([]*battle.Unit, 0, len(results))
	for _, result := range results {
		targets = append(targets, result.Target)
	}
	g.awardNativePlayerCommandExperience(actor, battle.NativeCommandHealExperience(results), nil, targets)
}

// 指令 24、28～31（0x276EC → 0x1C81F）。
func (g *Game) awardNativeCommandDerivedStrikeExperience(actor *battle.Unit, results []battle.NativeCommand24Damage) {
	targets := make([]*battle.Unit, 0, len(results))
	for _, result := range results {
		targets = append(targets, result.Target)
	}
	exp, err := battle.NativeCommandDerivedStrikeExperience(results)
	g.awardNativePlayerCommandExperience(actor, exp, err, targets)
}

// 指令 17～19（0x22721／0x22866／0x22997）：累加量已在結果的 Accumulator。
func (g *Game) awardNativeCommandModifierExperience(actor *battle.Unit, result battle.NativeCommandModifierResult, targets []*battle.Unit) {
	g.awardNativePlayerCommandExperience(actor, result.Accumulator, nil, targets)
}

// 指令 20～22、25～27（0x22AF6／0x22D1B／0x22C04）。
func (g *Game) awardNativeCommand2022Experience(actor *battle.Unit, plan *battle.NativeAICommand2022Plan) {
	var targets []*battle.Unit
	if plan != nil {
		targets = plan.Targets
	}
	exp, err := battle.NativeCommand2022Experience(plan)
	g.awardNativePlayerCommandExperience(actor, exp, err, targets)
}

// 指令 32（0x2111A → 0x1C75E → 0x1C81F）。
func (g *Game) awardNativeCompoundCommand32Experience(actor *battle.Unit, result battle.NativeCompoundCommand32Result) {
	targets := make([]*battle.Unit, 0, len(result.Targets))
	for _, target := range result.Targets {
		targets = append(targets, target.Target)
	}
	exp, err := battle.NativeCompoundCommand32Experience(result)
	g.awardNativePlayerCommandExperience(actor, exp, err, targets)
}

// 指令 33（0x211A4 → 0x1C916）。
func (g *Game) awardNativeCompoundCommand33Experience(actor *battle.Unit, result battle.NativeCompoundCommand33Result) {
	targets := make([]*battle.Unit, 0, len(result.Targets))
	for _, target := range result.Targets {
		targets = append(targets, target.Target)
	}
	g.awardNativePlayerCommandExperience(actor, battle.NativeCompoundCommand33Experience(result), nil, targets)
}

// 指令 34（0x27FC9 → 0x22721／0x22866／0x22997）。
func (g *Game) awardNativeCompoundCommand34Experience(actor *battle.Unit, result battle.NativeCompoundCommand34Result) {
	var targets []*battle.Unit
	if len(result.StageStates) != 0 {
		for _, state := range result.StageStates[0] {
			targets = append(targets, state.Target)
		}
	}
	g.awardNativePlayerCommandExperience(actor, battle.NativeCompoundCommand34Experience(result), nil, targets)
}

// 指令 35（0x27FC9 → 三次 0x22D1B）。
func (g *Game) awardNativeCompoundCommand35Experience(actor *battle.Unit, result battle.NativeCompoundCommand35Result) {
	var targets []*battle.Unit
	if len(result.StageStates) != 0 {
		for _, state := range result.StageStates[0] {
			targets = append(targets, state.Target)
		}
	}
	exp, err := battle.NativeCompoundCommand35Experience(result)
	g.awardNativePlayerCommandExperience(actor, exp, err, targets)
}
