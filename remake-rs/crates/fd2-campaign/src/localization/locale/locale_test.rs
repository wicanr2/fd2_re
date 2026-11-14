//! 移植自 Go `localization/locale_test.go`。

use super::*;
use crate::test_support;

/// Go `writePack`：寫進新的暫存目錄，回傳 `pack.json` 路徑。
fn write_pack(tag: &str, body: &str) -> String {
    let dir = test_support::temp_dir(tag);
    let path = dir.join("pack.json");
    std::fs::write(&path, body).unwrap();
    path.to_string_lossy().into_owned()
}

const VALID_PACK: &str = r#"{"schema_version":1,"pack_id":"test-en","locale":"en","kind":"official","layout_profile":"dos-8x8-latin","font":{"profile":"fd2-latin-8x8","path":"fonts/atlas.png"},"entries":{"battle.attack.hit":{"text":"%s deals %d damage","variables":["%s","%d"],"source_string_id":"native-1"}}}"#;

#[test]
fn test_load_and_format() {
    let catalog = load(&write_pack("load-format", VALID_PACK)).unwrap();
    let got = catalog.format("battle.attack.hit", &["A".into(), 3.into()]);
    assert_eq!(got.as_deref().ok(), Some("A deals 3 damage"), "format = {got:?}");
}

#[test]
fn test_format_fails_closed() {
    let catalog = load(&write_pack("format-closed", VALID_PACK)).unwrap();
    assert!(catalog.format("missing", &[]).is_err(), "unknown key accepted");
    assert!(catalog.format("battle.attack.hit", &["A".into()]).is_err(), "wrong argument count accepted");
    assert!(
        catalog.format("battle.attack.hit", &["A".into(), "not-a-number".into()]).is_err(),
        "incompatible argument type accepted"
    );
}

#[test]
fn test_load_rejects_unknown_and_unsafe_fields() {
    for (name, body) in [
        ("unknown top-level", VALID_PACK.replacen(r#""entries":"#, r#""future":true,"entries":"#, 1)),
        (
            "unknown entry",
            VALID_PACK.replacen(
                r#""source_string_id":"native-1""#,
                r#""source_string_id":"native-1","future":true"#,
                1,
            ),
        ),
        ("unsafe font", VALID_PACK.replacen("fonts/atlas.png", "../atlas.png", 1)),
        ("legacy key", VALID_PACK.replacen("battle.attack.hit", "legacy.go.main.l1", 1)),
    ] {
        assert!(load(&write_pack("load-reject", &body)).is_err(), "{name}: invalid pack accepted");
    }
}

#[test]
fn test_community_base_locale_rules() {
    let base = VALID_PACK.replacen(r#""kind":"official""#, r#""kind":"community","base_locale":"zh-Hant""#, 1);
    let base = base.replacen(r#""locale":"en""#, r#""locale":"zh-Hans""#, 1);
    load(&write_pack("community", &base)).unwrap();
    let bad = base.replacen(r#""base_locale":"zh-Hant""#, r#""base_locale":"zh-Hant-x-test""#, 1);
    assert!(load(&write_pack("community-bad", &bad)).is_err(), "same base locale accepted");
}

#[test]
fn test_load_official_uses_locale_path_and_identity() {
    let dir = test_support::temp_dir("load-official");
    let locale_dir = dir.join("en");
    std::fs::create_dir(&locale_dir).unwrap();
    std::fs::write(locale_dir.join("pack.json"), VALID_PACK).unwrap();
    let root = dir.to_string_lossy();
    load_official(&root, "en").unwrap();
    assert!(load_official(&root, "ja").is_err(), "missing official pack accepted");
}

#[test]
fn test_official_locale_assets_load() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    for locale in ["zh-Hant", "zh-Hans", "ja", "en"] {
        load_official(&root, locale).unwrap_or_else(|e| panic!("official {locale} pack: {e}"));
    }
}
