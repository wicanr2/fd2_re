package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// executeAISpell 是正規化（normalized）的 AI 法術消費端。它刻意使用與玩家路徑
// 相同的可編輯 SpellBook／CastArea transaction，但不把原版 0x1598A 命令評分或
// indexed 法術演出混進本後備。目標、距離、RNG 或法術列缺失時，必須在標記行動完成前
// 停止。
func (g *Game) executeAISpell(plan *battle.AIPlan) error {
	if g == nil || g.st == nil || plan == nil || plan.U == nil || plan.Target == nil {
		return errors.New("法術計畫上下文缺失")
	}
	if plan.SpellID < 0 || g.rng == nil {
		return errors.New("法術 ID 或決定性亂數來源不可用")
	}
	actor := plan.U
	target := plan.Target
	if !actor.OnField || !actor.Alive() || actor.Acted || actor.Sealed || !target.OnField || !target.Alive() {
		return errors.New("施法者或目標已失效")
	}

	var spell *battle.Spell
	for i := range g.st.SpellBook {
		if g.st.SpellBook[i].ID == plan.SpellID {
			spell = &g.st.SpellBook[i]
			break
		}
	}
	if spell == nil {
		return fmt.Errorf("法術資料列 %d 不可用", plan.SpellID)
	}
	if actor.MP < spell.MP || spell.MP < 0 {
		return fmt.Errorf("法術資料列 %d 的 MP 不足", spell.ID)
	}
	validTarget := false
	for _, candidate := range g.st.AISpellCandidates(actor, *spell) {
		if candidate == target {
			validTarget = true
			break
		}
	}
	if !validTarget {
		return fmt.Errorf("法術資料列 %d 的目標不合法", spell.ID)
	}
	distance := absInt(actor.X-target.X) + absInt(actor.Y-target.Y)
	if distance > spell.Dist {
		return fmt.Errorf("法術資料列 %d 的目標超出距離", spell.ID)
	}

	results := g.st.CastArea(actor, target.X, target.Y, *spell, g.rng)
	if len(results) == 0 {
		// CastArea 在上述 preflight 後才扣 MP，但變動中的 runtime roster 仍可能
		// 沒有合法結果；還原成本，讓失敗行動保持原子性。
		actor.MP += spell.MP
		return fmt.Errorf("法術資料列 %d 未產生效果結果", spell.ID)
	}

	hitN, missN, total := 0, 0, 0
	var first *battle.CastResult
	for i := range results {
		result := &results[i]
		if result.Missed {
			missN++
			continue
		}
		hitN++
		total += result.Amount
		if first == nil {
			first = result
		}
		g.awardDeathReward(result.Target, actor)
	}
	name := spell.Name
	if name == "" {
		name = fmt.Sprintf("法術 %d", spell.ID)
	}
	verb := "造成"
	if spell.Target == 1 {
		verb = "回復"
	}
	g.msg = fmt.Sprintf("%s 施放 %s：命中 %d（%s %d）", actorDisplayName(actor), name, hitN, verb, total)
	if missN > 0 {
		g.msg += fmt.Sprintf("、未命中 %d", missN)
	}
	actor.SetMapPose(dirToward(actor.X, actor.Y, target.X, target.Y))

	finish := func() {
		g.finishSuccessfulUnitAction(actor, nil)
		g.checkResult()
	}
	if spell.Target == 0 && first != nil && first.Target != nil && first.Amount > 0 {
		def := first.Target
		attackerName, defenderName := actorDisplayName(actor), actorDisplayName(def)
		g.atk = g.newAtkAnim(actor.BattleFig, def.BattleFig, attackerName, defenderName,
			actor.HP, actor.MaxHP, actor.Lv, actor.MP, def.Lv, def.MP,
			def.HP+first.Amount, def.HP, def.MaxHP, g.terrainAt(def.X, def.Y), actor.Camp == battle.Own)
		if g.atk != nil {
			g.atk.after = finish
			return nil
		}
	}
	finish()
	return nil
}

func actorDisplayName(u *battle.Unit) string {
	if u == nil {
		return "單位"
	}
	if u.Name != "" {
		return u.Name
	}
	if u.ClsName != "" {
		return u.ClsName
	}
	return "單位"
}
