//! 移植自 `native_compound_palette_test.go`。

use crate::fdother::*;

// Go: TestInterpolateNativeCompoundDACPreservesRawFormulaAndHalfOpenRange
#[test]
fn interpolate_native_compound_dac_preserves_raw_formula_and_half_open_range() {
    let mut baseline = vec![0u8; 256 * 3];
    baseline[..6].copy_from_slice(&[3, 33, 63, 63, 3, 33]);
    baseline[255 * 3] = 17;
    let raw = [63u8, 63, 63];
    let zero = interpolate_native_compound_dac(&baseline, 0, 255, 0, raw).unwrap();
    let full = interpolate_native_compound_dac(&baseline, 0, 255, 40, raw).unwrap();
    assert!(
        zero[..3] == [63, 63, 63] && full[..3] == [3, 33, 63] && zero[255 * 3] == 17 && full[255 * 3] == 17,
        "compound DAC zero={:?} full={:?}",
        &zero[..6],
        &full[..6]
    );
    let mid = interpolate_native_compound_dac(&baseline, 1, 2, 20, raw).unwrap();
    assert!(
        mid[0] == 3 && mid[3] == 63 && mid[4] == 33 && mid[5] == 48,
        "compound DAC signed interpolation={:?}",
        &mid[..6]
    );
}

// Go: TestInterpolateNativeCompoundDACFailsClosed
#[test]
fn interpolate_native_compound_dac_fails_closed() {
    let baseline = vec![0u8; 256 * 3];
    for (start, end, delta) in [(-1, 255, 1), (0, 257, 1), (2, 2, 1), (0, 255, 41)] {
        assert!(
            interpolate_native_compound_dac(&baseline, start, end, delta, [0; 3]).is_err(),
            "accepted invalid interpolation {start},{end},{delta}"
        );
    }
}
