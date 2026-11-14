//! Go 來源：`remake/cmd/fd2/settings_test.go` 與 `font_scale_test.go` 裡直接呼叫 F 組函式的測試。
//!
//! `settings_test.go:11` 的 `TestOfficialLocalesCoverSystemAndSaveMessages` 測的是 A 組的語系目錄，
//! `font_scale_test.go` 其餘三個測 A 組的 `Font` 方法、R8 的 `cycleFontScale` 與真字型量寬（R5），
//! 不在這裡（見 PORTING.md 的 F 組段落）。

use super::*;

/// Go：`settings_test.go:99`
#[test]
fn settings_round_trip_keeps_audio_and_locale_outside_save() {
    let name = "settings_round_trip_keeps_audio_and_locale_outside_save";
    let Some(root) = f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]) else {
        return;
    };
    save_settings(&Settings { bgm_source: "mt32".to_string(), locale_id: "ja".to_string(), ..Default::default() });
    let got = load_settings();
    if got.bgm_source != "mt32" || got.locale_id != "ja" {
        panic!("settings={got:?}");
    }
    let raw = std::fs::read(root.join("fd2_re").join("fd2_settings.json")).unwrap();
    let document: serde_json::Map<String, serde_json::Value> = serde_json::from_slice(&raw).unwrap();
    if !document.contains_key("locale_id") {
        panic!("settings lost locale_id");
    }
    // Go `json.Marshal(saveData{})`。
    let save_raw = serde_json::to_vec(&SaveData::default()).unwrap();
    let save_document: serde_json::Map<String, serde_json::Value> = serde_json::from_slice(&save_raw).unwrap();
    if save_document.contains_key("locale_id") {
        panic!("campaign save contains locale_id");
    }
}

/// Go：`settings_test.go:146`
#[test]
fn settings_reject_unsupported_locale() {
    let name = "settings_reject_unsupported_locale";
    let Some(root) = f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]) else {
        return;
    };
    let dir = root.join("fd2_re");
    std::fs::create_dir_all(&dir).unwrap();
    std::fs::write(dir.join("fd2_settings.json"), br#"{"bgm_source":"fm","locale_id":"xx"}"#).unwrap();
    let got = load_settings();
    if got.locale_id != "zh-Hant" {
        panic!("unsupported locale accepted: {got:?}");
    }
}

/// Go：`font_scale_test.go:10`
#[test]
fn normalize_font_scale_snaps_to_offered() {
    for (input, want) in [
        (0.0, DEFAULT_FONT_SCALE),  // 舊設定檔沒有這個欄位
        (-1.0, DEFAULT_FONT_SCALE), // 手改壞的值
        (1.0, 1.0),
        (0.83, 0.8),
        (1.19, 1.25),
        (5.0, 1.25), // 超出範圍取最接近的一格
    ] {
        let got = normalize_font_scale(input);
        assert_eq!(got, want, "normalizeFontScale({input:.2}) = {got:.2}，應為 {want:.2}");
    }
    for &scale in FONT_SCALES.iter() {
        let got = normalize_font_scale(scale);
        assert_eq!(got, scale, "提供的 {scale:.2} 被改成 {got:.2}");
    }
}

/// F 組補充：`loadSettings` 起點上的解碼（Go `json.Unmarshal` 只改寫到的欄位；完整的 Go 黃金在
/// `save/golden_save_test.rs`）。
#[test]
fn decode_settings_keeps_base_for_missing_and_null_fields() {
    let base = settings_load_base();
    let got = decode_settings_onto(&base, br#"{"locale_id":"ja","font_scale":null,"poster":null}"#).unwrap();
    assert_eq!(got.bgm_source, "fm");
    assert_eq!(got.locale_id, "ja");
    assert_eq!(got.font_scale, DEFAULT_FONT_SCALE);
    assert!(got.poster.is_none());
    assert_eq!(decode_settings_onto(&base, b"null"), Some(base.clone()));
    assert_eq!(decode_settings_onto(&base, b"[]"), None);
    assert_eq!(decode_settings_onto(&base, br#"{"locale_id":"ja","locale_id":"en"}"#).unwrap().locale_id, "en");
}
