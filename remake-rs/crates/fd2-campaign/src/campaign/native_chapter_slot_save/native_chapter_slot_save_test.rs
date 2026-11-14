//! 移植自 `remake/internal/campaign/native_chapter_slot_save_test.go`。

use super::*;
use crate::campaign::native_current_snapshot::native_current_snapshot_test::complete_native_current_save_unit;
use crate::test_support::remake_asset;

/// 空的 `fdsave.ChapterSlotSnapshot{}`（Rust 型別沒有 `Default`）。
pub(crate) fn empty_chapter_slot_snapshot() -> save::ChapterSlotSnapshot {
    save::ChapterSlotSnapshot {
        slot: 0,
        verified: save::VerifiedMetadata::default(),
        metadata: [0; save::METADATA_SIZE],
        records: [save::PersistentRecord::default(); save::ROSTER_UNITS],
        source_save_sha256: String::new(),
    }
}

// #87：未出戰record16的五個呈現欄位須留在原始持續槽，
// 同一角色仍可在戰後商店更改物品與能力值。
#[test]
fn test_encode_native_persistent_record_preserves_unmaterialized_presentation() {
    let mut baseline = save::PersistentRecord::default();
    (baseline.raw[0], baseline.raw[1]) = (133, 216);
    (baseline.raw[2], baseline.raw[3], baseline.raw[4]) = (204, 189, 189);
    (baseline.raw[7], baseline.raw[8]) = (15, 4);
    let mut unit = complete_native_current_save_unit();
    (unit.x, unit.y, unit.battle_fig) = (0, 0, 0);
    (unit.has_native_map_presentation, unit.has_map_selector_slot, unit.has_battle_fig) = (false, false, false);
    unit.native_map_presentation = battle::NativeMapPresentationState::default();
    unit.inventory_slots[0] = 24;
    unit.hp = 27;
    let got = encode_native_persistent_record(baseline, Some(&unit)).unwrap();
    for offset in [0, 1, 2, 3, 4, 7] {
        assert_eq!(got.raw[offset], baseline.raw[offset], "未物化+{offset:x}");
    }
    assert!(
        got.raw[0x0b] == 24 && u16::from_le_bytes([got.raw[0x40], got.raw[0x41]]) == 27,
        "未出戰者的物品或能力值沒有寫回"
    );
    // OnField不代表有raw呈現來源，不能用它覆寫保留欄位。
    unit.on_field = false;
    let again = encode_native_persistent_record(baseline, Some(&unit));
    assert!(again.as_ref().is_ok_and(|again| *again == got), "OnField改變存檔出處政策：err={:?}", again.err());
}

// TestBuildNativeChapterSlotOverlaysRosterOnBaseline 釘住酒店存檔寫回：已證實的欄位
// 由隊伍覆寫，其餘 byte 照讀檔基底，metadata 只改章節、筆數與金幣。第四章 r9 收據
// （state-sample9/FD2.SAV）與重製端 r38 寫出的整檔 sha256 相同就是這條規則的來源。
#[test]
fn test_build_native_chapter_slot_overlays_roster_on_baseline() {
    let Some(table_path) = remake_asset("data/native_join_constructor.json") else { return };
    let Some(rows_path) = remake_asset("data/native_item_effect_rows.json") else { return };
    let table = load_native_join_constructor_table(&table_path).unwrap();
    let item_rows = battle::load_native_item_effect_row_prefix(&rows_path).unwrap();
    let mut baseline = empty_chapter_slot_snapshot();
    baseline.verified.roster_count = 1;
    for (i, b) in baseline.records[0].raw.iter_mut().enumerate() {
        *b = 0xa0u8.wrapping_add(i as u8); // 不透明 byte 都要留下來
    }
    baseline.records[0].raw[8] = 4;
    for (i, b) in baseline.metadata.iter_mut().enumerate() {
        *b = 0xb0u8.wrapping_add(i as u8);
    }
    let mut unit = complete_native_current_save_unit();
    (unit.native_identity, unit.has_native_identity) = (4, true);
    unit.map_selector_slot = 3;
    (unit.native_map_presentation.pose, unit.native_map_presentation.motion) = (0, 0);
    (unit.base_ap, unit.base_dp, unit.equipment_base_set) = (78, 48, true);
    let roster = BTreeMap::from([(4, unit.clone())]);
    let slot = build_native_chapter_slot(&baseline, &roster, &[4], 4, 2037, &table, &item_rows).unwrap();
    let rec = &slot.roster[..save::UNIT_SIZE];
    let word = |off: usize| u16::from_le_bytes([rec[off], rec[off + 1]]);
    assert!(
        rec[0] == 7
            && rec[1] == 8
            && rec[2] == 3
            && rec[3] == 0
            && rec[4] == 0
            && rec[5] == 0x80
            && rec[6] == 2
            && rec[7] == 9
            && rec[8] == 4,
        "header={:x?}",
        &rec[..9]
    );
    assert!(
        rec[0x0a] == 0x40 && rec[0x0b] == 1 && rec[0x0c] == 0x80 && rec[0x0d] == 0xff && rec[0x21] == 12,
        "inventory/level={:x?} {}",
        &rec[0x0a..0x0e],
        rec[0x21]
    );
    assert!(
        rec[0x28] == 0xa0 + 0x28 && rec[0x30] == 0xa0 + 0x30 && rec[0x3d] == 7,
        "opaque bytes changed: +28={:#x} +30={:#x} +3d={:#x}",
        rec[0x28],
        rec[0x30],
        rec[0x3d]
    );
    assert!(
        word(0x37) == 78 && word(0x39) == 48 && rec[0x3c] == 13 && word(0x40) == 15 && word(0x48) == 19,
        "stats={:x?}",
        &rec[0x37..0x50]
    );
    for i in save::UNIT_SIZE..save::ROSTER_SIZE {
        assert_eq!(slot.roster[i], 0, "unused record byte {i:#x} changed");
    }
    assert!(
        slot.metadata[0] == 4
            && slot.metadata[1] == 1
            && u32::from_le_bytes(slot.metadata[2..6].try_into().unwrap()) == 2037
            && slot.metadata[6] == 0xb6
            && slot.metadata[0x27] == 0xb0 + 0x27,
        "metadata={:x?}",
        &slot.metadata[..8]
    );

    // 隊伍裡找不到基底身分、或多出來的成員不在加入順序裡，都整筆拒絕。
    assert!(
        build_native_chapter_slot(&baseline, &BTreeMap::new(), &[], 4, 0, &table, &item_rows).is_err(),
        "缺少基底身分應拒絕"
    );
    let mut extra = unit.clone();
    (extra.native_identity, extra.native_record_byte8) = (12, 12);
    let both = BTreeMap::from([(4, unit.clone()), (12, extra)]);
    assert!(
        build_native_chapter_slot(&baseline, &both, &[4], 4, 0, &table, &item_rows).is_err(),
        "不在加入順序的成員應拒絕"
    );
    let joined = build_native_chapter_slot(&baseline, &both, &[4, 12], 4, 0, &table, &item_rows).unwrap();
    assert!(
        joined.metadata[1] == 2 && joined.roster[save::UNIT_SIZE + 8] == 12,
        "JOIN 成員未補在第 2 筆：count={} +8={}",
        joined.metadata[1],
        joined.roster[save::UNIT_SIZE + 8]
    );
}
