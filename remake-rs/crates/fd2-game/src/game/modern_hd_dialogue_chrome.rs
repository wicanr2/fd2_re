//! Go 來源：`remake/cmd/fd2/modern_hd_dialogue_chrome.go`（247 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// This freezes FDOTHER#5's source validation and native panel colors.
/// Status/item panels keep the thin native bevel; dialogue owners explicitly
/// select the approved gold material in the same 18px border band.
///
/// Go：`modern_hd_dialogue_chrome.go:21`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDDialogueChrome {
    pub(crate) background: ColorRgba,
    /// top, right, bottom, left
    pub(crate) edges: [[ColorRgba; 3]; 4],
    /// top-left, top-right, bottom-left, bottom-right
    pub(crate) corners: [[ColorRgba; 9]; 4],
    pub(crate) gpu: Option<Arc<Picture>>,
    pub(crate) gold: Option<Arc<Picture>>,
    pub(crate) gold_gpu: BTreeMap<Point, Option<Arc<Picture>>>,
}

impl ModernHDDialogueChrome {
    pub(crate) fn thin_border(&mut self, area: Rectangle) -> Arc<Picture> {
        self.gpu
            .get_or_insert_with(|| {
                let (w, h) = (rect_dx(&area) * 3, rect_dy(&area) * 3);
                let mut canvas = Picture::new_image(w, h);
                for band in
                    [rect(0, 0, w, 18), rect(0, h - 18, w, h), rect(0, 18, 18, h - 18), rect(w - 18, 18, w, h - 18)]
                {
                    canvas.sub_image_mut(band).fill(self.background);
                }
                for i in 0..3 {
                    for (side, band) in [
                        rect(3, i, w - 3, i + 1),
                        rect(w - 3 + i, 3, w - 2 + i, h - 3),
                        rect(3, h - 3 + i, w - 3, h - 2 + i),
                        rect(i, 3, i + 1, h - 3),
                    ]
                    .into_iter()
                    .enumerate()
                    {
                        canvas.sub_image_mut(band).fill(self.edges[side][i as usize]);
                    }
                }
                for (corner, (x, y)) in [(0, 0), (w - 3, 0), (0, h - 3), (w - 3, h - 3)].into_iter().enumerate() {
                    for (i, &color) in self.corners[corner].iter().enumerate() {
                        canvas.set(x + (i % 3) as i64, y + (i / 3) as i64, color);
                    }
                }
                Arc::new(canvas)
            })
            .clone()
    }

    pub(crate) fn gold_border(&mut self, area: Rectangle) -> Option<Arc<Picture>> {
        self.gold_border_for_size(Point { x: 6 * rect_dx(&area), y: 6 * rect_dy(&area) })
    }

    pub(crate) fn gold_border_for_size(&mut self, size: Point) -> Option<Arc<Picture>> {
        if size.x < 36 || size.y < 36 {
            return None;
        }
        let tile = self.gold.as_ref()?;
        self.gold_gpu
            .entry(size)
            .or_insert_with(|| {
                let axis = |at, size| {
                    if at < 18 {
                        at
                    } else if at >= size - 18 {
                        48 - size + at
                    } else {
                        24
                    }
                };
                let source = tile.read_pixels().as_chunks::<4>().0;
                let mut pixels = vec![[0; 4]; (size.x * size.y) as usize];
                for y in 0..size.y {
                    for x in 0..size.x {
                        if x < 18 || x >= size.x - 18 || y < 18 || y >= size.y - 18 {
                            let from = axis(y, size.y) * 48 + axis(x, size.x);
                            pixels[(y * size.x + x) as usize] = source[from as usize];
                        }
                    }
                }
                let mut border = Picture::new_image(size.x, size.y);
                border.write_pixels(pixels.as_flattened());
                Some(Arc::new(border))
            })
            .clone()
    }
}

/// Go：`modern_hd_dialogue_chrome.go:30`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDDialoguePanel {
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    /// Frozen native 320x200 coordinates, before clipping.
    pub(crate) rect: Rectangle,
    /// Only dialogue owners use the approved antique-gold material.
    pub(crate) gold: bool,
    /// Legacy story box geometry, snapped to HD pixels.
    pub(crate) physical_rect: Rectangle,
}

/// Go：`modern_hd_dialogue_chrome.go:41`，`func freezeModernHDDialogueChrome(cells []fdother.RawCell, palette color.Palette) (*modernHDDialogueChrome, error)`
pub(crate) fn freeze_modern_hd_dialogue_chrome(
    cells: &[fdother::RawCell],
    palette: &ColorPalette,
) -> Result<Option<ModernHDDialogueChrome>> {
    if cells.len() <= 13 || palette.len() != 256 {
        return Err(game_err!("modern HD dialogue: source unavailable"));
    }
    // Go 以 map 字面值逐格檢查（順序隨機）；Rust 依格號升冪（只影響多格同時不符時回報哪一格）。
    for (i, (w, h)) in [
        (1, (3, 3)),
        (2, (3, 3)),
        (3, (3, 3)),
        (4, (3, 3)),
        (9, (16, 3)),
        (10, (3, 16)),
        (11, (3, 16)),
        (12, (16, 3)),
        (13, (16, 16)),
    ] {
        let cell = &cells[i];
        if cell.width != w || cell.height != h || cell.pixels.len() as i64 != w * h {
            return Err(game_err!("modern HD dialogue: raw cell{} shape differs", i));
        }
    }
    let bg = cells[13].pixels[0];
    if cells[13].pixels.iter().any(|&pixel| pixel != bg) {
        return Err(game_err!("modern HD dialogue: background is not uniform"));
    }
    // Go `palette[index].RGBA()`：`color.Palette` 的元素在 Rust 是 `ColorRgba`，不會是 nil；alpha 不是 255 就不是不透明。
    let rgba = |index: u8| -> Result<ColorRgba> {
        let c = palette[index as usize];
        if c.a != 255 {
            return Err(game_err!("modern HD dialogue: source color is not opaque"));
        }
        Ok(ColorRgba { r: c.r, g: c.g, b: c.b, a: 255 })
    };
    let tile = match decode_png_picture(&MODERN_HD_GOLD_DIALOGUE_TILE) {
        Some(tile) if tile.width == 48 && tile.height == 48 => tile,
        _ => return Err(game_err!("modern HD dialogue: embedded gold material invalid")),
    };
    let mut result = ModernHDDialogueChrome { gold: Some(Arc::new(tile)), ..Default::default() };
    result.background = rgba(bg)?;
    for (side, source) in [9usize, 11, 12, 10].into_iter().enumerate() {
        let cell = &cells[source];
        for band in 0..3usize {
            let index = if cell.width == 16 { cell.pixels[band * 16 + 8] } else { cell.pixels[8 * 3 + band] };
            for along in 0..16usize {
                let at = if cell.width == 3 { along * 3 + band } else { band * 16 + along };
                if cell.pixels[at] != index {
                    return Err(game_err!("modern HD dialogue: raw cell{} bevel is not uniform", source));
                }
            }
            result.edges[side][band] = rgba(index)?;
        }
    }
    for corner in 0..result.corners.len() {
        for (i, &index) in cells[corner + 1].pixels.iter().enumerate() {
            result.corners[corner][i] = rgba(index)?;
        }
    }
    Ok(Some(result))
}

/// Code-generated from the approved SVG, not original game artwork. The 18px
/// corners stay fixed; the flat center strips repeat without scaling the bevel.
///
/// Go：`modern_hd_dialogue_chrome.go:208`
///
/// Go 以 `//go:embed modern_hd_gold_dialogue.png` 嵌入 `remake/cmd/fd2/modern_hd_gold_dialogue.png`；素材放哪裡由擁有本檔的組決定（`include_bytes!` 或執行期讀檔）。
///
/// R4 D 組：Rust 容器只掛 `remake-rs`，`include_bytes!` 讀不到 `remake/cmd/fd2`，所以把這張 310 位元組、由 SVG 產生的
/// 程式素材（不是原版美術）照原樣寫成位元組常數。來源 `remake/cmd/fd2/modern_hd_gold_dialogue.png`，
/// SHA-256 `e03b4aa090221e8145d36e1541c42490546cc37197279327af81df3d0d09afb8`（單元測試核對）。
pub(crate) static MODERN_HD_GOLD_DIALOGUE_TILE: LazyLock<Vec<u8>> =
    LazyLock::new(|| MODERN_HD_GOLD_DIALOGUE_TILE_BYTES.to_vec());

/// [`MODERN_HD_GOLD_DIALOGUE_TILE`] 的位元組。
const MODERN_HD_GOLD_DIALOGUE_TILE_BYTES: [u8; 310] = [
    0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00,
    0x30, 0x00, 0x00, 0x00, 0x30, 0x08, 0x06, 0x00, 0x00, 0x00, 0x57, 0x02, 0xf9, 0x87, 0x00, 0x00, 0x00, 0x09, 0x70,
    0x48, 0x59, 0x73, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x84, 0x79, 0x17, 0x73, 0x00, 0x00, 0x00,
    0xe8, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0xed, 0xda, 0x31, 0x0a, 0xc2, 0x40, 0x10, 0x85, 0xe1, 0xbd, 0x85, 0x68,
    0x2b, 0x22, 0x69, 0x44, 0xb0, 0xb1, 0xb1, 0x11, 0x8b, 0x94, 0xa2, 0x8d, 0x58, 0xa5, 0xb3, 0x15, 0xc5, 0xd6, 0x1b,
    0x58, 0x59, 0x08, 0x56, 0xf6, 0x9e, 0xc0, 0x6b, 0x78, 0x08, 0xaf, 0xb1, 0xf2, 0x84, 0x34, 0xa2, 0xb0, 0x6b, 0x76,
    0x9d, 0x9d, 0xf0, 0x02, 0x3f, 0x84, 0x40, 0xc2, 0x7c, 0x90, 0x90, 0x90, 0xc4, 0xcc, 0x26, 0x99, 0xdd, 0x15, 0x43,
    0x95, 0x61, 0x76, 0x83, 0x95, 0xdb, 0x79, 0xa1, 0x32, 0xcc, 0x4e, 0x00, 0x01, 0x31, 0x00, 0xa7, 0x7d, 0x2e, 0x7e,
    0x81, 0xbe, 0x87, 0x99, 0x9c, 0x01, 0xd8, 0xf6, 0xb8, 0x5f, 0x92, 0xa9, 0xdf, 0x6d, 0xda, 0x6f, 0x73, 0x7e, 0x04,
    0x48, 0x0f, 0x5c, 0x76, 0x3d, 0xae, 0xec, 0x7a, 0xde, 0x7b, 0xb5, 0x59, 0x0e, 0xfc, 0x01, 0x90, 0x4b, 0x36, 0x1d,
    0xb5, 0x09, 0x20, 0x80, 0x80, 0x18, 0x80, 0x58, 0x37, 0x26, 0x02, 0x7c, 0x01, 0xa1, 0x4e, 0x15, 0x51, 0x40, 0xa3,
    0xd5, 0xa9, 0x14, 0x01, 0xa1, 0x00, 0xc6, 0x73, 0x21, 0x80, 0x00, 0x02, 0x08, 0x20, 0x80, 0x00, 0x02, 0x08, 0xa8,
    0x13, 0x40, 0xfd, 0xc3, 0x9c, 0x5a, 0x40, 0x88, 0xc4, 0x00, 0xa1, 0x23, 0xc0, 0x15, 0xf0, 0xaf, 0x08, 0x20, 0x40,
    0x12, 0x80, 0x77, 0xf3, 0x38, 0x80, 0x64, 0x45, 0x9e, 0xfd, 0x0e, 0x28, 0x77, 0x4c, 0x25, 0x2f, 0x00, 0xb6, 0x49,
    0x0f, 0x5c, 0x09, 0x80, 0x0f, 0x6a, 0xd8, 0x21, 0xa5, 0x0e, 0xdb, 0xb1, 0x3b, 0x40, 0x4b, 0x04, 0x48, 0x57, 0x0f,
    0x80, 0xf6, 0xdf, 0x6d, 0x9e, 0x87, 0x50, 0x18, 0xdd, 0xd4, 0x21, 0x1e, 0x5c, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
    0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
];

#[cfg(test)]
#[path = "modern_hd_dialogue_chrome/modern_hd_dialogue_chrome_test.rs"]
mod modern_hd_dialogue_chrome_test;
