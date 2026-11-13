//! 移植自 `remake/internal/battle/native_raw_word_step_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawWordStepMatchesMarkerDeltaAndScore`
#[test]
fn apply_native_raw_word_step_matches_marker_delta_and_score() {
    let mut records = vec![0u8; 2 * RS];
    (records[0x20], records[0x21]) = (9, 2);
    (records[0x48], records[0x49]) = (100, 0);
    records[RS + 0x22] = 1;
    (records[RS + 0x48], records[RS + 0x49]) = (200, 0);
    let (got, final_state, score) = apply_native_raw_word_step(&mut records, &[0, 1], 0).unwrap();
    assert!(
        final_state == 0x80a4
            && got.len() == 2
            && got[0].processed
            && got[0].marker == 2
            && got[0].delta == 16
            && !got[1].processed
            && score == 64,
        "results={got:?} state={final_state:#x} score={score}"
    );
    assert!(records[0x48] == 116 && records[RS + 0x48] == 200, "words={},{}", records[0x48], records[RS + 0x48]);
}

/// Go：`TestApplyNativeRawWordStepRejectsBadIndexBeforeMutation`
#[test]
fn apply_native_raw_word_step_rejects_bad_index_before_mutation() {
    let mut records = vec![0u8; RS];
    (records[0x22], records[0x48]) = (0, 100);
    assert!(apply_native_raw_word_step(&mut records, &[0, 1], 0).is_err(), "expected bad index to fail closed");
    assert!(records[0x22] == 0 && records[0x48] == 100, "bad input mutated records");
}

/// Go：`TestApplyNativeRawWordStepAtOffsetsMatches22866Family`
#[test]
fn apply_native_raw_word_step_at_offsets_matches22866_family() {
    let mut records = vec![0u8; RS];
    (records[0x20], records[0x21]) = (10, 1);
    (records[0x4a], records[0x4b]) = (100, 0);
    let (got, state, score) = apply_native_raw_word_step_at_offsets(&mut records, &[0], 0, 0x23, 0x4a).unwrap();
    assert!(
        state == 0x80a4
            && got.len() == 1
            && got[0].processed
            && got[0].marker == 2
            && got[0].delta == 16
            && score == 62,
        "results={got:?} state={state:#x} score={score}"
    );
    assert!(records[0x23] == 2 && records[0x4a] == 116, "raw offsets = {:#x} {:#x}", records[0x23], records[0x4a]);
}
