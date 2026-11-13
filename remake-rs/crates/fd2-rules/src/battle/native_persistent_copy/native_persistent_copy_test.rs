//! 移植自 `remake/internal/battle/native_persistent_copy_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

fn put16(b: &mut [u8], off: usize, v: u16) {
    b[off..off + 2].copy_from_slice(&v.to_le_bytes());
}

fn get16(b: &[u8], off: usize) -> u16 {
    u16::from_le_bytes([b[off], b[off + 1]])
}

/// Go：`TestApplyNativePersistentRecordCopyPreservesProvenMutationCore`
#[test]
fn apply_native_persistent_record_copy_preserves_proven_mutation_core() {
    let mut runtime = vec![0u8; RS];
    let mut persistent = vec![0u8; RS];
    runtime[5] = 0xff;
    (runtime[0x22], runtime[0x27]) = (7, 9);
    put16(&mut runtime, 0x40, 11);
    put16(&mut runtime, 0x42, 22);
    put16(&mut runtime, 0x44, 33);
    put16(&mut runtime, 0x46, 44);
    apply_native_persistent_record_copy(&runtime, &mut persistent, 0, 0).unwrap();
    assert!(
        persistent[0x22] == 0 && persistent[0x27] == 0 && persistent[5] == 1,
        "status bytes={:#x}/{:#x}/{:#x}",
        persistent[5],
        persistent[0x22],
        persistent[0x27]
    );
    assert_eq!(get16(&persistent, 0x40), 11, "+40 want copied runtime value 11 when byte5==1");
    assert_eq!(get16(&persistent, 0x44), 44, "+44");
}

/// Go：`TestApplyNativePersistentRecordCopyNonOneByte5CopiesWord40`
#[test]
fn apply_native_persistent_record_copy_non_one_byte5_copies_word40() {
    let mut runtime = vec![0u8; RS];
    let mut persistent = vec![0u8; RS];
    runtime[5] = 0x80;
    put16(&mut runtime, 0x42, 0x1234);
    put16(&mut runtime, 0x46, 0x5678);
    apply_native_persistent_record_copy(&runtime, &mut persistent, 0, 0).unwrap();
    assert_eq!(get16(&persistent, 0x40), 0x1234, "+40");
}

/// Go：`TestApplyNativePersistentRecordCopyPreflightsBothBuffers`
#[test]
fn apply_native_persistent_record_copy_preflights_both_buffers() {
    assert!(
        apply_native_persistent_record_copy(&[0u8; RS - 1], &mut [0u8; RS], 0, 0).is_err(),
        "short runtime record unexpectedly accepted"
    );
    assert!(
        apply_native_persistent_record_copy(&[0u8; RS], &mut [0u8; RS - 1], 0, 0).is_err(),
        "short persistent record unexpectedly accepted"
    );
}
