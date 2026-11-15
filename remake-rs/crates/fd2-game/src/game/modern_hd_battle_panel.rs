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
        record: &[u8],
        dst: &mut [u8],
        index: i64,
    ) -> Result<()> {
        let chapter = self.handler_chapter;
        self.render_battle_panel_text(&scene.panel_assets, record, dst, index, chapter, Some(&mut scene.hd_panel_text))
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
///
/// R4 D 組：`scene` 只讀、`sources` 可為 nil（Go 的 `*modernHDBattleSources`），簽名變更紀錄。
pub(crate) fn bind_modern_physical_panel_text(
    scene: Option<&NativePhysicalScene>,
    sources: Option<&mut ModernHDBattleSources>,
) {
    if let Some(scene) = scene {
        bind_modern_battle_panel_text(&scene.hd_panel_text, sources);
    }
}

/// Go：`modern_hd_battle_panel.go:129`，`func bindModernBattlePanelText(panels []*modernBattlePanelText, sources *modernHDBattleSources)`
///
/// R4 D 組：`panels` 只讀（簽名變更紀錄）。Go 以指標相等（`layer.PanelText == p`）判斷面板是否已綁；
/// Rust 的來源存複本，改以 [`same_panel_text`] 比對（`panel-text-identity`）。
pub(crate) fn bind_modern_battle_panel_text(
    panels: &[ModernBattlePanelText],
    sources: Option<&mut ModernHDBattleSources>,
) {
    let Some(sources) = sources.filter(|s| s.base.len() == 64000) else {
        return;
    };
    for p in panels {
        let f = &p.native;
        let mut matches = true;
        for row in 0..f.height {
            let src = ((f.y + row) * 320 + f.x) as usize;
            if f.pixels[(row * f.width) as usize..((row + 1) * f.width) as usize]
                != sources.base[src..src + f.width as usize]
            {
                matches = false;
                break;
            }
        }
        if !matches {
            continue;
        }
        let found = sources.base_scenery.iter().chain(sources.layers.iter()).any(|layer| {
            layer.panel_text.as_ref().is_some_and(|text| same_panel_text(text, p)) && layer.dx == 0 && layer.dy == 0
        });
        if !found {
            // Go 第一次追加前先複製切片（`cloned`），避免改到別的來源共用的底層陣列；Rust 的 Vec 本來就各自擁有。
            sources.base_scenery.push(ModernHDBattleSource {
                frame: f.clone(),
                resource: -1,
                fill: -1,
                panel_text: Some(p.clone()),
                ..Default::default()
            });
        }
    }
}

/// R4 D 組新增，沒有 Go 對應：Go 以 `*modernBattlePanelText` 指標相等辨認同一塊面板；Rust 存複本，比對
/// `renderBattlePanelText` 去重時用的同一組欄位（原點、名稱、數值、外框、字型與原生像素），同一場景內兩者等價。
pub(crate) fn same_panel_text(a: &ModernBattlePanelText, b: &ModernBattlePanelText) -> bool {
    a.native.x == b.native.x
        && a.native.y == b.native.y
        && a.name == b.name
        && a.values == b.values
        && a.chrome == b.chrome
        && same_panel_font(a.font.as_ref(), b.font.as_ref())
        && a.native.pixels == b.native.pixels
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
    // R4 E 組（主代理指派）：Go 的 `prepareNativePlayerMovement` 與物理演出在字型有 TTF、`localeEntities`
    // 存在時就收集 HD 面板文字，`renderBattlePanelText` 以 `raster` 預檢（忠實模式也一樣），預檢失敗會讓流程
    // 回錯誤。這裡照 Go 移植「找得到不超框字級」的量測與兩個錯誤；字形點陣化（`font.Drawer` 的陰影與前景
    // 兩次 `DrawString`）只有 HD 繪製讀（R6），Rust 不在畫布上畫字、直接回成功，所以 `raster` 的畫布與 `key` 沒有文字墨跡
    // （KNOWN_DIFFERENCES `panel-ink-measure-only`；只有 HD 繪製讀它們）。
    // Go `int(f.effectiveScale(float64(maximum)))`：轉整數時向零截斷。
    let maximum = f.effective_scale(maximum as f64) as i64;
    let mut size = maximum;
    while size >= 24 {
        let (face, _) = f.face_for(size);
        let Some(face) = face else {
            return Err(GameError::new("modern HD panel: font unavailable"));
        };
        // Go `font.BoundString(face, value)` 再取 `Floor`／`Ceil`。量測器的 `bound_string` 對應 Ebiten
        // `text.BoundString`（v2.6.6 `text/text.go:281`）；單行字串兩者的聯集相同（`fixed.Rectangle26_6.Union`
        // 略過空矩形，x/image v0.12.0 `font/font.go:245` 另以 `!b.Empty()` 略過），面板字串都沒有換行。
        // R5 r5f 起 `load_font_sized` 讀到字型就注入量測器，`face_for` 回 `Some` 時一定有（同 `face_for` 的 `expect`）。
        let measure = f
            .measure
            .0
            .clone()
            .expect("modern_hd_battle_panel.go:drawModernBattlePanelInk：face_for 成功卻沒有 TextMeasure");
        let bounds = measure.bound_string(&face, value);
        if rect_dx(&bounds) + 2 > rect_dx(&rect) || rect_dy(&bounds) + 2 > rect_dy(&rect) {
            size -= 1;
            continue;
        }
        // R6：`pen := fixed.P(rect.Min.X-box.Min.X, rect.Min.Y-box.Min.Y)`，先以 RGBA{12,22,38,ink.A} 在
        // pen+(2,2) 畫陰影，再以 `ink` 在 pen 畫前景，目的端是 `dst.SubImage(rect)`。
        return Ok(());
    }
    Err(game_err!("modern HD panel: text exceeds safe rectangle: {:?}", value))
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

#[cfg(test)]
mod modern_hd_battle_panel_ink_test;
