//! Go 來源：`remake/cmd/fd2/native_command_experience.go`（113 行）。玩家指令的經驗發放。
//!
//! 原版玩家控制器 0x117E7 在 0x18890（玩家行動）返回後才 0x11959 壓 99、0x1196D 呼叫 0x1E292 發經驗；物理攻擊由
//! battle 結算直接發放，這裡處理其餘指令家族。各家族的累加量與寫入端位址見 `fd2-rules` 的
//! `native_command_experience.rs`。時機和物理攻擊相同：在指令結果回呼裡、`finish_successful_unit_action` 之前
//! 發放並排入升級對話。

use super::*;

impl Game {
    /// 只處理原版 +6 == 2 的玩家單位；累加量算不出來時失敗即關閉，寫 `load_err` 且不發經驗。
    ///
    /// Go：`native_command_experience.go:17`，`func (g *Game) awardNativePlayerCommandExperience(actor *battle.Unit, delta int, err error, targets []*battle.Unit)`
    pub(crate) fn award_native_player_command_experience(
        &mut self,
        actor: UnitIdx,
        delta: battle::Result<i64>,
        targets: &[UnitIdx],
    ) {
        let Some(st) = self.st.as_mut() else { return };
        let unit = &st.units[actor];
        if !unit.has_native_record_byte6 || unit.native_record_byte6 != 2 {
            return;
        }
        let delta = match delta {
            Ok(delta) => delta,
            Err(err) => {
                self.load_err = err.to_string();
                return;
            }
        };
        let (got, ups, next) =
            st.award_native_player_action_experience(Some(actor), delta, self.native_rng_state, targets);
        self.native_rng_state = next;
        self.queue_native_level_up_dialogue(Some(actor), got, &ups);
    }

    /// 指令 0～12（0x2A6BD → 0x1C75E → 0x1C81F）。
    ///
    /// Go：`native_command_experience.go:31`，`func (g *Game) awardNativeCommandDamageExperience(actor *battle.Unit, results []battle.NativeCommandDamageResult)`
    pub(crate) fn award_native_command_damage_experience(
        &mut self,
        actor: UnitIdx,
        results: &[battle::NativeCommandDamageResult],
    ) {
        let targets: Vec<UnitIdx> = results.iter().filter_map(|result| result.target).collect();
        let exp = battle::native_command_damage_experience(&self.st.as_ref().unwrap().units, results);
        self.award_native_player_command_experience(actor, exp, &targets);
    }

    /// 指令 13～16（0x1C916）。
    ///
    /// Go：`native_command_experience.go:41`，`func (g *Game) awardNativeCommandHealExperience(actor *battle.Unit, results []battle.NativeCommandHealResult)`
    pub(crate) fn award_native_command_heal_experience(
        &mut self,
        actor: UnitIdx,
        results: &[battle::NativeCommandHealResult],
    ) {
        let targets: Vec<UnitIdx> = results.iter().filter_map(|result| result.target).collect();
        let exp = battle::native_command_heal_experience(&self.st.as_ref().unwrap().units, results);
        self.award_native_player_command_experience(actor, Ok(exp), &targets);
    }

    /// 指令 24、28～31（0x276EC → 0x1C81F）。
    ///
    /// Go：`native_command_experience.go:50`，`func (g *Game) awardNativeCommandDerivedStrikeExperience(actor *battle.Unit, results []battle.NativeCommand24Damage)`
    pub(crate) fn award_native_command_derived_strike_experience(
        &mut self,
        actor: UnitIdx,
        results: &[battle::NativeCommand24Damage],
    ) {
        let targets: Vec<UnitIdx> = results.iter().filter_map(|result| result.target).collect();
        let exp = battle::native_command_derived_strike_experience(&self.st.as_ref().unwrap().units, results);
        self.award_native_player_command_experience(actor, exp, &targets);
    }

    /// 指令 17～19（0x22721／0x22866／0x22997）：累加量已在結果的 accumulator。
    ///
    /// Go：`native_command_experience.go:60`，`func (g *Game) awardNativeCommandModifierExperience(actor *battle.Unit, result battle.NativeCommandModifierResult, targets []*battle.Unit)`
    pub(crate) fn award_native_command_modifier_experience(
        &mut self,
        actor: UnitIdx,
        result: &battle::NativeCommandModifierResult,
        targets: &[UnitIdx],
    ) {
        self.award_native_player_command_experience(actor, Ok(result.accumulator), targets);
    }

    /// 指令 20～22、25～27（0x22AF6／0x22D1B／0x22C04）。
    ///
    /// Go：`native_command_experience.go:65`，`func (g *Game) awardNativeCommand2022Experience(actor *battle.Unit, plan *battle.NativeAICommand2022Plan)`
    pub(crate) fn award_native_command2022_experience(
        &mut self,
        actor: UnitIdx,
        plan: &battle::NativeAICommand2022Plan,
    ) {
        let exp = battle::native_command2022_experience(&self.st.as_ref().unwrap().units, plan);
        let targets = plan.targets.clone();
        self.award_native_player_command_experience(actor, exp, &targets);
    }
}
