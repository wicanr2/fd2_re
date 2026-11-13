//! Go 來源：`remake/internal/battle/native_physical_attack.go`（377 行）。

use super::*;

// native_physical_attack.go — 把 `sub_28A6C` 的兩次 `sub_2939D` 接成一次完整的
// 物理攻擊：攻方先打，守方符合條件就還手。證據見
// docs/knowledge-base/106-physical-attack-counterattack-20260910.md。

/// NativePhysicalStrike 是一次揮擊：一次 `sub_29F72` 加上把傷害寫回守方。
/// 原版同一次 `sub_2939D` 可能揮兩次——開頭 3% 的機率，或武器 record `+9`＝3。
///
/// Go：`native_physical_attack.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalStrike {
    pub roll: NativePhysicalRollResult,
    /// 實際扣掉的 HP（守方剩餘不足時小於 Roll.Damage）
    pub damage: i64,
    /// 這一擊之後守方的 HP
    pub defender_hp: i64,
}

/// NativePhysicalAttackResult 是一次物理攻擊指令的完整結果。
///
/// Go：`native_physical_attack.go:23`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct NativePhysicalAttackResult {
    /// 主攻：ExpGained／LevelUps 是攻方拿到的
    pub attack: AttackResult,
    pub attack_strikes: Vec<NativePhysicalStrike>,
    /// Counter 為 nil 表示沒有反擊。反擊的經驗給守方：`sub_29F72` 每一擊都以「揮擊
    /// 者 `+6` ＝ 2」決定要不要把經驗寫進 `[0x53EC8]`，敵方行動結束時 `0x1566A`
    /// 對被打的那個單位呼叫 `0x1E292`，所以我方單位反擊敵人一樣有經驗、一樣會
    /// 當場升級。CounterExpGained／CounterLevelUps 就是那一份。
    pub counter: Option<AttackResult>,
    pub counter_strikes: Vec<NativePhysicalStrike>,
    pub counter_exp_gained: i64,
    pub counter_level_ups: Vec<LevelUpEvent>,
    /// RNGState 是整次結算（含升級成長擲骰）之後的原版全域 RNG。
    pub rng_state: u16,
}

/// nativeIgnoresTerrainModifier 重現 `sub_1F183`：回真就不套地形攻防修正，也就是
/// 飛行。缺欄位時各欄位當 0，那正好落在「非飛行」那一邊（0 ≠ 28、0 ≠ 19、
/// 0 ∉ {4,5}），與重製端既有行為一致。
///
/// Go：`native_physical_attack.go:43`，`func nativeIgnoresTerrainModifier(u *Unit) bool`
pub(crate) fn native_ignores_terrain_modifier(u: Option<&Unit>) -> bool {
    let Some(u) = u else {
        return false;
    };
    let fig = if u.has_battle_fig { u.battle_fig } else { 0 };
    if fig == 28 {
        return false;
    }
    if u.has_native_record_class && u.native_record_class == 19 {
        return true;
    }
    if u.has_native_record_race {
        let race = u.native_record_race;
        return race == 4 || race == 5;
    }
    false
}

/// nativeEquippedWeaponSlot 重現 `sub_1B83D(unit, 0)`：找第一個「已裝備且 ID 小於
/// 0x80」的槽。原版看的是 `unit+10+2i` 的 bit 0x40；有原始旗標就用原始旗標，沒有
/// 才退回正規化的 Equipped。
///
/// Go：`native_physical_attack.go:67`，`func nativeEquippedWeaponSlot(u *Unit) (int, bool)`
pub(crate) fn native_equipped_weapon_slot(u: Option<&Unit>) -> (i64, bool) {
    let Some(u) = u else {
        return (0, false);
    };
    let mut slot = 0usize;
    while slot < 8 && slot < u.inventory.len() {
        if u.inventory[slot] >= 0x80 {
            slot += 1;
            continue;
        }
        if slot < u.native_inventory_flags.len() {
            if u.native_inventory_flags[slot] & 0x40 != 0 {
                return (slot as i64, true);
            }
            slot += 1;
            continue;
        }
        if slot < u.equipped.len() && u.equipped[slot] {
            return (slot as i64, true);
        }
        slot += 1;
    }
    (0, false)
}

impl State {
    /// nativeEquippedWeapon 取出攻方武器 record 的 `+9`／`+10`／`+11`。拿不到原版物品表
    /// 時回零值，也就是沒有武器特效——那與重製端接線前的行為相同，不是新的猜測。
    ///
    /// Go：`native_physical_attack.go:90`，`func (s *State) nativeEquippedWeapon(u *Unit) (NativePhysicalWeapon, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_equipped_weapon(&self, u: Option<UnitIdx>) -> (NativePhysicalWeapon, bool) {
        let unit = u.map(|i| &self.units[i]);
        let (slot, ok) = native_equipped_weapon_slot(unit);
        if !ok {
            return (NativePhysicalWeapon::default(), false);
        }
        let unit = unit.expect("nativeEquippedWeaponSlot 已確認單位存在");
        let offset = match native_item_effect_row_offset(unit.inventory[slot as usize]) {
            Ok(offset) if offset + NATIVE_ITEM_EFFECT_ROW_SIZE <= self.native_future_item_rows.len() as i64 => offset,
            _ => return (NativePhysicalWeapon::default(), false),
        };
        let row = &self.native_future_item_rows[offset as usize..(offset + NATIVE_ITEM_EFFECT_ROW_SIZE) as usize];
        (NativePhysicalWeapon { effect: i64::from(row[9]), param: i64::from(row[10]), reach: i64::from(row[11]) }, true)
    }

    /// buildNativePhysicalRoll 組出一次擲骰的輸入。
    ///
    /// Go：`native_physical_attack.go:106`，`func (s *State) buildNativePhysicalRoll(a, d *Unit, rngState uint16) NativePhysicalRoll`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn build_native_physical_roll(
        &self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
    ) -> NativePhysicalRoll {
        let (au, du) = (a.map(|i| &self.units[i]), d.map(|i| &self.units[i]));
        let (au_ref, du_ref) = (au.expect("攻方不可為 nil"), du.expect("守方不可為 nil"));
        let (mut ap_pct, mut dp_pct) = (0, 0);
        if !native_ignores_terrain_modifier(au) {
            ap_pct = self.terrain_apdp_pct(au_ref.x, au_ref.y).0;
        }
        if !native_ignores_terrain_modifier(du) {
            dp_pct = self.terrain_apdp_pct(du_ref.x, du_ref.y).1;
        }
        let (weapon, _) = self.native_equipped_weapon(a);
        NativePhysicalRoll {
            attacker_ap: au_ref.effective_ap(),
            defender_dp: du_ref.effective_dp(),
            attacker_hit: au_ref.effective_hit(),
            defender_ev: du_ref.effective_ev(),
            attacker_crit_pct: native_class_crit_pct(au),
            attacker_terrain_ap_pct: ap_pct,
            defender_terrain_dp_pct: dp_pct,
            weapon,
            rng_state,
        }
    }

    /// nativePhysicalExchange 重現一次 `sub_2939D`：開頭 3% 決定要不要揮兩次，之後每
    /// 一輪擲一次 `sub_29F72` 並把傷害寫回；武器 record `+9`＝3 會再多一輪，但只多一次
    /// （原版的 `v40` 旗標）。每一輪都重讀守方當下的 HP，所以第二擊打的是已扣血的目標。
    ///
    /// Go：`native_physical_attack.go:128`，`func (s *State) nativePhysicalExchange(a, d *Unit, rngState uint16) ([]NativePhysicalStrike, uint16, error)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_physical_exchange(
        &mut self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<(Vec<NativePhysicalStrike>, u16)> {
        let di = d.expect("守方不可為 nil");
        let mut rng = fd2_formats::native_rng::native_rng_step(rng_state);
        let mut budget = if i64::from(rng) % 100 < 3 { 2 } else { 1 };
        let mut strikes = Vec::with_capacity(2);
        let mut extra_used = false;
        while budget > 0 {
            budget -= 1;
            let input = self.build_native_physical_roll(a, d, rng);
            let debug = DEBUG_AI.read().unwrap_or_else(|e| e.into_inner()).clone();
            if let Some(debug) = debug {
                // Go 以 %+v 印出輸入結構；Rust 印 Debug 格式，只供診斷。
                let (au, du) = (&self.units[a.expect("攻方不可為 nil")], &self.units[di]);
                debug(format_args!(
                    "native physical roll: actor=({},{}) target=({},{}) input={:?}",
                    au.x, au.y, du.x, du.y, input
                ));
            }
            let roll = roll_native_physical_damage(input)?;
            rng = roll.rng_state;
            let du = &mut self.units[di];
            if roll.status {
                // 0x2A12F：武器 +9＝2 發動時在命中判定之前就寫守方 +0x25＝rand()%4+2，
                // 未命中也照寫（第十章 r2 第 1 回合記錄 5、52 在下一個 0x1A866 扣 MaxHP/10）。
                du.set_native_transient_duration(0x25, roll.status_value as u8);
            }
            let mut strike = NativePhysicalStrike { roll, ..Default::default() };
            if !roll.missed {
                let before = du.hp;
                du.apply_hp_damage(roll.damage);
                strike.damage = before - du.hp;
            }
            strike.defender_hp = du.hp;
            strikes.push(strike);
            if du.hp == 0 {
                // 0x29B4C cmp [esp+0x58],0 → 把次數歸零：守方倒下就不再揮。
                break;
            }
            if !extra_used && roll.extra {
                extra_used = true;
                budget += 1;
            }
        }
        Ok((strikes, rng))
    }

    /// counterattackDefender 把守方的狀態整理成反擊資格判定的輸入。
    ///
    /// 原版 `sub_1F0DC` 讀的 `unit+38` 就是 `+0x26`——六個暫時狀態剩餘回合數
    /// （`+0x22..+0x27`）中的一個，由 `0x1A866` 的 camp-phase 掃描逐一遞減。非零代表
    /// 那個狀態還在生效，同一組條件也擋掉回合開始的自動回復（`sub_1A30B` 的
    /// `[ebx+25h]`／`[ebx+26h]` 兩個閘門）。
    ///
    /// Go：`native_physical_attack.go:178`，`func (s *State) counterattackDefender(d *Unit) NativeCounterattackDefender`
    /// 所有權：`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn counterattack_defender(&self, d: Option<UnitIdx>) -> NativeCounterattackDefender {
        let (weapon, ok) = self.native_equipped_weapon(d);
        let du = &self.units[d.expect("守方不可為 nil")];
        let (transient, _) = du.native_transient_duration(0x26);
        NativeCounterattackDefender {
            x: du.x,
            y: du.y,
            alive_hp: du.hp,
            byte38: i64::from(transient),
            has_weapon: ok,
            weapon_reach: weapon.reach,
        }
    }

    /// NativePhysicalCounterattackEligible 供呈現預檢沿用結算的資格來源。
    /// 這裡使用攻擊前HP；主攻結束後仍由結算端重新檢查生存條件。
    ///
    /// Go：`native_physical_attack.go:189`，`func (s *State) NativePhysicalCounterattackEligible(actor, target *Unit) bool`
    /// 所有權：`actor`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_physical_counterattack_eligible(&self, actor: Option<UnitIdx>, target: Option<UnitIdx>) -> bool {
        let (Some(actor), Some(_)) = (actor, target) else {
            return false;
        };
        let au = &self.units[actor];
        native_counterattack_eligible(au.x, au.y, self.counterattack_defender(target))
    }

    /// AttackNativePhysical 依原版結算一次物理攻擊，含反擊。rngState 是原版的全域
    /// `0x627B8`；呼叫者要把回傳的新狀態存回去，否則之後每一次結算都會偏掉。
    ///
    /// rng 只用在經驗值與升級（那條仍是重製端既有的規則），傷害本身完全走原版 RNG。
    ///
    /// Go：`native_physical_attack.go:198`，`func (s *State) AttackNativePhysical(a, d *Unit, rngState uint16, rng *rand.Rand) (NativePhysicalAttackResult, error)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn attack_native_physical(
        &mut self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
        rng: &mut dyn GoRand,
    ) -> Result<NativePhysicalAttackResult> {
        self.attack_native_physical_inner(a, d, rng_state, rng, None)
    }

    /// Go：`native_physical_attack.go:202`，`func (s *State) attackNativePhysical( a, d *Unit, rngState uint16, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan, ) (NativePhysicalAttackResult, error)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn attack_native_physical_inner(
        &mut self,
        a: Option<UnitIdx>,
        d: Option<UnitIdx>,
        rng_state: u16,
        rng: &mut dyn GoRand,
        native_exp: Option<&NativePhysicalExperiencePlan>,
    ) -> Result<NativePhysicalAttackResult> {
        let ai = a.expect("攻方不可為 nil");
        let di = d.expect("守方不可為 nil");
        self.units[ai].acted = true;
        if native_exp.is_some() && self.units[ai].has_native_record_byte5 {
            self.units[ai].native_record_byte5 |= 0x80;
        }

        let (strikes, next) = self.native_physical_exchange(a, d, rng_state)?;
        let mut result = NativePhysicalAttackResult { attack_strikes: strikes, rng_state: next, ..Default::default() };
        result.attack = summarize_native_strikes(&result.attack_strikes);

        // 反擊的經驗計畫要在改動任何經驗之前備妥：守方是我方時它和主攻一樣走
        // planNativePhysicalExperience，缺 raw 資料就整筆拒絕。
        let mut counter_exp = None;
        if native_exp.is_some() && self.units[di].has_native_record_byte6 {
            let plan = plan_native_physical_experience(Some(&self.units[di]), Some(&self.units[ai]))
                .map_err(|err| err.wrap("counterattack EXP: "))?;
            counter_exp = Some(plan);
        }

        let attacker = &self.units[ai];
        if native_counterattack_eligible(attacker.x, attacker.y, self.counterattack_defender(d)) {
            let (counter_strikes, next) = self.native_physical_exchange(d, a, result.rng_state)?;
            let counter = summarize_native_strikes(&counter_strikes);
            result.counter = Some(counter.clone());
            result.counter_strikes = counter_strikes;
            result.attack.counter = Some(Box::new(counter));
            result.rng_state = next;
        }

        // 經驗與升級在兩段交鋒都結束之後才發（玩家路徑 0x1196D、敵方路徑 0x1566A 都在
        // `sub_28A6C` 返回之後才呼叫 `0x1E292`），升級成長的擲骰接在反擊之後的 RNG 上。
        let Some(native_exp) = native_exp else {
            let (exp, ups) = self.award_native_physical_experience(a, di, result.attack.amount, rng, None);
            result.attack.exp_gained = exp;
            result.attack.level_ups = ups;
            return Ok(result);
        };
        // 0x18DC4 玩家入口重設；敵方 sub_1548E 沒有重設。兩段揮擊結束後
        // 留下最後一個我方 writer 的值，沒有反擊也可能保留前一動作的累計。
        if self.units[ai].native_record_byte6 == 2 {
            self.native_experience_accumulator = 0;
        }
        if native_exp.writes_accumulator {
            self.native_experience_accumulator =
                native_last_strike_experience(Some(native_exp), &result.attack_strikes);
        }
        if let (Some(_), Some(counter_exp)) = (&result.counter, counter_exp.as_ref())
            && counter_exp.writes_accumulator
        {
            self.native_experience_accumulator =
                native_last_strike_experience(Some(counter_exp), &result.counter_strikes);
        }
        if self.kill_cancels_exp(&[di]) || self.kill_cancels_exp(&[ai]) {
            self.native_experience_accumulator = 0;
        }
        if self.units[ai].native_record_byte6 == 2 {
            if self.native_experience_accumulator > 99 {
                self.native_experience_accumulator = 99; // 0x11959：只限玩家 caller。
            }
            let (got, ups, next) = self.award_exp_native(a, self.native_experience_accumulator, result.rng_state);
            result.attack.exp_gained = got as f64;
            result.attack.level_ups = ups;
            result.rng_state = next;
            if got > 0 {
                self.native_experience_accumulator = 0; // 0x1E51A；滿級／死亡早退不抵達。
            }
        } else {
            let (got, ups, next) = self.award_exp_native(d, self.native_experience_accumulator, result.rng_state);
            result.counter_exp_gained = got;
            result.counter_level_ups = ups;
            result.rng_state = next;
            if got > 0 {
                self.native_experience_accumulator = 0;
            }
        }
        Ok(result)
    }
}

/// nativeLastStrikeExperience 重現 `[0x53EC8]` 的寫法：`sub_29F72` 每一擊都整個覆寫，
/// 不累加，所以留下來的是最後一擊算出的值——最後一擊打空就是 0，最後一擊把守方
/// 打倒就是整份 base。
///
/// Go：`native_physical_attack.go:283`，`func nativeLastStrikeExperience(plan *nativePhysicalExperiencePlan, strikes []NativePhysicalStrike) int`
pub(crate) fn native_last_strike_experience(
    plan: Option<&NativePhysicalExperiencePlan>,
    strikes: &[NativePhysicalStrike],
) -> i64 {
    let (Some(plan), Some(last)) = (plan, strikes.last()) else {
        return 0;
    };
    plan.award(last.damage, last.defender_hp == 0)
}

/// summarizeNativeStrikes 把一次交鋒的每一擊併成既有的 AttackResult 形狀：傷害相加，
/// 全部落空才算 Missed，任一擊暴擊就算 Crit。
///
/// Go：`native_physical_attack.go:293`，`func summarizeNativeStrikes(strikes []NativePhysicalStrike) AttackResult`
pub(crate) fn summarize_native_strikes(strikes: &[NativePhysicalStrike]) -> AttackResult {
    let mut out = AttackResult { missed: true, ..Default::default() };
    for strike in strikes {
        out.amount += strike.damage;
        if !strike.roll.missed {
            out.missed = false;
        }
        if strike.roll.crit {
            out.crit = true;
        }
    }
    out
}

impl State {
    /// awardNativePhysicalExperience 是沒有原版 `+6` 來源的舊可編輯單位走的經驗規則
    /// （重製端既有公式、Go RNG 升級）。原生單位改走 nativeLastStrikeExperience 加
    /// AwardExpNative。
    ///
    /// Go：`native_physical_attack.go:310`，`func (s *State) awardNativePhysicalExperience( a, d *Unit, damage int, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan, ) (float64, []LevelUpEvent)`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn award_native_physical_experience(
        &mut self,
        a: Option<UnitIdx>,
        d: UnitIdx,
        damage: i64,
        rng: &mut dyn GoRand,
        native_exp: Option<&NativePhysicalExperiencePlan>,
    ) -> (f64, Vec<LevelUpEvent>) {
        let ai = a.expect("攻方不可為 nil");
        let (au, du) = (&self.units[ai], &self.units[d]);
        if au.camp != OWN && au.camp != ALLY {
            return (0.0, Vec::new());
        }
        let for_exp = if du.hp == 0 { du.max_hp } else { damage };
        let mut exp = attack_exp(au.lv, du.lv, for_exp, du.max_hp, du.exp_per_level);
        if let Some(native_exp) = native_exp {
            exp = native_exp.award(damage, du.hp == 0).min(99) as f64;
        }
        if self.kill_cancels_exp(&[d]) {
            exp = 0.0;
        }
        self.award_exp(a, exp, rng)
    }

    /// AttackNativePhysicalWithExperience 是正式物理入口：傷害與反擊照原版，經驗值在
    /// 攻方帶有原版 `+6` 來源時走 planNativePhysicalExperience，否則走重製端既有規則。
    ///
    /// 預檢在改動任何 HP 之前完成：raw 資料缺項時整筆拒絕，不猜用相容欄位。
    ///
    /// Go：`native_physical_attack.go:334`，`func (s *State) AttackNativePhysicalWithExperience( actor, target *Unit, rngState uint16, rng *rand.Rand, ) (NativePhysicalAttackResult, error)`
    /// 所有權：`actor`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn attack_native_physical_with_experience(
        &mut self,
        actor: Option<UnitIdx>,
        target: Option<UnitIdx>,
        rng_state: u16,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativePhysicalAttackResult> {
        let Some(rng) = rng else {
            return Err(battle_err!("native physical attack: state or RNG unavailable"));
        };
        let (Some(actor_idx), Some(_)) = (actor, target) else {
            return Err(battle_err!("native physical attack: actor or target unavailable"));
        };
        if !self.units[actor_idx].has_native_record_byte6 {
            return self.attack_native_physical_inner(actor, target, rng_state, rng, None);
        }
        let plan = plan_native_physical_experience(actor.map(|i| &self.units[i]), target.map(|i| &self.units[i]))?;
        self.attack_native_physical_inner(actor, target, rng_state, rng, Some(&plan))
    }
}

/// nativeClassCritPctTable 是 FD2.EXE 0x5239B 的 24 個職業暴擊率（索引 = record
/// +0x20 職業減一；docs/data/exe_tables/native_combat_tables.json 釘住同一組位元組）。
/// `sub_29F72` 以 `[ebp+0x20]-1` 直接索引這張表，不讀 unit 記錄裡的任何欄位。
///
/// Go：`native_physical_attack.go:358`
pub(crate) static NATIVE_CLASS_CRIT_PCT_TABLE: [i64; 24] =
    [5, 3, 3, 5, 3, 3, 0, 18, 5, 3, 3, 12, 3, 3, 12, 10, 6, 3, 3, 7, 3, 3, 30, 18];

/// nativeClassCritPct 依原版以職業查暴擊率；沒有 raw +0x20 的舊可編輯單位沿用
/// authored CritPct。
///
/// Go：`native_physical_attack.go:362`，`func nativeClassCritPct(u *Unit) int`
pub(crate) fn native_class_crit_pct(u: Option<&Unit>) -> i64 {
    let Some(u) = u else {
        return 0;
    };
    let class = if u.has_native_record_class {
        i64::from(u.native_record_class)
    } else if u.class_id > 0 {
        u.class_id
    } else {
        -1
    };
    if class >= 1 && class <= NATIVE_CLASS_CRIT_PCT_TABLE.len() as i64 {
        return NATIVE_CLASS_CRIT_PCT_TABLE[(class - 1) as usize];
    }
    u.crit_pct
}
