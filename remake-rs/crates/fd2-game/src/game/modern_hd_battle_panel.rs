//! Go 來源：`remake/cmd/fd2/modern_hd_battle_panel.go`（246 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Frozen at the native panel write, before settlement or a later HP step.
/// The native image is also the exact source identity; no glyph recognition.
///
/// Go：`modern_hd_battle_panel.go:19`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernBattlePanelText {
    pub(crate) native: figani::Frame,
    pub(crate) name: String,
    pub(crate) values: battle::NativeBattlePanelValues,
    pub(crate) chrome: ModernBattlePanelChrome,
    pub(crate) background: u8,
    pub(crate) digit_ink: [u8; 2],
    pub(crate) font: Option<Font>,
    pub(crate) last_palette: [u8; 32],
    pub(crate) last_scale: f64,
    pub(crate) picture: Option<Rgba>,
    pub(crate) key: String,
}

impl Game {
    /// Go：`modern_hd_battle_panel.go:56`，`func (g *Game) renderPhysicalBattlePanel(scene *nativePhysicalScene, record, dst []byte, index int) error`
    pub(crate) fn render_physical_battle_panel(
        &mut self,
        scene: &mut NativePhysicalScene,
        record: &mut [u8],
        dst: &mut [u8],
        index: i64,
    ) -> Result<()> {
        todo!("modern_hd_battle_panel.go:renderPhysicalBattlePanel")
    }

    /// Go：`modern_hd_battle_panel.go:60`，`func (g *Game) renderBattlePanelText(assets battle.NativeItemPanelDataAssets, record, dst []byte, index, chapter int, collected *[]*modernBattlePanelText) error`
    pub(crate) fn render_battle_panel_text(
        &mut self,
        assets: &battle::NativeItemPanelDataAssets,
        record: &[u8],
        dst: &mut [u8],
        index: i64,
        chapter: i64,
        collected: Option<&mut Vec<ModernBattlePanelText>>,
    ) -> Result<()> {
        self.render_localized_native_battle_panel(assets, record, dst, index, chapter)?;
        let Some(collected) = collected else {
            return Ok(());
        };
        if self.font.is_none() || self.locale_entities.is_none() {
            return Ok(());
        }
        let (x, y) = battle::native_battle_panel_origin(record, index, chapter)?;
        let Ok(name) = self.locale_entities.as_ref().unwrap().battle_name(i64::from(record[8])) else {
            return Ok(()); // An unresolved optional name must not change native reachability.
        };
        let base_font = self.font.as_ref().unwrap();
        let f = base_font.hd_fonts.get(&self.locale_id).unwrap_or(base_font).clone();
        let mut p = ModernBattlePanelText {
            font: Some(f),
            name,
            background: assets.battle_panel.pixels[10 * 149 + 100],
            ..Default::default()
        };
        for (i, base) in [31i64, 42].into_iter().enumerate() {
            let mut counts = [0i64; 256];
            if let Some(frame) = assets.frames.as_ref().and_then(|frames| frames.get(&base)) {
                for (k, &pixel) in frame.indexed.iter().enumerate() {
                    if frame.mask[k] != 0 && pixel != p.background {
                        counts[pixel as usize] += 1;
                    }
                }
            }
            let mut brightness = 0u32;
            for (pixel, &count) in counts.iter().enumerate() {
                // Go `color.RGBA.RGBA()`：16 位元預乘值 `v * 0x101`。
                let c = self.native_ui_palette[pixel];
                let (r, green, b) = (u32::from(c.r) * 0x101, u32::from(c.g) * 0x101, u32::from(c.b) * 0x101);
                let light = 30 * r + 59 * green + 11 * b;
                if count > 0 && light > brightness {
                    p.digit_ink[i] = pixel as u8;
                    brightness = light;
                }
            }
        }
        p.native = figani::Frame {
            x,
            y,
            width: 149,
            height: 42,
            pixels: vec![0; 149 * 42],
            mask: vec![255; 149 * 42],
            ..Default::default()
        };
        for row in 0..42i64 {
            let src = ((y + row) * 320 + x) as usize;
            p.native.pixels[(row * 149) as usize..((row + 1) * 149) as usize].copy_from_slice(&dst[src..src + 149]);
        }
        let word = |offset: usize| i64::from(i16::from_le_bytes([record[offset], record[offset + 1]]));
        p.values = battle::NativeBattlePanelValues {
            level: i64::from(record[0x21]),
            hp: word(0x40),
            max_hp: word(0x42),
            mp: word(0x44),
            max_mp: word(0x46),
        };
        p.chrome = freeze_modern_battle_panel_chrome(assets, p.values)?;
        for previous in collected.iter() {
            if previous.native.x == x
                && previous.native.y == y
                && previous.name == p.name
                && previous.values == p.values
                && previous.chrome == p.chrome
                && same_panel_font(previous.font.as_ref(), p.font.as_ref())
                && previous.native.pixels == p.native.pixels
            {
                return Ok(());
            }
        }
        // Validate before publishing this scene or committing any combat state.
        let palette = self.native_ui_palette.clone();
        p.raster(&palette)?;
        collected.push(p);
        Ok(())
    }
}

/// Bind only a complete, byte-identical panel at its proven native origin.
/// Partial scroll views inherit the explicit layer and clip from their producer.
///
/// Go：`modern_hd_battle_panel.go:123`，`func bindModernPhysicalPanelText(scene *nativePhysicalScene, sources *modernHDBattleSources)`
pub(crate) fn bind_modern_physical_panel_text(
    scene: Option<&mut NativePhysicalScene>,
    sources: &mut ModernHDBattleSources,
) {
    todo!("modern_hd_battle_panel.go:bindModernPhysicalPanelText")
}

/// Go：`modern_hd_battle_panel.go:129`，`func bindModernBattlePanelText(panels []*modernBattlePanelText, sources *modernHDBattleSources)`
pub(crate) fn bind_modern_battle_panel_text(
    panels: &mut [ModernBattlePanelText],
    sources: Option<&mut ModernHDBattleSources>,
) {
    todo!("modern_hd_battle_panel.go:bindModernBattlePanelText")
}

impl ModernBattlePanelText {
    /// Go：`modern_hd_battle_panel.go:163`，`func (p *modernBattlePanelText) raster(palette color.Palette) (image.Image, string, error)`
    ///
    /// Go 回傳 `image.Image`（`*image.RGBA` 畫布）；Rust 回傳同一份像素的 [`Picture`]，畫布本身存在 `picture`。
    pub(crate) fn raster(&mut self, palette: &ColorPalette) -> Result<(Option<Arc<Picture>>, String)> {
        let raw = modern_hd_battle_palette(palette)?;
        let key: [u8; 32] = {
            use sha2::Digest;
            sha2::Sha256::digest(&raw).into()
        };
        let mut scale = 1.0;
        if let Some(font) = &self.font
            && font.user_scale > 0.0
        {
            scale = font.user_scale;
        }
        if let Some(picture) = &self.picture
            && key == self.last_palette
            && scale == self.last_scale
        {
            return Ok((Some(Arc::new(picture_from_rgba(picture))), self.key.clone()));
        }
        if self.font.is_none()
            || self.native.width != 149
            || self.native.height != 42
            || self.native.pixels.len() != 149 * 42
        {
            return Err(game_err!("modern HD panel: frozen source unavailable"));
        }
        let mut canvas = new_rgba_canvas(894, 252);
        self.chrome.paint(&mut canvas, palette, self.background)?;
        struct Field {
            value: String,
            rect: Rectangle,
            size: i64,
            colour: u8,
        }
        let ink0 = self.digit_ink[0];
        let mut fields = vec![
            Field { value: self.name.clone(), rect: rgba_rect(5, 4, 114, 20), size: 96, colour: ink0 },
            Field { value: String::from("LV·"), rect: rgba_rect(115, 4, 131, 13), size: 48, colour: ink0 },
            Field { value: String::from("HP"), rect: rgba_rect(3, 21, 20, 30), size: 48, colour: ink0 },
            Field { value: String::from("MP"), rect: rgba_rect(3, 30, 20, 39), size: 48, colour: ink0 },
        ];
        for (value, max, width, x, y) in
            [(self.values.level, 99, 2, 132, 4), (self.values.hp, 999, 3, 126, 21), (self.values.mp, 999, 3, 126, 30)]
        {
            if value > max {
                self.preserve_native_panel_marker(&mut canvas, palette, rgba_rect(x, y, 147, y + 9));
                continue;
            } // Keep the original overflow glyph.
            let value = value.max(0);
            let mut ink = ink0;
            if y == 21 && self.values.hp != self.values.max_hp || y == 30 && self.values.mp != self.values.max_mp {
                ink = self.digit_ink[1];
            }
            fields.push(Field {
                value: format!("{value:0width$}", width = width as usize),
                rect: rgba_rect(x, y, x + width * 6 + 1, y + 9),
                size: 48,
                colour: ink,
            });
        }
        for field in &fields {
            let r = rgba_rect(field.rect.min.x * 6, field.rect.min.y * 6, field.rect.max.x * 6, field.rect.max.y * 6);
            let ink = native_palette_color(palette, field.colour, ColorRgba { r: 240, g: 244, b: 255, a: 255 });
            let font = self.font.as_mut().unwrap();
            let face = if field.value != self.name && font.hd_heavy.is_some() {
                font.hd_heavy.as_deref_mut().unwrap()
            } else {
                font
            };
            draw_modern_battle_panel_ink(&mut canvas, face, &field.value, r, field.size, ink)?;
        }
        let digest: [u8; 32] = {
            use sha2::Digest;
            sha2::Sha256::digest(&canvas.pix).into()
        };
        self.key = format!("panel-text/{}", digest.iter().map(|b| format!("{b:02x}")).collect::<String>());
        self.last_palette = key;
        self.last_scale = scale;
        let picture = picture_from_rgba(&canvas);
        self.picture = Some(canvas);
        Ok((Some(Arc::new(picture)), self.key.clone()))
    }

    /// Go：`modern_hd_battle_panel_chrome.go:105`，`func (p *modernBattlePanelText) preserveNativePanelMarker(canvas *image.RGBA, palette color.Palette, r image.Rectangle)`
    /// （`r8`，隨 `raster` 補移植）。
    fn preserve_native_panel_marker(&self, canvas: &mut Rgba, palette: &ColorPalette, r: Rectangle) {
        for y in r.min.y..r.max.y {
            for x in r.min.x..r.max.x {
                let c = palette[self.native.pixels[(y * 149 + x) as usize] as usize];
                for py in y * 6..(y + 1) * 6 {
                    for px in x * 6..(x + 1) * 6 {
                        if px < canvas.rect.min.x
                            || px >= canvas.rect.max.x
                            || py < canvas.rect.min.y
                            || py >= canvas.rect.max.y
                        {
                            continue;
                        }
                        let o = ((py - canvas.rect.min.y) * canvas.stride + (px - canvas.rect.min.x) * 4) as usize;
                        canvas.pix[o..o + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
                    }
                }
            }
        }
    }
}

/// Go：`modern_hd_battle_panel.go:224`，`func drawModernBattlePanelInk(dst *image.RGBA, f *Font, value string, rect image.Rectangle, maximum int, ink color.RGBA) error`
pub(crate) fn draw_modern_battle_panel_ink(
    dst: &mut Rgba,
    f: &mut Font,
    value: &str,
    rect: Rectangle,
    maximum: i64,
    ink: ColorRgba,
) -> Result<()> {
    // Go 以 `font.Drawer` 把 HD 面板文字畫進 RGBA 畫布（只在 HD 介面呼叫）。
    todo!("R6: modern_hd_battle_panel.go:drawModernBattlePanelInk（HD 面板文字繪製）")
}

/// Rust 新增：Go 以 `previous.font == p.font` 比較 `*Font` 指標。Rust 的 `ModernBattlePanelText` 存字型複本，
/// 改以「同一份 TTF（`sf` 的 `Arc` 相同）且基準尺寸與字級倍率相同」視為同一個字型；同一次面板收集裡 Go 的
/// 指標都是 `g.font` 或同一個區域字型，兩種比法結果相同。
pub(crate) fn same_panel_font(a: Option<&Font>, b: Option<&Font>) -> bool {
    match (a, b) {
        (None, None) => true,
        (Some(a), Some(b)) => {
            let same_sf = match (&a.sf, &b.sf) {
                (None, None) => true,
                (Some(x), Some(y)) => Arc::ptr_eq(x, y),
                _ => false,
            };
            same_sf && a.base == b.base && a.user_scale == b.user_scale
        }
        _ => false,
    }
}

/// Rust 新增：Go `*image.RGBA` 畫布轉成 [`Picture`]（逐位元組複製；畫布底色不透明，預乘與否相同）。
pub(crate) fn picture_from_rgba(canvas: &Rgba) -> Picture {
    let (w, h) = (canvas.rect.max.x - canvas.rect.min.x, canvas.rect.max.y - canvas.rect.min.y);
    let mut pix = Vec::with_capacity((w * h * 4) as usize);
    for y in 0..h {
        let o = (y * canvas.stride) as usize;
        pix.extend_from_slice(&canvas.pix[o..o + (w * 4) as usize]);
    }
    Picture { width: w, height: h, pix }
}
