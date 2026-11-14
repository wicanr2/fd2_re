//! `locale.go` 載入函式的測試。Go 的對應測試在 `story_locale_test.go`（逐句譯文核對）與
//! `assets_test.go`（`packageSelfCheck`，`r8`）；逐句譯文核對見 `PORTING.md`「A 組測試」。

use super::*;
use crate::test_support;

/// Go `settings.go:28` `localeIDs`（`settings.go` 是 F 組的檔，這裡照抄常值）。
const LOCALE_IDS: [&str; 4] = ["zh-Hant", "zh-Hans", "ja", "en"];

/// 受版控的四個官方語系都通過 `loadOfficialLocale` 的契約核對（Go `packageSelfCheck` 的語系段）。
#[test]
fn official_locales_satisfy_contract() {
    if test_support::remake_asset("locales/en/pack.json").is_none() {
        return;
    }
    for locale_id in LOCALE_IDS {
        let catalog = load_official_locale(locale_id).unwrap_or_else(|e| panic!("{locale_id}: {e}")).unwrap();
        assert_eq!(catalog.entries.len(), OFFICIAL_LOCALE_CONTRACT.read().unwrap().len());
        load_official_locale_content(locale_id).unwrap().unwrap();
        load_official_locale_entities(locale_id).unwrap().unwrap();
    }
    assert!(load_official_locale("xx").is_err());
}

/// Go：`story_locale_test.go:970` `TestStoryLocaleFailsClosedForMissingIdentity` 的 `localizedStoryText`
/// 部分（Go 經 `resolveCampaignDialogLine`，B 組）。
#[test]
fn localized_story_text_fails_closed_without_identity() {
    if test_support::remake_asset("locales/en/content.json").is_none() {
        return;
    }
    let content = load_official_locale_content("en").unwrap();
    let g = Game { locale_id: String::from("en"), locale_content: content, ..Default::default() };
    let line = campaign::Line { text: String::from("來源"), ..Default::default() };
    assert!(g.localized_story_text(&line).is_err(), "story line without canonical identity was accepted");
    // 繁中且沒有 line_id：照 Go 直接用劇本文字。
    let g = Game { locale_id: String::from("zh-Hant"), ..Default::default() };
    assert_eq!(g.localized_story_text(&line).unwrap(), "來源");
    let line = campaign::Line { line_id: String::from("x"), ..Default::default() };
    assert_eq!(g.localized_story_text(&line).unwrap_err().message(), "official locale content is unavailable");
}

#[test]
fn locale_message_reports_missing_catalog_and_bad_keys() {
    let mut g = Game::default();
    assert_eq!(g.locale_message("any", &[]), (String::new(), false));
    assert_eq!(g.load_err, "locale message: official catalog is unavailable");
    if test_support::remake_asset("locales/en/pack.json").is_none() {
        return;
    }
    g.locale_catalog = load_official_locale("en").unwrap();
    g.load_err.clear();
    assert!(!g.locale_message("no.such.key", &[]).1);
    assert!(g.load_err.starts_with("locale message: "), "{}", g.load_err);
}

#[test]
fn clean_slash_path_matches_filepath_clean() {
    for p in [".", "/", "a", "a/b", "../a", "../../a/b", "/a/b", "fonts/cjk.ttc"] {
        assert!(is_clean_slash_path(p), "{p}");
    }
    for p in ["", "a/", "a//b", "./a", "a/./b", "a/../b", "/../a", "/a/", "a/.."] {
        assert!(!is_clean_slash_path(p), "{p}");
    }
}
