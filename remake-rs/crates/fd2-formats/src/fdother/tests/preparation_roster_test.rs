//! 移植自 `preparation_roster_test.go`。

use super::original;
use crate::fdicon::{Bank, NATIVE_SIZE, Sprite};
use crate::fdother::preparation_roster::{
    blit_native_preparation_roster_layers, blit_native_preparation_roster_units, blit_native_preparation_two_digits,
};
use crate::fdother::*;

fn solid_preparation_sprite(pixel: u8) -> Sprite {
    let n = (NATIVE_SIZE * NATIVE_SIZE) as usize;
    Sprite { pixels: vec![pixel; n], mask: vec![1; n], remap_mask: vec![0; n] }
}

fn preparation_unit_bank(keys: &[usize]) -> Bank {
    let max_key = keys.iter().copied().max().unwrap_or(0);
    let mut bank = Bank { sprites: vec![Sprite::default(); (max_key + 1) * 12] };
    for &key in keys {
        for cycle in 0..3 {
            bank.sprites[key * 12 + cycle] = solid_preparation_sprite((key * 8 + cycle + 1) as u8);
        }
    }
    bank
}

fn cursor_bank() -> Bank {
    Bank { sprites: vec![solid_preparation_sprite(0x55)] }
}

// Go: TestNativePreparationRosterPositions
#[test]
fn native_preparation_roster_positions() {
    let tests = [
        (0, false, false, 23, 100),
        (9, false, false, 275, 100),
        (10, false, false, 23, 130),
        (29, false, false, 275, 160),
        (10, true, false, 23, 133),
        (10, false, true, 23, 134),
    ];
    for (index, selected, cursor, x, y) in tests {
        let got = native_preparation_roster_position(index, selected, cursor);
        assert_eq!(got.ok(), Some((x, y)), "index {index} selected={selected} cursor={cursor}");
    }
}

// Go: TestMoveNativePreparationRosterCursorUsesTenColumnGrid
#[test]
fn move_native_preparation_roster_cursor_uses_ten_column_grid() {
    let tests = [
        (0, 16, 0x4b, 0),
        (0, 16, 0x4d, 1),
        (15, 16, 0x4d, 15),
        (9, 16, 0x48, 9),
        (10, 16, 0x48, 0),
        (5, 16, 0x50, 15),
        (6, 16, 0x50, 6),
    ];
    for (cursor, count, scan, want) in tests {
        let got = move_native_preparation_roster_cursor(cursor, count, scan);
        assert_eq!(got.ok(), Some(want), "cursor={cursor} count={count} scan={scan:#x}");
    }
}

// Go: TestBlitNativePreparationRosterUsesNativeSelectedAndUnselectedPaths
#[test]
fn blit_native_preparation_roster_uses_native_selected_and_unselected_paths() {
    let mut frame = vec![0u8; 320 * 200];
    let units = preparation_unit_bank(&[2, 3]);
    blit_native_preparation_roster(&mut frame, 320, &units, &cursor_bank(), &[2, 3], &[false, true], 1, 2).unwrap();
    assert_eq!(frame[100 * 320 + 23], ((2 * 8 + 3) as u8 & 7) + 0x18, "unselected palette-band pixel");
    assert_eq!(frame[103 * 320 + 51], (3 * 8 + 3) as u8, "selected raised pixel");
    // 0x31e80 先畫游標。已選單位蓋住它的中段，游標最後一列在抬高三像素的單位下方仍可見。
    assert_eq!(frame[127 * 320 + 51], 0x55, "cursor pixel");
    assert_eq!(frame[100 * 320 + 51], 0, "selected sprite was not raised by three pixels");
}

// Go: TestBlitNativePreparationRosterFailsAtomically
#[test]
fn blit_native_preparation_roster_fails_atomically() {
    let mut frame = vec![0x44u8; 320 * 200];
    let before = frame.clone();
    let mut units = preparation_unit_bank(&[1]);
    units.sprites[12] = Sprite::default();
    let r = blit_native_preparation_roster(&mut frame, 320, &units, &cursor_bank(), &[1], &[false], 0, 0);
    assert!(r.is_err(), "malformed unit sprite unexpectedly rendered");
    assert_eq!(frame, before, "failed preparation render changed destination");
}

// Go: TestDecodeAndComposeNativePreparationAssetsFromPlayerArchive
#[test]
fn decode_and_compose_native_preparation_assets_from_player_archive() {
    let Some(fdother_path) = original("FDOTHER.DAT") else { return };
    let Some(fdicon_path) = original("FDICON.B24") else { return };
    let assets = decode_native_preparation_assets_archive(&fdother_path, &fdicon_path).unwrap();
    assert!(
        assets.upper_right.width == 223
            && assets.upper_right.height == 86
            && assets.lower.width == 310
            && assets.lower.height == 99
            && assets.upper_left.width == 86
            && assets.upper_left.height == 86,
        "native preparation geometry changed"
    );
    let frame = compose_native_preparation_frame(&assets, &[0, 9, 4], &[false, true, false], 1, 0, 15).unwrap();
    assert_eq!(frame.len(), 320 * 200, "native preparation frame length");
    assert!(frame.iter().any(|&b| b != 0), "native preparation frame is empty");
    // 0x31ea9 畫 quota（15），0x31edb 畫 quota-selected（14）。兩個數字格都和原始字形項目比較，避免
    // 第一組再次被已選人數悄悄取代。
    let mut want = frame.clone();
    blit_native_preparation_two_digits(&assets.quota_digits, &mut want, 61, 35, 15).unwrap();
    blit_native_preparation_two_digits(&assets.remaining_digits, &mut want, 61, 73, 14).unwrap();
    for (bx, by, bw, bh) in [(61, 35, 12, 8), (61, 73, 12, 8)] {
        for y in by..by + bh {
            let start = y * 320 + bx;
            assert_eq!(frame[start..start + bw], want[start..start + bw], "native preparation count differs at box");
        }
    }
    let mut wrong = frame.clone();
    blit_native_preparation_two_digits(&assets.quota_digits, &mut wrong, 61, 73, 14).unwrap();
    assert_ne!(frame[73 * 320..81 * 320], wrong[73 * 320..81 * 320], "整備剩餘人數錯用 quota 字形時畫面未改變");
}

fn alternating_roster() -> (Vec<i64>, Vec<bool>) {
    ((0..30).map(|i| 2 + i % 2).collect(), (0..30).map(|i| i % 2 == 0).collect())
}

// Go: TestNativePreparationRosterLayerReconstructsOriginal
#[test]
fn native_preparation_roster_layer_reconstructs_original() {
    let mut units = preparation_unit_bank(&[2, 3]);
    let cursor = cursor_bank();
    let (keys, selected) = alternating_roster();
    for cycle in 0..3 {
        let mut full = vec![0x44u8; 64000];
        let mut layer = full.clone();
        blit_native_preparation_roster(&mut full, 320, &units, &cursor, &keys, &selected, 29, cycle).unwrap();
        blit_native_preparation_roster_units(&mut layer, 320, &units, &cursor, &keys, &selected, 29, cycle, false)
            .unwrap();
        for (i, &key) in keys.iter().enumerate() {
            let sprite = units.sprite_for(key, 0, cycle).unwrap();
            let (x, y) = native_preparation_roster_position(i as i64, selected[i], false).unwrap();
            if selected[i] {
                sprite.blit_at(&mut layer, 320, x, y).unwrap();
            } else {
                sprite.blit_palette_band(&mut layer, 320, x, y).unwrap();
            }
        }
        assert_eq!(full, layer, "original unit order or background changed {cycle}");
    }
    units.sprites[2 * 12] = Sprite::default();
    let mut untouched = vec![0x44u8; 64000];
    let r = blit_native_preparation_roster_units(&mut untouched, 320, &units, &cursor, &[2], &[true], 0, 0, false);
    assert!(r.is_err() && untouched.iter().all(|&b| b == 0x44), "omitted malformed unit bypassed native preflight");
    let mut invalid = vec![0x44u8; 64000];
    let r = blit_native_preparation_roster_units(&mut invalid, 320, &units, &cursor, &[999], &[true], 0, 0, false);
    assert!(r.is_err() && invalid.iter().all(|&b| b == 0x44), "hidden units bypassed native preflight");
}

// Go: TestNativePreparationCursorLayersReconstructOriginal
#[test]
fn native_preparation_cursor_layers_reconstruct_original() {
    let units = preparation_unit_bank(&[2, 3]);
    let mut cursor = cursor_bank();
    let (keys, selected) = alternating_roster();
    for cycle in 0..3 {
        for slot in [0, 9, 10, 19, 20, 29] {
            for draw_units in [false, true] {
                let mut full = vec![0x44u8; 64000];
                blit_native_preparation_roster(&mut full, 320, &units, &cursor, &keys, &selected, slot, cycle).unwrap();
                let mut layer = vec![0x44u8; 64000];
                blit_native_preparation_roster_layers(
                    &mut layer, 320, &units, &cursor, &keys, &selected, slot, cycle, draw_units, false,
                )
                .unwrap();
                // 先游標後單位重播即重建原生順序，包括不透明來源值零與未選單位的色帶運算。
                blit_native_preparation_roster(&mut layer, 320, &units, &cursor, &keys, &selected, slot, cycle)
                    .unwrap();
                assert_eq!(layer, full, "cursor layer changed native order {cycle} {slot} {draw_units}");
            }
        }
    }
    cursor.sprites[0].mask = Vec::new();
    let mut layer = vec![0x44u8; 64000];
    let r =
        blit_native_preparation_roster_layers(&mut layer, 320, &units, &cursor, &keys, &selected, 0, 0, false, false);
    assert!(r.is_err() && layer.iter().all(|&b| b == 0x44), "hidden cursor bypassed atomic source preflight");
}
