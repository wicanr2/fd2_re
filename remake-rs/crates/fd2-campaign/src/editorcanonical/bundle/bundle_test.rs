//! Go `remake/internal/editorcanonical/bundle_test.go` 的逐條移植。

use super::*;

/// Go：`TestCheckedInBundleValidatesAllCanonicalDocuments`
#[test]
fn checked_in_bundle_validates_all_canonical_documents() {
    let Some(root) = crate::test_support::remake_asset("editor-canonical") else {
        return;
    };
    let summary = validate_bundle(&root).unwrap();
    assert!(
        summary.documents.len() == 66 && summary.identity_diagnostics == 4,
        "summary documents={} identity diagnostics={}",
        summary.documents.len(),
        summary.identity_diagnostics
    );
}

/// Go：`TestBundleHashMismatchFailsClosed`
#[test]
fn bundle_hash_mismatch_fails_closed() {
    let Some(source) = crate::test_support::remake_asset("editor-canonical") else {
        return;
    };
    let root = crate::test_support::temp_dir("bundle-incomplete");
    std::os::unix::fs::symlink(
        std::path::Path::new(&source).join("bundle-summary.json"),
        root.join("bundle-summary.json"),
    )
    .expect("建立符號連結");
    assert!(validate_bundle(&root.to_string_lossy()).is_err(), "incomplete bundle was accepted");
}
