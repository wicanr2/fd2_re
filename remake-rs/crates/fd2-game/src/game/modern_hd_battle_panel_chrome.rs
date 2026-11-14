//! Go 來源：`remake/cmd/fd2/modern_hd_battle_panel_chrome.go`（112 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Frozen with the panel, so later settlement cannot change a bar already shown.
/// Colors retain the source palette; geometry is the HD presentation contract.
///
/// Go：`modern_hd_battle_panel_chrome.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct ModernBattlePanelChrome {
    pub(crate) ready: bool,
    pub(crate) border: u8,
    pub(crate) track: u8,
    pub(crate) ink: [u8; 2],
    pub(crate) width: [i64; 2],
    pub(crate) frame: [u8; 6258],
}

impl Default for ModernBattlePanelChrome {
    fn default() -> Self {
        Self {
            ready: Default::default(),
            border: Default::default(),
            track: Default::default(),
            ink: Default::default(),
            width: Default::default(),
            frame: std::array::from_fn(|_| Default::default()),
        }
    }
}

/// Go：`modern_hd_battle_panel_chrome.go:22`，`func freezeModernBattlePanelChrome(assets battle.NativeItemPanelDataAssets, values battle.NativeBattlePanelValues) (modernBattlePanelChrome, error)`
pub(crate) fn freeze_modern_battle_panel_chrome(
    assets: &battle::NativeItemPanelDataAssets,
    values: battle::NativeBattlePanelValues,
) -> Result<ModernBattlePanelChrome> {
    let panel = &assets.battle_panel;
    if panel.width != 149 || panel.height != 42 || panel.pixels.len() != 149 * 42 {
        return Err(game_err!("modern HD panel: chrome source unavailable"));
    }
    let cell = |index: i64| assets.raw_cells.as_ref().and_then(|cells| cells.get(&index));
    for index in [24, 27, 29] {
        match cell(index) {
            Some(c) if c.width == 1 && c.height == 5 && c.pixels.len() == 5 => {}
            _ => return Err(game_err!("modern HD panel: chrome raw cell {index} unavailable")),
        }
    }
    let mut c = ModernBattlePanelChrome {
        ready: true,
        border: panel.pixels[149 + 1],
        track: cell(29).unwrap().pixels[1],
        ink: [cell(24).unwrap().pixels[1], cell(27).unwrap().pixels[1]],
        ..Default::default()
    };
    c.frame.copy_from_slice(&panel.pixels);
    for (i, bar) in [[values.hp, values.max_hp], [values.mp, values.max_mp]].into_iter().enumerate() {
        if bar[0] == 0 || bar[1] == 0 {
            continue;
        }
        // The native writer emits a left cap, repeated body, then a right cap.
        // Even a negative length still emits both caps. Clip only presentation.
        let mut length = 101i64.wrapping_mul(bar[0]) / bar[1] + 1;
        if length < 1 {
            length = 1;
        }
        c.width[i] = (length + 1) * 6;
        if c.width[i] > 103 * 6 {
            c.width[i] = 103 * 6;
        }
    }
    Ok(c)
}

impl ModernBattlePanelChrome {
    /// Go：`modern_hd_battle_panel_chrome.go:55`，`func (c modernBattlePanelChrome) paint(canvas *image.RGBA, palette color.Palette, background byte) error`
    /// Go 值接收者：本體對接收者的改寫只影響副本。
    pub(crate) fn paint(&self, canvas: &mut Rgba, palette: &ColorPalette, background: u8) -> Result<()> {
        if !self.ready || canvas.rect != rgba_rect(0, 0, 894, 252) {
            return Err(game_err!("modern HD panel: frozen chrome unavailable"));
        }
        let colour = |index: u8| palette[index as usize];
        // Go `draw.Draw(canvas, r, image.NewUniform(c), image.Point{}, draw.Src)`：裁到畫布範圍後整塊寫入。
        let fill_colour = |canvas: &mut Rgba, r: Rectangle, c: ColorRgba| {
            let (x0, y0) = (r.min.x.max(canvas.rect.min.x), r.min.y.max(canvas.rect.min.y));
            let (x1, y1) = (r.max.x.min(canvas.rect.max.x), r.max.y.min(canvas.rect.max.y));
            for y in y0..y1 {
                for x in x0..x1 {
                    let o = ((y - canvas.rect.min.y) * canvas.stride + (x - canvas.rect.min.x) * 4) as usize;
                    canvas.pix[o..o + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
                }
            }
        };
        let fill = |canvas: &mut Rgba, r: Rectangle, index: u8| fill_colour(canvas, r, colour(index));
        let stroke = |canvas: &mut Rgba, r: Rectangle, width: i64, index: u8| {
            fill(canvas, rgba_rect(r.min.x, r.min.y, r.max.x, r.min.y + width), index);
            fill(canvas, rgba_rect(r.min.x, r.max.y - width, r.max.x, r.max.y), index);
            fill(canvas, rgba_rect(r.min.x, r.min.y, r.min.x + width, r.max.y), index);
            fill(canvas, rgba_rect(r.max.x - width, r.min.y, r.max.x, r.max.y), index);
        };
        let bounds = canvas.rect;
        fill(canvas, bounds, background);
        // FDOTHER#5/22 has three distinct perimeter pixels, including the
        // asymmetric corner highlights and lower shadow. Keep the complete source
        // bevel instead of collapsing it to one uniform outline.
        for y in 0..42i64 {
            for x in 0..149i64 {
                if !(3..146).contains(&x) || !(3..39).contains(&y) {
                    fill(canvas, rgba_rect(x * 6, y * 6, (x + 1) * 6, (y + 1) * 6), self.frame[(y * 149 + x) as usize]);
                }
            }
        }
        fill(canvas, rgba_rect(24, 120, 870, 122), self.border);
        for (i, y) in [22i64, 31].into_iter().enumerate() {
            let r = rgba_rect(21 * 6, y * 6, 124 * 6, (y + 6) * 6);
            fill(canvas, r, self.track);
            if self.width[i] > 0 {
                let body = rgba_rect(r.min.x, r.min.y, r.min.x + self.width[i], r.max.y);
                fill(canvas, body, self.ink[i]);
                let base = colour(self.ink[i]);
                for row in 0..6i64 {
                    // Go 以 `int` 計算後 `uint8(...)` 截斷。
                    let light = |v: u8| (i64::from(v) + (255 - i64::from(v)) * (6 - row) / 15) as u8;
                    let dark = |v: u8| (i64::from(v) * (12 + row) / 18) as u8;
                    for (edge_y, c) in [
                        (r.min.y + row, ColorRgba { r: light(base.r), g: light(base.g), b: light(base.b), a: base.a }),
                        (r.max.y - 1 - row, ColorRgba { r: dark(base.r), g: dark(base.g), b: dark(base.b), a: base.a }),
                    ] {
                        fill_colour(canvas, rgba_rect(body.min.x, edge_y, body.max.x, edge_y + 1), c);
                    }
                }
            }
            stroke(canvas, r, 2, self.border);
        }
        Ok(())
    }
}

/// Go `image.Rect(x0, y0, x1, y1)`（座標已是正規化的順序）。
pub(crate) fn rgba_rect(x0: i64, y0: i64, x1: i64, y1: i64) -> Rectangle {
    Rectangle { min: Point { x: x0, y: y0 }, max: Point { x: x1, y: y1 } }
}

/// Go `image.NewRGBA(image.Rect(0, 0, w, h))`：全透明黑的畫布。
pub(crate) fn new_rgba_canvas(w: i64, h: i64) -> Rgba {
    Rgba { pix: vec![0; (w * h * 4) as usize], stride: w * 4, rect: rgba_rect(0, 0, w, h) }
}

#[cfg(test)]
#[path = "modern_hd_battle_panel_chrome/modern_hd_battle_panel_chrome_test.rs"]
mod modern_hd_battle_panel_chrome_test;
