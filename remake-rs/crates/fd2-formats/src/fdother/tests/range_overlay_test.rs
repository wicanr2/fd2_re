//! 移植自 `range_overlay_test.go`。

use super::original;
use crate::fdicon::{Bank, NATIVE_SIZE, Sprite};
use crate::fdother::range_overlay::NATIVE_RANGE_OVERLAY_STRIDE;
use crate::fdother::*;

fn p(x: i64, y: i64, descriptor: i64) -> NativeRangeOverlayPlacement {
    NativeRangeOverlayPlacement { x, y, descriptor }
}

#[test]
fn native_range_overlay_placements_match_122dc() {
    let tests: [(i64, Vec<NativeRangeOverlayPlacement>); 5] = [
        (1, vec![p(10, 20, 0)]),
        (2, vec![p(10, 20, 1)]),
        (3, vec![p(10, 20, 14), p(10, 19, 2), p(9, 20, 3), p(11, 20, 4), p(10, 21, 5)]),
        (
            4,
            vec![
                p(10, 20, 1),
                p(10, 18, 2),
                p(8, 20, 3),
                p(12, 20, 4),
                p(10, 22, 5),
                p(9, 19, 6),
                p(11, 19, 7),
                p(9, 21, 8),
                p(11, 21, 9),
                p(10, 19, 10),
                p(9, 20, 11),
                p(11, 20, 12),
                p(10, 21, 13),
            ],
        ),
        (
            5,
            vec![
                p(10, 20, 1),
                p(10, 17, 2),
                p(7, 20, 3),
                p(13, 20, 4),
                p(10, 23, 5),
                p(9, 18, 6),
                p(8, 19, 6),
                p(11, 18, 7),
                p(8, 22, 7),
                p(11, 22, 8),
                p(11, 18, 8),
                p(12, 22, 9),
                p(11, 22, 9),
                p(10, 18, 10),
                p(8, 20, 11),
                p(12, 20, 12),
                p(10, 22, 13),
                p(9, 19, 15),
                p(11, 19, 16),
                p(9, 21, 17),
                p(11, 21, 18),
            ],
        ),
    ];
    for (mode, want) in tests {
        assert_eq!(native_range_overlay_placements(mode, 10, 20).unwrap(), want, "mode {mode}");
    }
}

#[test]
fn native_range_overlay_rejects_mode6_and_invalid() {
    for mode in [0, 6, 7] {
        assert!(native_range_overlay_placements(mode, 0, 0).is_err(), "mode {mode} accepted");
    }
}

#[test]
fn native_range_overlay_mode6_byte_address_formula() {
    assert_eq!(native_range_overlay_mode6_byte_address(17, 3, 2).unwrap(), 4 * (3 + 2 * 17) + 7);
    assert!(native_range_overlay_mode6_byte_address(0, 3, 2).is_err(), "zero width accepted");
}

#[test]
fn clear_native_range_overlay_mode6_field_byte_clears_fourth_byte() {
    let mut field = vec![0u8; 4 + 3 * 2 * 4];
    field[0..2].copy_from_slice(&3u16.to_le_bytes());
    field[2..4].copy_from_slice(&2u16.to_le_bytes());
    for i in 0..6 {
        field[4 + 4 * i + 3] = 0xff;
    }
    clear_native_range_overlay_mode6_field_byte(&mut field, 1, 1).unwrap();
    for i in 0..6 {
        let want = if i == 4 { 0 } else { 0xff }; // x + y*width = 1 + 1*3
        assert_eq!(field[4 + 4 * i + 3], want, "cell {i} high byte");
    }
    let before = field.clone();
    assert!(clear_native_range_overlay_mode6_field_byte(&mut field, 3, 1).is_err(), "out-of-range coordinate accepted");
    assert_eq!(field, before, "invalid field mutation changed data");
}

#[test]
fn native_range_overlay_bank_and_blit() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let bank = decode_native_range_overlay_bank(&path).unwrap();
    assert_eq!(bank.sprites.len(), 20, "FDOTHER#1 descriptor count");
    for (i, sprite) in bank.sprites.iter().enumerate() {
        assert_eq!(sprite.pixels.len() as i64, NATIVE_SIZE * NATIVE_SIZE, "FDOTHER#1 descriptor {i}");
    }
    let mut dst = vec![0u8; (NATIVE_RANGE_OVERLAY_STRIDE * 300) as usize];
    blit_native_range_overlay(&bank, &mut dst, 10, 20, 13, 8, 4, 10, 20).unwrap();
    let before = dst.clone();
    assert!(
        blit_native_range_overlay(&bank, &mut dst, 10, 20, 13, 8, 6, 10, 20).is_err(),
        "mode 6 was accepted as drawable"
    );
    assert_eq!(dst, before, "rejected mode mutated framebuffer");
}

#[test]
fn blit_native_range_overlay_preflights_destination() {
    let blank = Sprite { pixels: vec![0; 24 * 24], mask: vec![0; 24 * 24], remap_mask: vec![0; 24 * 24] };
    let bank = Bank { sprites: vec![blank; 20] };
    let mut dst = vec![0u8; (NATIVE_RANGE_OVERLAY_STRIDE * 80) as usize];
    let before = dst.clone();
    assert!(
        blit_native_range_overlay(&bank, &mut dst, 0, 0, 13, 8, 1, 0, 0).is_err(),
        "undersized framebuffer accepted"
    );
    assert_eq!(dst, before, "failed preflight mutated framebuffer");
}

#[test]
fn blit_native_range_overlay_mode_zero_is_exact_no_op() {
    let bank = Bank { sprites: vec![Sprite::default(); 20] };
    let mut dst: Vec<u8> = (0..NATIVE_RANGE_OVERLAY_STRIDE * 200).map(|i| i as u8).collect();
    let before = dst.clone();
    blit_native_range_overlay(&bank, &mut dst, 0, 0, 13, 8, 0, 0, 0).unwrap();
    assert_eq!(dst, before, "raw range mode zero mutated the frame");
}
