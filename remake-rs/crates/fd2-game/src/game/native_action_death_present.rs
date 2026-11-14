//! Go 來源：`remake/cmd/fd2/native_action_death_present.go`（241 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Settlement already writes HP and may mirror +5=1. Preserve the actual entry
/// flag for private presentation snapshots; never reconstruct it from a corpse.
///
/// Go：`native_action_death_present.go:11`，`func captureNativeActionDeathFlags(units ...*battle.Unit) map[*battle.Unit]byte`
pub(crate) fn capture_native_action_death_flags(units: &[UnitIdx]) -> BTreeMap<UnitIdx, u8> {
    todo!("native_action_death_present.go:captureNativeActionDeathFlags")
}

impl Game {
    /// Go：`native_action_death_present.go:21`，`func (g *Game) queueNativeActionDeathPresentation(before map[*battle.Unit]byte)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn queue_native_action_death_presentation(&mut self, before: &BTreeMap<UnitIdx, u8>) {
        todo!("native_action_death_present.go:queueNativeActionDeathPresentation")
    }
}

/// Go：`native_action_death_present.go:37`，`func nativeDeathFlagPrelude(flags map[int]byte) func(*battle.State) error`
pub(crate) fn native_death_flag_prelude(
    flags: &BTreeMap<i64, u8>,
) -> Option<Box<dyn FnMut(&mut Game, &battle::State) -> Result<()> + Send>> {
    todo!("native_action_death_present.go:nativeDeathFlagPrelude")
}

impl Game {
    /// Go：`native_action_death_present.go:56`，`func (g *Game) nativeMapDeathPlan(flags map[int]byte) (bool, error)`
    pub(crate) fn native_map_death_plan(&mut self, flags: &BTreeMap<i64, u8>) -> Result<bool> {
        todo!("native_action_death_present.go:nativeMapDeathPlan")
    }

    /// Native and HD frames share the existing typed producer. The retained live
    /// state blocks through the same owner, without replacing actor/target pointers.
    ///
    /// Go：`native_action_death_present.go:71`，`func (g *Game) startNativeMapDeathVisual(flags map[int]byte, kind string, then func()) (bool, error)`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_map_death_visual(
        &mut self,
        flags: &mut BTreeMap<i64, u8>,
        kind: &str,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<bool> {
        todo!("native_action_death_present.go:startNativeMapDeathVisual")
    }
}

/// Go：`native_action_death_present.go:83`，`func nativeTransientDeathFlags(before, after *battle.State) map[int]byte`
pub(crate) fn native_transient_death_flags(
    before: Option<&battle::State>,
    after: Option<&battle::State>,
) -> BTreeMap<i64, u8> {
    todo!("native_action_death_present.go:nativeTransientDeathFlags")
}

impl Game {
    /// Preflight before the status transaction is published. Final focus can change
    /// the viewport after damage messages, so activation rebuilds from that view.
    ///
    /// Go：`native_action_death_present.go:99`，`func (g *Game) preflightNativeTransientDeath(candidate *battle.State, flags map[int]byte, damage []battle.NativeTransientDamage) error`
    pub(crate) fn preflight_native_transient_death(
        &self,
        candidate: &mut battle::State,
        flags: &BTreeMap<i64, u8>,
        damage: &mut [battle::NativeTransientDamage],
    ) -> Result<()> {
        todo!("native_action_death_present.go:preflightNativeTransientDeath")
    }

    /// Go：`native_action_death_present.go:110`，`func (g *Game) nativeTransientDeathPreview(candidate *battle.State, damage []battle.NativeTransientDamage) (*Game, error)`
    pub(crate) fn native_transient_death_preview(
        &self,
        candidate: &mut battle::State,
        damage: &[battle::NativeTransientDamage],
    ) -> Result<Option<Game>> {
        todo!("native_action_death_present.go:nativeTransientDeathPreview")
    }

    /// Private focus uses the same cursor primitives without emitting replay input.
    ///
    /// Go：`native_action_death_present.go:119`，`func (g *Game) nativeTransientFocusPreview(candidate *battle.State, slots []int) (*Game, error)`
    pub(crate) fn native_transient_focus_preview(
        &self,
        candidate: &mut battle::State,
        slots: &[i64],
    ) -> Result<Option<Game>> {
        todo!("native_action_death_present.go:nativeTransientFocusPreview")
    }

    /// Go：`native_action_death_present.go:179`，`func (g *Game) beginNativeActionDeathPresentation(actor *battle.Unit, after func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_action_death_presentation(
        &mut self,
        actor: UnitIdx,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        todo!("native_action_death_present.go:beginNativeActionDeathPresentation")
    }

    /// Go：`native_action_death_present.go:228`，`func (g *Game) nativeActionDeathPresentationPending() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_action_death_presentation_pending(&self) -> bool {
        todo!("native_action_death_present.go:nativeActionDeathPresentationPending")
    }
}
