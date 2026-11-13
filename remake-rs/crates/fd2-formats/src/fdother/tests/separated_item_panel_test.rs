//! 移植自 `separated_item_panel_test.go`。

use super::{TempDir, original, original_entry, pack};
use crate::fdother::separated_item_panel::{
    ITEM_PANEL_FRAME_ENTRIES, ITEM_PANEL_OPAQUE_ENTRIES, ITEM_PANEL_RAW_ENTRIES,
};
use crate::fdother::*;

#[test]
fn separated_item_panel_entries_match_original_codecs() {
    let Some(raw) = original_entry("FDOTHER.DAT", 5) else { return };
    let Some(ui_root) = pack("ui", "fdother_005_item_panel/resource.json") else { return };
    let separated = load_separated_item_panel_entries(&ui_root).unwrap();
    for index in ITEM_PANEL_OPAQUE_ENTRIES {
        let want = parse_lmi1_opaque_entry(&raw, index).unwrap();
        assert_eq!(separated.opaque.get(&index), Some(&want), "opaque entry {index} differs");
    }
    for index in ITEM_PANEL_RAW_ENTRIES {
        let want = parse_lmi1_raw_entry(&raw, index).unwrap();
        assert_eq!(separated.raw.get(&index), Some(&want), "raw entry {index} differs");
    }
    for index in ITEM_PANEL_FRAME_ENTRIES {
        let want = parse_lmi1_frame_entry(&raw, index).unwrap();
        let (want_indexed, want_mask) = want.indexed_layers().unwrap();
        let got = separated.frames.get(&index).unwrap_or_else(|| panic!("frame entry {index} missing"));
        assert!(
            got.width == want.width
                && got.height == want.height
                && got.indexed == want_indexed
                && got.mask == want_mask,
            "frame entry {index} differs"
        );
    }
}

#[test]
fn separated_system_info_panels_match_fixed_archive() {
    let Some(ui_root) = pack("ui", "fdother_005_item_panel/resource.json") else { return };
    let Some(archive) = original("FDOTHER.DAT") else { return };
    let got = load_separated_system_info_panels(&ui_root).unwrap();
    let raw = read_resource(archive, 5).unwrap();
    for (offset, panel) in got.iter().enumerate() {
        let index = 0x85 + offset as i64;
        let want = parse_lmi1_frame_entry(&raw, index).unwrap();
        let (indexed, mask) = want.indexed_layers().unwrap();
        assert!(
            panel.width == want.width && panel.height == want.height && panel.indexed == indexed && panel.mask == mask,
            "separated system info entry {index:#x} differs from fixed archive"
        );
    }
}

#[test]
fn separated_system_info_panels_fails_closed_without_complete_bank() {
    let dir = TempDir::new("system-info");
    assert!(load_separated_system_info_panels(dir.path()).is_err(), "missing separated system info bank was accepted");
}

#[test]
fn separated_item_panel_entries_fail_closed_without_complete_pack() {
    let dir = TempDir::new("item-panel");
    assert!(
        load_separated_item_panel_entries(dir.path()).is_err(),
        "incomplete separated item panel pack was accepted"
    );
}
