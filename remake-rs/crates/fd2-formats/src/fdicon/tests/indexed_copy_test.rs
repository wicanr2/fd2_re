//! 移植自 `indexed_copy_test.go`。

use crate::fdicon::*;

#[test]
fn copy_native_indexed_region_matches_11eb0() {
    let src = [0u8, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];
    let mut dst = vec![0u8; 12];
    copy_native_indexed_region(&mut dst, 6, &src[1..], 6, 4, 2).unwrap();
    let want = [1u8, 2, 3, 4, 0, 0, 7, 8, 9, 10, 0, 0];
    for i in 0..want.len() {
        assert_eq!(dst[i], want[i], "dst[{i}]");
    }
    assert!(copy_native_indexed_region(&mut [0u8; 3], 3, &src, 6, 4, 1).is_err(), "short destination accepted");
    assert!(copy_native_indexed_region(&mut dst, 3, &src, 6, 4, 1).is_err(), "short stride accepted");
}
