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
    let mut current = current;
    if !(0..=3).contains(&current) {
        current = 0;
    }
    let delta = match delta.signum() {
        0 => return current,
        d => d,
    };
    current += delta;
    if current < 0 {
        return 3;
    }
    if current > 3 {
        return 0;
    }
    current
}

/// AdvanceNativeClassConfirmation mirrors 0x19953's two-choice prompt:
/// left selects zero and right selects one. There is no wrap or up/down path.
///
/// Go：`church.go:42`，`func AdvanceNativeClassConfirmation(current, horizontal int) int`
pub fn advance_native_class_confirmation(current: i64, horizontal: i64) -> i64 {
    let mut current = current;
    if current != 0 && current != 1 {
        current = 0;
    }
    if horizontal < 0 {
        return 0;
    }
    if horizontal > 0 {
        return 1;
    }
    current
}

/// NativeClassCandidateWindow mirrors 0x311DC's three-row viewport. The
/// selection is bounded by the caller; once it moves below the third visible
/// row, the window follows it one row at a time.
///
/// Go：`church.go:58`，`func NativeClassCandidateWindow(count, selected int) (start, visible int)`
/// Go 具名回傳值依序為：start、visible。
pub fn native_class_candidate_window(count: i64, selected: i64) -> (i64, i64) {
    if count <= 0 || selected < 0 || selected >= count {
        return (0, 0);
    }
    let start = (selected - 2).max(0);
    let visible = (count - start).min(3);
    (start, visible)
}

/// NativeThreeRowWindow mirrors the stateful vertical viewport shared by
/// 0x30c22 and 0x311dc. The caller preserves start between moves; it changes
/// by one only after the selection crosses the current three-row window.
///
/// Go：`church.go:76`，`func NativeThreeRowWindow(count, selected, start int) (nextStart, visible int)`
/// Go 具名回傳值依序為：nextStart、visible。
pub fn native_three_row_window(count: i64, selected: i64, start: i64) -> (i64, i64) {
    if count <= 0 || selected < 0 || selected >= count || start < 0 {
        return (0, 0);
    }
    let max_start = (count - 3).max(0);
    let mut start = start.min(max_start);
    while selected < start && start > 0 {
        start -= 1;
    }
    while selected >= start + 3 && start < max_start {
        start += 1;
    }
    let visible = (count - start).min(3);
    (start, visible)
}

/// AdvanceNativeTwoColumnSelection mirrors the bounded list movement used by
/// 0x2e6b8/0x2df6b. delta is one of -2,-1,+1,+2 (left/up/right/down); invalid
/// counts or moves leave the cursor unchanged, and there is no wrap.
///
/// Go：`church.go:103`，`func AdvanceNativeTwoColumnSelection(current, count, delta int) int`
pub fn advance_native_two_column_selection(current: i64, count: i64, delta: i64) -> i64 {
    if count <= 0 {
        return 0;
    }
    if current < 0 {
        return 0;
    }
    if current >= count {
        return count - 1;
    }
    if !matches!(delta, -2 | -1 | 1 | 2) {
        return current;
    }
    let next = current + delta;
    if next < 0 || next >= count {
        return current;
    }
    next
}

/// ClassName is the EXE mechanical class-name table used by target class IDs.
///
/// Go：`church.go:124`，`func ClassName(classID int) string`
pub fn class_name(class_id: i64) -> String {
    if class_id >= 0 && (class_id as usize) < CLASS_NAMES.len() {
        return CLASS_NAMES[class_id as usize].to_string();
    }
    format!("職業{class_id}")
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
/// Rust 收 `&mut Option<..>`：同一個 rng 要連續借給五次擲骰。
pub(crate) fn roll_class_change_range(r: [i64; 2], rng: &mut Option<&mut dyn GoRand>) -> Result<i64> {
    if r[1] < r[0] {
        return Err(campaign_err!("class change: invalid range [{},{})", r[0], r[1]));
    }
    if r[1] == r[0] {
        return Ok(r[0]);
    }
    let Some(rng) = rng.as_deref_mut() else {
        return Err(campaign_err!("class change: missing rng"));
    };
    Ok(r[0].wrapping_add(rng.intn(r[1].wrapping_sub(r[0]))))
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
    let Some(u) = u else {
        return Err(campaign_err!("class change: missing unit"));
    };
    if !(0..=0xff).contains(&target_portrait) || !(0..=0xff).contains(&class_id) || !(0..=0xff).contains(&growth_group)
    {
        return Err(campaign_err!("class change: invalid target/class/group"));
    }
    if remove_item_index >= u.inventory.len() as i64 || remove_item_index < -1 {
        return Err(campaign_err!("class change: invalid item index"));
    }
    let mut rng = rng;
    let ap = roll_class_change_range(row.ap, &mut rng)?;
    let dp = roll_class_change_range(row.dp, &mut rng)?;
    let dx = roll_class_change_range(row.dx, &mut rng)?;
    let hp = roll_class_change_range(row.hp, &mut rng)?;
    let mp = roll_class_change_range(row.mp, &mut rng)?;
    if remove_item_index >= 0 && !u.remove_inventory_index(remove_item_index) {
        return Err(campaign_err!("class change: item removal failed"));
    }
    u.ap = u.ap.wrapping_add(ap);
    u.dp = u.dp.wrapping_add(dp);
    u.dx = u.dx.wrapping_add(dx);
    u.max_hp = u.max_hp.wrapping_add(hp);
    u.max_mp = u.max_mp.wrapping_add(mp);
    u.hp = u.max_hp;
    u.mp = u.max_mp;
    // The second byte returned by 0x4e48d is the class mobility increment;
    // native unit+0x3b is the movement budget used by pathfinding.
    u.mv = u.mv.wrapping_add(growth_group);
    u.exp = 0.0;
    (u.portrait, u.class_id) = (target_portrait, class_id);
    // 0x31571..0x3157a rewrites raw +0x20 and +7, but not +0x1f.
    (u.native_record_class, u.has_native_record_class) = (class_id as u8, true);
    // Native class-change flow 0x31576..0x3157a writes the resolved target
    // byte to live unit+7. For player construction that byte is both the
    // FIGANI selector and the next 0x11019 raw map key; stable Fig remains the
    // JOIN/+8 identity. A previous cache slot belongs to the old key.
    u.battle_fig = target_portrait;
    (u.map_selector_key, u.has_map_selector_key) = (target_portrait, true);
    (u.map_selector_slot, u.has_map_selector_slot) = (0, false);
    (u.native_map_presentation, u.has_native_map_presentation) = (battle::NativeMapPresentationState::default(), false);
    Ok(())
}

/// Go：`church.go:215`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
pub(crate) struct ReviveFeeTable {
    /// Go 標籤：`json:"rates"`
    #[serde(default, deserialize_with = "battle::null_as_default")]
    pub rates: Vec<i64>,
}

/// LoadReviveFeeRates loads the direct EXE-derived class fee words. The source
/// field is intentionally documentation-only; callers receive an indexed copy.
///
/// Go：`church.go:221`，`func LoadReviveFeeRates(path string) ([]int, error)`
pub fn load_revive_fee_rates(path: &str) -> Result<Vec<i64>> {
    let raw = battle::read_file(path)?;
    let table: ReviveFeeTable = battle::decode_json(&raw)?;
    if table.rates.is_empty() {
        return Err(campaign_err!("revive fee table is empty"));
    }
    Ok(table.rates)
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
    let mut out = Vec::new();
    let mut seen = BTreeSet::new();
    for &id in order {
        if !seen.insert(id) {
            continue;
        }
        if roster.get(&id).is_some_and(|u| can_change_class(Some(u))) {
            out.push(id);
        }
    }
    // Go `church.go:257` 依 map 迭代順序附加不在 order 裡的 id（順序不固定）；BTreeMap 依遞增順序。
    for (&id, u) in roster {
        if !seen.contains(&id) && can_change_class(Some(u)) {
            out.push(id);
        }
    }
    out
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
/// Rust 回傳 `(gold, cost, Result)`：Go 失敗時仍回傳有意義的 `cost`（金幣不足時是應付金額），
/// 包成 `Result<(i64, i64)>` 會丟掉它。
pub fn revive_unit(gold: i64, u: Option<&mut Unit>, fee_rate: i64) -> (i64, i64, Result<()>) {
    let Some(u) = u else {
        return (gold, 0, Err(campaign_err!("revive: missing unit")));
    };
    if !can_revive(Some(u)) {
        return (gold, 0, Err(campaign_err!("revive: unit is not a candidate")));
    }
    if fee_rate < 0 {
        return (gold, 0, Err(campaign_err!("revive: invalid fee rate")));
    }
    if u.lv < 0 {
        return (gold, 0, Err(campaign_err!("revive: invalid level")));
    }
    let cost = fee_rate.wrapping_mul(u.lv);
    if gold < cost {
        return (gold, cost, Err(campaign_err!("revive: insufficient gold")));
    }
    let gold = gold - cost;
    // 0x30f9c clears the raw death/inactive bit; 0x30fa0 copies max HP to
    // current HP. OnField is retained as the remake projection.
    u.restore_native_hp();
    u.on_field = true;
    (gold, cost, Ok(()))
}

#[cfg(test)]
pub(crate) mod church_test;
