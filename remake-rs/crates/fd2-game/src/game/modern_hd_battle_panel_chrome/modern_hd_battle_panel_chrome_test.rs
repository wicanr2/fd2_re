//! C 組：HD 戰鬥面板外框與調色盤的 Rust 新增測試。Go 的 `modern_hd_battle_panel_test.go` 經
//! `renderPhysicalBattlePanel`（D 組）與真 TTF 繪字（`drawModernBattlePanelInk`），列在 PORTING.md
//! 「C 組 R5 測試清單」；這裡只驗 `freezeModernBattlePanelChrome`、`paint` 與 `modernHDBattlePalette`
//! 不經繪字的部分，期望值由 Go 原始碼的算式推得。

use super::*;

fn chrome_assets() -> battle::NativeItemPanelDataAssets {
    let mut assets = battle::NativeItemPanelDataAssets::default();
    let pixels = (0..149 * 42).map(|i| (i % 7) as u8 + 10).collect();
    assets.battle_panel = fdother::LMI1Entry { width: 149, height: 42, pixels };
    let cell = |v: u8| fdother::RawCell { width: 1, height: 5, pixels: vec![v; 5] };
    assets.raw_cells = Some(BTreeMap::from([(24, cell(40)), (27, cell(41)), (29, cell(42))]));
    assets
}

fn grey_palette() -> ColorPalette {
    (0..256).map(|i| ColorRgba { r: i as u8, g: i as u8, b: i as u8, a: 255 }).collect()
}

fn at(canvas: &Rgba, x: i64, y: i64) -> [u8; 4] {
    let o = (y * canvas.stride + x * 4) as usize;
    canvas.pix[o..o + 4].try_into().unwrap()
}

/// `freezeModernBattlePanelChrome` 凍結外框、bar 色與長度；長度照原版左帽＋本體＋右帽，夾到 103 格。
#[test]
fn freeze_chrome_bar_widths_follow_native_writer() {
    let assets = chrome_assets();
    let values = battle::NativeBattlePanelValues { level: 1, hp: 40, max_hp: 80, mp: -5, max_mp: 30 };
    let c = freeze_modern_battle_panel_chrome(&assets, values).expect("chrome");
    assert!(c.ready);
    assert_eq!((c.border, c.track, c.ink), (assets.battle_panel.pixels[150], 42, [40, 41]));
    // HP：101*40/80+1 = 51 → (51+1)*6；MP：101*(-5)/30+1 = -15 → 夾到 1 → (1+1)*6。
    assert_eq!(c.width, [52 * 6, 2 * 6]);
    let full = battle::NativeBattlePanelValues { hp: 999, max_hp: 1, mp: 0, max_mp: 30, ..values };
    let c = freeze_modern_battle_panel_chrome(&assets, full).expect("chrome");
    assert_eq!(c.width, [103 * 6, 0], "超出夾到 103 格；值為零時不畫");
}

/// 原始格尺寸不對或面板尺寸不對都拒收（Go `TestModernBattlePanelChromeFrozenPaletteAndReject` 的最後一段）。
#[test]
fn freeze_chrome_rejects_malformed_sources() {
    let mut assets = chrome_assets();
    assets.raw_cells.as_mut().unwrap().get_mut(&27).unwrap().height = 4;
    let err = freeze_modern_battle_panel_chrome(&assets, battle::NativeBattlePanelValues::default()).unwrap_err();
    assert_eq!(err.message(), "modern HD panel: chrome raw cell 27 unavailable");
    let mut assets = chrome_assets();
    assets.battle_panel.width = 148;
    let err = freeze_modern_battle_panel_chrome(&assets, battle::NativeBattlePanelValues::default()).unwrap_err();
    assert_eq!(err.message(), "modern HD panel: chrome source unavailable");
}

/// `paint` 保留原版外圈斜面像素、以凍結的索引取目前調色盤，並拒收未凍結的外框或錯誤尺寸的畫布。
#[test]
fn paint_keeps_native_bevel_and_bar_colours() {
    let assets = chrome_assets();
    let values = battle::NativeBattlePanelValues { level: 1, hp: 80, max_hp: 80, mp: 30, max_mp: 30 };
    let c = freeze_modern_battle_panel_chrome(&assets, values).expect("chrome");
    let palette = grey_palette();
    let mut canvas = new_rgba_canvas(894, 252);
    c.paint(&mut canvas, &palette, 7).expect("paint");
    for (x, y) in [(0, 0), (148, 0), (0, 41), (2, 20), (147, 20)] {
        let index = assets.battle_panel.pixels[(y * 149 + x) as usize];
        assert_eq!(at(&canvas, x * 6 + 3, y * 6 + 3), [index, index, index, 255], "bevel {x},{y}");
    }
    assert_eq!(at(&canvas, 60, 60), [7, 7, 7, 255], "面板內部是底色");
    assert_eq!(at(&canvas, 600, 120), [c.border, c.border, c.border, 255], "分隔線");
    // HP bar 本體在 y=22*6..28*6，第 3 列（row=3）的上緣漸亮：v + (255-v)*(6-3)/15。
    let ink = c.ink[0];
    let light = (i64::from(ink) + (255 - i64::from(ink)) * 3 / 15) as u8;
    assert_eq!(at(&canvas, 21 * 6 + 10, 22 * 6 + 3), [light, light, light, 255]);
    // 外框線寬 2：bar 矩形的左上角是 border。
    assert_eq!(at(&canvas, 21 * 6, 22 * 6), [c.border, c.border, c.border, 255]);
    let mut wrong = new_rgba_canvas(893, 252);
    assert!(c.paint(&mut wrong, &palette, 7).is_err());
    assert!(ModernBattlePanelChrome::default().paint(&mut canvas, &palette, 7).is_err());
}

/// `modernHDBattlePalette` 要 256 個不透明色，寫成 RGBA 位元組；`nativePaletteColor` 越界回備用色。
#[test]
fn battle_palette_requires_256_opaque_colours() {
    let palette = grey_palette();
    let raw = modern_hd_battle_palette(&palette).expect("palette");
    assert_eq!((raw.len(), &raw[4 * 9..4 * 10]), (1024, &[9u8, 9, 9, 255][..]));
    assert_eq!(
        modern_hd_battle_palette(&palette[..255].to_vec()).unwrap_err().message(),
        "modern HD battle: palette unavailable"
    );
    let mut translucent = palette.clone();
    translucent[3].a = 254;
    assert_eq!(
        modern_hd_battle_palette(&translucent).unwrap_err().message(),
        "modern HD battle: native palette must be opaque"
    );
    let fallback = ColorRgba { r: 1, g: 2, b: 3, a: 4 };
    assert_eq!(native_palette_color(&palette[..10].to_vec(), 0xcd, fallback), fallback);
    assert_eq!(native_palette_color(&palette, 0xcd, fallback), palette[0xcd]);
}
