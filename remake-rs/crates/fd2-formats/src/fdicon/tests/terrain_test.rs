//! 移植自 `terrain_test.go`。

use crate::fdicon::*;

const N: usize = NATIVE_SIZE as usize;

fn blank_sprite() -> Sprite {
    Sprite { pixels: vec![0; N * N], mask: vec![0; N * N], remap_mask: vec![0; N * N] }
}

#[test]
fn native_terrain_cursor_info_for_cell_matches_12e38() {
    let mut controls = vec![0u8; 0x12 * 4];
    controls[0x11 * 4..0x11 * 4 + 4].copy_from_slice(&[0x80, 0x12, 0x34, 0x56]);
    let got = native_terrain_cursor_info_for_cell(0xfc11, 0xffe5, &controls).unwrap();
    assert!(got.tile == 0x11 && got.event_low == 5 && got.control == [0x80, 0x12, 0x34, 0x56], "info={got:?}");
    assert!(native_terrain_cursor_info_for_cell(0x12, 0, &controls).is_err(), "out-of-range control accepted");
}

#[test]
fn native_map_hud_unit_frame_index_matches_1ae4d() {
    for (group, state, want) in [(0, 0, 0), (2, 1, 25), (2, 2, 26), (2, 3, 25)] {
        assert_eq!(native_map_hud_unit_frame_index(group, state), Ok(want), "({group},{state})");
    }
    assert!(native_map_hud_unit_frame_index(0, 4).is_err(), "out-of-range raw state was accepted");
}

#[test]
fn native_map_hud_layout_for_matches_1acf3() {
    let l = native_map_hud_layout_for(1, NATIVE_MAP_STRIDE).unwrap();
    assert_eq!(
        l,
        NativeMapHUDLayout {
            frame: NATIVE_MAP_STRIDE * 157 + 1,
            terrain: NATIVE_MAP_STRIDE * 162 + 7,
            ap: NATIVE_MAP_STRIDE * 165 + 0x2c,
            dp: NATIVE_MAP_STRIDE * 176 + 0x2c,
            unit: NATIVE_MAP_STRIDE * 162 + 7,
            hp: NATIVE_MAP_STRIDE * 178 + 10,
        }
    );
    assert!(native_map_hud_layout_for(252, NATIVE_MAP_STRIDE).is_err(), "off-screen native HUD frame was accepted");
    assert!(native_map_hud_layout_for(1, 320).is_err(), "non-native stride was accepted");
}

#[test]
fn native_command_background_selector_matches_2b5e1() {
    let targets = [
        NativeCommandBackgroundTarget { gate: true, control: [0, 0, 0x11, 0] },
        NativeCommandBackgroundTarget { gate: false, control: [0, 0, 0x22, 0] },
        NativeCommandBackgroundTarget { gate: true, control: [0, 0, 0x33, 0] },
    ];
    // 原版由後往前：#2 換不掉非 0 的 0x07，#1 因 gate 為 false 而取代，#0 再保留該結果。
    assert_eq!(native_command_background_selector(0x07, &targets), 0x22);
    // 初值為 0 時，最後一個目標會先填入選擇器，之後通過閘門的目標保留它。
    let seeded = native_command_background_selector(
        0,
        &[
            NativeCommandBackgroundTarget { gate: true, control: [0, 0, 0x11, 0] },
            NativeCommandBackgroundTarget { gate: true, control: [0, 0, 0x33, 0] },
        ],
    );
    assert_eq!(seeded, 0x33, "zero-initial selector");
}

#[test]
fn native_foreground_redraw_eligible_matches_129ec_gates() {
    for (name, inactive, unit7, race, class, want) in [
        ("inactive always skips", 1u8, 0x1cu8, 0u8, 0u8, false),
        ("ordinary active", 0, 0, 0, 0, true),
        ("unit plus 7 1c overrides raw gate", 0, 0x1c, 4, 0x13, true),
        ("class 13 skips", 0, 0, 0, 0x13, false),
        ("race 4 skips", 0, 0, 4, 0, false),
        ("race 5 skips", 0, 0, 5, 0, false),
    ] {
        assert_eq!(native_foreground_redraw_eligible(inactive != 0, unit7, race, class), want, "{name}");
    }
}

#[test]
fn native_foreground_redraw_cells_matches_129ec() {
    let c = |x, y| NativeCellCoordinate { x, y };
    for (name, pose, movement_offset, want) in [
        ("stationary", 0u8, 0, vec![c(10, 20), c(10, 19)]),
        ("down", 0, 1, vec![c(10, 20), c(10, 19), c(10, 21)]),
        ("left", 1, -1, vec![c(10, 20), c(10, 19), c(9, 20)]),
        ("up", 2, 1, vec![c(10, 20), c(10, 19), c(10, 18)]),
        ("right", 3, 1, vec![c(10, 20), c(10, 19), c(11, 20)]),
        ("native default right", 0xff, 1, vec![c(10, 20), c(10, 19), c(11, 20)]),
        ("unclipped", 1, 1, vec![c(0, 0), c(0, -1), c(-1, 0)]),
    ] {
        let (x, y) = if name == "unclipped" { (0, 0) } else { (10, 20) };
        let (got, n) = native_foreground_redraw_cells(x, y, pose, movement_offset);
        assert_eq!(n, want.len(), "{name}: count");
        assert_eq!(&got[..n], &want[..], "{name}");
    }
}

#[test]
fn native_terrain_frame_index_matches_11eee() {
    for (flags, flip, cycle, want) in [
        (0x00, 1, 9, 100),
        (0x04, 1, 9, 101),
        (0x10, 1, 9, 104),
        (0x08, 1, 9, 102),
        (0x1c, 1, 9, 102),
        (0x10, 0, -3, 99),
    ] {
        assert_eq!(native_terrain_frame_index(100, flags, flip, cycle), Ok(want), "flags={flags:#x}");
    }
    assert!(native_terrain_frame_index(0x400, 0, 0, 0).is_err(), "out-of-range tile accepted");
    assert!(native_terrain_frame_index(0, 0, 2, 0).is_err(), "invalid flip accepted");
}

#[test]
fn native_foreground_frame_index_matches_12ac6() {
    for (flags, flip, want, present) in
        [(0x00, 0, 0, false), (0x80, 0, 101, true), (0x88, 1, 103, true), (0x88, 0, 101, true)]
    {
        assert_eq!(native_foreground_frame_index(100, flags, flip), Ok((want, present)), "flags={flags:#x}");
    }
    assert!(native_foreground_frame_index(0x400, 0x80, 0).is_err(), "out-of-range tile accepted");
}

#[test]
fn blit_native_terrain_cell_selects_raw_or_lut_branch() {
    let mut sprites = vec![blank_sprite(); 12];
    sprites[0].pixels[0] = 7;
    sprites[0].mask[0] = 1;
    sprites[1].pixels[0] = 9;
    sprites[1].mask[0] = 1;
    sprites[1].remap_mask[1] = 1;
    let b = Bank { sprites };
    let mut dst = vec![1u8; N * N];
    b.blit_native_terrain_cell(&mut dst, NATIVE_SIZE, 0, 0, 0, 0, 0xff, 0, 0, &[]).unwrap();
    assert_eq!(dst[0], 7, "raw branch");
    let lut: Vec<u8> = (0..256).map(|i| (i + 0x10) as u8).collect();
    dst.fill(1);
    b.blit_native_terrain_cell(&mut dst, NATIVE_SIZE, 0, 0, 0, 0x04, 0, 1, 0, &lut).unwrap();
    assert!(dst[0] == 0x19 && dst[1] == 0x11, "LUT branch={:#x},{:#x}", dst[0], dst[1]);
}

#[test]
fn blit_native_terrain_region_matches_11eee_cell_order() {
    let mut pixels = vec![0u8; N * N];
    let mut mask = vec![0u8; N * N];
    pixels[0] = 7;
    mask[0] = 1;
    let b = Bank { sprites: vec![Sprite { pixels, mask, remap_mask: vec![0; N * N] }; 12] };
    let lut: Vec<u8> = (0..256).map(|i| (i + 0x10) as u8).collect();
    let mut dst = vec![0u8; N * N * 2];
    let cells = [NativeTerrainCell { tile: 0, blit_mode: 0xff }, NativeTerrainCell { tile: 0, blit_mode: 0 }];
    b.blit_native_terrain_region(&mut dst, NATIVE_SIZE * 2, 0, 0, 2, &cells, &[0, 0, 0, 0], 0, 0, 2, 1, 0, 0, &lut)
        .unwrap();
    assert!(dst[0] == 7 && dst[N] == 0x17, "region cells={:#x},{:#x}", dst[0], dst[N]);
    let result = b.blit_native_terrain_region(
        &mut dst,
        NATIVE_SIZE,
        0,
        0,
        1,
        &[NativeTerrainCell::default()],
        &[],
        0,
        0,
        1,
        1,
        0,
        0,
        &lut,
    );
    assert!(result.is_err(), "short control table accepted");
}

#[test]
fn blit_native_foreground_cell_matches_12ac6() {
    let mut b = Bank { sprites: vec![blank_sprite(); 12] };
    b.sprites[1].pixels[0] = 7;
    b.sprites[1].mask[0] = 1;
    b.sprites[1].remap_mask[1] = 1;
    let mut dst = vec![1u8; N * N];
    let lut: Vec<u8> = (0..256).map(|i| (i + 0x10) as u8).collect();
    b.blit_native_foreground_cell(&mut dst, NATIVE_SIZE, 0, 0, 0, 0x80, 0, 0, &lut).unwrap();
    assert!(dst[0] == 0x17 && dst[1] == 1, "foreground={:#x},{:#x}", dst[0], dst[1]);
}
