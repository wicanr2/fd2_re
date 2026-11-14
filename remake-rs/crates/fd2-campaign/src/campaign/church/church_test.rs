//! 移植自 `remake/internal/campaign/church_test.go`。

use super::*;
use crate::test_support::docs_file;
use fd2_rules::battle::GoMathRand;

/// Go 測試的 `battle.Unit{Lv: 4, ...}` 字面值：`Unit` 有私有欄位，不能用 `..Default::default()`，
/// 改成先取零值再逐欄指定。B 組各測試檔共用。
macro_rules! unit {
    ($($field:ident: $value:expr),* $(,)?) => {{
        #[allow(clippy::field_reassign_with_default)]
        let mut built = ::fd2_rules::battle::Unit::default();
        $(built.$field = $value;)*
        built
    }};
}
pub(crate) use unit;

/// Go：`TestClassNameUsesNativeFDTXTClassRange`
#[test]
fn class_name_uses_native_fdtxt_class_range() {
    for (class, want) in [(26, "？？？"), (27, "　　"), (28, "？？？"), (29, "職業29")] {
        assert_eq!(class_name(class), want, "class {class}");
    }
}

/// Go：`TestLoadReviveFeeRatesUsesExportedEXETable`
#[test]
fn load_revive_fee_rates_uses_exported_exe_table() {
    let Some(path) = docs_file("data/exe_tables/revive_fee_rates.json") else { return };
    let rates = load_revive_fee_rates(&path).unwrap();
    assert!(
        rates.len() == 29 && rates[0] == 506 && rates[9] == 1200 && rates[25] == 350,
        "unexpected fee table length/values: len={} rates={rates:?}",
        rates.len()
    );
}

/// Go：`TestAdvanceNativeChurchServiceSelectionWrapsLeftRight`
#[test]
fn advance_native_church_service_selection_wraps_left_right() {
    for (current, delta, want) in [(0, -1, 3), (3, 1, 0), (1, -1, 0), (2, 1, 3), (99, 1, 1), (-1, -1, 3), (2, 0, 2)] {
        assert_eq!(advance_native_church_service_selection(current, delta), want, "current={current} delta={delta}");
    }
}

/// Go：`TestAdvanceNativeTwoColumnSelectionIsBounded`
#[test]
fn advance_native_two_column_selection_is_bounded() {
    for (current, count, delta, want) in [
        (0, 8, -1, 0),
        (0, 8, -2, 0),
        (1, 8, 1, 2),
        (2, 8, -2, 0),
        (6, 8, 2, 6),
        (5, 8, 2, 7),
        (7, 8, 1, 7),
        (99, 2, -1, 1),
    ] {
        assert_eq!(
            advance_native_two_column_selection(current, count, delta),
            want,
            "two-column current={current} count={count} delta={delta}"
        );
    }
}

/// Go：`TestReviveUnitUsesLevelFeeAndRestoresNativeFields`
#[test]
fn revive_unit_uses_level_fee_and_restores_native_fields() {
    let mut u = unit! {
        fig: 9,
        lv: 4,
        hp: 0,
        max_hp: 37,
        on_field: false,
        native_record_byte5: 1,
        has_native_record_byte5: true,
    };
    let (gold, cost, err) = revive_unit(321, Some(&mut u), 7);
    assert!(err.is_ok() && cost == 28 && gold == 293, "revive result gold={gold} cost={cost} err={err:?}");
    assert!(u.hp == 37 && u.on_field, "revive fields hp={} on_field={}", u.hp, u.on_field);
}

/// Go：`TestReviveUnitIsAtomicOnInsufficientGoldOrInvalidCandidate`
#[test]
fn revive_unit_is_atomic_on_insufficient_gold_or_invalid_candidate() {
    let mut dead = unit! {
        lv: 4,
        hp: 0,
        max_hp: 37,
        on_field: false,
        native_record_byte5: 1,
        has_native_record_byte5: true,
    };
    let (gold, cost, err) = revive_unit(27, Some(&mut dead), 7);
    assert!(
        err.is_err() && gold == 27 && cost == 28 && dead.hp == 0 && !dead.on_field,
        "insufficient-gold mutation gold={gold} cost={cost} err={err:?} unit={dead:?}"
    );
    let mut alive = unit! {
        lv: 4,
        hp: 1,
        max_hp: 37,
        on_field: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
    };
    let (gold, cost, err) = revive_unit(100, Some(&mut alive), 7);
    assert!(
        err.is_err() && gold == 100 && cost == 0 && alive.hp == 1,
        "alive candidate mutation gold={gold} cost={cost} err={err:?} unit={alive:?}"
    );
    let legacy_dead = unit! { lv: 4, hp: 0, max_hp: 37, };
    assert!(!can_revive(Some(&legacy_dead)), "HP projection without raw byte5 provenance must fail closed");
}

/// Go：`TestReviveUnitUsesRawLevelWithoutInventedMinimum`
#[test]
fn revive_unit_uses_raw_level_without_invented_minimum() {
    let mut u = unit! {
        lv: 0,
        hp: 0,
        max_hp: 9,
        native_record_byte5: 1,
        has_native_record_byte5: true,
    };
    let (gold, cost, err) = revive_unit(12, Some(&mut u), 7);
    assert!(
        err.is_ok() && gold == 12 && cost == 0 && u.hp == 9 && u.native_record_byte5 & 1 == 0,
        "level-zero raw revive gold={gold} cost={cost} err={err:?} unit={u:?}"
    );
}

/// Go：`TestClassChangeCandidatesMatchOriginal31793Predicate`
///
/// 期望值只有一個候選，不受 Go `church.go:257` map 迭代順序影響。
#[test]
fn class_change_candidates_match_original_31793_predicate() {
    let roster = BTreeMap::from([
        (0, unit! { fig: 0, lv: 20, portrait: 0, }),
        (9, unit! { fig: 9, lv: 19, portrait: 9, }),
        (4, unit! { fig: 4, lv: 20, portrait: 7, }),
        (30, unit! { fig: 30, lv: 20, portrait: 18, }),
    ]);
    let got = class_change_candidates(&roster, &[0, 9, 4, 30]);
    assert_eq!(got, vec![0], "class change candidates={got:?}, want [0]");
}

/// Go：`TestNativeClassConfirmationUsesLeftRightWithoutWrap`
#[test]
fn native_class_confirmation_uses_left_right_without_wrap() {
    assert_eq!(advance_native_class_confirmation(1, -1), 0, "left");
    assert_eq!(advance_native_class_confirmation(0, 1), 1, "right");
    assert_eq!(advance_native_class_confirmation(1, 0), 1, "no input");
}

/// Go：`TestNativeClassCandidateWindowShowsThreeRowsAndScrolls`
#[test]
fn native_class_candidate_window_shows_three_rows_and_scrolls() {
    for (count, selected, start, visible) in
        [(0, 0, 0, 0), (2, 1, 0, 2), (5, 0, 0, 3), (5, 2, 0, 3), (5, 3, 1, 3), (5, 4, 2, 3)]
    {
        assert_eq!(native_class_candidate_window(count, selected), (start, visible), "window({count},{selected})");
    }
}

/// Go：`TestNativeThreeRowWindowPreservesStatefulOrigin`
#[test]
fn native_three_row_window_preserves_stateful_origin() {
    let mut start = 0;
    for (selected, want) in [(0, 0), (2, 0), (3, 1), (4, 2), (3, 2), (1, 1), (0, 0)] {
        let visible;
        (start, visible) = native_three_row_window(6, selected, start);
        assert!(start == want && visible == 3, "selected{selected} window=({start},{visible}), want ({want},3)");
    }
}

/// Go：`TestApplyClassChangeAddsGrowthAndConsumesItem`
#[test]
fn apply_class_change_adds_growth_and_consumes_item() {
    let mut u = unit! {
        fig: 9,
        portrait: 9,
        battle_fig: 9,
        map_selector_key: 9,
        has_map_selector_key: true,
        map_selector_slot: 3,
        has_map_selector_slot: true,
        class_id: 5,
        native_record_race: 1,
        has_native_record_race: true,
        native_record_class: 5,
        has_native_record_class: true,
        lv: 27,
        exp: 73.0,
        ap: 99,
        dp: 88,
        dx: 77,
        mv: 5,
        inventory: vec![0x5a, 0x20],
        equipped: vec![true, false],
        inventory_slots: vec![0x5a, 0x20],
    };
    let row = ClassChangeGrowth { ap: [10, 11], dp: [20, 21], dx: [30, 31], hp: [40, 41], mp: [50, 51] };
    let mut rng = GoMathRand::new(1);
    apply_class_change(Some(&mut u), 0x34, 21, 2, row, Some(&mut rng), 0).unwrap();
    assert!(
        u.ap == 109 && u.dp == 108 && u.dx == 107 && u.max_hp == 40 && u.hp == 40 && u.max_mp == 50 && u.mp == 50,
        "incremented stats AP/DP/DX={}/{}/{} HP={}/{} MP={}/{}",
        u.ap,
        u.dp,
        u.dx,
        u.hp,
        u.max_hp,
        u.mp,
        u.max_mp
    );
    assert!(
        u.lv == 27 && u.exp == 0.0 && u.mv == 7 && u.portrait == 0x34 && u.class_id == 21,
        "metadata lv={} exp={} mv={} portrait={:x} class={}",
        u.lv,
        u.exp,
        u.mv,
        u.portrait,
        u.class_id
    );
    assert!(
        u.fig == 9
            && u.battle_fig == 0x34
            && u.has_map_selector_key
            && u.map_selector_key == 0x34
            && !u.has_map_selector_slot,
        "class-change selectors fig={} battle={} key={} known={} slot-known={}",
        u.fig,
        u.battle_fig,
        u.map_selector_key,
        u.has_map_selector_key,
        u.has_map_selector_slot
    );
    assert!(
        u.has_native_record_race
            && u.native_record_race == 1
            && u.has_native_record_class
            && u.native_record_class == 21,
        "class-change raw race/class={}/{} known={}/{}",
        u.native_record_race,
        u.native_record_class,
        u.has_native_record_race,
        u.has_native_record_class
    );
    assert!(
        u.inventory.len() == 1 && u.inventory[0] == 0x20 && !u.equipped[0],
        "item removal inventory={:?} equipped={:?}",
        u.inventory,
        u.equipped
    );
}

/// Go：`TestApplyClassChangeRejectsInvalidRangeAtomically`
#[test]
fn apply_class_change_rejects_invalid_range_atomically() {
    let mut u = unit! { lv: 20, ap: 7, inventory: vec![9], };
    let mut rng = GoMathRand::new(1);
    let row = ClassChangeGrowth { ap: [2, 1], ..Default::default() };
    let err = apply_class_change(Some(&mut u), 0x20, 9, 1, row, Some(&mut rng), 0);
    assert!(err.is_err() && u.ap == 7 && u.inventory.len() == 1, "invalid range mutated unit: err={err:?} unit={u:?}");
}
