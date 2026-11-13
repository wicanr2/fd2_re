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
        todo!("battle/native_transient.go:Unit.NativeTransientDuration")
    }

    /// SetNativeTransientDuration writes one recovered raw duration byte.  It is a
    /// bounded storage primitive; command-specific gates and side effects remain
    /// in their separately recovered executors.
    ///
    /// Go：`native_transient.go:55`，`func (u *Unit) SetNativeTransientDuration(offset int, duration byte) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_native_transient_duration(&mut self, offset: i64, duration: u8) -> bool {
        todo!("battle/native_transient.go:Unit.SetNativeTransientDuration")
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
        todo!("battle/native_transient.go:State.TickNativeTransientsRaw")
    }

    /// AdvanceNativeTransientPhaseRaw 重現 sub_1A866 的狀態寫入順序：先依 record 順序
    /// 對 +0x25 非零的同 selector 單位扣 MaxHP/10，再執行 sub_1DB65 的 HP==0 →
    /// raw +5=1，最後只替仍 active 的單位遞減 +0x22..+0x27。typed 與完整 raw
    /// projection 在同一個私有 State 中同步；任何矛盾都在 mutation 前拒絕。
    ///
    /// Go：`native_transient.go:94`，`func (s *State) AdvanceNativeTransientPhaseRaw(selector byte) (NativeTransientPhaseResult, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_transient_phase_raw(&mut self, selector: u8) -> Result<NativeTransientPhaseResult> {
        todo!("battle/native_transient.go:State.AdvanceNativeTransientPhaseRaw")
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
        todo!("battle/native_transient.go:State.AdvanceNativeTransientPhaseTyped")
    }
}
