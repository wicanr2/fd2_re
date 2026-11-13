//! Go `separated_test.go`。

use super::*;

// Go: TestSeparatedResourceMatchesOriginalIndexedAnimation
#[test]
fn separated_resource_matches_original_indexed_animation() {
    let Some(archive) = original("FIGANI.DAT") else { return };
    let Some(pack) = original_pack() else { return };
    let root = format!("{pack}/animations");
    let want = decode_resource(&archive, 4).unwrap();
    let got = load_separated_resource(&root, 4).unwrap();
    assert_eq!(got, want, "separated FIGANI#4 differs from archive decoder");
}

// Go: TestSeparatedZeroHeaderFallbackMatchesOriginalPreviousResource
#[test]
fn separated_zero_header_fallback_matches_original_previous_resource() {
    let Some(archive) = original("FIGANI.DAT") else { return };
    let Some(pack) = original_pack() else { return };
    let root = format!("{pack}/animations");
    let want = decode_resource(&archive, 1).unwrap();
    let got = load_separated_resource_with_zero_header_fallback(&root, 2).unwrap();
    assert_eq!(got, want, "separated zero-header fallback differs from original previous resource");
}

// Go: TestSeparatedFDOTHERAnimationMatchesOriginalResource
#[test]
fn separated_fdother_animation_matches_original_resource() {
    let Some(archive) = original("FDOTHER.DAT") else { return };
    let Some(pack) = original_pack() else { return };
    let root = format!("{pack}/animations");
    let want = decode_resource(&archive, 18).unwrap();
    let got = load_separated_archive_resource(&root, "FDOTHER.DAT", 18).unwrap();
    assert_eq!(got, want, "separated FDOTHER#18 differs from archive decoder");
}

// Go: TestSeparatedResourceFailsClosed
#[test]
fn separated_resource_fails_closed() {
    let dir = TempDir::new("fails-closed");
    assert!(load_separated_resource(&dir.path(), 4).is_err(), "missing separated animation was accepted");
}

// Go: TestSeparatedZeroHeaderFallbackRequiresConfirmedStatus
#[test]
fn separated_zero_header_fallback_requires_confirmed_status() {
    let root = TempDir::new("zero-header-status");
    let directory = root.0.join("FIGANI_002");
    std::fs::create_dir_all(&directory).unwrap();
    // 與 Go 版 json.Marshal(map) 的輸出相同：鍵依字母排序、無空白。
    let document = |evidence: &str| {
        format!(
            concat!(
                r#"{{"evidence":"{}","extensions":{{}},"header_word_le":0,"kind":"animation_resource_status","#,
                r#""raw_size":3,"reason_code":"zero_header_word","resource_id":"animation-resource/figani_002","#,
                r#""schema_version":1,"source":{{"file":"FIGANI.DAT","resource":2}},"status":"empty_header_zero"}}"#
            ),
            evidence
        )
    };
    std::fs::write(directory.join("resource.json"), document("confirmed")).unwrap();
    assert!(
        load_separated_resource_with_zero_header_fallback(&root.path(), 2).is_err(),
        "confirmed zero header accepted without the fallback animation"
    );
    std::fs::write(directory.join("resource.json"), document("hypothesis")).unwrap();
    let zero = separated_resource_has_zero_header(&root.path(), 2);
    assert!(matches!(zero, Ok(false)), "unconfirmed zero header accepted: {zero:?}");
}
