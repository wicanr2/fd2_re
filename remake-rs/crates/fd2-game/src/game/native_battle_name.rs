//! Go 來源：`remake/cmd/fd2/native_battle_name.go`（135 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeBattleNameAssets 是全螢幕戰鬥狀態欄的原版姓名字模來源。
/// 原版 0x18c6d→0x15f84 走 FDOTHER#4 的 16×16 1bpp glyph，不與對話／現代
/// UTF-8 字型共用；索引表仍是可編輯資產，未知字元一律拒絕 native 路徑。
///
/// Go：`native_battle_name.go:22`，`func loadNativeBattleNameAssets() (*fdtxt.Font, map[string]int, error)`
pub(crate) fn load_native_battle_name_assets() -> Result<(Option<fdtxt::Font>, BTreeMap<String, i64>)> {
    let font = fdtxt::load_separated_font(separated_asset_path("fonts"))?;
    let path = asset_path("assets/fonts/unicode_to_glyph.json");
    let index_raw = std::fs::read(&path).map_err(|e| game_err!("open {path}: {e}"))?;
    let encoded: BTreeMap<String, serde_json::Value> =
        serde_json::from_slice(&index_raw).map_err(|e| game_err!("{e}"))?;
    let mut index = BTreeMap::new();
    for (key, value) in encoded {
        if key == "_comment" {
            continue;
        }
        // Go `json.Unmarshal(value, &glyph)`（`int`）：只接受整數值。
        let glyph = value.as_i64().ok_or_else(|| {
            game_err!("native battle name glyph {}: {}", battle::go_quote(&key), "json: cannot unmarshal")
        })?;
        index.insert(key, glyph);
    }
    for (key, &glyph) in &index {
        if glyph < 0 || glyph >= font.glyph_count() as i64 {
            return Err(game_err!("native battle name glyph {}={glyph} is outside font", battle::go_quote(key)));
        }
    }
    Ok((Some(font), index))
}

/// Go：`native_battle_name.go:87`，`func nativePaletteColor(palette color.Palette, index byte, fallback color.RGBA) color.RGBA`
pub(crate) fn native_palette_color(palette: &ColorPalette, index: u8, fallback: ColorRgba) -> ColorRgba {
    if index as usize >= palette.len() {
        return fallback;
    }
    // Go `palette[index].RGBA()` 是 16 位元預乘值，`>> 8` 即 `color.RGBA` 的原位元組。
    palette[index as usize]
}
