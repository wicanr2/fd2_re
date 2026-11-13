//! 移植自 `palette_test.go`。

use crate::fdother::*;

#[test]
fn parse_vga_palette_preserves_six_bit_scale_as_opaque_vga() {
    let mut raw = vec![0u8; 256 * 3];
    raw[3..6].copy_from_slice(&[63, 32, 1]);
    let palette = parse_vga_palette(&raw).unwrap();
    // Go 另外檢查 palette[0] 不是 nil 介面；Rust 的 Nrgba 恆存在。
    assert_eq!(palette[0].rgba().3, 0xffff, "VGA index-zero alpha");
    assert_eq!(palette[1].rgba(), (0xffff, 0x8282, 0x0404, 0xffff), "palette[1]");
}

#[test]
fn raw_cell_paletted_preserves_zero_index() {
    let palette = parse_vga_palette(&[0u8; 256 * 3]).unwrap();
    let im = RawCell { width: 2, height: 1, pixels: vec![0, 7] }.paletted(&palette).unwrap();
    assert_eq!(im.color_index_at(0, 0), 0, "zero index");
    assert_eq!(im.color_index_at(1, 0), 7, "opaque index");
    let (raw_alpha, cell_alpha) = (palette[0].rgba().3, im.palette[0].rgba().3);
    assert!(
        raw_alpha == 0xffff && cell_alpha == 0,
        "palette alpha leaked across codec boundary: VGA={raw_alpha:#x} cell={cell_alpha:#x}"
    );
}

#[test]
fn apply_vga_palette_delta_uses_immutable_baseline_inclusive_and_atomic() {
    let mut baseline = vec![0u8; 256 * 3];
    baseline[3..6].copy_from_slice(&[1, 62, 63]);
    let mut dac = vec![9u8; 256 * 3];
    apply_vga_palette_delta(&mut dac, &baseline, 1, 1, 4).unwrap();
    assert_eq!(&dac[3..9], &[5, 63, 63, 9, 9, 9], "first delta");
    apply_vga_palette_delta(&mut dac, &baseline, 1, 1, 2).unwrap();
    assert_eq!(&dac[3..9], &[3, 63, 63, 9, 9, 9], "baseline-derived second delta");
    let before = dac.clone();
    assert!(apply_vga_palette_delta(&mut dac, &baseline, 2, 1, 1).is_err(), "invalid range accepted");
    assert_eq!(dac, before, "rejected delta mutated DAC");
}

#[test]
fn apply_vga_palette_delta_preserves_byte_sized_white_flash() {
    let mut baseline = vec![0u8; 256 * 3];
    baseline[..3].copy_from_slice(&[0, 1, 63]);
    let mut dac = vec![7u8; 256 * 3];
    apply_vga_palette_delta(&mut dac, &baseline, 0, 255, 255).unwrap();
    assert!(dac.iter().all(|&c| c == 63), "white flash");
    let before = dac.clone();
    assert!(
        apply_vga_palette_delta(&mut dac, &baseline, 0, 255, 256).is_err(),
        "delta wider than the recovered byte argument was accepted"
    );
    assert_eq!(dac, before, "rejected byte-sized delta boundary mutated DAC");
}

#[test]
fn apply_vga_palette_subtraction_uses_immutable_baseline_inclusive_and_atomic() {
    let mut baseline = vec![0u8; 256 * 3];
    baseline[3..6].copy_from_slice(&[1, 62, 63]);
    let mut dac = vec![9u8; 256 * 3];
    apply_vga_palette_subtraction(&mut dac, &baseline, 1, 1, 4).unwrap();
    assert_eq!(&dac[3..9], &[0, 58, 59, 9, 9, 9], "first subtraction");
    apply_vga_palette_subtraction(&mut dac, &baseline, 1, 1, 2).unwrap();
    assert_eq!(&dac[3..9], &[0, 60, 61, 9, 9, 9], "baseline-derived second subtraction");
    let before = dac.clone();
    assert!(apply_vga_palette_subtraction(&mut dac, &baseline, 2, 1, 1).is_err(), "invalid subtraction range accepted");
    assert_eq!(dac, before, "rejected subtraction mutated DAC");
}
