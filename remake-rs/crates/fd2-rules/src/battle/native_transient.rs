//! Go 來源：`remake/internal/battle/native_transient.go`（230 行）。

use super::*;

// NativeTransientOffset is the first byte of the original six-byte transient
// interval at runtime unit+0x22..+0x27.  Callers use raw offsets deliberately:
// only some effects have a recovered gameplay label, and the data layer must
// not invent one for the rest.

pub const NATIVE_TRANSIENT_OFFSET: i64 = 0x22;

pub const NATIVE_TRANSIENT_COUNT: i64 = 6;

/// NativeTransientExpiry records one original raw interval byte which reached
/// zero during the recovered 0x1A866 camp-phase sweep.  Recalculation of
/// derived stats/render feedback is intentionally owned by higher layers;
/// battle cannot import campaign's equipment table without creating a cycle.
///
/// Go：`native_transient.go:21`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTransientExpiry {
    pub unit: Option<UnitIdx>,
    pub offset: i64,
}

/// NativeTransientDamage 保存 sub_1A866 第一段對 raw +0x25 記錄的逐筆 HP writer。
/// Amount 即 MaxHP/10；即使 Amount 為 0，原版仍會走該筆回覆，因此照樣保留事件。
///
/// Go：`native_transient.go:28`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTransientDamage {
    pub unit: Option<UnitIdx>,
    pub slot: i64,
    pub amount: i64,
    pub before: i64,
    pub after: i64,
}

/// NativeTransientPhaseResult 是單次 sub_1A866(selector) 的純狀態結果。它不帶
/// killer，也不分派死亡效果；該 phase 的原版 ABI 根本沒有這兩項輸入。
///
/// Go：`native_transient.go:36`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTransientPhaseResult {
    pub damage: Vec<NativeTransientDamage>,
    pub inactive_slots: Vec<i64>,
    pub expired: Vec<NativeTransientExpiry>,
}

impl Unit {
    /// NativeTransientDuration returns a raw duration byte by its original unit
    /// offset.  It rejects anything outside +0x22..+0x27 rather than silently
    /// mapping a guessed status field.
    ///
    /// Go：`native_transient.go:45`，`func (u *Unit) NativeTransientDuration(offset int) (byte, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_transient_duration(&self, offset: i64) -> (u8, bool) {
        if !(NATIVE_TRANSIENT_OFFSET..NATIVE_TRANSIENT_OFFSET + NATIVE_TRANSIENT_COUNT).contains(&offset) {
            return (0, false);
        }
        (self.native_transient[(offset - NATIVE_TRANSIENT_OFFSET) as usize], true)
    }

    /// SetNativeTransientDuration writes one recovered raw duration byte.  It is a
    /// bounded storage primitive; command-specific gates and side effects remain
    /// in their separately recovered executors.
    ///
    /// Go：`native_transient.go:55`，`func (u *Unit) SetNativeTransientDuration(offset int, duration byte) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_native_transient_duration(&mut self, offset: i64, duration: u8) -> bool {
        if !(NATIVE_TRANSIENT_OFFSET..NATIVE_TRANSIENT_OFFSET + NATIVE_TRANSIENT_COUNT).contains(&offset) {
            return false;
        }
        self.native_transient[(offset - NATIVE_TRANSIENT_OFFSET) as usize] = duration;
        true
    }
}
impl State {
    /// TickNativeTransientsRaw mirrors the recovered raw mutation portion of
    /// 0x1A866. The native routine gates on record+6 == selector and
    /// (record+5 & 1) == 0; it does not prove an OnField/Alive/Camp equivalence.
    /// Every nonzero byte +0x22..+0x27 is decremented independently.
    ///
    /// Go：`native_transient.go:67`，`func (s *State) TickNativeTransientsRaw(selector byte) []NativeTransientExpiry`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn tick_native_transients_raw(&mut self, selector: u8) -> Vec<NativeTransientExpiry> {
        let mut expired = Vec::new();
        for (slot, u) in self.units.iter_mut().enumerate() {
            if !u.has_native_record_byte6
                || u.native_record_byte6 != selector
                || !u.has_native_record_byte5
                || u.native_record_byte5 & 1 != 0
            {
                continue;
            }
            for (i, duration) in u.native_transient.iter_mut().enumerate() {
                if *duration == 0 {
                    continue;
                }
                *duration -= 1;
                if *duration == 0 {
                    expired
                        .push(NativeTransientExpiry { unit: Some(slot), offset: NATIVE_TRANSIENT_OFFSET + i as i64 });
                }
            }
        }
        expired
    }

    /// AdvanceNativeTransientPhaseRaw 重現 sub_1A866 的狀態寫入順序：先依 record 順序
    /// 對 +0x25 非零的同 selector 單位扣 MaxHP/10，再執行 sub_1DB65 的 HP==0 →
    /// raw +5=1，最後只替仍 active 的單位遞減 +0x22..+0x27。typed 與完整 raw
    /// projection 在同一個私有 State 中同步；任何矛盾都在 mutation 前拒絕。
    ///
    /// Go：`native_transient.go:94`，`func (s *State) AdvanceNativeTransientPhaseRaw(selector byte) (NativeTransientPhaseResult, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_transient_phase_raw(&mut self, selector: u8) -> Result<NativeTransientPhaseResult> {
        if !self.has_native_runtime_unit_projection
            || self.units.is_empty()
            || self.units.len() != self.native_runtime_records.len()
        {
            return Err(battle_err!("native transient phase: runtime raw projection is incomplete"));
        }
        for (slot, unit) in self.units.iter().enumerate() {
            if !unit.has_native_record_byte5
                || !unit.has_native_record_byte6
                || !(0..=0xffff).contains(&unit.hp)
                || !(0..=0xffff).contains(&unit.max_hp)
            {
                return Err(battle_err!("native transient phase: unit {slot} lacks consumed raw fields"));
            }
            let raw = &self.native_runtime_records[slot].raw;
            if raw[5] != unit.native_record_byte5
                || raw[6] != unit.native_record_byte6
                || u16::from_le_bytes([raw[0x40], raw[0x41]]) != unit.hp as u16
                || u16::from_le_bytes([raw[0x42], raw[0x43]]) != unit.max_hp as u16
            {
                return Err(battle_err!("native transient phase: unit {slot} raw projection disagrees"));
            }
            for (index, &duration) in unit.native_transient.iter().enumerate() {
                if raw[NATIVE_TRANSIENT_OFFSET as usize + index] != duration {
                    return Err(battle_err!("native transient phase: unit {slot} transient projection disagrees"));
                }
            }
        }

        let mut result = NativeTransientPhaseResult::default();
        let poison = (0x25 - NATIVE_TRANSIENT_OFFSET) as usize;
        for (slot, unit) in self.units.iter_mut().enumerate() {
            if unit.native_transient[poison] == 0
                || unit.native_record_byte6 != selector
                || unit.native_record_byte5 & 1 != 0
            {
                continue;
            }
            let (amount, before) = (unit.max_hp / 10, unit.hp);
            let after = (before - amount).max(0);
            unit.hp = after;
            self.native_runtime_records[slot].raw[0x40..0x42].copy_from_slice(&(after as u16).to_le_bytes());
            result.damage.push(NativeTransientDamage { unit: Some(slot), slot: slot as i64, amount, before, after });
        }

        // sub_1DB65 的兩個收尾 writer 都掃全部記錄並把整個 +5 byte 覆寫成 1；
        // 不只處理這次 selector，也不保留 bit7 等舊值。
        for (slot, unit) in self.units.iter_mut().enumerate() {
            if unit.hp != 0 {
                continue;
            }
            if unit.native_record_byte5 & 1 == 0 {
                result.inactive_slots.push(slot as i64);
            }
            unit.native_record_byte5 = 1;
            self.native_runtime_records[slot].raw[5] = 1;
        }

        for (slot, unit) in self.units.iter_mut().enumerate() {
            if unit.native_record_byte6 != selector || unit.native_record_byte5 & 1 != 0 {
                continue;
            }
            for index in 0..unit.native_transient.len() {
                if unit.native_transient[index] == 0 {
                    continue;
                }
                unit.native_transient[index] -= 1;
                let raw = &mut self.native_runtime_records[slot].raw[NATIVE_TRANSIENT_OFFSET as usize + index];
                *raw = raw.wrapping_sub(1);
                if unit.native_transient[index] == 0 {
                    result.expired.push(NativeTransientExpiry {
                        unit: Some(slot),
                        offset: NATIVE_TRANSIENT_OFFSET + index as i64,
                    });
                }
            }
        }
        Ok(result)
    }

    /// TickNativeTransients is retained for source compatibility only. Camp is a
    /// normalized remake enum, not the raw selector passed as 0x1A866's argument;
    /// mapping one to the other would reintroduce the withdrawn assertion. Callers
    /// must provide the recovered raw selector through TickNativeTransientsRaw.
    ///
    /// Go：`native_transient.go:172`，`func (s *State) TickNativeTransients(_ Camp) []NativeTransientExpiry`
    pub fn tick_native_transients(&self, _arg0: Camp) -> Vec<NativeTransientExpiry> {
        Vec::new()
    }

    /// AdvanceNativeTransientPhaseTyped 是 AdvanceNativeTransientPhaseRaw 在沒有 saved
    /// runtime raw 投影（從城鎮正常進戰場）時的版本：同一個 sub_1A866(selector) 的三段
    /// 順序（+0x25 扣 MaxHP/10 → sub_1DB65 標記 HP==0 → 遞減 +0x22..+0x27），只寫 typed
    /// 欄位。每個單位都要有 raw +5／+6 出處，缺了整段拒絕，不做部分掃描。
    ///
    /// Go：`native_transient.go:178`，`func (s *State) AdvanceNativeTransientPhaseTyped(selector byte) (NativeTransientPhaseResult, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_transient_phase_typed(&mut self, selector: u8) -> Result<NativeTransientPhaseResult> {
        if self.units.is_empty() {
            return Err(battle_err!("native transient phase: roster is empty"));
        }
        for (slot, unit) in self.units.iter().enumerate() {
            if !unit.has_native_record_byte5
                || !unit.has_native_record_byte6
                || !(0..=0xffff).contains(&unit.hp)
                || !(0..=0xffff).contains(&unit.max_hp)
            {
                return Err(battle_err!("native transient phase: unit {slot} lacks consumed raw fields"));
            }
        }
        let mut result = NativeTransientPhaseResult::default();
        let poison = (0x25 - NATIVE_TRANSIENT_OFFSET) as usize;
        for (slot, unit) in self.units.iter_mut().enumerate() {
            if unit.native_transient[poison] == 0
                || unit.native_record_byte6 != selector
                || unit.native_record_byte5 & 1 != 0
            {
                continue;
            }
            let (amount, before) = (unit.max_hp / 10, unit.hp);
            let after = (before - amount).max(0);
            unit.hp = after;
            result.damage.push(NativeTransientDamage { unit: Some(slot), slot: slot as i64, amount, before, after });
        }
        for (slot, unit) in self.units.iter_mut().enumerate() {
            if unit.hp != 0 {
                continue;
            }
            if unit.native_record_byte5 & 1 == 0 {
                result.inactive_slots.push(slot as i64);
            }
            unit.native_record_byte5 = 1;
        }
        for (slot, unit) in self.units.iter_mut().enumerate() {
            if unit.native_record_byte6 != selector || unit.native_record_byte5 & 1 != 0 {
                continue;
            }
            for (index, duration) in unit.native_transient.iter_mut().enumerate() {
                if *duration == 0 {
                    continue;
                }
                *duration -= 1;
                if *duration == 0 {
                    result.expired.push(NativeTransientExpiry {
                        unit: Some(slot),
                        offset: NATIVE_TRANSIENT_OFFSET + index as i64,
                    });
                }
            }
        }
        Ok(result)
    }
}

#[cfg(test)]
mod native_transient_test;
