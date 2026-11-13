//! 移植自 `separated_palette_test.go`。

use super::{TempDir, original, pack};
use crate::fdother::*;

#[test]
fn separated_ending_palette_matches_fixed_archive() {
    let Some(root) = pack("palette", "fdother_057.json") else { return };
    let Some(archive) = original("FDOTHER.DAT") else { return };
    let want = read_resource(archive, 57).unwrap();
    let (got, palette) = load_separated_fdother_palette(&root, 57).unwrap();
    assert!(
        got == want && palette.len() == 256,
        "separated ending palette bytes={} colors={}",
        got.len(),
        palette.len()
    );
}

#[test]
fn separated_ending_palette_rejects_out_of_range_component() {
    let Some(root) = pack("palette", "fdother_057.json") else { return };
    // Go 解碼成文件結構、改一個分量再編碼；這裡在 JSON 值上做同樣的修改。
    let mut doc: serde_json::Value =
        serde_json::from_slice(&std::fs::read(root.join("fdother_057.json")).unwrap()).unwrap();
    doc["dac_6bit_components"][0] = 64.into();
    let temp = TempDir::new("palette");
    std::fs::write(temp.path().join("fdother_057.json"), serde_json::to_vec(&doc).unwrap()).unwrap();
    assert!(
        load_separated_fdother_palette(temp.path(), 57).is_err(),
        "out-of-range ending palette component unexpectedly passed"
    );
}
