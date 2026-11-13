//! 移植自 `separated_sound_test.go`。

use super::TempDir;
use crate::fdother::separated_sound::{
    SEPARATED_FDOTHER_MD5, SEPARATED_FDOTHER_SHA256, SEPARATED_FDOTHER_SIZE, separated_sound_count,
};
use crate::fdother::*;
use serde_json::json;

fn write_separated_sound_fixture(resource: i64) -> TempDir {
    let count = separated_sound_count(resource).unwrap_or(0);
    let root = TempDir::new("sound");
    let bank_root = root.path().join(format!("FDOTHER_{resource:03}"));
    std::fs::create_dir_all(&bank_root).unwrap();
    let mut samples = Vec::new();
    for index in 0..count {
        // Go：`"sample_00" + string(rune('0'+index)) + ".ogg"`；本測試只用 count <= 4。
        let path = format!("sample_00{}.ogg", char::from(b'0' + index as u8));
        samples.push(json!({
            "subresource": index, "source_byte_count": 4,
            "source_pcm_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
            "path": path, "cue_evidence": "typed_schedule", "classification_evidence": "confirmed",
        }));
        std::fs::write(bank_root.join(&path), b"OggSfixture").unwrap();
    }
    let doc = json!({
        "schema_version": 2, "kind": "fd2_pcm_sound_bank", "asset_id": format!("sfx/FDOTHER_{resource:03}"),
        "status": "converted",
        "source": {"name": "FDOTHER.DAT", "size": SEPARATED_FDOTHER_SIZE, "md5": SEPARATED_FDOTHER_MD5, "sha256": SEPARATED_FDOTHER_SHA256},
        "resource": resource, "container_count": count + 1, "zero_length_tail_index": count,
        "sample_rate": 11025, "channels": 1, "sample_format": "unsigned_u8",
        "timing_evidence": "hardware-spec_approximation", "samples": samples,
    });
    std::fs::write(bank_root.join("resource.json"), serde_json::to_vec(&doc).unwrap()).unwrap();
    root
}

fn check_bank(resource: i64, count: usize, probe: i64) {
    let root = write_separated_sound_fixture(resource);
    let bank = load_separated_sound_bank(root.path(), resource).unwrap();
    assert!(
        bank.resource == resource && bank.encoded.len() == count && &bank.encoded[&probe][..4] == b"OggS",
        "bank={bank:?}"
    );
}

#[test]
fn load_separated_sound_bank_fixture() {
    check_bank(82, 2, 1);
}

#[test]
fn load_separated_title_sound_bank_contract() {
    check_bank(77, 4, 2);
}

#[test]
fn load_separated_ani1_companion_sound_contract() {
    check_bank(78, 1, 0);
}

#[test]
fn load_separated_sound_bank_fails_closed() {
    let root = write_separated_sound_fixture(82);
    std::fs::remove_file(root.path().join("FDOTHER_082").join("sample_001.ogg")).unwrap();
    assert!(load_separated_sound_bank(root.path(), 82).is_err(), "missing OGG was accepted");
    assert!(load_separated_sound_bank(root.path(), 81).is_err(), "unsupported resource was accepted");
}
