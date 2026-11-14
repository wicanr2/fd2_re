//! 移植自 `remake/internal/campaign/native_persistent_party_test.go`。
//!
//! Go 的 `TestMaterializeNativePersistentPartyOriginalSnapshot` 讀環境變數
//! `FD2_NATIVE_SAVE_FIXTURE` 指定的使用者原版 FD2.SAV。Rust 測試環境只用四個統一變數
//! （`test_support.rs`），所以改讀 `FD2_FLAME2_DIR/FD2.SAV`；這就是知識庫 `56` 記載的那份
//! checksum 正確的使用者原版快照（實得索爾、悠妮、亞雷斯、蓋亞），期望值不變。

use super::*;
use crate::test_support::{input, remake_asset};

fn catalog() -> Option<NativeCharacterCatalog> {
    let path = remake_asset("data/native_character_catalog.json")?;
    Some(load_native_character_catalog(&path).unwrap())
}

#[test]
fn test_materialize_native_persistent_party_original_snapshot() {
    let Some(path) = input("FD2_FLAME2_DIR", "FD2.SAV") else { return };
    let Some(mut catalog) = catalog() else { return };
    let stored = std::fs::read(path).unwrap();
    let plain = save::decode(&stored).unwrap();
    let snapshot = save::inspect_current_snapshot(&plain).unwrap();
    let records = snapshot.active_persistent_records();
    let mut got = Vec::new();
    for (index, record) in records.into_iter().enumerate() {
        let unit = materialize_native_persistent_party_record(record, &mut catalog)
            .unwrap_or_else(|err| panic!("record {index}: {err}"));
        got.push(unit.name);
    }
    assert_eq!(got, ["索爾", "悠妮", "亞雷斯", "蓋亞"], "party={got:?}");
}

#[test]
fn test_native_character_catalog_asset_is_complete() {
    let Some(catalog) = catalog() else { return };
    for (identity, want) in [(0, "索爾"), (4, "亞雷斯"), (9, "悠妮"), (30, "蓋亞")] {
        assert_eq!(catalog.identity_names[identity], want, "identity {identity}");
    }
    assert_eq!(catalog.class_names[27], "　　", "class 27 want two full-width spaces");
    assert_eq!(catalog.class_names[28], "？？？", "class 28");
}

#[test]
fn test_materialize_native_persistent_party_record_preserves_proven_fields() {
    let Some(mut catalog) = catalog() else { return };
    let mut record = save::PersistentRecord::default();
    (record.raw[5], record.raw[6], record.raw[7], record.raw[8]) = (0x81, 2, 0x44, 9);
    for slot in 0..8 {
        record.raw[0x0a + slot * 2] = 0x80;
        record.raw[0x0b + slot * 2] = 0xff;
    }
    (record.raw[0x0a], record.raw[0x0b]) = (0x40, 0x12);
    record.raw[0x1a..0x1f].copy_from_slice(&[1, 2, 3, 4, 5]);
    (record.raw[0x1f], record.raw[0x20], record.raw[0x21]) = (1, 5, 7);
    record.raw[0x22..0x28].copy_from_slice(&[6, 7, 8, 9, 10, 11]);
    (record.raw[0x31], record.raw[0x32], record.raw[0x33]) = (3, 17, 42);
    (record.raw[0x34], record.raw[0x35], record.raw[0x36]) = (0x81, 0x22, 0x33);
    (record.raw[0x3b], record.raw[0x3c]) = (6, 42);
    for (offset, value) in [
        (0x37, 10i16),
        (0x39, 11),
        (0x3e, 12),
        (0x40, 13),
        (0x42, 14),
        (0x44, 15),
        (0x46, 16),
        (0x48, 17),
        (0x4a, 18),
        (0x4c, 19),
        (0x4e, 20),
    ] {
        record.raw[offset..offset + 2].copy_from_slice(&(value as u16).to_le_bytes());
    }

    let unit = materialize_native_persistent_party_record(record, &mut catalog).unwrap();
    assert!(
        unit.name == "悠妮"
            && unit.cls_name == "法師"
            && unit.native_identity == 9
            && unit.class_id == 5
            && unit.has_map_selector_key
            && unit.map_selector_key == 0x44
            && unit.hp == 13
            && unit.max_hp == 14
            && unit.ap == 17
            && unit.base_ap == 10
            && unit.base_hit == 12
            && unit.has_native_record_word42
            && unit.native_record_word42 == 14
            && unit.has_native_record_word46
            && unit.native_record_word46 == 16
            && unit.has_native_record_death_effect
            && unit.native_record_death_effect == [3, 17, 42]
            && unit.has_native_record_byte34
            && unit.native_record_byte34 == 0x81
            && unit.has_native_record_byte35
            && unit.native_record_byte35 == 0x22
            && unit.has_native_record_byte36
            && unit.native_record_byte36 == 0x33
            && unit.dx == 12
            && unit.exp == 42.0
            && unit.inventory == vec![0x12]
            && unit.equipped[0],
        "materialized unit={unit:?}"
    );
    assert!(
        unit.portrait == 0 && unit.fig == 0 && !unit.on_field && !unit.has_battle_fig,
        "unproven presentation fields were materialized: {unit:?}"
    );
}

#[test]
fn test_materialize_native_persistent_party_record_rejects_unknown_class() {
    let Some(mut catalog) = catalog() else { return };
    let mut record = save::PersistentRecord::default();
    record.raw[8] = 0;
    record.raw[0x20] = 0xff;
    assert!(
        materialize_native_persistent_party_record(record, &mut catalog).is_err(),
        "unknown class unexpectedly accepted"
    );
}

#[test]
fn test_native_character_catalog_rejects_reordered_identity() {
    let Some(mut catalog) = catalog() else { return };
    catalog.identities[9].id = 10;
    assert!(catalog.validate().is_err(), "reordered identity unexpectedly accepted");
}

#[test]
fn test_native_character_catalog_rejects_wrong_exe_version() {
    let Some(mut catalog) = catalog() else { return };
    catalog.source.exesha256 = "wrong".into();
    assert!(catalog.validate().is_err(), "wrong EXE version unexpectedly accepted");
}
