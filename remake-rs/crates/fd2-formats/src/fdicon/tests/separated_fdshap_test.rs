//! 移植自 `separated_fdshap_test.go`。

use super::TempDir;
use crate::fdicon::*;

#[test]
fn separated_fdshap_rejects_incomplete_pack() {
    let root = TempDir::new("fdshap-incomplete");
    let directory = root.0.join("map_00");
    std::fs::create_dir_all(&directory).unwrap();
    let document = r#"{"schema_version":1,"kind":"fdshap_terrain_bank","asset_id":"tilesets/fdshap/map_00","status":"decoded","evidence":"confirmed","source":{"file":"FDSHAP.DAT","size":3557794,"md5":"9b0d356074f57cc27aebf3bb89aae247","sha256":"901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c"},"map_index":0,"image_resource":0,"control_resource":1,"controls":[0,0,0,0],"sprites":[]}"#;
    std::fs::write(directory.join("bank.json"), document).unwrap();
    assert!(load_separated_fdshap_bank(root.path(), 0).is_err(), "incomplete separated FDSHAP bank was accepted");
}

#[test]
fn separated_fdshap_rejects_invalid_map_index() {
    for index in [-1, SEPARATED_FDSHAP_MAP_COUNT] {
        let root = TempDir::new("fdshap-index");
        assert!(load_separated_fdshap_bank(root.path(), index).is_err(), "map index {index} was accepted");
    }
}
