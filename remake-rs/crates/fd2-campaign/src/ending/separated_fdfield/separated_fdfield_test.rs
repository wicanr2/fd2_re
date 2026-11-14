//! 移植自 `remake/internal/ending/separated_fdfield_test.go`。

use super::*;
use crate::test_support::{input, temp_dir};

/// Go `selector30SeparatedRoot`：`FD2_ASSET_PACK/fields/fdfield`。
pub(crate) fn selector30_separated_root() -> Option<String> {
    input("FD2_ASSET_PACK", "fields/fdfield/selector_30/field.json")?;
    input("FD2_ASSET_PACK", "fields/fdfield").map(|p| p.to_string_lossy().into_owned())
}

#[test]
fn test_separated_fdfield_selector30_matches_fixed_archive_resources() {
    let Some(root) = selector30_separated_root() else { return };
    let Some(archive) = input("FD2_FLAME2_DIR", "FDFIELD.DAT") else { return };
    let (got_map, got_control, got_positions) = load_separated_fdfield_selector30(&root).unwrap();
    for (offset, got) in [got_map, got_control, got_positions].iter().enumerate() {
        let want = fdother::read_resource(&archive, 90 + offset).unwrap();
        assert_eq!(*got, want, "separated FDFIELD #{} differs from fixed archive", 90 + offset);
    }
}

#[test]
fn test_separated_fdfield_selector30_fails_closed_on_edited_content_without_updated_provenance() {
    let Some(root) = selector30_separated_root() else { return };
    let raw = std::fs::read(std::path::Path::new(&root).join("selector_30").join("field.json")).unwrap();
    let mut doc: SeparatedFDFIELDDocument = serde_json::from_slice(&raw).unwrap();
    doc.tiles[0] ^= 1;
    let temp_root = temp_dir("selector30");
    let directory = temp_root.join("selector_30");
    std::fs::create_dir_all(&directory).unwrap();
    std::fs::write(directory.join("field.json"), serde_json::to_vec(&doc).unwrap()).unwrap();
    assert!(
        load_separated_fdfield_selector30(&temp_root.to_string_lossy()).is_err(),
        "edited selector 30 content unexpectedly bypassed source hash gate"
    );
}
