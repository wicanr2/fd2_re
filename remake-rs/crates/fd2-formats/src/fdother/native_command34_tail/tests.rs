//! 移植自 `native_command34_tail_test.go`。

use crate::fdother::*;

// Go: TestBuildNativeCommand34TailSchedulePreservesRawRows
#[test]
fn build_native_command34_tail_schedule_preserves_raw_rows() {
    let rows = build_native_command34_tail_schedule().unwrap();
    assert_eq!(rows.len(), 3);
    let want = [[17, 0xb7, 8, 6, 0x92], [18, 0x7e, 12, 7, 0x48], [19, 0x93, 11, 8, 0xd8]];
    for (index, row) in rows.iter().enumerate() {
        let got = [row.command_id, row.effect_start, row.effect_frames, row.effect_sample, row.mask_index];
        assert!(
            got == want[index]
                && row.mask_pairs == 5
                && row.digit_bias == 0x69
                && row.digit_frames == 22
                && row.digit_hold_milliseconds == 500,
            "row {index}={row:?}"
        );
    }
    assert!(rows[0].extra_sample_frame_indices.is_empty());
    assert_eq!(rows[1].extra_sample_frame_indices, [4]);
    assert_eq!(rows[2].extra_sample_frame_indices, [3, 6]);
}
