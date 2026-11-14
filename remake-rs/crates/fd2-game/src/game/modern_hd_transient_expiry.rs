//! Go 來源：`remake/cmd/fd2/modern_hd_transient_expiry.go`（254 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// This state belongs to one expiry job, including its final restore frame.
/// The scene is captured after that record's native focus, never during Draw.
///
/// Go：`modern_hd_transient_expiry.go:53`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeTransientExpiryPresentation {}

impl Game {
    /// Check the new HD sources on private states before publishing the countdown.
    /// Activation seals them again after the actual damage/death/focus sequence.
    ///
    /// Go：`modern_hd_transient_expiry.go:109`，`func (g *Game) preflightModernHDTransientExpiry(candidate *battle.State, expired []battle.NativeTransientExpiry, damage []battle.NativeTransientDamage) error`
    ///
    /// R4 D 組：只讀參數改成 `&`（簽名變更紀錄）。忠實模式（沒有 HD 呈現、沒有 HD 地圖人物）在第一行就返回；
    /// HD 開啟時的預覽與到期框預檢在 R6。
    pub(crate) fn preflight_modern_hd_transient_expiry(
        &self,
        candidate: &battle::State,
        expired: &[battle::NativeTransientExpiry],
        damage: &[battle::NativeTransientDamage],
    ) -> Result<()> {
        // Go `hdPresentationEnabled()`（`player_interface.go:32`，`r8`）：介面 HD 或美術 HD。
        let hd_presentation =
            self.modern_hd_interface_enabled() || self.modern_story_portraits.as_ref().is_some_and(|set| set.has_hd);
        if expired.is_empty()
            || (!hd_presentation
                && self.modern_story_portraits.as_ref().is_none_or(|set| set.map_sprites_hd.is_empty()))
        {
            return Ok(());
        }
        let _ = (candidate, damage);
        todo!(
            "R6: modern_hd_transient_expiry.go:113 HD 到期框預檢（nativeTransientDeathPreview、newNativeTransientExpiryPresentation）"
        )
    }
}
