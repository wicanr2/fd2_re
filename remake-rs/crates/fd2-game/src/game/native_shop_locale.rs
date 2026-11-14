//! Go 來源：`remake/cmd/fd2/native_shop_locale.go`（317 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const LOCALIZED_SHOP_MESSAGE_X: i64 = 12;

pub(crate) const LOCALIZED_SHOP_MESSAGE_Y: i64 = 119;

pub(crate) const LOCALIZED_SHOP_MESSAGE_WIDTH: i64 = 292;

pub(crate) const LOCALIZED_SHOP_MESSAGE_LINE_STEP: i64 = 19;

pub(crate) const LOCALIZED_SHOP_MESSAGE_ROWS: i64 = 3;

pub(crate) const LOCALIZED_SHOP_MESSAGE_SCALE: f64 = 0.78;

impl Game {
    /// Go：`native_shop_locale.go:151`，`func (g *Game) localizedShopKey(weaponKey, otherKey string) string`
    pub(crate) fn localized_shop_key(&self, weapon_key: &str, other_key: &str) -> String {
        todo!("native_shop_locale.go:localizedShopKey")
    }

    /// Go：`native_shop_locale.go:158`，`func (g *Game) composeLocalizedNativeShopStable( assets *campaign.NativeShopAssets, portrait dato.Frame, portraitID int, ) ([]byte, error)`
    pub(crate) fn compose_localized_native_shop_stable(
        &mut self,
        assets: &campaign::NativeShopAssets,
        portrait: &dato::Frame,
        portrait_id: i64,
    ) -> Result<Vec<u8>> {
        todo!("native_shop_locale.go:composeLocalizedNativeShopStable")
    }

    /// Go：`native_shop_locale.go:224`，`func (g *Game) drawLocalizedShopMessage(frame []byte, message string, y, maxRows int) ([]byte, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn draw_localized_shop_message(
        &mut self,
        frame: &[u8],
        message: &str,
        y: i64,
        max_rows: i64,
    ) -> Result<Vec<u8>> {
        todo!("native_shop_locale.go:drawLocalizedShopMessage")
    }
}
