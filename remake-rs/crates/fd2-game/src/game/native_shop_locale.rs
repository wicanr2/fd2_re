//! Go 來源：`remake/cmd/fd2/native_shop_locale.go`（317 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! 這裡是非繁中語系（或 HD 擷取文字時）的商店訊息：斷行經 `Font::wrap`（外殼注入的量測器），
//! 點陣化經 B 組的 `draw_indexed_localized_text`（R5）。忠實模式繁中不經過這裡。

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
        if self.native_shop_variant == 1 {
            return weapon_key.to_string();
        }
        other_key.to_string()
    }

    /// Go：`native_shop_locale.go:158`，`func (g *Game) composeLocalizedNativeShopStable( assets *campaign.NativeShopAssets, portrait dato.Frame, portraitID int, ) ([]byte, error)`
    pub(crate) fn compose_localized_native_shop_stable(
        &mut self,
        assets: &campaign::NativeShopAssets,
        portrait: &dato::Frame,
        portrait_id: i64,
    ) -> Result<Vec<u8>> {
        let key = self.localized_shop_key("shop.greeting.weapon", "shop.greeting.item");
        let (message, ok) = self.locale_message(&key, &[]);
        if !ok {
            return Err(game_err!("localized shop greeting is unavailable"));
        }
        // Go 直接解參考 `g.nativeClassUI`（呼叫端 `composeNativeShopStable` 已確認它存在）。
        let shared = self.native_class_ui.as_ref().expect("composeLocalizedNativeShopStable: nativeClassUI is nil");
        let frame = campaign::compose_native_shop_scene_without_text(
            Some(assets),
            &shared.dialogue,
            &shared.digits,
            portrait,
            portrait_id,
            self.gold,
        )?;
        self.draw_localized_shop_message(&frame, &message, LOCALIZED_SHOP_MESSAGE_Y, LOCALIZED_SHOP_MESSAGE_ROWS)
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
        self.clear_modern_hd_shop_text();
        if self.font.is_none() || frame.len() != 320 * 200 || message.is_empty() || max_rows <= 0 {
            return Err(game_err!("localized shop message state is invalid"));
        }
        let font = self.font.as_mut().unwrap();
        let lines = font.wrap(message, LOCALIZED_SHOP_MESSAGE_SCALE, LOCALIZED_SHOP_MESSAGE_WIDTH as f64);
        if lines.is_empty() || lines.len() as i64 > max_rows {
            return Err(game_err!("localized shop message requires {} rows, max {}", lines.len(), max_rows));
        }
        let mut result = frame.to_vec();
        for (index, line) in lines.iter().enumerate() {
            if line.is_empty() {
                return Err(game_err!("localized shop message contains an empty row"));
            }
            draw_indexed_localized_text(
                &mut result,
                Some(&mut *font),
                line,
                LOCALIZED_SHOP_MESSAGE_X,
                y + index as i64 * LOCALIZED_SHOP_MESSAGE_LINE_STEP,
                LOCALIZED_SHOP_MESSAGE_WIDTH,
                LOCALIZED_SHOP_MESSAGE_LINE_STEP,
                LOCALIZED_SHOP_MESSAGE_SCALE,
                0xcd,
                0x4c,
            )?;
        }
        Ok(result)
    }
}
