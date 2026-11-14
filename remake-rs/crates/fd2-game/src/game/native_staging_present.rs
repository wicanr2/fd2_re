//! Go 來源：`remake/cmd/fd2/native_staging_present.go`（178 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// nativeFocusEndpoint predicts the exact 0x12CEA safe-band endpoint without
    /// publishing a focus step.  sub_33F78 needs this private endpoint to preflight
    /// the following 0x22253 transaction before the first visible mutation.
    ///
    /// Go：`native_staging_present.go:15`，`func (g *Game) nativeFocusEndpoint(targetX, targetY int) (int, int, battle.NativeMapViewState, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_focus_endpoint(
        &self,
        target_x: i64,
        target_y: i64,
    ) -> Result<(i64, i64, battle::NativeMapViewState)> {
        todo!("native_staging_present.go:nativeFocusEndpoint")
    }
}
