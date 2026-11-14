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
    pub(crate) fn preflight_modern_hd_transient_expiry(
        &self,
        candidate: &mut battle::State,
        expired: &[battle::NativeTransientExpiry],
        damage: &mut [battle::NativeTransientDamage],
    ) -> Result<()> {
        todo!("modern_hd_transient_expiry.go:preflightModernHDTransientExpiry")
    }
}
