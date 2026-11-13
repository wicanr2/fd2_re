//! 移植自 `lutbank_test.go`。

use super::original;
use crate::fdother::*;

#[test]
fn parse_lut_bank_directory() {
    let mut data = vec![0u8; 6 + 2 * 4 + 2 * 256];
    data[..4].copy_from_slice(b"LMI1");
    data[4] = 2;
    data[6..10].copy_from_slice(&14u32.to_le_bytes());
    data[10..14].copy_from_slice(&(14u32 + 256).to_le_bytes());
    data[14] = 7;
    data[14 + 256] = 9;
    let luts = parse_lut_bank(&data).unwrap();
    assert!(luts.len() == 2 && luts[0].len() == 256 && luts[0][0] == 7 && luts[1][0] == 9, "luts={luts:?}");
    assert!(parse_lut_bank(&data[..20]).is_err(), "truncated LUT bank accepted");
}

#[test]
fn native_unit_present_bridge_lut_crosses_entry_boundary() {
    let mut luts = vec![(0..=255u8).collect::<Vec<u8>>(), vec![0u8; 256]];
    luts[1][0] = 0xa7;
    let bridge = native_unit_present_bridge_lut(&luts).unwrap();
    assert!(bridge[0] == 1 && bridge[254] == 255 && bridge[255] == 0xa7, "shifted bridge boundaries");
    assert!(native_unit_present_bridge_lut(&luts[..1]).is_err(), "accepted bridge source without adjacent entry");
}

#[test]
fn fdother_003_lut_bank() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let luts = decode_lut_resource(&path, 3).unwrap();
    assert_eq!(luts.len(), 23, "FDOTHER#3 LUT count");
    for (i, lut) in luts.iter().enumerate() {
        assert_eq!(lut.len(), 256, "FDOTHER#3 LUT {i} length");
    }
    let bridge = native_unit_present_bridge_lut(&luts).unwrap();
    assert!(
        bridge[0] == luts[0][1] && bridge[254] == luts[0][255] && bridge[255] == luts[1][0],
        "real FDOTHER#3 bridge LUT does not match entry0 pointer+1"
    );
    assert!(bridge != luts[0] && bridge != luts[1], "real bridge LUT incorrectly aliases an aligned directory entry");
}
