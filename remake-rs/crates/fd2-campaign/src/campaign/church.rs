//! Go 來源：`remake/internal/campaign/church.go`（298 行）。

use super::*;

/// Go：`church.go:12`
pub(crate) static CLASS_NAMES: &[&str] = &[
    "龍",
    "劍士",
    "戰士",
    "騎士",
    "弓兵",
    "法師",
    "僧侶",
    "盜賊",
    "武者",
    "劍聖",
    "聖戰士",
    "聖騎士",
    "狙擊手",
    "大法師",
    "祭師",
    "龍劍士",
    "鬥士",
    "英雄",
    "魔戰士",
    "龍騎士",
    "神射手",
    "召喚師",
    "聖者",
    "忍者",
    "武聖",
    "機兵",
    "？？？",
    "　　",
    "？？？",
];

/// AdvanceNativeChurchServiceSelection mirrors 0x2d7bd's four-entry service
/// cursor. The native selector reacts to left (raw scancode 75) and right
/// (raw scancode 77), wrapping 0..3; it does not use the up/down movement
/// contract of the party/character lists. Keep the operation pure so the UI
/// can share the recovered ABI without assigning names to the four services.
///
/// Go：`church.go:19`，`func AdvanceNativeChurchServiceSelection(current, delta int) int`
pub fn advance_native_church_service_selection(current: i64, delta: i64) -> i64 {
    todo!("campaign/church.go:AdvanceNativeChurchServiceSelection")
}

/// AdvanceNativeClassConfirmation mirrors 0x19953's two-choice prompt:
/// left selects zero and right selects one. There is no wrap or up/down path.
///
/// Go：`church.go:42`，`func AdvanceNativeClassConfirmation(current, horizontal int) int`
pub fn advance_native_class_confirmation(current: i64, horizontal: i64) -> i64 {
    todo!("campaign/church.go:AdvanceNativeClassConfirmation")
}

/// NativeClassCandidateWindow mirrors 0x311DC's three-row viewport. The
/// selection is bounded by the caller; once it moves below the third visible
/// row, the window follows it one row at a time.
///
/// Go：`church.go:58`，`func NativeClassCandidateWindow(count, selected int) (start, visible int)`
/// Go 具名回傳值依序為：start、visible。
pub fn native_class_candidate_window(count: i64, selected: i64) -> (i64, i64) {
    todo!("campaign/church.go:NativeClassCandidateWindow")
}

/// NativeThreeRowWindow mirrors the stateful vertical viewport shared by
/// 0x30c22 and 0x311dc. The caller preserves start between moves; it changes
/// by one only after the selection crosses the current three-row window.
///
/// Go：`church.go:76`，`func NativeThreeRowWindow(count, selected, start int) (nextStart, visible int)`
/// Go 具名回傳值依序為：nextStart、visible。
pub fn native_three_row_window(count: i64, selected: i64, start: i64) -> (i64, i64) {
    todo!("campaign/church.go:NativeThreeRowWindow")
}

/// AdvanceNativeTwoColumnSelection mirrors the bounded list movement used by
/// 0x2e6b8/0x2df6b. delta is one of -2,-1,+1,+2 (left/up/right/down); invalid
/// counts or moves leave the cursor unchanged, and there is no wrap.
///
/// Go：`church.go:103`，`func AdvanceNativeTwoColumnSelection(current, count, delta int) int`
pub fn advance_native_two_column_selection(current: i64, count: i64, delta: i64) -> i64 {
    todo!("campaign/church.go:AdvanceNativeTwoColumnSelection")
}

/// ClassName is the EXE mechanical class-name table used by target class IDs.
///
/// Go：`church.go:124`，`func ClassName(classID int) string`
pub fn class_name(class_id: i64) -> String {
    todo!("campaign/church.go:ClassName")
}

/// ClassChangeGrowth is one 11-byte EXE growth row (0x4e4d1).  The five
/// pairs are encoded as [min,max), matching 0x1e529's idiv(max-min) path.
///
/// Go：`church.go:133`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ClassChangeGrowth {
    pub ap: [i64; 2],
    pub dp: [i64; 2],
    pub dx: [i64; 2],
    pub hp: [i64; 2],
    pub mp: [i64; 2],
}

/// Go：`church.go:137`，`func rollClassChangeRange(r [2]int, rng *rand.Rand) (int, error)`
pub(crate) fn roll_class_change_range(r: [i64; 2], rng: Option<&mut dyn GoRand>) -> Result<i64> {
    todo!("campaign/church.go:rollClassChangeRange")
}

/// ApplyClassChange mirrors the proven 0x31602 state writes.  0x1e529 adds the
/// rolled row value to the existing raw AP/DP/DX/MaxHP/MaxMP words; it does not
/// replace them. The native routine leaves the level byte untouched and clears
/// EXP, then copies the new maxima to current HP/MP. The caller owns the
/// editable class-name lookup and subsequent 0x1b750-equivalent equipment
/// recomputation. removeItemIndex is the compact Inventory index returned by
/// the church item scan, or -1 when this branch consumed no item.
///
/// Go：`church.go:157`，`func ApplyClassChange(u *battle.Unit, targetPortrait, classID, growthGroup int, row ClassChangeGrowth, rng *rand.Rand, removeItemIndex int) error`
pub fn apply_class_change(
    u: Option<&mut Unit>,
    target_portrait: i64,
    class_id: i64,
    growth_group: i64,
    row: ClassChangeGrowth,
    rng: Option<&mut dyn GoRand>,
    remove_item_index: i64,
) -> Result<()> {
    todo!("campaign/church.go:ApplyClassChange")
}

/// Go：`church.go:215`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ReviveFeeTable {
    /// Go 標籤：`json:"rates"`
    pub rates: Vec<i64>,
}

/// LoadReviveFeeRates loads the direct EXE-derived class fee words. The source
/// field is intentionally documentation-only; callers receive an indexed copy.
///
/// Go：`church.go:221`，`func LoadReviveFeeRates(path string) ([]int, error)`
pub fn load_revive_fee_rates(path: &str) -> Result<Vec<i64>> {
    todo!("campaign/church.go:LoadReviveFeeRates")
}

/// CanChangeClass is the proven 0x31793 candidate predicate. The native
/// routine uses the roster record's level and portrait byte; it does not list
/// already promoted portrait groups (>=0x12) or portrait 7.
///
/// Go：`church.go:239`，`func CanChangeClass(u *battle.Unit) bool`
pub fn can_change_class(u: Option<&Unit>) -> bool {
    u.is_some_and(|u| u.lv >= 20 && u.portrait < 0x12 && u.portrait != 7)
}

/// ClassChangeCandidates preserves caller order (normally JOIN chronology).
///
/// Go：`church.go:244`，`func ClassChangeCandidates(roster map[int]battle.Unit, order []int) []int`
pub fn class_change_candidates(roster: &BTreeMap<i64, Unit>, order: &[i64]) -> Vec<i64> {
    todo!("campaign/church.go:ClassChangeCandidates")
}

/// CanRevive reproduces 0x309ff -> 0x3453e: the roster record is a candidate
/// exactly when raw byte +5 has bit 0 set. Unknown raw provenance fails closed.
///
/// Go：`church.go:267`，`func CanRevive(u *battle.Unit) bool`
pub fn can_revive(u: Option<&Unit>) -> bool {
    u.is_some_and(|u| u.has_native_record_byte5 && u.native_record_byte5 & 1 != 0)
}

/// ReviveUnit applies the proven 0x30dc3 write-back sequence. feeRate is the
/// original class fee word loaded from the editable class-fee table; keeping it
/// as an argument prevents the engine from inventing values until that table is
/// exported. Native cost is feeRate * unit level, checked before any mutation.
///
/// Go：`church.go:275`，`func ReviveUnit(gold int, u *battle.Unit, feeRate int) (int, int, error)`
pub fn revive_unit(gold: i64, u: Option<&mut Unit>, fee_rate: i64) -> Result<(i64, i64)> {
    todo!("campaign/church.go:ReviveUnit")
}
