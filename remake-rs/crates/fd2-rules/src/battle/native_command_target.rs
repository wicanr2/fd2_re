//! Go 來源：`remake/internal/battle/native_command_target.go`（575 行）。

use super::*;

/// NativeCommandGridBlocked is original grid flag bit 0x40: 0x4e16e
/// refuses to enter this cell during the dist<0x10 flood-fill.
pub const NATIVE_COMMAND_GRID_BLOCKED: u8 = 0x40;

/// NativeCommandGridZeroBudget is original grid flag bit 0x80: 0x4e16e
/// forces the destination cell's remaining byte to zero after subtracting
/// the terrain cost. It does not make a zero-cost chain.
pub const NATIVE_COMMAND_GRID_ZERO_BUDGET: u8 = 0x80;

/// NativeCompositionBaseFlags reproduces the +2 part of 0x4dbfc. The exported
/// FDFIELD byte may carry archive bits outside low5, but the live composition
/// buffer clears them before any targeting writer runs.
///
/// Go：`native_command_target.go:18`，`func NativeCompositionBaseFlags(w, h int, eventBytes []byte) ([]byte, error)`
pub fn native_composition_base_flags(w: i64, h: i64, event_bytes: &[u8]) -> Result<Vec<u8>> {
    todo!("battle/native_command_target.go:NativeCompositionBaseFlags")
}

impl State {
    /// NativeCommandBaseFlags reproduces the baseline seen by 0x14818 callers.
    /// 0x1cff0, 0x1bbdc, 0x1598a and 0x1567e reset the mutable composition buffer
    /// with 0x4dbfc after each candidate lifetime, so State must not retain a
    /// caller-owned live flag slice between commands.
    ///
    /// Go：`native_command_target.go:33`，`func (s *State) NativeCommandBaseFlags() ([]byte, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_base_flags(&self) -> Result<Vec<u8>> {
        todo!("battle/native_command_target.go:State.NativeCommandBaseFlags")
    }
}

/// NativeCommandRuntimeFlags reproduces the verified 0x145cd→0x14625/
/// 0x146a7 writer on top of the 0x4dbfc base. selector zero marks active raw
/// +6!=0 records; any nonzero selector marks active raw +6==0 records. Each
/// selected unit sets 0x40 on its cell and 0x80 on its in-bounds cardinal
/// neighbours. It requires raw presentation and record provenance throughout.
///
/// Go：`native_command_target.go:47`，`func NativeCommandRuntimeFlags( w, h int, eventBytes []byte, units []*Unit, selector byte, ) ([]byte, error)`
pub fn native_command_runtime_flags(
    w: i64,
    h: i64,
    event_bytes: &[u8],
    units: &[Unit],
    selector: u8,
) -> Result<Vec<u8>> {
    todo!("battle/native_command_target.go:NativeCommandRuntimeFlags")
}

/// applyNativeRuntimeFlagWriter mutates one caller-owned live composition
/// slice. Native code may call 0x145cd more than once before 0x4dbfc resets
/// the slice, so rebuilding the archive baseline inside this helper would lose
/// earlier writers.
///
/// Go：`native_command_target.go:67`，`func applyNativeRuntimeFlagWriter( w, h int, flags []byte, units []*Unit, selector byte, ) error`
pub(crate) fn apply_native_runtime_flag_writer(
    w: i64,
    h: i64,
    flags: &mut [u8],
    units: &[Unit],
    selector: u8,
) -> Result<()> {
    todo!("battle/native_command_target.go:applyNativeRuntimeFlagWriter")
}

/// NativeFutureGroupPlacement reproduces the placement prefix of
/// 0x10c50. rawGate is the exact byte read from [0x53afa]: zero runs
/// 0x145cd(0), then 0x145cd(1), and selects the unoccupied cell with the
/// smallest Manhattan distance from the low bytes of the paired six-byte
/// position row. The native jg branch replaces equal-distance candidates, so
/// the last cell in row-major order wins a tie. Any nonzero gate uses the
/// authored low-byte coordinate directly.
///
/// The mutable 0x40/0x80 flags are caller-local. This function never writes
/// State.NativeCompositionEventBytes, which is immutable archive provenance.
///
/// Go：`native_command_target.go:118`，`func NativeFutureGroupPlacement( w, h int, eventBytes []byte, units []*Unit, position NativePositionRecord, rawGate byte, ) (Cell, error)`
pub fn native_future_group_placement(
    w: i64,
    h: i64,
    event_bytes: &[u8],
    units: &[Unit],
    position: NativePositionRecord,
    raw_gate: u8,
) -> Result<Cell> {
    todo!("battle/native_command_target.go:NativeFutureGroupPlacement")
}

/// NativeCommandTargetCells mirrors one 0x14818 geometry invocation before
/// roster filtering.  It intentionally accepts the raw fourth argument
/// (called mode in the original callsite), rather than declaring it a command
/// range: generic 0x1cff0 first calls it with record+3 from the actor, lets the
/// cursor confirm a candidate, then calls it again with record+4 from that
/// cursor cell to build the effect list.  Callers must select the correct stage.
///
/// flags is the caller-owned live per-cell +2 byte after 0x4dbfc and any
/// caller-specific 0x145cd writer, not the immutable FDFIELD export or remake
/// movement cost.
///
/// Go：`native_command_target.go:179`，`func NativeCommandTargetCells(w, h int, origin Cell, dist int, flags []byte) (map[Cell]bool, error)`
pub fn native_command_target_cells(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    flags: &[u8],
) -> Result<BTreeMap<Cell, bool>> {
    todo!("battle/native_command_target.go:NativeCommandTargetCells")
}

/// Go：`native_command_target.go:191`，`func nativeCommandTargetBudgets(w, h int, origin Cell, dist int, flags []byte) (map[Cell]int, error)`
pub(crate) fn native_command_target_budgets(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    flags: &[u8],
) -> Result<BTreeMap<Cell, i64>> {
    todo!("battle/native_command_target.go:nativeCommandTargetBudgets")
}

/// NativeCommandTargetFieldBytes materializes the byte+3 state produced by
/// 0x14818 for one target stage. 0x4dbfc first fills every cell with 0xff;
/// 0x4e040 then writes remaining-budget bytes for dist<0x10, while the cross
/// branch writes zero. A nonzero inner radius restores the strict inner
/// Manhattan area to 0xff.
///
/// Go：`native_command_target.go:257`，`func NativeCommandTargetFieldBytes(w, h int, origin Cell, dist, innerRadius int, flags []byte) ([]byte, error)`
pub fn native_command_target_field_bytes(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    inner_radius: i64,
    flags: &[u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_command_target.go:NativeCommandTargetFieldBytes")
}

/// Go：`native_command_target.go:279`，`func absInt(n int) int`
pub(crate) fn abs_int(n: i64) -> i64 {
    n.wrapping_abs()
}

/// NativeCommandTargetMatches is the exact record+6 predicate in 0x14818.
/// The native constructor ABI is Enemy=0, Ally=1, Own=2, whereas the Go Camp
/// enum is Own=0, Ally=1, Enemy=2.  Use named values here rather than copying
/// the native ordinals into the remake model.
///
/// Go：`native_command_target.go:290`，`func NativeCommandTargetMatches(code int, camp Camp) bool`
pub fn native_command_target_matches(code: i64, camp: Camp) -> bool {
    todo!("battle/native_command_target.go:NativeCommandTargetMatches")
}

/// NativeCursorConfirmationAllowed preserves the non-relocation Enter/Space
/// gate in 0x115b6. fieldByte is the selected FDFIELD cell's fourth byte
/// ([0x53a51 + 4*cell + 7]), not composition cell +2 flags. targetCode 5 and a 0xff
/// cell reject before targetCode 4 accepts.
///
/// For target codes 0..3 on a non-0xff cell, 0x115b6 derives a strict
/// Manhattan radius from [0x51a83] (decrementing values above one) and accepts
/// only when 0x14742 finds at least one active matching unit inside that radius.
/// targetCode 6 owns a separate relocation legality branch and is rejected
/// here rather than being approximated.
///
/// Go：`native_command_target.go:315`，`func NativeCursorConfirmationAllowed(cursor Cell, fieldByte byte, overlaySelector, targetCode int, units []*Unit) (bool, error)`
pub fn native_cursor_confirmation_allowed(
    cursor: Cell,
    field_byte: u8,
    overlay_selector: i64,
    target_code: i64,
    units: &[Unit],
) -> Result<bool> {
    todo!("battle/native_command_target.go:NativeCursorConfirmationAllowed")
}

/// nativeTargetUnitUsable applies the raw 0x14818 inactive predicate whenever
/// the supplied roster is fully materialized. Legacy hand-built test/scenario
/// units without raw provenance retain the normalized projection only as an
/// explicit compatibility boundary.
///
/// Go：`native_command_target.go:351`，`func nativeTargetUnitUsable(unit *Unit, rawComplete bool) bool`
pub(crate) fn native_target_unit_usable(unit: Option<&Unit>, raw_complete: bool) -> bool {
    todo!("battle/native_command_target.go:nativeTargetUnitUsable")
}

/// Go：`native_command_target.go:364`，`func nativeTargetRosterRawComplete(units []*Unit) bool`
pub(crate) fn native_target_roster_raw_complete(units: &[Unit]) -> bool {
    todo!("battle/native_command_target.go:nativeTargetRosterRawComplete")
}

/// Go：`native_command_target.go:376`，`func nativeTargetActorUsable(unit *Unit, rawComplete bool) bool`
pub(crate) fn native_target_actor_usable(unit: Option<&Unit>, raw_complete: bool) -> bool {
    todo!("battle/native_command_target.go:nativeTargetActorUsable")
}

/// NativeCommandTargets applies one recovered 0x14818 invocation and its
/// record+6 camp predicate to runtime units.  It is deliberately independent
/// of CastArea; for generic commands its origin/mode must be the confirmed
/// cursor and record+4 to represent the final effect list.
///
/// Go：`native_command_target.go:392`，`func NativeCommandTargets(w, h int, origin Cell, dist, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_targets(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    todo!("battle/native_command_target.go:NativeCommandTargets")
}

/// NativeAttackCandidates preserves the additional 0x14818 geometry used by
/// 0x14237. mode is raw a4 and innerRadius is raw a5 from the item row; for
/// mode<0x10 the callee excludes cells with Manhattan distance strictly less
/// than innerRadius after the four-way grid pass. mode>=0x10 is the native
/// cross branch and does not apply the inner-radius marker.
///
/// Go：`native_command_target.go:415`，`func NativeAttackCandidates(w, h int, origin Cell, mode, innerRadius, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_attack_candidates(
    w: i64,
    h: i64,
    origin: Cell,
    mode: i64,
    inner_radius: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    todo!("battle/native_command_target.go:NativeAttackCandidates")
}

/// NativeCommandEffectTargets 保留既有直接單位入口的候選檢查。
/// 它不能表示空格或我方格的範圍中心；玩家指令6改用 AtCursor 入口。
/// 原版 caller 與取代原因見 fd2_player_command6_cursor_center_20261004.json。
///
/// Go：`native_command_target.go:446`，`func NativeCommandEffectTargets(w, h int, actor, confirmed *Unit, selectionMode, effectMode, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor`、`confirmed` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_effect_targets(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    confirmed: Option<UnitIdx>,
    selection_mode: i64,
    effect_mode: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    todo!("battle/native_command_target.go:NativeCommandEffectTargets")
}

/// NativeCommandEffectTargetsAtCursor 依 sub_1CFF0 的一般分支驗證游標中心。
/// 0x1D2BF 建 actor selection field，0x1D2E3 確認游標，0x1D32A 再以
/// cursor、record+4 建作用名單。中心格可為空格或我方格，不能先當成目標單位。
/// 證據：docs/data/ida/fd2_player_command6_cursor_center_20261004.json。
///
/// Go：`native_command_target.go:472`，`func NativeCommandEffectTargetsAtCursor(w, h int, actor *Unit, cursor Cell, selectionMode, effectMode, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_effect_targets_at_cursor(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    cursor: Cell,
    selection_mode: i64,
    effect_mode: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    todo!("battle/native_command_target.go:NativeCommandEffectTargetsAtCursor")
}

/// NativeCommandSelfCenteredTargets follows sub_1CFF0 0x1D366..0x1D3B4.
/// SelectionMode0 fixes the enemy list and effect field at the invoking actor.
/// A nonempty list confirms with literal targetCode4 and overlay1; an empty list
/// uses targetCode5. Moving the confirmation cursor never recenters the list.
///
/// Go：`native_command_target.go:503`，`func NativeCommandSelfCenteredTargets(w, h int, actor *Unit, cursor Cell, effectMode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_self_centered_targets(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    cursor: Cell,
    effect_mode: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    todo!("battle/native_command_target.go:NativeCommandSelfCenteredTargets")
}

/// NativeCommand30Targets mirrors the special 0x149F8 selector used only by
/// player command 30.  It starts at the cursor position saved before 0x115B6,
/// advances SelectionMode-0x10 cells towards the confirmed cursor, and keeps
/// only the first active unit found in each cell when it is an enemy.  X has
/// priority: only when both cursors share X does it use Y.  This deliberately
/// does not reuse the generic 0x14818 two-stage geometry.
///
/// Go：`native_command_target.go:537`，`func NativeCommand30Targets(w, h int, savedCursor, confirmedCursor Cell, steps int, units []*Unit) ([]*Unit, error)`
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command30_targets(
    w: i64,
    h: i64,
    saved_cursor: Cell,
    confirmed_cursor: Cell,
    steps: i64,
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    todo!("battle/native_command_target.go:NativeCommand30Targets")
}
