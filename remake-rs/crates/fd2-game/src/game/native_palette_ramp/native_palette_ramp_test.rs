//! Go 來源：`remake/cmd/fd2/native_palette_ramp_test.go`（R4 B 組，期望值不改）。
//!
//! 其餘測試先以 C 組的 `completeNativeMapFrameFixture` 建素材庫，見 `PORTING.md`「B 組測試」。

use super::*;

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
