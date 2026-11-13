//! 移植自 `foreground_layer_test.go`。

use super::unit_layer_test::native_layer_bank;
use crate::fdicon::*;

#[test]
fn blit_native_foreground_layer_matches_12ac6_schedule() {
    let mut bank = native_layer_bank();
    // index 1/2 分別由前景 tile 0/1 選中。
    bank.sprites[1].pixels[0] = 0x2a;
    bank.sprites[2].pixels[0] = 0x2b;
    let mut dst = vec![0u8; (NATIVE_MAP_STRIDE * 160) as usize];
    let cells = [NativeTerrainCell { tile: 0, blit_mode: 0xff }, NativeTerrainCell { tile: 1, blit_mode: 0xff }];
    let controls = [0x80, 0, 0, 0, 0x80, 0, 0, 0];
    let unit = NativeForegroundLayerEntry { x: 0, y: 1, pose: 0, ..Default::default() };
    bank.blit_native_foreground_layer(&mut dst, NATIVE_MAP_STRIDE, &[unit], 1, &cells, &controls, 0, 0, 12, 7, 0, &[])
        .unwrap();
    // 原版順序是 (x,y) 再 (x,y-1)；兩個座標各自選描述子，不沿用單位的地圖選擇器。
    let upper = 0x8088;
    assert_eq!(dst[upper], 0x2a, "upper foreground pixel");
    let current = 0x8088 + (NATIVE_SIZE * NATIVE_MAP_STRIDE) as usize;
    assert_eq!(dst[current], 0x2b, "current foreground pixel");
}

#[test]
fn blit_native_foreground_layer_fails_before_partial_frame() {
    let mut dst = vec![0u8; (NATIVE_MAP_STRIDE * 160) as usize];
    let cells = [NativeTerrainCell { tile: 0, blit_mode: 0 }];
    let controls = [0x80, 0, 0, 0];
    let unit = NativeForegroundLayerEntry::default();
    let result = native_layer_bank().blit_native_foreground_layer(
        &mut dst,
        NATIVE_MAP_STRIDE,
        &[unit],
        1,
        &cells,
        &controls,
        0,
        0,
        12,
        7,
        0,
        &[],
    );
    assert!(result.is_err(), "missing LUT was accepted");
    assert_eq!(dst[0x8088], 0, "failed foreground layer partially modified destination");
}
