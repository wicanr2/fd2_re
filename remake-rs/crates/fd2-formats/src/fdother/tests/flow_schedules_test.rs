//! 移植自 `ch28_post_present_test.go`、`indexed_transition_test.go`、`indexed_transition_pass_test.go`、
//! `native_ai_item_type21_tail_test.go`、`native_command20_22_tail_test.go`、`native_command32_tail_test.go`、
//! `native_command33_tail_test.go`、`native_command35_tail_test.go`、`native_command9_test.go` 與
//! `pending_code1_interlude_test.go`。這些檔案每個只有一到三項測試，合併成一個檔。

use super::original;
use crate::fdother::*;

// ---- ch28_post_present_test.go ----

// Go: TestNativeCh28PostOverlayFramesPreserveTwoSixFrameLoops
#[test]
fn native_ch28_post_overlay_frames_preserve_two_six_frame_loops() {
    let frames = native_ch28_post_overlay_frames();
    assert_eq!(frames.len(), 12);
    for (index, frame) in frames.iter().enumerate() {
        assert_eq!(frame.entry, NATIVE_CH28_POST_LMI_FIRST + index as i64, "frame{index} entry");
        let want_phase = if index >= 6 { "overlay_b" } else { "overlay_a" };
        assert_eq!(frame.phase, want_phase, "frame{index} phase");
    }
}

// Go: TestNativeCh28PostOverlayOriginMatchesDirectAddressExpression
#[test]
fn native_ch28_post_overlay_origin_matches_direct_address_expression() {
    let got = native_ch28_post_overlay_origin(10, 8, 9, 7).unwrap();
    assert_eq!(got, NATIVE_CH28_POST_VIEW_BASE - 6 * NATIVE_CH28_POST_STRIDE);
    assert!(native_ch28_post_overlay_origin(-100, -100, 0, 0).is_err(), "off-surface origin was accepted");
}

// Go: TestBlitNativeCh28PostOverlayUsesTransparent4E85BRule
#[test]
fn blit_native_ch28_post_overlay_uses_transparent_4e85b_rule() {
    let mut work = vec![0u8; NATIVE_CH28_POST_WORK_SIZE as usize];
    let origin = native_ch28_post_overlay_origin(10, 8, 9, 7).unwrap();
    let o = origin as usize;
    work[o] = 7;
    work[o + 1] = 8;
    let entry = LMI1Entry { width: 2, height: 1, pixels: vec![0, 9] };
    blit_native_ch28_post_overlay(&entry, &mut work, origin).unwrap();
    assert_eq!(&work[o..o + 2], [7, 9]);
}

// Go: TestValidateNativeCh28PostLMIRequiresExactRawEntryRange
#[test]
fn validate_native_ch28_post_lmi_requires_exact_raw_entry_range() {
    let mut entries = vec![LMI1Entry::default(); (NATIVE_CH28_POST_LMI_LAST + 1) as usize];
    for index in NATIVE_CH28_POST_LMI_FIRST..=NATIVE_CH28_POST_LMI_LAST {
        entries[index as usize] = LMI1Entry { width: 1, height: 1, pixels: vec![index as u8] };
    }
    validate_native_ch28_post_lmi(&entries).unwrap();
    entries[(NATIVE_CH28_POST_LMI_FIRST + 2) as usize] = LMI1Entry::default();
    assert!(validate_native_ch28_post_lmi(&entries).is_err(), "malformed required entry was accepted");
}

// ---- indexed_transition_test.go ----

// Go: TestApplyIndexedTransitionPassPreservesRadialRedrawOrdering
#[test]
fn apply_indexed_transition_pass_preserves_radial_redraw_ordering() {
    let mut dst: Vec<u8> = (0..64).map(|i| i as u8).collect();
    let lut: Vec<u8> = (0..256).map(|i| (255 - i) as u8).collect();
    let pass = IndexedTransitionPass {
        first_radial: RadialLUTRemap {
            center_x: 3,
            center_y: 3,
            radius: 3,
            scale: 10,
            start_y: 0,
            end_y: 8,
            clip_width: 8,
        },
        second_radial: RadialLUTRemap {
            center_x: 3,
            center_y: 3,
            radius: 2,
            scale: 10,
            start_y: 0,
            end_y: 8,
            clip_width: 8,
        },
        final_rect: CenteredRectLUTRemap { center_x: 3, horizontal_radius: 2, start_y: 0, end_y: 8, clip_width: 8 },
    };
    let mut order = Vec::new();
    let mut redraw = |buf: &mut [u8]| {
        order.push("redraw");
        buf[0] = 0x7e;
        Ok(())
    };
    apply_indexed_transition_pass(&mut dst, 8, &lut, pass, Some(&mut redraw)).unwrap();
    assert!(order.len() == 1 && dst[0] == 0x7e, "redraw ordering={order:?} first-byte={:#x}", dst[0]);
}

fn one_pixel_pass(second_center_x: i64) -> IndexedTransitionPass {
    let radial = |center_x| RadialLUTRemap {
        center_x,
        center_y: 1,
        radius: 1,
        scale: 10,
        end_y: 8,
        clip_width: 8,
        ..Default::default()
    };
    IndexedTransitionPass {
        first_radial: radial(1),
        second_radial: radial(second_center_x),
        final_rect: CenteredRectLUTRemap {
            center_x: 1,
            horizontal_radius: 1,
            end_y: 8,
            clip_width: 8,
            ..Default::default()
        },
    }
}

// Go: TestApplyIndexedTransitionPassRejectsMissingRedrawBeforeMutation
#[test]
fn apply_indexed_transition_pass_rejects_missing_redraw_before_mutation() {
    let mut dst = vec![0u8; 64];
    dst[0] = 7;
    assert!(apply_indexed_transition_pass(&mut dst, 8, &[0; 256], one_pixel_pass(1), None).is_err(), "missing redraw");
    assert_eq!(dst[0], 7, "missing redraw mutated buffer");
}

// Go: TestApplyIndexedTransitionPassPreflightsSecondPass
#[test]
fn apply_indexed_transition_pass_preflights_second_pass() {
    let mut dst = vec![0u8; 64];
    dst[0] = 9;
    let mut redraw = |_: &mut [u8]| Ok(());
    assert!(
        apply_indexed_transition_pass(&mut dst, 8, &[0; 256], one_pixel_pass(9), Some(&mut redraw)).is_err(),
        "invalid second pass accepted"
    );
    assert_eq!(dst[0], 9, "invalid second pass mutated first pass");
}

// ---- indexed_transition_pass_test.go ----

// Go: TestBuildNativeIndexedTransitionPassPreservesRawArgumentAliasing
#[test]
fn build_native_indexed_transition_pass_preserves_raw_argument_aliasing() {
    let pass = build_native_indexed_transition_pass(6, 6, 10, 0, 192).unwrap();
    let f = pass.first_radial;
    assert!(f.center_x == 6 && f.center_y == 6 && f.scale == 16 && f.start_y == 0 && f.end_y == 192, "first={f:?}");
    let s = pass.second_radial;
    assert!(s.start_y == 6 && s.end_y == 192, "second radial={s:?}");
    let r = pass.final_rect;
    assert!(r.horizontal_radius == 16 && r.start_y == 0 && r.end_y == 6, "final rect={r:?}");
}

// Go: TestBuildNativeIndexedTransitionPassRejectsInvalidRawBounds
#[test]
fn build_native_indexed_transition_pass_rejects_invalid_raw_bounds() {
    for a in [[-1, 6, 10, 0, 192], [6, 6, 0, 0, 192], [6, 6, 10, 193, 192], [312, 6, 10, 0, 192]] {
        assert!(build_native_indexed_transition_pass(a[0], a[1], a[2], a[3], a[4]).is_err(), "accepted args={a:?}");
    }
}

// ---- native_ai_item_type21_tail_test.go ----

/// Go `validType21Entries`／`command9LMIEntries`（兩者相同）。
fn valid_entries(count: usize) -> Vec<LMI1Entry> {
    (0..count).map(|index| LMI1Entry { width: 1, height: 1, pixels: vec![index as u8] }).collect()
}

// Go: TestBuildNativeAIItemType21TailSchedulePreservesCallerTables
#[test]
fn build_native_ai_item_type21_tail_schedule_preserves_caller_tables() {
    let (effect, digits) = (valid_entries(0x4c), valid_entries(0x68));
    for (id, start, frames, sample) in [(1, 0x31, 8, 6), (6, 0x40, 10, 9), (7, 0x40, 10, 9)] {
        let s = build_native_ai_item_type21_tail_schedule(id, &effect, &digits).unwrap();
        assert!(
            s.effect_start == start
                && s.effect_frames == frames
                && s.effect_sample == sample
                && s.toggle_a == 0x4a
                && s.toggle_b == 0x4b
                && s.toggle_pairs == 4
                && s.toggle_delay_ms == 90
                && s.digit_bias == 0x5e
                && s.digit_frames == 22
                && s.digit_hold_ms == 500,
            "command {id} schedule={s:?}"
        );
    }
}

// Go: TestBuildNativeAIItemType21TailScheduleFailsClosed
#[test]
fn build_native_ai_item_type21_tail_schedule_fails_closed() {
    let (mut effect, digits) = (valid_entries(0x4c), valid_entries(0x68));
    for command_id in [0, 2, 3, 32] {
        assert!(
            build_native_ai_item_type21_tail_schedule(command_id, &effect, &digits).is_err(),
            "command {command_id}"
        );
    }
    effect[0x4a] = LMI1Entry::default();
    assert!(build_native_ai_item_type21_tail_schedule(1, &effect, &digits).is_err(), "invalid toggle accepted");
}

// ---- native_command20_22_tail_test.go ----

// Go: TestBuildNativeCommand2022TailSchedulePreservesRawRows
#[test]
fn build_native_command2022_tail_schedule_preserves_raw_rows() {
    let rows = build_native_command2022_tail_schedule().unwrap();
    assert_eq!(rows.len(), 6);
    let want = [
        [20, 0xcc, 13, 4, 0xc0, 0x69],
        [21, 0xd9, 13, 4, 0xc0, 0x69],
        [22, 0xaa, 13, 3, 0x23, 0x5e],
        [25, 0xbf, 13, 5, 0xc0, 0x5e],
        [26, 0x8a, 9, 3, 0xc0, 0x5e],
        [27, 0x9e, 12, 2, 0xc0, 0x5e],
    ];
    for (index, row) in rows.iter().enumerate() {
        let got =
            [row.command_id, row.effect_start, row.effect_frames, row.effect_sample, row.mask_index, row.digit_bias];
        assert!(
            got == want[index] && row.mask_pairs == 5 && row.digit_frames == 22 && row.digit_hold_milliseconds == 500,
            "row {index}={row:?}"
        );
    }
    for (index, row) in rows.iter().enumerate() {
        let want: &[i64] = if index == 2 { &[7] } else { &[] };
        assert_eq!(row.extra_sample_frame_indices, want, "extra sample markers row {index}");
    }
}

// ---- native_command32_tail_test.go ----

// Go: TestBuildNativeCommand32TailSchedulePreservesRawContract
#[test]
fn build_native_command32_tail_schedule_preserves_raw_contract() {
    let s = build_native_command32_tail_schedule(&valid_entries(0x4c), &valid_entries(0x77)).unwrap();
    assert!(
        s.effect_resource == 6
            && s.effect_start == 0x40
            && s.effect_frames == 10
            && s.sound_resource == 80
            && s.effect_sample == 9
            && s.effect_delay_ticks == 1
            && s.toggle_a == 0x4a
            && s.toggle_b == 0x4b
            && s.toggle_pairs == 4
            && s.toggle_delay_ms == 90
            && s.digit_resource == 5
            && s.digit_bias == 0x5e
            && s.digit_frames == 22
            && s.digit_delay_ms == 2
            && s.digit_hold_ms == 500,
        "command32 tail schedule={s:?}"
    );
}

// Go: TestBuildNativeCommand32TailScheduleFailsClosed
#[test]
fn build_native_command32_tail_schedule_fails_closed() {
    assert!(
        build_native_command32_tail_schedule(&valid_entries(0x4b), &valid_entries(0x77)).is_err(),
        "command32 accepted a missing toggle descriptor"
    );
    assert!(
        build_native_command32_tail_schedule(&valid_entries(0x4c), &valid_entries(0x76)).is_err(),
        "command32 accepted incomplete miss digits"
    );
}

// Go: TestNativeCommand32TailOriginalAssets
#[test]
fn native_command32_tail_original_assets() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let effect = decode_lmi1_resource(&path, NATIVE_COMMAND32_TAIL_EFFECT_RESOURCE).unwrap();
    let digits = decode_lmi1_resource(&path, NATIVE_COMMAND32_TAIL_DIGIT_RESOURCE).unwrap();
    build_native_command32_tail_schedule(&effect, &digits).unwrap();
    let raw = read_nested_resource(
        &path,
        NATIVE_COMMAND32_TAIL_SOUND_RESOURCE as usize,
        NATIVE_COMMAND32_TAIL_EFFECT_SAMPLE as usize,
    )
    .unwrap();
    assert!(!raw.is_empty(), "FDOTHER #80 selector9 len=0");
}

// ---- native_command33_tail_test.go ----

// Go: TestBuildNativeCommand33TailScheduleUsesHardCodedCommand13Row
#[test]
fn build_native_command33_tail_schedule_uses_hard_coded_command13_row() {
    let s = build_native_command33_tail_schedule().unwrap();
    assert!(
        s.effect_start == 0x39
            && s.effect_frames == 7
            && s.effect_sample == 12
            && s.mask_sample == 1
            && s.mask_index == 0xc0
            && s.mask_pairs == 5
            && s.digit_bias == 0x69
            && s.digit_frames == 22
            && s.digit_hold_ms == 500,
        "command33 tail={s:?}"
    );
}

// ---- native_command35_tail_test.go ----

// Go: TestBuildNativeCommand35TailSchedulePreservesRawRows
#[test]
fn build_native_command35_tail_schedule_preserves_raw_rows() {
    let rows = build_native_command35_tail_schedule().unwrap();
    assert_eq!(rows.len(), 3);
    let want = [[26, 0x8a, 9, 3, 0xc0], [22, 0xaa, 13, 3, 0x23], [27, 0x9e, 12, 2, 0xc0]];
    for (index, row) in rows.iter().enumerate() {
        let got = [row.command_id, row.effect_start, row.effect_frames, row.effect_sample, row.mask_index];
        assert_eq!(got, want[index], "row {index}={row:?}");
        assert!(
            row.mask_pairs == 5
                && row.digit_bias == 0x5e
                && row.digit_frames == 22
                && row.digit_hold_milliseconds == 500,
            "tail row {index}={row:?}"
        );
    }
    assert!(
        rows[1].extra_sample_frame_indices == [7] && rows[1].extra_sample == 3,
        "command22 extra marker={:?}",
        rows[1]
    );
}

// ---- native_command9_test.go ----

// Go: TestBuildNativeCommand9PlayerSchedulePreservesRawTables
#[test]
fn build_native_command9_player_schedule_preserves_raw_tables() {
    let s = build_native_command9_player_schedule(&valid_entries(230), &valid_entries(138)).unwrap();
    assert!(
        s.effect_start == 87
            && s.effect_frames == 27
            && s.sound_resource == 80
            && s.initial_sample == 14
            && s.repeat_sample == 15
            && s.repeat_sample_frames == [15, 19]
            && s.result_digit_bias == 94
            && s.result_miss_descriptors == [116, 117, 118, 118]
            && s.result_frames == 22,
        "command9 player schedule={s:?}"
    );
}

// Go: TestBuildNativeCommand9PlayerScheduleFailsClosed
#[test]
fn build_native_command9_player_schedule_fails_closed() {
    assert!(
        build_native_command9_player_schedule(&valid_entries(113), &valid_entries(138)).is_err(),
        "short effect bank was accepted"
    );
    assert!(
        build_native_command9_player_schedule(&valid_entries(230), &valid_entries(118)).is_err(),
        "short result bank was accepted"
    );
}

// Go: TestNativeCommand9PlayerOriginalDescriptors
#[test]
fn native_command9_player_original_descriptors() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let effect = decode_lmi1_resource(&path, NATIVE_COMMAND9_PLAYER_EFFECT_RESOURCE).unwrap();
    let result = decode_lmi1_resource(&path, NATIVE_COMMAND9_RESULT_RESOURCE).unwrap();
    build_native_command9_player_schedule(&effect, &result).unwrap();
    for selector in [0, 14, 15] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND9_PLAYER_SOUND_RESOURCE as usize, selector).unwrap();
        assert!(!raw.is_empty(), "FDOTHER #80 selector {selector} len=0");
    }
}

// ---- pending_code1_interlude_test.go ----

// Go: TestNativePendingCode1PresentationPlanPreservesRawOrder
#[test]
fn native_pending_code1_presentation_plan_preserves_raw_order() {
    use PendingCode1PresentationKind::*;
    let got = native_pending_code1_presentation_plan();
    let want_kinds =
        [StopBgm, WaitTick, PreparePalette, ClearScreen, DrawFrame, FadeIn, WaitTick, DrawFrame, WaitTick, Release];
    assert_eq!(got.len(), want_kinds.len());
    for (index, kind) in want_kinds.iter().enumerate() {
        assert_eq!(got[index].kind, *kind, "step {index} kind");
    }
    assert!(
        got[1].count == 1
            && got[4].frame == 0
            && got[5].count == 65
            && got[5].duration_ms == 2
            && got[6].count == 9
            && got[7].frame == 1
            && got[8].count == 36,
        "raw schedule drifted: {got:?}"
    );
}
