//! Go 來源：`remake/cmd/fd2/ai_spell_execute.go`（118 行）。

use super::*;

impl Game {
    /// executeAISpell 是正規化（normalized）的 AI 法術消費端。它刻意使用與玩家路徑
    /// 相同的可編輯 SpellBook／CastArea transaction，但不把原版 0x1598A 命令評分或
    /// indexed 法術演出混進本後備。目標、距離、RNG 或法術列缺失時，必須在標記行動完成前
    /// 停止。
    ///
    /// Go：`ai_spell_execute.go:14`，`func (g *Game) executeAISpell(plan *battle.AIPlan) error`
    /// 所有權：`plan.u`／`plan.target` 是 `st.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn execute_ai_spell(&mut self, plan: &battle::AIPlan) -> Result<()> {
        let (Some(actor), Some(target)) = (plan.u, plan.target) else {
            return Err(game_err!("法術計畫上下文缺失"));
        };
        let Some(st) = self.st.as_ref() else {
            return Err(game_err!("法術計畫上下文缺失"));
        };
        if plan.spell_id < 0 || self.rng.is_none() {
            return Err(game_err!("法術 ID 或決定性亂數來源不可用"));
        }
        let (a, t) = (&st.units[actor], &st.units[target]);
        if !a.on_field || !a.alive() || a.acted || a.sealed || !t.on_field || !t.alive() {
            return Err(game_err!("施法者或目標已失效"));
        }
        let Some(spell) = st.spell_book.iter().find(|spell| spell.id == plan.spell_id).cloned() else {
            return Err(game_err!("法術資料列 {} 不可用", plan.spell_id));
        };
        if a.mp < spell.mp || spell.mp < 0 {
            return Err(game_err!("法術資料列 {} 的 MP 不足", spell.id));
        }
        if !st.ai_spell_candidates(Some(actor), &spell).contains(&target) {
            return Err(game_err!("法術資料列 {} 的目標不合法", spell.id));
        }
        let (tx, ty) = (t.x, t.y);
        let distance = abs_int(a.x - tx) + abs_int(a.y - ty);
        if distance > spell.dist {
            return Err(game_err!("法術資料列 {} 的目標超出距離", spell.id));
        }

        let rng = self.rng.as_mut().unwrap();
        let st = self.st.as_mut().unwrap();
        let results = st.cast_area(Some(actor), tx, ty, &spell, rng);
        if results.is_empty() {
            // CastArea 在上述 preflight 後才扣 MP，但變動中的 runtime roster 仍可能
            // 沒有合法結果；還原成本，讓失敗行動保持原子性。
            st.units[actor].mp += spell.mp;
            return Err(game_err!("法術資料列 {} 未產生效果結果", spell.id));
        }

        let (mut hit_n, mut miss_n, mut total) = (0, 0, 0);
        let mut first: Option<(Option<UnitIdx>, i64)> = None;
        for result in &results {
            if result.missed {
                miss_n += 1;
                continue;
            }
            hit_n += 1;
            total += result.amount;
            if first.is_none() {
                first = Some((result.target, result.amount));
            }
            self.award_death_reward(result.target, Some(actor));
        }
        let name = if spell.name.is_empty() { format!("法術 {}", spell.id) } else { spell.name.clone() };
        let actor_name = actor_display_name(Some(&self.st.as_ref().unwrap().units[actor]));
        if let (message, true) =
            self.localized_spell_result(&actor_name, &name, hit_n, total, spell.target == 1, miss_n)
        {
            self.msg = message;
        }
        let unit = &mut self.st.as_mut().unwrap().units[actor];
        let (ux, uy) = (unit.x, unit.y);
        unit.set_map_pose(dir_toward(ux, uy, tx, ty));

        let finish = move |g: &mut Game| {
            g.finish_successful_unit_action(Some(actor), None);
            g.check_result();
        };
        if spell.target == 0
            && let Some((Some(def), amount)) = first
            && amount > 0
        {
            let units = &self.st.as_ref().unwrap().units;
            let (a, d) = (&units[actor], &units[def]);
            let (attacker_name, defender_name) = (actor_display_name(Some(a)), actor_display_name(Some(d)));
            let terrain = self.terrain_at(d.x, d.y);
            self.atk = self.new_atk_anim(
                a.battle_fig,
                d.battle_fig,
                &attacker_name,
                &defender_name,
                a.hp,
                a.max_hp,
                a.lv,
                a.mp,
                a.max_mp,
                d.lv,
                d.mp,
                d.max_mp,
                d.hp + amount,
                d.hp,
                d.max_hp,
                terrain,
                a.camp == battle::OWN,
            );
            if let Some(atk) = self.atk.as_mut() {
                atk.after = Some(Box::new(finish));
                return Ok(());
            }
        }
        finish(self);
        Ok(())
    }
}

/// Go：`ai_spell_execute.go:107`，`func actorDisplayName(u *battle.Unit) string`
pub(crate) fn actor_display_name(u: Option<&battle::Unit>) -> String {
    let Some(u) = u else {
        return String::from("單位");
    };
    if !u.name.is_empty() {
        return u.name.clone();
    }
    if !u.cls_name.is_empty() {
        return u.cls_name.clone();
    }
    String::from("單位")
}

#[cfg(test)]
#[path = "ai_spell_execute/ai_spell_consumer_test.rs"]
mod ai_spell_consumer_test;
