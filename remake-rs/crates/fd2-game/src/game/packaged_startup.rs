//! Go 來源：`remake/cmd/fd2/packaged_startup.go`（42 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// canonicalCampaignReference 讓 FD2_CAMPAIGN 指向編輯器的 canonical bundle 而不是
/// 某個 JSON 檔。編輯器改的是 canonical 文件；玩家路徑要是繼續讀 legacy JSON，
/// 編輯的結果就永遠進不了遊戲，而兩邊看起來都正常。
pub(crate) const CANONICAL_CAMPAIGN_REFERENCE: &str = "canonical";

/// loadPlayerCampaign 依 FD2_CAMPAIGN 的值取得戰役圖。
///
/// `canonical` 走編輯器 bundle：編譯回 legacy 形狀之後交給 campaign.Decode，與直接
/// 讀 JSON 走的是同一道驗證。其餘值仍是明確路徑，測試與離屏 oracle 靠它指定自己
/// 的戰役檔，那個契約不動。
///
/// Go：`packaged_startup.go:33`，`func loadPlayerCampaign(reference string) (*campaign.Campaign, error)`
pub(crate) fn load_player_campaign(reference: &str) -> Result<Option<campaign::Campaign>> {
    todo!("packaged_startup.go:loadPlayerCampaign")
}
