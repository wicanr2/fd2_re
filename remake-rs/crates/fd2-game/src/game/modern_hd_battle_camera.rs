//! Go 來源：`remake/cmd/fd2/modern_hd_battle_camera.go`（112 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_battle_camera.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernBattlePresentationProfile {
    /// Go 標籤：`json:"impact_hold_frames"`
    pub(crate) impact_hold_frames: i64,
}

impl Game {
    /// Go：`modern_hd_battle_camera.go:107`，`func (g *Game) modernPhysicalImpactHoldMillis(impact bool) float64`
    pub(crate) fn modern_physical_impact_hold_millis(&self, impact: bool) -> f64 {
        let Some(profile) = self.modern_story_portraits.as_ref().and_then(|set| set.battle_presentation.as_ref())
        else {
            return 0.0;
        };
        if !impact {
            return 0.0;
        }
        profile.impact_hold_frames as f64 * 1000.0 / 60.0
    }
}
