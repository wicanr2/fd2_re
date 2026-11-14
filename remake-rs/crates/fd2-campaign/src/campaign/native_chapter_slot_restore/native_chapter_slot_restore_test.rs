//! 移植自 `remake/internal/campaign/native_chapter_slot_restore_test.go`。
//!
//! 兩個用 `campaign.Load` 讀完整戰役圖的測試等 A 組（PORTING.md「延後的測試」）。

use super::*;
use crate::campaign::native_chapter_slot_save::native_chapter_slot_save_test::empty_chapter_slot_snapshot;
use crate::test_support::remake_asset;

/// Go `loadNativeRestoreTestAssets`。
pub(crate) fn load_native_restore_test_assets() -> Option<(NativeCharacterCatalog, NativeIntermissionGateTable)> {
    let catalog_path = remake_asset("data/native_character_catalog.json")?;
    let gate_path = remake_asset("data/native_intermission_gate.json")?;
    Some((
        load_native_character_catalog(&catalog_path).unwrap(),
        load_native_intermission_gate_table(&gate_path).unwrap(),
    ))
}

/// Go `nativeRestoreRecord`。
pub(crate) fn native_restore_record(identity: u8, class: u8) -> save::PersistentRecord {
    let mut record = save::PersistentRecord::default();
    record.raw[8] = identity;
    record.raw[0x20] = class;
    record.raw[0x21] = 7;
    record.raw[0x3b] = 6;
    record
}

fn graph_with(nodes: &[(&str, &str)]) -> Campaign {
    let mut graph = Campaign::default();
    for (id, kind) in nodes {
        let node = Node { r#type: (*kind).to_string(), ..Default::default() };
        graph.nodes.insert((*id).to_string(), node);
    }
    graph
}

#[test]
fn test_native_intermission_gate_asset_matches_original_distribution() {
    let Some((_, table)) = load_native_restore_test_assets() else { return };
    for index in [22, 23, 24, 27, 28, 29] {
        assert_eq!(table.entries[index], 1, "gate[{index}]");
    }
    for (index, value) in table.entries.iter().enumerate() {
        assert!(*value == 0 || [22, 23, 24, 27, 28, 29].contains(&index), "unexpected gate[{index}]={value}");
    }
}

#[test]
fn test_native_intermission_gate_routes_every_saveable_chapter_to_campaign_node() {
    let Some((mut catalog, table)) = load_native_restore_test_assets() else { return };
    let Some(path) = remake_asset("scenarios/campaign_full.json") else { return };
    let graph = load(&path).unwrap();
    for raw_chapter in 1..table.entries.len() {
        let mut snapshot = empty_chapter_slot_snapshot();
        snapshot.verified.chapter = raw_chapter as u8;
        let plan = build_native_chapter_slot_restore_plan(&snapshot, &mut catalog, &table, Some(&graph))
            .unwrap_or_else(|err| panic!("raw chapter {raw_chapter}: {err}"));
        let want_type = if table.entries[raw_chapter] != 0 { "preparation" } else { "town" };
        let node = graph.nodes.get(&plan.entry_node);
        assert!(
            node.is_some_and(|node| node.r#type == want_type),
            "raw chapter {raw_chapter} route={:?} type={node:?}, want {want_type}",
            plan.entry_node
        );
    }
}

#[test]
fn test_native_intermission_gate_bounds_late_effective_route_override_to_its_source() {
    let Some((mut catalog, table)) = load_native_restore_test_assets() else { return };
    let Some(path) = remake_asset("scenarios/campaign_full.json") else { return };
    let graph = load(&path).unwrap();
    let mut snapshot = empty_chapter_slot_snapshot();
    snapshot.verified.chapter = 28;
    let plan = build_native_chapter_slot_restore_plan(&snapshot, &mut catalog, &table, Some(&graph)).unwrap();
    assert!(
        plan.native_chapter_index == 28 && plan.display_chapter == 29 && plan.entry_node == "preparation_ch29",
        "raw chapter 28 default route={plan:?}"
    );
    snapshot.source_save_sha256 = "f46d9c54d3037f84f05d72714569c282e63f39bf125251a9cf5cd9593ff3241f".into();
    let plan = build_native_chapter_slot_restore_plan(&snapshot, &mut catalog, &table, Some(&graph));
    assert!(
        plan.as_ref().is_ok_and(|plan| plan.entry_node == "preparation_ch30"),
        "fixed input effective route={plan:?}"
    );
    snapshot.source_save_sha256 = "df50919111cccb37a00215dea248a069617aa222545d0f4493ab4b00dc5b819f".into();
    let plan = build_native_chapter_slot_restore_plan(&snapshot, &mut catalog, &table, Some(&graph));
    assert!(
        plan.as_ref().is_ok_and(|plan| plan.entry_node == "preparation_ch29"),
        "chapter28 actual save skipped chapter29: route={plan:?}"
    );
}

#[test]
fn test_build_native_chapter_slot_restore_plan_routes_after_postbattle_gates() {
    let Some((mut catalog, table)) = load_native_restore_test_assets() else { return };
    let graph = graph_with(&[
        ("town_ch02", "town"),
        ("town_ch22", "town"),
        ("preparation_ch23", "preparation"),
        ("preparation_ch28", "preparation"),
    ]);
    for (raw, node) in [(1u8, "town_ch02"), (21, "town_ch22"), (22, "preparation_ch23"), (27, "preparation_ch28")] {
        let mut snapshot = empty_chapter_slot_snapshot();
        snapshot.slot = 2;
        snapshot.verified = save::VerifiedMetadata {
            chapter: raw,
            roster_count: 2,
            currency: 3456,
            hud_gate_a: 1,
            raw_53af9: 2,
            raw_51e61: 3,
            raw_51e62: 4,
        };
        snapshot.records[0] = native_restore_record(0, 1);
        snapshot.records[1] = native_restore_record(9, 5);
        let plan = build_native_chapter_slot_restore_plan(&snapshot, &mut catalog, &table, Some(&graph))
            .unwrap_or_else(|err| panic!("raw chapter {raw}: {err}"));
        assert!(
            plan.entry_node == node
                && plan.display_chapter == i64::from(raw) + 1
                && plan.native_chapter_index == i64::from(raw)
                && plan.currency == 3456
                && plan.hud_gate_a == 1
                && plan.raw53_af9 == 2
                && plan.raw51_e61 == 3
                && plan.raw51_e62 == 4
                && plan.party_join_order == vec![0, 9]
                && plan.party_roster.get(&9).is_some_and(|u| u.name == "悠妮"),
            "raw chapter {raw} plan={plan:?}"
        );
    }
}

#[test]
fn test_build_native_chapter_slot_restore_plan_fails_closed() {
    let Some((mut catalog, table)) = load_native_restore_test_assets() else { return };
    let graph = graph_with(&[("town_ch02", "town")]);
    let mut snapshot = empty_chapter_slot_snapshot();
    snapshot.verified.chapter = 1;
    snapshot.verified.roster_count = 2;
    snapshot.records[0] = native_restore_record(9, 5);
    snapshot.records[1] = native_restore_record(9, 5);
    let mut build = |snapshot: &save::ChapterSlotSnapshot| {
        build_native_chapter_slot_restore_plan(snapshot, &mut catalog, &table, Some(&graph))
    };
    assert!(build(&snapshot).is_err(), "duplicate native identity unexpectedly accepted");
    snapshot.verified.roster_count = 1;
    snapshot.verified.chapter = 0;
    assert!(build(&snapshot).is_err(), "missing town_ch01 unexpectedly accepted");
    snapshot.verified.chapter = 22;
    assert!(build(&snapshot).is_err(), "missing preparation_ch23 unexpectedly accepted");
    snapshot.slot = save::SLOT_COUNT;
    snapshot.verified.chapter = 1;
    assert!(build(&snapshot).is_err(), "out-of-range slot unexpectedly accepted");
    snapshot.slot = 0;
    snapshot.verified.roster_count = save::ROSTER_UNITS as u8 + 1;
    assert!(build(&snapshot).is_err(), "oversized roster count unexpectedly accepted");
}
