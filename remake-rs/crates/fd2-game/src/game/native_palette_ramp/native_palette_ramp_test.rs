//! Go 來源：`remake/cmd/fd2/native_palette_ramp_test.go`（R4 B 組，期望值不改）。
//!
//! `:14`、`:53` 兩項以 C 組的 `completeNativeMapFrameFixture` 建素材庫，R5 階段 2 r5p 移植。

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;

/// Go：`native_palette_ramp_test.go:200` `TestNativePaletteRampUsesBaselineSubtraction`
#[test]
fn test_native_palette_ramp_uses_baseline_subtraction() {
    let baseline = vec![63u8; 256 * 3];
    let mut dac = baseline.clone();
    let err = fdother::apply_vga_palette_subtraction(&mut dac, &baseline, 0, 255, 63);
    assert!(
        err.is_ok() && native_dacis_black(&dac),
        "terminal subtraction err={err:?} black={}",
        native_dacis_black(&dac)
    );
}

/// B 組補充：`buildInclusivePaletteDeltas`（Go `native_palette_ramp.go:33`）含兩端點、可遞減，越界拒收。
#[test]
fn test_build_inclusive_palette_deltas_endpoints() {
    let out = build_inclusive_palette_deltas(0, 63).unwrap();
    assert!(out.len() == 64 && out[0] == 0 && out[63] == 63);
    let back = build_inclusive_palette_deltas(64, 0).unwrap();
    assert!(back.len() == 65 && back[0] == 64 && back[64] == 0);
    assert_eq!(build_inclusive_palette_deltas(5, 5).unwrap(), vec![5]);
    let err = build_inclusive_palette_deltas(0, 256).unwrap_err();
    assert_eq!(err.message(), "palette delta endpoint outside byte range");
}

/// Go：`native_palette_ramp_test.go:14` `TestNativePaletteRampExecutesExactOutAndInEndpoints`（R5 階段 2 r5p：C 組的
/// `completeNativeMapFrameFixture` 已在 crate 裡，連同下一項移植）。Go 的 `drawn = true` 模擬每一步都已呈現。
#[test]
fn test_native_palette_ramp_executes_exact_out_and_in_endpoints() {
    use std::sync::atomic::{AtomicI64, Ordering};
    let (mut assets, _, _) = complete_native_map_frame_fixture();
    for (i, v) in assets.palette_dac.iter_mut().enumerate() {
        *v = (i % 64) as u8;
    }
    let baseline = assets.palette_dac.clone();
    let mut g = Game {
        native_map_dac: baseline.clone(),
        native_map_assets: Some(assets),
        native_map_vga: vec![7; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
        ..Default::default()
    };
    let advanced = Arc::new(AtomicI64::new(0));
    let counter = advanced.clone();
    g.start_native_palette_ramp(
        0,
        63,
        2,
        Some(Box::new(move |_| {
            counter.fetch_add(1, Ordering::SeqCst);
        })),
    )
    .unwrap();
    let deltas = &g.native_palette_ramp.as_ref().unwrap().deltas;
    assert!(deltas.len() == 64 && deltas[0] == 0 && deltas[63] == 63, "fade-out deltas={deltas:?}");
    while g.native_palette_ramp.is_some() {
        g.native_palette_ramp.as_mut().unwrap().drawn = true;
        g.step_native_palette_ramp();
    }
    assert!(
        advanced.load(Ordering::SeqCst) == 1 && native_dacis_black(&g.native_map_dac),
        "fade-out completion advanced={} black={}",
        advanced.load(Ordering::SeqCst),
        native_dacis_black(&g.native_map_dac)
    );
    let counter = advanced.clone();
    g.start_native_palette_ramp(
        64,
        0,
        2,
        Some(Box::new(move |_| {
            counter.fetch_add(1, Ordering::SeqCst);
        })),
    )
    .unwrap();
    let deltas = &g.native_palette_ramp.as_ref().unwrap().deltas;
    assert!(deltas.len() == 65 && deltas[0] == 64 && deltas[64] == 0, "fade-in deltas={deltas:?}");
    while g.native_palette_ramp.is_some() {
        g.native_palette_ramp.as_mut().unwrap().drawn = true;
        g.step_native_palette_ramp();
    }
    assert!(
        advanced.load(Ordering::SeqCst) == 2 && g.native_map_dac == baseline,
        "fade-in completion advanced={} baseline={}",
        advanced.load(Ordering::SeqCst),
        g.native_map_dac == baseline
    );
}

/// Go：`native_palette_ramp_test.go:53` `TestNativePaletteRampDrawAcknowledgesIndexedFrame`。Go 讀
/// `palette[7].RGBA()` 的 16 位元值（`r >= 0xff00`），Rust 的調色盤已是 8 位元（`0xff`）。另加：整張畫面是索引 7
/// 的顏色（Rust 新增）。
#[test]
fn test_native_palette_ramp_draw_acknowledges_indexed_frame() {
    let (mut assets, _, _) = complete_native_map_frame_fixture();
    assets.palette_dac.fill(0);
    assets.palette_dac[7 * 3] = 63;
    let mut g = Game {
        native_map_dac: assets.palette_dac.clone(),
        native_map_assets: Some(assets),
        native_map_vga: vec![7; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
        ..Default::default()
    };
    g.start_native_palette_ramp(0, 0, 2, None).unwrap();
    let c = g.native_palette_ramp.as_ref().unwrap().palette[7];
    assert!(c.r == 0xff && c.g == 0 && c.b == 0 && c.a == 0xff, "indexed DAC palette={c:?}, want opaque red");
    let mut screen = Picture::new_image(640, 400);
    assert!(
        g.draw_native_palette_ramp(&mut screen) && g.native_palette_ramp.as_ref().unwrap().drawn,
        "indexed palette frame was not acknowledged by Draw"
    );
    assert!(screen.pix.chunks(4).all(|p| p == [0xff, 0, 0, 0xff]), "整張是索引 7 的紅色");
}

/// Rust 新增：`drawNativePaletteRamp`（`native_palette_ramp.go:164`）的逐像素 2 倍貼上與兩個失敗條件。
#[test]
fn draw_native_palette_ramp_scales_and_fails_closed() {
    let palette: ColorPalette = (0..256).map(|i| ColorRgba { r: i as u8, g: 3, b: (255 - i) as u8, a: 0xff }).collect();
    let vga: Vec<u8> = (0..320 * 200).map(|i| (i % 253) as u8).collect();
    let mut g = Game {
        native_palette_ramp: Some(NativePaletteRampJob {
            vga: vga.clone(),
            palette: palette.clone(),
            ..Default::default()
        }),
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    assert!(g.draw_native_palette_ramp(&mut screen));
    assert!(g.native_palette_ramp.as_ref().unwrap().drawn);
    for (x, y) in [(0usize, 0usize), (1, 1), (2, 0), (639, 399), (321, 17), (100, 250)] {
        let c = palette[vga[(y / 2) * 320 + x / 2] as usize];
        let at = (y * 640 + x) * 4;
        assert_eq!(&screen.pix[at..at + 4], &[c.r, c.g, c.b, c.a], "({x},{y})");
    }
    let mut short = Game {
        native_palette_ramp: Some(NativePaletteRampJob { vga: vec![0; 10], palette, ..Default::default() }),
        ..Default::default()
    };
    assert!(!short.draw_native_palette_ramp(&mut screen) && !short.native_palette_ramp.as_ref().unwrap().drawn);
    let mut no_palette =
        Game { native_palette_ramp: Some(NativePaletteRampJob { vga, ..Default::default() }), ..Default::default() };
    assert!(!no_palette.draw_native_palette_ramp(&mut screen));
    assert!(!Game::default().draw_native_palette_ramp(&mut screen));
}
