//! 移植自 `remake/internal/campaign/class_change_table_test.go`。

use super::*;
use crate::campaign::church::church_test::unit;
use crate::test_support::docs_file;

/// Go：`classChangeFixture`。缺 `docs/` 時回 `None`（Go 直接 `t.Fatal`，那份檔案受版控）。
fn class_change_fixture() -> Option<ClassChangeTable> {
    let path = docs_file("data/exe_tables/class_change_targets.json")?;
    Some(load_class_change_table(&path).unwrap())
}

/// Go：`TestClassChangeTableSeparatesCurrentAndTargetIndexes`
#[test]
fn class_change_table_separates_current_and_target_indexes() {
    let Some(table) = class_change_fixture() else { return };
    assert!(
        table.current.len() == 0x12 && table.targets.len() == 0x22,
        "table lengths current={} target={}",
        table.current.len(),
        table.targets.len()
    );
    for i in 0..0x12 {
        let row = table.current.get(&i).unwrap_or_else(|| panic!("missing current row {i}"));
        assert!(row.portrait == i && row.default_target == i + 0x20, "current row {i}={row:?}");
        assert!(
            !(row.item_id == 0xff && row.optional_target.is_some()),
            "raw ff item unexpectedly has optional target: {row:?}"
        );
    }
    assert_eq!(table.current[&9].optional_target, Some(0x3b), "portrait 9 optional target, want 0x3b");
    for i in 0..0x22 {
        let row = table.targets.get(&(i + 0x20));
        assert!(row.is_some_and(|r| r.portrait == i + 0x20), "target row {}={row:?}", i + 0x20);
    }
}

/// Go：`TestNativeClassChangeTargetUsesOriginalOverwritePriority`
#[test]
fn native_class_change_target_uses_original_overwrite_priority() {
    let Some(table) = class_change_fixture() else { return };
    let mut u = unit! { portrait: 9, inventory: vec![0x5a, 0x58], };
    let (got, ok) = native_class_change_target(Some(&u), &table);
    assert!(
        ok && got.branch == "special"
            && got.required_item_id == 0x5a
            && got.inventory_index == 0
            && got.portrait == 0x34,
        "special override={got:?} ok={ok}"
    );
    u.inventory = vec![0x58];
    let (got, ok) = native_class_change_target(Some(&u), &table);
    assert!(
        ok && got.branch == "optional"
            && got.required_item_id == 0x58
            && got.inventory_index == 0
            && got.portrait == 0x3b,
        "optional override={got:?} ok={ok}"
    );
    u.inventory = Vec::new();
    let (got, ok) = native_class_change_target(Some(&u), &table);
    assert!(
        ok && got.branch == "default"
            && got.required_item_id == -1
            && got.inventory_index == -1
            && got.portrait == 0x29,
        "default target={got:?} ok={ok}"
    );
}

/// Go：`TestLoadClassChangeGrowthMapsRowsByPortraitIndex`
#[test]
fn load_class_change_growth_maps_rows_by_portrait_index() {
    let Some(path) = docs_file("data/exe_tables/growth.json") else { return };
    let growth = load_class_change_growth(&path).unwrap();
    assert_eq!(growth.len(), 36, "growth rows, want 36 rows idx>=32");
    assert!(
        growth[&0x20].ap == [8, 12] && growth[&0x41].mp == [0, 0],
        "unexpected mapped growth rows: 0x20={:?} 0x41={:?}",
        growth[&0x20],
        growth[&0x41]
    );
}
