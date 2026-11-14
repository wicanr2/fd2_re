//! 移植自 `remake/internal/campaign/native_join_base_test.go`。

use super::*;
use crate::test_support::remake_asset;

#[test]
fn test_native_join_base_table_loads_explicit_map17_records() {
    let Some(path) = remake_asset("data/native_join_base_units.json") else { return };
    let table = load_native_join_base_table(&path).unwrap();
    for id in [7, 21] {
        let unit = table.load_base_unit(id).unwrap_or_else(|err| panic!("base {id}: {err}"));
        assert!(
            unit.camp == battle::ALLY
                && unit.has_native_record_byte8
                && i64::from(unit.native_record_byte8) == id
                && unit.fig == id,
            "base {id} lost allied/raw identity: {unit:?}"
        );
    }
}

#[test]
fn test_native_join_base_table_rejects_unknown_identity() {
    let Some(path) = remake_asset("data/native_join_base_units.json") else { return };
    let table = load_native_join_base_table(&path).unwrap();
    assert!(table.load_base_unit(12).is_err(), "unindexed JOIN base was accepted");
}
