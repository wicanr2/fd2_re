//! Go 來源：`remake/cmd/fd2/modern_hd_title_interface.go`（281 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。
//!
//! 忠實模式下載入標題與讀檔槽素材時也會凍結 HD 資料（Go 在載入時就核對原版素材雜湊，失敗即整批
//! 拒絕），所以這裡的核對要照 Go 完整移植。`seal` 只和自己比對（Go `valid()` 重算後比較），
//! Rust 以自己的位元組序列計算 SHA-256，不重現 Go `json.Marshal` 的位元組
//! （KNOWN_DIFFERENCES `hd-title-seal-bytes`）。

use super::*;
use sha2::{Digest, Sha256};

/// Go：`modern_hd_title_interface.go:18`
pub(crate) static MODERN_HD_TITLE_ITEM_HASHES: LazyLock<[String; 6]> = LazyLock::new(|| {
    [
        String::from("384f6a9d22e430c71aff99587aa22dccc55ee24d474fa2ec563d73fde0da9014"),
        String::from("6b82c6b8693ec55b5cfcb9f1e74e1bc9e67f232483a36363f2c4f7c786da314a"),
        String::from("cf10d302eec5615156f8d6061bec36780662a2c62ba226a983da3387bcdd1a96"),
        String::from("f237dfcd75c398f841c9a7c12ecbd846d8a35bb67ec85180e7b31baef55cf3c8"),
        String::from("8ff71b5cc1af08bdeec80b23248d101049f8e8adb2d35336f41ab8dc75f542cb"),
        String::from("842648c39e8cf0a6d897ab1d7ae166d07194b256e34481910cd3303137fc77b8"),
    ]
});

/// Go：`modern_hd_title_interface.go:26`
pub(crate) static MODERN_HD_TITLE_MASK_HASHES: LazyLock<[String; 3]> = LazyLock::new(|| {
    [
        String::from("97762f67cb746e5c82034901351a88ea591ac674323a745bfce7772d84ea0af1"),
        String::from("6c1d60b8f28ad79070aa42defb4681bc7ffe060277f44fae6f3b1f73dea31b04"),
        String::from("1dd7b5aec086135580c588a9e91cb20983cfdb38ce06d49567355c7944980ffc"),
    ]
});

/// Go：`modern_hd_title_interface.go:32`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDTitleMenuItem {
    pub(crate) item: i64,
    pub(crate) state: i64,
    pub(crate) source: fdother::Frame,
    pub(crate) blank: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) original: Option<Arc<Picture>>,
    pub(crate) picture: Option<Arc<Picture>>,
    pub(crate) picture_identity: Option<Arc<Picture>>,
    pub(crate) foreground: ColorRgba,
    pub(crate) shadow: ColorRgba,
    pub(crate) seal: [u8; 32],
}

/// Go `fmt.Sprintf("%x", sha256.Sum256(b))`。
pub(crate) fn sha256_hex(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn hash_i64(h: &mut Sha256, v: i64) {
    h.update(v.to_le_bytes());
}

fn hash_bytes(h: &mut Sha256, b: &[u8]) {
    hash_i64(h, b.len() as i64);
    h.update(b);
}

fn hash_colors(h: &mut Sha256, colors: &[ColorRgba]) {
    hash_i64(h, colors.len() as i64);
    for c in colors {
        h.update([c.r, c.g, c.b, c.a]);
    }
}

/// Go：`modern_hd_title_interface.go:42`，`func titleMenuHDSeal(h *modernHDTitleMenuItem) [32]byte`
/// Go 參數是 `*modernHDTitleMenuItem`，Rust 收唯讀借用（簽名變更見 `PORTING.md`）。
pub(crate) fn title_menu_hd_seal(h: Option<&ModernHDTitleMenuItem>) -> [u8; 32] {
    let Some(h) = h else {
        return [0; 32];
    };
    let mut d = Sha256::new();
    for v in [h.item, h.state, h.source.width, h.source.height] {
        hash_i64(&mut d, v);
    }
    hash_bytes(&mut d, &h.source.indexed);
    hash_bytes(&mut d, &h.source.mask);
    hash_bytes(&mut d, &h.blank);
    hash_colors(&mut d, &h.palette);
    hash_colors(&mut d, &[h.foreground, h.shadow]);
    d.finalize().into()
}

/// Go `color.Color.RGBA()` 的 16 位元預乘值（`ColorRgba` 已預乘，乘 0x101）。
fn rgba16(c: ColorRgba) -> (u32, u32, u32, u32) {
    (c.r as u32 * 0x101, c.g as u32 * 0x101, c.b as u32 * 0x101, c.a as u32 * 0x101)
}

/// Go：`modern_hd_title_interface.go:50`，`func freezeModernHDTitleMenuItem(source fdother.Frame, dac []byte, palette color.Palette, item, state int, original *ebiten.Image) (*modernHDTitleMenuItem, error)`
/// `original` 收 `&Arc<Picture>`（Go 以指標身分比對，Rust 存同一個 `Arc`；簽名變更見 `PORTING.md`）。
pub(crate) fn freeze_modern_hd_title_menu_item(
    source: &fdother::Frame,
    dac: &[u8],
    palette: &ColorPalette,
    item: i64,
    state: i64,
    original: Option<&Arc<Picture>>,
) -> Result<Option<ModernHDTitleMenuItem>> {
    let source_ok = (0..3).contains(&item)
        && (0..2).contains(&state)
        && original.is_some()
        && !(item == 0 && source.width != 61)
        && !(item != 0 && source.width != 62)
        && !(item < 2 && source.height != 7)
        && !(item == 2 && source.height != 8)
        && source.indexed.len() as i64 == source.width * source.height
        && source.mask.len() == source.indexed.len()
        && palette.len() == 256;
    if !source_ok
        || sha256_hex(dac) != "435ff992367240d0f195f62983288741548fa9c9a6d0293ebd8735b69ad40b51"
        || sha256_hex(&source.indexed) != MODERN_HD_TITLE_ITEM_HASHES[(item * 2 + state) as usize]
        || sha256_hex(&source.mask) != MODERN_HD_TITLE_MASK_HASHES[item as usize]
    {
        return Err(GameError::new("modern HD title: original item source differs"));
    }
    let expected = color_palette_from_nrgba(&fdother::vga_palette_from_dac(dac)?);
    if &expected != palette {
        return Err(GameError::new("modern HD title: palette differs"));
    }
    let mut h = ModernHDTitleMenuItem {
        item,
        state,
        source: source.clone(),
        palette: palette.clone(),
        original: original.cloned(),
        blank: source.indexed.clone(),
        ..Default::default()
    };
    // x0..6 and the last seven columns contain the selected jewels and shadows.
    // The verified x7 background is uniform across the remaining non-letter area.
    let mut brightest: i64 = -1;
    let w = source.width;
    for y in 0..source.height {
        let bg = source.indexed[(y * w + 7) as usize];
        let (r, _, b, _) = rgba16(palette[bg as usize]);
        if b > r {
            return Err(GameError::new("modern HD title: background sample differs"));
        }
        for x in 7..w - 7 {
            let at = (y * w + x) as usize;
            let index = source.indexed[at];
            if index == bg {
                continue;
            }
            let (r, g, b, _) = rgba16(palette[index as usize]);
            if b <= r {
                return Err(GameError::new("modern HD title: non-letter source in text region"));
            }
            let value = (r + g + b) as i64;
            if value > brightest {
                brightest = value;
                h.foreground = modern_hd_native_text_color(palette, index);
            }
            h.blank[at] = bg;
        }
    }
    if brightest < 0 {
        return Err(GameError::new("modern HD title: text source missing"));
    }
    h.shadow = modern_hd_native_text_color(palette, 253);
    let mut blank = h.source.clone();
    blank.indexed = h.blank.clone();
    h.picture = masked_rgba(&blank, palette);
    h.picture_identity = h.picture.clone();
    h.seal = title_menu_hd_seal(Some(&h));
    Ok(Some(h))
}

/// 與原生選單的中心位置及最後一列下緣相同，座標以 640×400 為基準。
pub(crate) fn title_interface_rect() -> Rectangle {
    rect(258, 328, 382, 380)
}

impl ModernHDTitleMenuItem {
    fn render(
        &self,
        font: &mut Font,
        scale: f64,
        item: usize,
        state: usize,
        original: &Arc<Picture>,
    ) -> Result<Picture> {
        let picture = self
            .picture
            .as_ref()
            .zip(self.picture_identity.as_ref())
            .filter(|(p, identity)| Arc::ptr_eq(p, identity))
            .map(|(p, _)| p)
            .filter(|_| {
                (self.item, self.state) == (item as i64, state as i64)
                    && self.original.as_ref().is_some_and(|p| Arc::ptr_eq(p, original))
                    && self.seal == title_menu_hd_seal(Some(self))
            })
            .ok_or_else(|| game_err!("modern HD title: source unavailable or changed"))?;
        let measure = font.measure.0.clone().ok_or_else(|| game_err!("modern HD title: font unavailable"))?;
        let label = ["START", "LOAD", "CONTINUE"][item];
        let (width, height) = ((picture.width - 14) * 6, picture.height * 6);
        let mut ink = Picture::new_image(width, height);
        let size = (54.0 * normalize_font_scale(scale)).round() as i64;
        let (face, bounds) = (6..=size)
            .rev()
            .find_map(|size| {
                let (face, _) = font.face_for(size);
                let face = face?;
                let bounds = measure.bound_string(&face, label);
                (rect_dx(&bounds) <= width && rect_dy(&bounds) <= height).then_some((face, bounds))
            })
            .ok_or_else(|| game_err!("modern HD title: text does not fit"))?;
        let x = (width - rect_dx(&bounds)) / 2 - bounds.min.x;
        let y = (height - rect_dy(&bounds)) / 2 - bounds.min.y;
        for (offset, color) in [(1, self.shadow), (0, self.foreground)] {
            let mut op = DrawImageOptions::default();
            op.geo_m.translate((x + offset) as f64, (y + offset) as f64);
            op.color_scale.scale_with_color(color);
            measure.draw_text(&face, &mut ink, label, &op);
        }
        let mut row = Picture::new_image(picture.width * 6, height);
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(6.0, 6.0);
        row.draw_image(picture, &op);
        let mut op = DrawImageOptions::default();
        op.geo_m.translate(42.0, 0.0);
        row.draw_image(&ink, &op);
        Ok(row)
    }
}

impl Game {
    pub(crate) fn prepare_hd_title_interface(&self) -> Result<[Arc<Picture>; 4]> {
        let assets = self.title_assets.as_ref().ok_or_else(|| game_err!("modern HD title: missing menu"))?;
        let font = self.font.as_ref().ok_or_else(|| game_err!("modern HD title: font unavailable"))?;
        let mut font = font.hd_fonts.get(&self.locale_id).unwrap_or(font).clone();
        let mut rows = Vec::with_capacity(3);
        for (item, entries) in assets.hd_items.iter().enumerate() {
            let mut render = |state: usize| {
                let original = assets.items[item][state]
                    .as_ref()
                    .ok_or_else(|| game_err!("modern HD title: missing original item"))?;
                entries[state].render(&mut font, self.font_scale, item, state, original)
            };
            rows.push([render(0)?, render(1)?]);
        }
        let rect = title_interface_rect();
        Ok(std::array::from_fn(|selection| {
            let mut menu = Picture::new_image(rect_dx(&rect) * 3, rect_dy(&rect) * 3);
            for (item, states) in rows.iter().enumerate() {
                let row = &states[usize::from(item == selection)];
                let mut op = DrawImageOptions::default();
                op.geo_m.translate(
                    ((320 - row.width / 6) / 2 * 6 - rect.min.x * 3) as f64,
                    ((title_menu_y(item as i64) * 2 - rect.min.y) * 3) as f64,
                );
                menu.draw_image(row, &op);
            }
            Arc::new(menu)
        }))
    }

    pub(crate) fn hd_title_interface_picture(&self) -> Option<&Arc<Picture>> {
        if self.help_visible || self.native_full_dac_black || self.native_full_dac_white || self.title_phase != "menu" {
            return None;
        }
        let selected = (0..3).contains(&self.title_sel) && (self.title_flash == 0 || self.title_flash / 3 % 2 == 0);
        let index = if selected { self.title_sel as usize } else { 3 };
        self.hd_title_menu.as_ref().map(|pictures| &pictures[index])
    }
}

/// Go：`modern_hd_title_interface.go:167`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDTitleLoadSlots {
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) chrome_identity: Option<ModernHDDialogueChrome>,
    pub(crate) json_chrome: Option<ModernHDDialogueChrome>,
    pub(crate) json_identity: Option<ModernHDDialogueChrome>,
    pub(crate) cells: Vec<fdother::RawCell>,
    pub(crate) seal: [u8; 32],
}

/// Go：`modern_hd_title_interface.go:176`，`func titleLoadSlotsHDSeal(a *nativeLoadSlotsUIAssets, h *modernHDTitleLoadSlots) [32]byte`
/// Rust 收唯讀借用（簽名變更見 `PORTING.md`）。
pub(crate) fn title_load_slots_hd_seal(
    a: Option<&NativeLoadSlotsUIAssets>,
    h: Option<&ModernHDTitleLoadSlots>,
) -> [u8; 32] {
    let (Some(a), Some(h)) = (a, h) else {
        return [0; 32];
    };
    let Some(chrome) = h.chrome.as_ref() else {
        return [0; 32];
    };
    let mut d = Sha256::new();
    hash_i64(&mut d, a.dialogue_box.width);
    hash_i64(&mut d, a.dialogue_box.height);
    hash_bytes(&mut d, &a.dialogue_box.pixels);
    hash_colors(&mut d, &a.palette);
    hash_i64(&mut d, h.cells.len() as i64);
    for cell in &h.cells {
        hash_i64(&mut d, cell.width);
        hash_i64(&mut d, cell.height);
        hash_bytes(&mut d, &cell.pixels);
    }
    hash_colors(&mut d, &[chrome.background]);
    for edge in &chrome.edges {
        hash_colors(&mut d, edge);
    }
    for corner in &chrome.corners {
        hash_colors(&mut d, corner);
    }
    d.finalize().into()
}

/// Go：`modern_hd_title_interface.go:184`，`func freezeModernHDTitleLoadSlots(a *nativeLoadSlotsUIAssets, dac []byte) (*modernHDTitleLoadSlots, error)`
/// Rust 收唯讀借用（簽名變更見 `PORTING.md`）。
pub(crate) fn freeze_modern_hd_title_load_slots(
    a: Option<&NativeLoadSlotsUIAssets>,
    dac: &[u8],
) -> Result<Option<ModernHDTitleLoadSlots>> {
    let a = match a {
        Some(a)
            if a.dialogue_box.width == 310
                && a.dialogue_box.height == 86
                && sha256_hex(&a.dialogue_box.pixels)
                    == "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00"
                && sha256_hex(dac) == "b89f66fd8cec0b38e9e8697cc72ee63095f0a61b14edcee10d6ea908fff429ae" =>
        {
            a
        }
        _ => return Err(GameError::new("modern HD title slots: original panel or palette differs")),
    };
    let mut expected = color_palette_from_nrgba(&fdother::vga_palette_from_dac(dac)?);
    expected[0] = ColorRgba { a: 255, ..Default::default() };
    if expected != a.palette {
        return Err(GameError::new("modern HD title slots: palette differs"));
    }
    let shared = fdother::load_separated_item_panel_entries(separated_asset_path("ui"))?;
    let mut h = ModernHDTitleLoadSlots { cells: Vec::with_capacity(20), ..Default::default() };
    for i in 0..20 {
        let Some(cell) = shared.raw.get(&i) else {
            return Err(GameError::new("modern HD title slots: dialogue cells missing"));
        };
        h.cells.push(cell.clone());
    }
    let mut hash = Sha256::new();
    for i in [1usize, 2, 3, 4, 9, 10, 11, 12, 13] {
        let cell = &h.cells[i];
        hash.update([i as u8, cell.width as u8, cell.height as u8]);
        hash.update(&cell.pixels);
    }
    let digest: String = hash.finalize().iter().map(|x| format!("{x:02x}")).collect();
    if digest != "3171adb1c2c47ddb7a8e3057460efdefdf42fc6176f12083687272f55700e22f" {
        return Err(GameError::new("modern HD title slots: original dialogue cells differ"));
    }
    h.chrome = freeze_modern_hd_dialogue_chrome(&h.cells, &a.palette)?;
    h.chrome_identity = h.chrome.clone();
    let mut json_chrome = h.chrome.clone().unwrap_or_default();
    json_chrome.background = ColorRgba::default();
    json_chrome.gpu = None;
    json_chrome.gold_gpu = BTreeMap::new();
    h.json_chrome = Some(json_chrome.clone());
    h.json_identity = Some(json_chrome);
    h.seal = title_load_slots_hd_seal(Some(a), Some(&h));
    Ok(Some(h))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_support;

    /// Go：`modern_hd_title_interface_test.go:98` `TestModernHDTitleInterfaceRejectsSource` 的工廠部分
    /// （五種來源錯誤在算顏色之前就拒收）。前半段經 `setInterfaceMode`／`drawModernHDTitleMenuItem`（R6）。
    #[test]
    fn freeze_modern_hd_title_menu_item_rejects_factory_faults() {
        if test_support::input("FD2_ASSET_PACK", "surfaces").is_none() {
            return;
        }
        let pack = std::env::var("FD2_ASSET_PACK").unwrap();
        let source = fdother::load_separated_nested_single_frame(join_path(&[&pack, "surfaces"]), 7, 1).unwrap();
        let (dac, palette) = fdother::load_separated_fdother_palette(join_path(&[&pack, "palette"]), 8).unwrap();
        let palette = color_palette_from_nrgba(&palette);
        let original = masked_rgba(&source, &palette).unwrap();
        let mut bad_shape = source.clone();
        (bad_shape.width, bad_shape.height) = (7, 61);
        assert!(
            freeze_modern_hd_title_menu_item(&bad_shape, &dac, &palette, 0, 0, Some(&original)).is_err(),
            "same count wrong geometry admitted"
        );
        for field in ["pixels", "mask", "dac", "palette", "original"] {
            let (mut s, mut d, mut p) = (source.clone(), dac.clone(), palette.clone());
            let mut o = Some(&original);
            match field {
                "pixels" => s.indexed[0] ^= 1,
                "mask" => s.mask[0] ^= 255,
                "dac" => d[0] ^= 1,
                "palette" => p[0] = ColorRgba { r: 1, a: 255, ..Default::default() },
                _ => o = None,
            }
            let err = freeze_modern_hd_title_menu_item(&s, &d, &p, 0, 0, o).unwrap_err();
            let want = if field == "palette" {
                "modern HD title: palette differs"
            } else {
                "modern HD title: original item source differs"
            };
            assert_eq!(err.message(), want, "factory admitted {field}");
        }
    }
}
