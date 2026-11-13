//! Go 來源：`remake/internal/battle/native_end_turn_recovery.go`（127 行）。

use super::*;

// native_end_turn_recovery.go — 0x1A30B 開頭的我方回復掃描。
//
// 0x1A332..0x1A3A2 先掃所有候選並畫 0x1DA16(mode2)；整輪發布後只播一次 sample4。
// 0x1A3F2..0x1A484 再掃一次把數值寫回：`+6`＝2、`(+5 & 0x81)`＝0、`+0x25`＝0、
// `+0x26`＝0、且 `+0x40`≠`+0x42` 的記錄，`+0x40 += +0x42 / 5`，超過就取 `+0x42`。
// 這段在 selector 1 的回合事件（0x1A813(1)）之前，玩家按 END、全員行動完的 0x13565
// 都會經過；已行動（0x80）的單位不回復，所以「全員行動完自動換手」那一回合沒有人回血。
// 回復過的單位由 0x13512 設 +5 bit7（0x1A477），橫幅之後 0x13536 才清。
// 數值部分由 fdother.NativeBattleEntryStep 釘住 raw 版面，這裡是 Unit 投影上的同一件事。

/// NativeEndTurnRecovery 是一個單位在 0x1A30B 回復掃描的結果。
///
/// Go：`native_end_turn_recovery.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeEndTurnRecovery {
    pub unit: Option<UnitIdx>,
    pub before: i64,
    pub after: i64,
    pub max_hp: i64,
}

impl State {
    /// PlanNativeEndTurnRecovery 計算符合 0x1A30B 閘門的我方單位回復 MaxHP/5 後的狀態，
    /// 依記錄順序回傳計畫，不寫入單位。沒有原版 `+6`／`+5` 的舊可編輯單位用 Camp／Acted
    /// ／Alive 對應同一組條件。
    ///
    /// Go：`native_end_turn_recovery.go:26`，`func (s *State) PlanNativeEndTurnRecovery() []NativeEndTurnRecovery`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_end_turn_recovery(&self) -> Vec<NativeEndTurnRecovery> {
        let mut out = Vec::new();
        for (index, u) in self.units.iter().enumerate() {
            if !u.on_field {
                continue;
            }
            if u.has_native_record_byte6 {
                if u.native_record_byte6 != 2 {
                    continue;
                }
            } else if u.camp != OWN {
                continue;
            }
            if !u.alive() {
                continue;
            }
            if u.has_native_record_byte5 {
                if u.native_record_byte5 & 0x81 != 0 {
                    continue;
                }
            } else if u.acted {
                continue;
            }
            if let (d, true) = u.native_transient_duration(0x25)
                && d != 0
            {
                continue;
            }
            if let (d, true) = u.native_transient_duration(0x26)
                && d != 0
            {
                continue;
            }
            if u.hp == u.max_hp || u.max_hp <= 0 {
                continue;
            }
            let before = u.hp;
            let next = u.hp.wrapping_add(u.max_hp / 5).min(u.max_hp);
            out.push(NativeEndTurnRecovery { unit: Some(index), before, after: next, max_hp: u.max_hp });
        }
        out
    }

    /// CommitNativeEndTurnRecovery rechecks the whole scan before the first write.
    /// A changed gate or HP rejects every candidate, including preceding records.
    ///
    /// Go：`native_end_turn_recovery.go:73`，`func (s *State) CommitNativeEndTurnRecovery(plan []NativeEndTurnRecovery) error`
    pub fn commit_native_end_turn_recovery(&mut self, plan: &[NativeEndTurnRecovery]) -> Result<()> {
        self.validate_native_end_turn_recovery(plan)?;
        for rec in plan {
            // 計畫剛重驗過，`unit` 一定是 `self.units` 的有效索引。
            let unit = &mut self.units[rec.unit.expect("重驗後的計畫一定有單位")];
            unit.hp = rec.after;
            if unit.has_native_record_byte5 {
                unit.native_record_byte5 |= 0x80; // 0x1A477 → 0x13512
                unit.acted = true; // live projection of the same bit; no extra raw write
            }
        }
        Ok(())
    }

    /// Go：`native_end_turn_recovery.go:87`，`func (s *State) ValidateNativeEndTurnRecovery(plan []NativeEndTurnRecovery) error`
    pub fn validate_native_end_turn_recovery(&self, plan: &[NativeEndTurnRecovery]) -> Result<()> {
        let current = self.plan_native_end_turn_recovery();
        if current.len() != plan.len() {
            return Err(battle_err!("native END recovery: candidate set changed"));
        }
        for (i, (cur, want)) in current.iter().zip(plan).enumerate() {
            if cur != want {
                return Err(battle_err!("native END recovery: candidate {i} changed"));
            }
        }
        Ok(())
    }

    /// ApplyNativeEndTurnRecovery preserves the synchronous legacy caller.
    ///
    /// Go：`native_end_turn_recovery.go:101`，`func (s *State) ApplyNativeEndTurnRecovery() []NativeEndTurnRecovery`
    pub fn apply_native_end_turn_recovery(&mut self) -> Vec<NativeEndTurnRecovery> {
        let plan = self.plan_native_end_turn_recovery();
        if self.commit_native_end_turn_recovery(&plan).is_err() {
            return Vec::new();
        }
        plan
    }

    /// MarkNativeDeadRecords 是 0x1DB65 的狀態部分：每次行動結算後（玩家路徑與敵方
    /// 路徑 0x15643 都呼叫）掃全部記錄，`+0x40` 為 0 的記錄把 `+5` 整個 byte 寫成 1
    /// （0x1DC61／0x1DD4C），之後的 AI 目標掃描、回合回復與勝負判定都以這個 bit 為準。
    /// 物理攻擊在扣血時已鏡射 bit0，這裡補的是指令傷害等只寫 HP 的路徑。回傳新標記的單位。
    ///
    /// Go：`native_end_turn_recovery.go:113`，`func (s *State) MarkNativeDeadRecords() []*Unit`
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn mark_native_dead_records(&mut self) -> Vec<UnitIdx> {
        let mut marked = Vec::new();
        for index in 0..self.units.len() {
            let u = &mut self.units[index];
            if !u.has_native_record_byte5 || u.hp != 0 || u.native_record_byte5 == 1 {
                continue;
            }
            u.native_record_byte5 = 1;
            self.sync_native_runtime_record(index as i64, 0x05, 1, 1);
            marked.push(index);
        }
        marked
    }
}

#[cfg(test)]
mod native_end_turn_recovery_test;
