//! 移植自 `separated_church_ui_test.go`。

use super::{TempDir, original_entry, pack};
use crate::fdother::separated_church_ui::{CHURCH_UI_FRAME_ENTRIES, CHURCH_UI_OPAQUE_ENTRIES};
use crate::fdother::*;
use std::path::Path;

#[test]
fn separated_church_ui_assets_match_original_codecs() {
    let Some(raw) = original_entry("FDOTHER.DAT", 14) else { return };
    let Some(root) = pack("ui", "fdother_014_church/resource.json") else { return };
    // Go 在設定 FD2_ASSET_PACK 時載入失敗即 Fatal；這裡素材包一定來自 FD2_ASSET_PACK。
    let got = load_separated_church_ui_assets(&root).unwrap();
    for index in CHURCH_UI_OPAQUE_ENTRIES {
        let want = parse_lmi1_opaque_entry(&raw, index).unwrap();
        assert_eq!(got.entries[index as usize], want, "opaque entry {index} differs");
    }
    for index in 3..=10 {
        let want = parse_lmi1_raw_entry(&raw, index).unwrap();
        let e = &got.entries[index as usize];
        assert!(e.width == 24 && e.height == 20 && e.pixels == want.pixels, "raw service{index} differs");
        if let Ok(wrong) = parse_lmi1_opaque_entry(&raw, index) {
            assert_ne!(wrong.pixels, want.pixels, "test did not distinguish wrong codec{index}");
        }
    }
    let want_price = parse_lmi1_raw_entry(&raw, 15).unwrap();
    assert_eq!(got.price_cell, want_price, "raw entry 15 differs");
    for index in CHURCH_UI_FRAME_ENTRIES {
        let want = parse_lmi1_frame_entry(&raw, index).unwrap();
        let (want_indexed, want_mask) = want.indexed_layers().unwrap();
        let frame = if index == 0 { &got.background } else { &got.revive_fx[(index - 23) as usize] };
        assert!(
            frame.width == want.width
                && frame.height == want.height
                && frame.indexed == want_indexed
                && frame.mask == want_mask,
            "frame entry {index} differs"
        );
    }
}

#[test]
fn separated_church_ui_assets_fail_closed_without_pack() {
    let dir = TempDir::new("church");
    assert!(load_separated_church_ui_assets(dir.path()).is_err(), "incomplete separated church UI pack was accepted");
}

fn copy_tree(source: &Path, target: &Path) {
    std::fs::create_dir_all(target).unwrap();
    for entry in std::fs::read_dir(source).unwrap() {
        let entry = entry.unwrap();
        let out = target.join(entry.file_name());
        if entry.file_type().unwrap().is_dir() {
            copy_tree(&entry.path(), &out);
        } else {
            std::fs::write(&out, std::fs::read(entry.path()).unwrap()).unwrap();
        }
    }
}

/// 舊素材包把服務格當成高位連續段解讀；即使每張必要 PNG 都在，也必須因那個明確的編碼拒收。
#[test]
fn separated_church_ui_assets_rejects_high_run_service_metadata() {
    let Some(ui) = pack("ui", "fdother_014_church/resource.json") else { return };
    let temp = TempDir::new("church-high-run");
    let target = temp.path().join("fdother_014_church");
    copy_tree(&ui.join("fdother_014_church"), &target);
    let metadata = target.join("resource.json");
    let mut doc: serde_json::Value = serde_json::from_slice(&std::fs::read(&metadata).unwrap()).unwrap();
    for entry in doc["entries"].as_array_mut().unwrap() {
        if entry["index"] == 3 {
            entry["codec"] = "opaque_high_run".into();
        }
    }
    std::fs::write(&metadata, serde_json::to_vec(&doc).unwrap()).unwrap();
    let err = load_separated_church_ui_assets(temp.path()).err().map(|e| e.to_string()).unwrap_or_default();
    assert!(
        err.contains("entry3 metadata mismatch") || err.contains("entry 3 metadata mismatch"),
        "old codec not explicitly rejected: {err}"
    );
}
