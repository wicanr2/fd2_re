//! Go 來源：`remake/cmd/fd2/settings.go`（230 行）。
//!
//! settings.go — 全域系統設定(跨存檔,獨立於 fd2_save.json)。
//! 目前提供重製端 F3 設定與兩套預錄 OGG，在操作意圖上近似原版
//! SETSOUND.EXE 的音源選擇；它不執行原版程式、驅動或即時合成。
//! 兩套 OGG 以 Sound Blaster（FM）與 Roland MT-32 身分並存；重製端預設選 FM。
//! 這個預設與音色描述不是原版 SETSOUND 的動態證據，也不代表即時合成 parity。
//!
//! 骨架：Go 檔的 8 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// settingsPath 設定檔位置:$XDG_DATA_HOME/fd2_re/fd2_settings.json(理由同 savePath,見 assets.go)。
///
/// Go：`settings.go:18`，`func settingsPath() string`
pub(crate) fn settings_path() -> String {
    todo!("settings.go:settingsPath")
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

/// Go：`settings.go:43`
#[derive(Debug, Clone, Default, PartialEq)]
pub(crate) struct Settings {
    /// "fm"(預設)或 "mt32"
    ///
    /// Go 標籤：`json:"bgm_source"`
    pub(crate) bgm_source: String,
    /// BCP 47；不屬於戰役存檔
    ///
    /// Go 標籤：`json:"locale_id"`
    pub(crate) locale_id: String,
    /// 字級倍率；0 或不合法值視同 1.0
    ///
    /// Go 標籤：`json:"font_scale"`
    pub(crate) font_scale: f64,
    /// Go 標籤：`json:"interface_mode,omitempty"`
    pub(crate) interface_mode: String,
    /// Go 標籤：`json:"theme_id,omitempty"`
    pub(crate) theme_id: String,
    /// Go 標籤：`json:"game_speed,omitempty"`
    pub(crate) game_speed: i64,
    /// Go 標籤：`json:"battle_animation_speed,omitempty"`
    pub(crate) battle_animation_speed: i64,
    /// Go 標籤：`json:"hd_texture_quality,omitempty"`
    pub(crate) hd_texture_quality: String,
    /// Go 標籤：`json:"poster,omitempty"`
    pub(crate) poster: Option<PlayerPosterSettings>,
}

/// normalizeFontScale 把設定值收斂到 fontScales 裡最接近的一格。舊的設定檔沒有
/// 這個欄位，讀出來是 0，這裡會變成預設值。
///
/// Go：`settings.go:57`，`func normalizeFontScale(v float64) float64`
pub(crate) fn normalize_font_scale(v: f64) -> f64 {
    todo!("settings.go:normalizeFontScale")
}

/// loadSettings 讀 fd2_settings.json；無檔或不合法時回重製端預設 FM。
///
/// Go：`settings.go:71`，`func loadSettings() settings`
pub(crate) fn load_settings() -> Settings {
    todo!("settings.go:loadSettings")
}

impl Game {
    /// applyFontScale 把字級倍率套到所有以 TTF 繪製的字型。索引畫面用原版字模，不走
    /// 這裡，所以調字級不會動到與原版對拍的畫面。
    ///
    /// Go：`settings.go:181`，`func (g *Game) applyFontScale(scale float64)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn apply_font_scale(&mut self, scale: f64) {
        todo!("settings.go:applyFontScale")
    }
}
