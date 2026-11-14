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
    if reference != CANONICAL_CAMPAIGN_REFERENCE {
        return Ok(Some(campaign::load(&asset_path(reference))?));
    }
    let compiled = editorcanonical::compile_campaign_json(&asset_path("assets/editor-canonical"))?;
    Ok(Some(campaign::decode(&compiled)?))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_support;

    /// Go `TestCanonicalCampaignMatchesLegacyDocument` 的對應：受版控的 canonical bundle 經
    /// `load_player_campaign` 編出的戰役起點與 legacy JSON 相同。Go `packaged_startup_test.go`
    /// 的三個測試改寫環境變數或驅動 `Update`，列在 `PORTING.md`「A 組測試」。
    #[test]
    fn canonical_campaign_loads_with_chapter_zero_start() {
        let (Some(bundle), Some(legacy)) = (
            test_support::remake_asset("editor-canonical"),
            test_support::remake_asset("scenarios/campaign_full.json"),
        ) else {
            return;
        };
        let compiled = editorcanonical::compile_campaign_json(&bundle).unwrap();
        let canonical = campaign::decode(&compiled).unwrap();
        let legacy = load_player_campaign(&legacy).unwrap().unwrap();
        assert_eq!(canonical.start, "story_ch00_handler");
        assert_eq!(canonical.start, legacy.start);
        assert_eq!(canonical.nodes.len(), legacy.nodes.len());
    }
}
