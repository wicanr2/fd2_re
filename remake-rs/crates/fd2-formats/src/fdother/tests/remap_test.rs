//! 移植自 `remap_test.go`。

use crate::fdother::*;

#[test]
fn apply_radial_lut_remap_uses_strict_radius_and_in_place_lut() {
    let mut dst: Vec<u8> = (0..100).map(|i| (i % 10) as u8).collect();
    let lut: Vec<u8> = (0..256).map(|i| ((i + 20) & 0xff) as u8).collect();
    let spec = RadialLUTRemap { center_x: 5, center_y: 5, radius: 4, scale: 10, start_y: 0, end_y: 10, clip_width: 10 };
    apply_radial_lut_remap(&mut dst, 10, &lut, spec).unwrap();
    // dy=4 是原生的嚴格邊界，不映射。
    assert_eq!(dst[10 + 5], 5, "radius boundary changed");
    // dy=3 時原生區間是 trunc(sqrt(7)*10/10)=2：x=3..6。
    for x in 0..10 {
        let want = x as u8 + if (3..7).contains(&x) { 20 } else { 0 };
        assert_eq!(dst[2 * 10 + x], want, "dy=3 x={x}");
    }
    // 中心列的區間是 x=1..8（右端不含）。
    for x in 0..10 {
        let want = x as u8 + if (1..9).contains(&x) { 20 } else { 0 };
        assert_eq!(dst[5 * 10 + x], want, "center x={x}");
    }
}

#[test]
fn apply_radial_lut_remap_clips_to_visible_width() {
    let mut dst: Vec<u8> = (0..32).map(|i| i as u8).collect();
    let lut: Vec<u8> = (0..256).map(|i| (255 - i) as u8).collect();
    let spec = RadialLUTRemap { center_x: 1, center_y: 2, radius: 3, scale: 10, start_y: 0, end_y: 4, clip_width: 6 };
    apply_radial_lut_remap(&mut dst, 8, &lut, spec).unwrap();
    // 中心列區間是三，從 [-2,4) 裁成 [0,4)。即使底層 stride 更寬，clip_width 之外的位元組不動。
    for x in 0..8 {
        let mut want = (2 * 8 + x) as u8;
        if x < 4 {
            want = 255 - want;
        }
        assert_eq!(dst[2 * 8 + x], want, "clip x={x}");
    }
}

#[test]
fn apply_radial_lut_remap_rejects_incomplete_abi() {
    let spec = |center_x, radius| RadialLUTRemap {
        center_x,
        center_y: 1,
        radius,
        scale: 16,
        start_y: 0,
        end_y: 4,
        clip_width: 4,
    };
    assert!(apply_radial_lut_remap(&mut [0u8; 16], 4, &[0u8; 255], spec(1, 1)).is_err(), "short LUT was accepted");
    assert!(
        apply_radial_lut_remap(&mut [0u8; 16], 4, &[0u8; 256], spec(4, 1)).is_err(),
        "out-of-clip center was accepted"
    );
    assert!(
        apply_radial_lut_remap(&mut [0u8; 16], 4, &[0u8; 256], spec(1, 0x8000)).is_err(),
        "overflowing radius was accepted"
    );
}

#[test]
fn apply_centered_rect_lut_remap_clips_and_keeps_end_exclusive() {
    let mut dst: Vec<u8> = (0..32).map(|i| i as u8).collect();
    let lut: Vec<u8> = (0..256).map(|i| (200 + i % 20) as u8).collect();
    let spec = CenteredRectLUTRemap { center_x: 1, horizontal_radius: 3, start_y: 1, end_y: 3, clip_width: 6 };
    apply_centered_rect_lut_remap(&mut dst, 8, &lut, spec).unwrap();
    for y in 0..4 {
        for x in 0..8 {
            let mut want = (y * 8 + x) as u8;
            if (1..3).contains(&y) && x < 4 {
                // [-2,4) 裁成 [0,4)
                want = lut[want as usize];
            }
            assert_eq!(dst[y * 8 + x], want, "y={y} x={x}");
        }
    }
}
