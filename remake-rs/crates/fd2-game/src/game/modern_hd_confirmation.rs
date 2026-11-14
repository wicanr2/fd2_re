//! Go 來源：`remake/cmd/fd2/modern_hd_confirmation.go`（280 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_confirmation.go:18`
pub(crate) static MODERN_HD_CONFIRMATION_INDICES: LazyLock<[i64; 6]> = LazyLock::new(|| [16, 17, 48, 49, 51, 52]);

/// Go：`modern_hd_confirmation.go:20`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDConfirmationCell {
    pub(crate) height: i64,
    pub(crate) background: ColorRgba,
    pub(crate) ink: ColorRgba,
    pub(crate) shadow: ColorRgba,
}

/// Go：`modern_hd_confirmation.go:25`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDConfirmation {
    pub(crate) cells: BTreeMap<i64, ModernHDConfirmationCell>,
    pub(crate) labels: BTreeMap<String, [String; 2]>,
    pub(crate) fonts: BTreeMap<String, Font>,
    pub(crate) bounds: BTreeMap<String, [Rectangle; 2]>,
    pub(crate) images: BTreeMap<ModernHDConfirmationImageKey, Option<Arc<Picture>>>,
}

/// Go：`modern_hd_confirmation.go:33`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct ModernHDConfirmationImageKey {}

/// Go：`modern_hd_confirmation.go:51`，`func freezeModernHDConfirmation(cells []fdother.RawCell, palette color.Palette, font *Font) (*modernHDConfirmation, error)`
pub(crate) fn freeze_modern_hd_confirmation(
    cells: &[fdother::RawCell],
    palette: &ColorPalette,
    font: Option<&mut Font>,
) -> Result<Option<ModernHDConfirmation>> {
    if cells.len() <= 52 || palette.len() != 256 {
        return Err(GameError::new("modern HD confirmation: source unavailable"));
    }
    let mut result = ModernHDConfirmation::default();
    for &index in MODERN_HD_CONFIRMATION_INDICES.iter() {
        let cell = &cells[index as usize];
        let height = if index == 16 || index == 17 { 20 } else { 16 };
        if cell.width != 24 || cell.height != height || cell.pixels.len() as i64 != 24 * height {
            return Err(game_err!("modern HD confirmation: cell{index} shape differs"));
        }
        let color_at = |x: usize, y: usize| -> Result<ColorRgba> {
            let ink = palette[cell.pixels[y * 24 + x] as usize];
            if ink.a != 255 {
                return Err(GameError::new("modern HD confirmation: source color is not opaque"));
            }
            Ok(ColorRgba { r: ink.r, g: ink.g, b: ink.b, a: 255 })
        };
        let art = ModernHDConfirmationCell {
            height,
            background: color_at(3, 3)?,
            ink: color_at(12, 0)?,
            shadow: color_at(12, 1)?,
        };
        result.cells.insert(index, art);
    }
    let font = match font {
        Some(font) if font.sf.is_some() => font,
        _ => return Err(GameError::new("modern HD confirmation: font unavailable")),
    };
    for locale in ["zh-Hant", "zh-Hans", "en", "ja"] {
        let Some(catalog) = load_official_locale(locale)? else {
            return Err(GameError::new("modern HD confirmation: official locale unavailable"));
        };
        // Go：`font.hdFonts[locale]` 優先（HD 區域字型在 R6，忠實模式下 `hd_fonts` 是空的）。
        let mut f = font.hd_fonts.get(locale).cloned().unwrap_or_else(|| font.clone());
        let Some(sf) = f.sf.clone() else {
            return Err(GameError::new("modern HD confirmation: regional font unavailable"));
        };
        let (face, _) = f.face_for(40);
        let Some(face) = face else {
            return Err(GameError::new("modern HD confirmation: font face unavailable"));
        };
        // face_for 成功代表已注入量測器（見 font.rs）。
        let measure = f.measure.0.clone().expect("face_for 成功時已注入 TextMeasure");
        let mut labels: [String; 2] = Default::default();
        let mut bounds: [Rectangle; 2] = Default::default();
        for (option, key) in ["common.yes", "common.no"].into_iter().enumerate() {
            let label = match catalog.format(key, &[]) {
                Ok(label) if !label.is_empty() => label,
                _ => return Err(game_err!("modern HD confirmation: {locale}/{key} unavailable")),
            };
            if label.chars().any(|r| !measure.has_glyph(&sf, r)) {
                return Err(GameError::new("modern HD confirmation: label glyph unavailable"));
            }
            bounds[option] = measure.bound_string(&face, &label);
            // The smallest native cell is 24x16. Include a one-HD-pixel
            // shadow and the selected six-HD-pixel downward pulse.
            let b = &bounds[option];
            let (left, top) = ((144 - rect_dx(b)) / 2, (96 - rect_dy(b)) / 2);
            let ink = Rectangle {
                min: Point { x: left, y: top },
                max: Point { x: left + rect_dx(b) + 1, y: top + rect_dy(b) + 7 },
            };
            // Go `ink.In(image.Rect(12, 12, 132, 84))`（ink 不會是空矩形）。
            let inside = ink.min.x >= 12 && ink.max.x <= 132 && ink.min.y >= 12 && ink.max.y <= 84;
            if rect_empty(b) || !inside {
                return Err(game_err!("modern HD confirmation: {locale} label exceeds text-safe area"));
            }
            labels[option] = label;
        }
        result.labels.insert(locale.to_string(), labels);
        result.bounds.insert(locale.to_string(), bounds);
        result.fonts.insert(locale.to_string(), f);
    }
    Ok(Some(result))
}
