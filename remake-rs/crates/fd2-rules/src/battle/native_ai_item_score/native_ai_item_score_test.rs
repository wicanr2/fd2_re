//! 移植自 `remake/internal/battle/native_ai_item_score_test.go`。

use super::*;
use crate::test_support::put16;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestScoreNativeAIItemCommandTargetsType5And13`
#[test]
fn score_native_ai_item_command_targets_type5_and13() {
    let mut records = vec![0u8; 3 * RS];
    let mut set_hp = |index: usize, current: u16, maximum: u16, raw34: u8| {
        put16(&mut records, index * RS + 0x40, current);
        put16(&mut records, index * RS + 0x42, maximum);
        records[index * RS + 0x34] = raw34;
    };
    set_hp(0, 30, 90, 0);
    set_hp(1, 45, 90, 0);
    set_hp(2, 46, 90, 0x80);
    for item_type in [5, 0x0d] {
        let got = score_native_ai_item_command_targets(&records, &[0, 1, 2], item_type, 0, &[]);
        assert!(matches!(got, Ok(11)), "type={item_type:#x} score={got:?} want 11");
    }
}

/// Go：`TestScoreNativeAIItemCommandTargetsThresholdFamilies`
#[test]
fn score_native_ai_item_command_targets_threshold_families() {
    let mut records = vec![0u8; 2 * RS];
    put16(&mut records, 0x40, 20);
    put16(&mut records, RS + 0x40, 21);
    let mut book: Vec<NativeCommandRecord> =
        (0..NATIVE_COMMAND_RECORD_COUNT).map(|id| NativeCommandRecord { id, ..Default::default() }).collect();
    book[2].damage = 20;
    for item_type in [0x14, 0x15] {
        let got = score_native_ai_item_command_targets(&records, &[0, 1], item_type, 2, &book);
        assert!(matches!(got, Ok(0x1a)), "type={item_type:#x} score={got:?} want 26");
    }
    let got = score_native_ai_item_command_targets(&records, &[0, 1], 0x18, 20, &[]);
    assert!(matches!(got, Ok(0x1a)), "type=0x18 score={got:?} want 26");
}

/// Go：`TestScoreNativeAIItemCommandTargetsFailsClosed`
#[test]
fn score_native_ai_item_command_targets_fails_closed() {
    let records = vec![0u8; RS];
    assert!(score_native_ai_item_command_targets(&records, &[1], 5, 0, &[]).is_err(), "out-of-range target accepted");
    assert!(
        score_native_ai_item_command_targets(&records, &[0], 0x14, 2, &[]).is_err(),
        "missing 0x4e516 command record accepted"
    );
}
