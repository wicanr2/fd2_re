//! 移植自 `remake/internal/battle/native_raw_pair_step_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawPairStepMatches22997Offsets`
#[test]
fn apply_native_raw_pair_step_matches22997_offsets() {
    let mut records = vec![0u8; RS];
    (records[0x20], records[0x21]) = (9, 2);
    (records[0x4c], records[0x4d]) = (0xfe, 0xff);
    (records[0x4e], records[0x4f]) = (1, 0);
    let (got, state, score) = apply_native_raw_pair_step(&mut records, &[0], 0).unwrap();
    assert!(
        state == 0x80a4 && got.len() == 1 && got[0].processed && got[0].marker == 2 && score == 64,
        "results={got:?} state={state:#x} score={score}"
    );
    assert!(
        records[0x24] == 2 && records[0x4c] == 13 && records[0x4d] == 0 && records[0x4e] == 16 && records[0x4f] == 0,
        "raw pair = {:#x} {:#x} {:#x} {:#x}",
        records[0x4c],
        records[0x4d],
        records[0x4e],
        records[0x4f]
    );
}

/// Go：`TestApplyNativeRawPairStepSkipsMarkedAndPreflights`
#[test]
fn apply_native_raw_pair_step_skips_marked_and_preflights() {
    let mut records = vec![0u8; RS];
    records[0x24] = 1;
    let (got, state, score) = apply_native_raw_pair_step(&mut records, &[0], 0).unwrap();
    assert!(state == 0 && score == 0 && !got[0].processed, "marked result={got:?} state={state:#x} score={score}");
    assert!(apply_native_raw_pair_step(&mut records, &[1], 0).is_err(), "expected bad index to fail closed");
}
