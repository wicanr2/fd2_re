//! Go 來源：`remake/cmd/fd2/native_attack_assets.go`（77 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// ensureNativeAttackPresentation supplements the checked-in export subset
    /// from the player's separated FIGANI pack. All resources are decoded and
    /// scheduled before any Game map is changed, so a malformed pair cannot leave
    /// a half-published presentation.
    ///
    /// Go：`native_attack_assets.go:22`，`func (g *Game) ensureNativeAttackPresentation(atkGroup, defGroup int) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ensure_native_attack_presentation(&mut self, atk_group: i64, def_group: i64) -> Result<()> {
        todo!("native_attack_assets.go:ensureNativeAttackPresentation")
    }
}
