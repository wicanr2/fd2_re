//! 移植自 `indexed_transition_buffer_test.go`。

use crate::fdother::*;

// Go: TestCopyNativeTransitionBufferPreservesFullSeed
#[test]
fn copy_native_transition_buffer_preserves_full_seed() {
    let src = [1u8, 2, 3, 4];
    let mut dst = vec![0u8; 8];
    copy_native_transition_buffer(&mut dst, &src).unwrap();
    assert_eq!(&dst[..4], &src);
    assert_eq!(dst[7], 0, "copy escaped source length");
}

// Go: TestCopyNativeTransitionViewportUsesNativeStrides
#[test]
fn copy_native_transition_viewport_uses_native_strides() {
    let src: Vec<u8> = (0..4 * 6).map(|i| i as u8 + 1).collect();
    let mut dst = vec![0u8; 4 * 4];
    copy_native_transition_viewport(&mut dst, 4, &src, 1, 6, 3, 2).unwrap();
    assert_eq!(&dst[0..3], &[2, 3, 4], "viewport row0");
    assert_eq!(&dst[4..7], &[8, 9, 10], "viewport row1");
}

// Go: TestCopyNativeTransitionViewportRejectsBeforeMutation
#[test]
fn copy_native_transition_viewport_rejects_before_mutation() {
    let mut dst = vec![9u8, 9, 9, 9];
    let snapshot = dst.clone();
    assert!(copy_native_transition_viewport(&mut dst, 2, &[1, 2], 0, 2, 3, 1).is_err(), "invalid width accepted");
    assert_eq!(dst, snapshot, "invalid geometry mutated destination");
}
