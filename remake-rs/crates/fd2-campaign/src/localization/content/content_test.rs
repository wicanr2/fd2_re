//! 移植自 Go `localization/content_test.go`。

use super::*;
use crate::test_support;

#[test]
fn test_official_content_assets_load_and_resolve_story() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    for locale in ["zh-Hant", "zh-Hans", "ja", "en"] {
        let catalog = load_official_content(&root, locale).unwrap_or_else(|e| panic!("official {locale} content: {e}"));
        catalog
            .story_text("legacy/line/7ecb566a60db/scenes/0/lines/0")
            .unwrap_or_else(|e| panic!("official {locale} first ch01 line: {e}"));
    }
}

#[test]
fn test_story_text_fails_closed() {
    let entry = |id: &str, id_status: &str, text: &str, status: &str| ContentEntry {
        string_id: id.to_string(),
        id_status: id_status.to_string(),
        role: "dialogue".to_string(),
        text: text.to_string(),
        status: status.to_string(),
        ..Default::default()
    };
    let catalog = ContentCatalog {
        entries: [
            entry("line/a/text", "stable_canonical", "ok", "reviewed"),
            entry("line/b/text", "provisional_legacy_path", "bad", "machine_draft"),
            entry("line/c/text", "stable_canonical", "bad", "blocked"),
        ]
        .into_iter()
        .map(|e| (e.string_id.clone(), e))
        .collect(),
        ..Default::default()
    };
    let got = catalog.story_text("line/a");
    assert_eq!(got.as_deref().ok(), Some("ok"), "resolved = {got:?}");
    for id in ["missing", "line/b", "line/c"] {
        assert!(catalog.story_text(id).is_err(), "{id} unexpectedly resolved");
    }
}

#[test]
fn test_load_official_content_rejects_unknown_field() {
    let Some(path) = test_support::remake_asset("locales/en/content.json") else {
        return;
    };
    let raw = std::fs::read_to_string(path).unwrap();
    // 筆數會隨字串清冊重生而變，不寫死；只要在 entry_count 後面塞一個未知欄位。
    let bad = raw.replacen(r#""entry_count": "#, r#""future": true, "entry_count": "#, 1);
    assert_ne!(bad, raw, "content.json lacks entry_count");
    let dir = test_support::temp_dir("content-unknown");
    std::fs::create_dir(dir.join("en")).unwrap();
    std::fs::write(dir.join("en").join("content.json"), bad).unwrap();
    assert!(load_official_content(&dir.to_string_lossy(), "en").is_err(), "unknown top-level content field accepted");
}
