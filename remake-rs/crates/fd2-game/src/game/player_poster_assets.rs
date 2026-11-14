//! Go 來源：`remake/cmd/fd2/player_poster_assets.go`（159 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`player_poster_assets.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// R4 F 組：`settings.json` 的 `poster` 欄位要能讀回、照原樣寫出，所以三個欄位與 serde 先補上
/// （欄位名照 Go 標籤）；`normalizePlayerPosters` 由 `loadSettings` 呼叫，一併移植在下方。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct PlayerPosterSettings {
    /// Go 標籤：`json:"set"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) set: String,
    /// Go 標籤：`json:"edge"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) edge: i64,
    /// Go 標籤：`json:"fade_ms"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) fade_ms: i64,
}

/// Go：`player_poster_assets.go:24`，`func defaultPlayerPosters() playerPosterSettings`（R4 F 組補移植，原標 `r8`）
pub(crate) fn default_player_posters() -> PlayerPosterSettings {
    PlayerPosterSettings { set: "off".to_string(), edge: 48, fade_ms: 600 }
}

/// Go：`player_poster_assets.go:28`，`func validPosterMode(mode string) bool`（R4 F 組補移植，原標 `r8`）
pub(crate) fn valid_poster_mode(mode: &str) -> bool {
    mode == "off" || mode == "hd" || mode == "classic"
}

/// Go：`player_poster_assets.go:32`，`func normalizePlayerPosters(p *playerPosterSettings) playerPosterSettings`
/// （R4 F 組補移植，原標 `r8`：`loadSettings` 讀到 `poster` 時呼叫）
pub(crate) fn normalize_player_posters(p: Option<&PlayerPosterSettings>) -> PlayerPosterSettings {
    let Some(p) = p else {
        return default_player_posters();
    };
    let mut v = p.clone();
    if !valid_poster_mode(&v.set) {
        v.set = "off".to_string();
    }
    if v.edge < 0 || v.edge > 96 {
        v.edge = 48;
    }
    match v.fade_ms {
        0 | 300 | 600 | 1200 => {}
        _ => v.fade_ms = 600,
    }
    v
}
