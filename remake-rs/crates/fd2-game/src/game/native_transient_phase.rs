//! Go 來源：`remake/cmd/fd2/native_transient_phase.go`（198 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// applyNativeTransientPhases owns one or more ordered sub_1A866(selector)
    /// sweeps as a single publication. Each sweep applies the proven +0x25 HP
    /// writer, sub_1DB65 inactive mark and raw +0x22..+0x27 countdown. When any
    /// byte expires, it runs the proven 0x1B750 equipment/derived-stat
    /// recalculation before publishing the state. This path deliberately has no
    /// killer and never enters the action-only 0x1B6B7/0x1AA1D dispatch.
    ///
    /// Go：`native_transient_phase.go:17`，`func (g *Game) buildNativeTransientPhases(selectors ...byte) (*battle.State, []battle.NativeTransientExpiry, []battle.NativeTransientDamage, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn build_native_transient_phases(
        &self,
        selectors: &[u8],
    ) -> Result<(Option<battle::State>, Vec<battle::NativeTransientExpiry>, Vec<battle::NativeTransientDamage>)> {
        todo!("native_transient_phase.go:buildNativeTransientPhases")
    }

    /// adoptNativeStateCandidate 把候選狀態寫回 g.st，但保留原本的 Unit 指標：候選是整份
    /// 複製出來的（buildNativeTransientPhases 逐筆 clone），直接 `*g.st = *candidate` 會把
    /// Units 換成一批新指標，凡是握著舊指標的（g.sel、AI 計畫、deathRewarded、選擇器快取、
    /// 重播端的 actor）都會指到過期的副本。第七章 r6 seq 1763：selector 0 掃描搬到橫幅之後
    /// 才跑，換指標讓整回合的單位畫格都對不上。
    ///
    /// Go：`native_transient_phase.go:117`，`func (g *Game) adoptNativeStateCandidate(candidate *battle.State) map[*battle.Unit]*battle.Unit`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn adopt_native_state_candidate(
        &mut self,
        candidate: Option<&mut battle::State>,
    ) -> BTreeMap<UnitIdx, UnitIdx> {
        todo!("native_transient_phase.go:adoptNativeStateCandidate")
    }

    /// buildNativeTransientPhasesTyped 是從城鎮正常進戰場（沒有 saved runtime raw 投影）
    /// 時的 sub_1A866 掃描：同樣的 +0x25 扣血、sub_1DB65 標記與 +0x22..+0x27 遞減，
    /// 到期的單位以 0x1B750 的規則從基礎攻防＋裝備重算 +0x48..+0x4E（強化到期時
    /// AP 從 1.15 倍回到基礎值；第六章 r4 收據 seq 177：記錄 15 的 +0x22 歸零、AP 93→80）。
    /// 每個單位都要有 raw +5／+6 與基礎攻防出處，缺了整段拒絕。
    ///
    /// Go：`native_transient_phase.go:145`，`func (g *Game) buildNativeTransientPhasesTyped(selectors ...byte) (*battle.State, []battle.NativeTransientExpiry, []battle.NativeTransientDamage, error)`
    pub(crate) fn build_native_transient_phases_typed(
        &self,
        selectors: &[u8],
    ) -> Result<(Option<battle::State>, Vec<battle::NativeTransientExpiry>, Vec<battle::NativeTransientDamage>)> {
        todo!("native_transient_phase.go:buildNativeTransientPhasesTyped")
    }
}
