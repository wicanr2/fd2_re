//! Go 來源：`remake/cmd/fd2/settings.go`（230 行）。
//!
//! settings.go — 全域系統設定(跨存檔,獨立於 fd2_save.json)。
//! 目前提供重製端 F3 設定與兩套預錄 OGG，在操作意圖上近似原版
//! SETSOUND.EXE 的音源選擇；它不執行原版程式、驅動或即時合成。
//! 兩套 OGG 以 Sound Blaster（FM）與 Roland MT-32 身分並存；重製端預設選 FM。
//! 這個預設與音色描述不是原版 SETSOUND 的動態證據，也不代表即時合成 parity。
//!
//! 骨架：Go 檔的 8 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! F 組另外移植 `saveSettings`（設定檔寫出端；Go 測試與黃金需要它）。`cycleBGMSource`、
//! `cycleLocale`、`cycleFontScale` 是 F3／F4 等按鍵的處理器，第一章受控流程不按，留給 R8。
//! JSON 對應規則見 `save.rs` 的模組文件。

use super::*;

/// settingsPath 設定檔位置:$XDG_DATA_HOME/fd2_re/fd2_settings.json(理由同 savePath,見 assets.go)。
///
/// Go：`settings.go:18`，`func settingsPath() string`
pub(crate) fn settings_path() -> String {
    user_data_path("fd2_settings.json")
}

/// Go：`settings.go:23`
pub(crate) static BGM_SOURCE_NAME: LazyLock<BTreeMap<String, String>> = LazyLock::new(|| {
    BTreeMap::from([
        (String::from("fm"), String::from("Sound Blaster (FM)")),
        (String::from("mt32"), String::from("Roland MT-32")),
    ])
});

/// Go：`settings.go:30`
pub(crate) static LOCALE_DISPLAY_NAME: LazyLock<BTreeMap<String, String>> = LazyLock::new(|| {
    BTreeMap::from([
        (String::from("zh-Hant"), String::from("繁體中文")),
        (String::from("zh-Hans"), String::from("简体中文")),
        (String::from("ja"), String::from("日本語")),
        (String::from("en"), String::from("English")),
    ])
});

/// fontScales 是可選的字級倍率。英日文譯文比中文長，同一個框裡常常斷不完；
/// 調小一格能多塞一行。原版字模畫的索引畫面不受影響。
///
/// Go：`settings.go:39`
pub(crate) static FONT_SCALES: LazyLock<Vec<f64>> = LazyLock::new(|| vec![0.8, 0.9, 1.0, 1.1, 1.25]);

pub(crate) const DEFAULT_FONT_SCALE: f64 = 1.0;

/// Go `encoding/json` 寫 `float64` 的寫法：整數值不帶小數點（`1` 而不是 serde_json 的 `1.0`）。
/// 只處理 `|v| < 2^53` 的整數值，這個範圍內 Go 的 `'f'` 格式就是十進位整數；其他值照 serde_json
/// 的最短表示（`1e-6 ≤ |v| < 1e21` 的非整數與 Go 相同，更小或更大的指數寫法不同，設定檔不會出現）。
pub(crate) mod go_json_float64 {
    use serde::{Deserialize, Deserializer, Serializer};

    pub(crate) fn serialize<S: Serializer>(v: &f64, s: S) -> std::result::Result<S::Ok, S::Error> {
        const EXACT: f64 = 9_007_199_254_740_992.0; // 2^53
        if v.is_finite() && v.fract() == 0.0 && v.abs() < EXACT && !(*v == 0.0 && v.is_sign_negative()) {
            s.serialize_i64(*v as i64)
        } else {
            s.serialize_f64(*v)
        }
    }

    pub(crate) fn deserialize<'de, D: Deserializer<'de>>(d: D) -> std::result::Result<f64, D::Error> {
        // Go 讀 null 到 float64 不改原值；起點是零值，所以是 0。
        Ok(Option::<f64>::deserialize(d)?.unwrap_or_default())
    }
}

/// Go：`settings.go:43`
#[derive(Debug, Clone, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct Settings {
    /// "fm"(預設)或 "mt32"
    ///
    /// Go 標籤：`json:"bgm_source"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) bgm_source: String,
    /// BCP 47；不屬於戰役存檔
    ///
    /// Go 標籤：`json:"locale_id"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) locale_id: String,
    /// 字級倍率；0 或不合法值視同 1.0
    ///
    /// Go 標籤：`json:"font_scale"`
    #[serde(with = "go_json_float64")]
    pub(crate) font_scale: f64,
    /// Go 標籤：`json:"interface_mode,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub(crate) interface_mode: String,
    /// Go 標籤：`json:"theme_id,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub(crate) theme_id: String,
    /// Go 標籤：`json:"game_speed,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "battle::go_is_zero")]
    pub(crate) game_speed: i64,
    /// Go 標籤：`json:"battle_animation_speed,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "battle::go_is_zero")]
    pub(crate) battle_animation_speed: i64,
    /// Go 標籤：`json:"hd_texture_quality,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub(crate) hd_texture_quality: String,
    /// Go 標籤：`json:"poster,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) poster: Option<PlayerPosterSettings>,
}

/// `loadSettings` 解碼前的起點（Go `settings.go:72`）：`json.Unmarshal` 只改 JSON 寫到的欄位，
/// 其他欄位保留這些值。
pub(crate) fn settings_load_base() -> Settings {
    Settings {
        bgm_source: "fm".to_string(),
        locale_id: "zh-Hant".to_string(),
        font_scale: DEFAULT_FONT_SCALE,
        ..Default::default()
    }
}

/// 以 `base` 為起點解碼設定 JSON（Go `json.Unmarshal(raw, &s)`，`s` 已有值）。
///
/// Rust：serde 的 `#[serde(default)]` 只能從 `Default` 起算，所以先把 JSON 讀成物件，沒寫到的欄位
/// 與寫成 `null` 的欄位再從 `base` 補上（Go 把 `null` 解到字串、整數、小數時不改原值，解到指標時設
/// nil，而 `base.poster` 本來就是 nil），然後整份解碼；重複的鍵與 Go 一樣以最後一個為準。JSON 是
/// `null` 時 Go 不改任何欄位，結果是 `base`。Go 遇到型別錯誤時會保留其他欄位繼續填
/// （`SAVE_JSON_KNOWN_DIFFERENCES` 的「型別錯誤時的部分填入」），Rust 整份失敗。
pub(crate) fn go_json_unmarshal_settings(base: &Settings, raw: &[u8]) -> serde_json::Result<Settings> {
    use serde::de::Error as _;
    let mut document = match serde_json::from_slice::<serde_json::Value>(raw)? {
        serde_json::Value::Null => return Ok(base.clone()),
        serde_json::Value::Object(document) => document,
        _ => return Err(serde_json::Error::custom("cannot unmarshal non-object into Go struct settings")),
    };
    let serde_json::Value::Object(defaults) = serde_json::to_value(base)? else {
        return Err(serde_json::Error::custom("settings 不是物件"));
    };
    for (key, value) in defaults {
        let slot = document.entry(key).or_insert(serde_json::Value::Null);
        if slot.is_null() {
            *slot = value;
        }
    }
    // `base` 寫出時省略的零值欄位（omitempty）：`null` 拿掉後 serde 補 `Default`，就是 `base` 的值。
    document.retain(|_, value| !value.is_null());
    serde_json::from_value(serde_json::Value::Object(document))
}

/// `loadSettings` 的解碼：失敗時回 `None`，呼叫端保留 `base`（Go 忽略 `json.Unmarshal` 的錯誤）。
pub(crate) fn decode_settings_onto(base: &Settings, raw: &[u8]) -> Option<Settings> {
    go_json_unmarshal_settings(base, raw).ok()
}

/// normalizeFontScale 把設定值收斂到 fontScales 裡最接近的一格。舊的設定檔沒有
/// 這個欄位，讀出來是 0，這裡會變成預設值。
///
/// Go：`settings.go:57`，`func normalizeFontScale(v float64) float64`
pub(crate) fn normalize_font_scale(v: f64) -> f64 {
    // Go `v <= 0`：NaN 不成立，往下走；NaN 與任何候選的差都是 NaN，`<` 不成立，結果是預設值。
    if v <= 0.0 {
        return DEFAULT_FONT_SCALE;
    }
    let (mut best, mut best_diff) = (DEFAULT_FONT_SCALE, f64::INFINITY);
    for &candidate in FONT_SCALES.iter() {
        let diff = (candidate - v).abs();
        if diff < best_diff {
            best = candidate;
            best_diff = diff;
        }
    }
    best
}

/// loadSettings 讀 fd2_settings.json；無檔或不合法時回重製端預設 FM。
///
/// Go：`settings.go:71`，`func loadSettings() settings`
pub(crate) fn load_settings() -> Settings {
    let mut s = settings_load_base();
    if let Ok(raw) = std::fs::read(settings_path())
        && let Some(decoded) = decode_settings_onto(&s, &raw)
    {
        s = decoded;
    }
    normalize_loaded_settings(s)
}

/// `loadSettings` 解碼後的正規化（Go `settings.go:76`～`95`），拆出來讓黃金不經檔案系統也能比對。
pub(crate) fn normalize_loaded_settings(mut s: Settings) -> Settings {
    s.font_scale = normalize_font_scale(s.font_scale);
    s.game_speed = normalize_player_speed(s.game_speed);
    s.battle_animation_speed = normalize_battle_animation_speed(s.battle_animation_speed);
    s.hd_texture_quality = parse_hd_texture_quality(&s.hd_texture_quality).name();
    if s.poster.is_some() {
        s.poster = Some(normalize_player_posters(s.poster.as_ref()));
    }
    if s.interface_mode != PLAYER_INTERFACE_ORIGINAL && s.interface_mode != PLAYER_INTERFACE_HD {
        s.interface_mode = String::new();
    }
    if s.theme_id != "faithful" && s.theme_id != MODERN_HANDPAINTED_THEME_ID {
        s.theme_id = String::new();
    }
    if BGM_SOURCE_NAME.get(&s.bgm_source).is_none_or(|name| name.is_empty()) {
        s.bgm_source = "fm".to_string();
    }
    if LOCALE_DISPLAY_NAME.get(&s.locale_id).is_none_or(|name| name.is_empty()) {
        s.locale_id = "zh-Hant".to_string();
    }
    s
}

/// Go：`settings.go:99`，`func saveSettings(s settings)`（F 組補移植，原標 `r8`）
///
/// Go 忽略 `os.WriteFile` 的錯誤；它不是原子取代（與存檔的 `writeSaveFile` 不同），照 Go 直接寫。
pub(crate) fn save_settings(s: &Settings) {
    if let Ok(raw) = go_json_marshal_indent(s) {
        let _ = write_file_mode(&settings_path(), &raw, 0o644);
    }
}

/// Go `os.WriteFile(name, data, perm)`：建立或截斷，新檔的權限是 `perm`（受 umask 影響）。
pub(crate) fn write_file_mode(path: &str, data: &[u8], mode: u32) -> std::io::Result<()> {
    use std::io::Write as _;
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create(true).truncate(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(mode);
    }
    #[cfg(not(unix))]
    let _ = mode;
    options.open(path)?.write_all(data)
}

impl Game {
    /// applyFontScale 把字級倍率套到所有以 TTF 繪製的字型。索引畫面用原版字模，不走
    /// 這裡，所以調字級不會動到與原版對拍的畫面。
    ///
    /// Go：`settings.go:181`，`func (g *Game) applyFontScale(scale float64)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn apply_font_scale(&mut self, scale: f64) {
        self.font_scale = normalize_font_scale(scale);
        let scale = self.font_scale;
        // Go 對 nil 的 *Font 呼叫 SetUserScale 不做事（font.go:45）。
        for primary in [self.font.as_mut(), self.font_nm.as_mut()].into_iter().flatten() {
            primary.set_user_scale(scale);
            for regional in primary.hd_fonts.values_mut() {
                regional.set_user_scale(scale);
                if let Some(heavy) = regional.hd_heavy.as_mut() {
                    heavy.set_user_scale(scale);
                }
            }
        }
    }
}

#[cfg(test)]
mod settings_test;
