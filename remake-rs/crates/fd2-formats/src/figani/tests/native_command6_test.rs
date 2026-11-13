//! Go `native_command6_test.go`。

use super::*;
use crate::archive::read_nested_resource;

fn command6_test_animation() -> Animation {
    test_animation(NATIVE_COMMAND6_EFFECT_FRAME_COUNT as usize, 0)
}

fn schedule(side: u8) -> NativeCommand6PresentationSchedule {
    build_native_command6_presentation_schedule(side, Some(&command6_test_animation())).unwrap()
}

fn p(x: i64, y: i64) -> NativeCommand6Point {
    NativeCommand6Point { x, y }
}

// Go: TestNativeCommand6SchedulePreservesRawSideTables
#[test]
fn native_command6_schedule_preserves_raw_side_tables() {
    let nonzero = schedule(1);
    assert!(
        nonzero.effect_resource == 32
            && nonzero.sound_resource == 87
            && nonzero.sound_indices == [1, 2, 3]
            && nonzero.base_byte == 30
            && nonzero.dword_table == [10, 8, 3, 0, 0]
            && nonzero.byte_table == [10, 8, 3, 0, 0],
        "nonzero side schedule={nonzero:?}"
    );
    let zero = schedule(0);
    assert!(
        zero.effect_resource == 33
            && zero.base_byte == 90
            && zero.dword_table == [-10, -8, -3, 0, 0]
            && zero.byte_table == [0, 0, 0, 0, 0],
        "zero side schedule={zero:?}"
    );
}

// Go: TestNativeCommand6CoordinatesPreserveFivePointFormula
#[test]
fn native_command6_coordinates_preserve_five_point_formula() {
    assert_eq!(native_command6_coordinates(10, 30), [p(40, 30), p(33, 41), p(21, 37), p(21, 22), p(33, 18)]);
    assert_eq!(native_command6_coordinates(10, 90), [p(100, 30), p(93, 41), p(81, 37), p(81, 22), p(93, 18)]);
}

// Go: TestNativeCommand6TargetsPreserveMode3GeometryAndState
#[test]
fn native_command6_targets_preserve_mode3_geometry_and_state() {
    // 手算對照原始 0x26F13..0x26F3A；保留負 Y，不猜補裁切。
    let front = [p(66, 30), p(41, 71), p(0, 55), p(0, 4), p(41, -11)];
    let want = [p(66, 30), p(41, 71), p(41, -11), p(0, 4), p(0, 42)];
    assert_eq!(native_command6_target_coordinates(front, 42), want, "mode3 coordinates");
    assert_eq!(front[2], p(0, 55), "輸入陣列被改寫");
    for side in [0u8, 1] {
        let schedule = schedule(side);
        let first = build_native_command6_target_sequence(schedule, want, side).unwrap();
        for layer in &first[0].mode4 {
            assert!(!(side == 0 || layer.channel > 1), "mode4 side{side} channel{}", layer.channel);
        }
        for layer in &first[0].mode5 {
            assert!(!(side != 0 && layer.channel < 2), "mode5 side{side} channel{}", layer.channel);
        }
        let transition = build_native_command6_transition_sequence(first[11].next, schedule, want, side).unwrap();
        let state = transition[8].next;
        assert_eq!(state.counters, [1, 0, 4, 3, 2], "12+9 calls counters");
        let second = build_native_command6_target_sequence_from_state(state, schedule, want, side).unwrap();
        assert!(
            second[0].hp_stage == 1 && second[0].next.counters == [2, 1, 0, 4, 3],
            "next target reset state: {:?}",
            second[0]
        );
    }
}

// Go: TestNativeCommand6OrbitPlansPreserveModeOrderAndSideSplit
#[test]
fn native_command6_orbit_plans_preserve_mode_order_and_side_split() {
    let front = plan_native_command6_prelude_frame(0, 1, schedule(1)).unwrap();
    assert!(
        front.first.len() == 2
            && front.second.len() == 3
            && front.first[0].mode == 1
            && front.second[0].mode == 2
            && front.next_radius == 6,
        "front={front:?}"
    );
    let tail = plan_native_command6_tail_frame(42, 1, schedule(1)).unwrap();
    assert!(
        tail.first.len() == 2
            && tail.second.len() == 3
            && tail.first[0].mode == 7
            && tail.second[0].mode == 8
            && tail.next_radius == 36
            && tail.draw_target,
        "tail={tail:?}"
    );
    let zero = plan_native_command6_prelude_frame(0, 0, schedule(0)).unwrap();
    assert!(zero.first.is_empty() && zero.second.len() == 5, "zero-side front={zero:?}");
}

// Go: TestNativeCommand6TargetPlanMarksNumericBoundaryWithoutEndingLoop
#[test]
fn native_command6_target_plan_marks_numeric_boundary_without_ending_loop() {
    let s = schedule(1);
    let points = native_command6_coordinates(10, s.base_byte);
    let first = plan_native_command6_target_frame(new_native_command6_target_state(), s, points, 1).unwrap();
    assert!(
        !first.numeric_marker
            && first.mode4.len() == 2
            && first.mode5.len() == 3
            && first.next.counters == [1, 0, -1, -2, -3],
        "first target frame={first:?}"
    );
    let second = plan_native_command6_target_frame(first.next, s, points, 1).unwrap();
    assert!(second.numeric_marker && second.next.counters == [2, 1, 0, -1, -2], "second target frame={second:?}");
    assert!(
        second.mode5.len() == 4
            && second.mode5[3].secondary
            && second.mode5[3].frame == 5
            && second.mode5[3].channel == 0,
        "second secondary layers={:?}",
        second.mode5
    );
}

// Go: TestNativeCommand6TargetPlanRunsMode3TwelveFrameBudget
#[test]
fn native_command6_target_plan_runs_mode3_twelve_frame_budget() {
    let s = schedule(1);
    let points = native_command6_coordinates(10, s.base_byte);
    let mut state = new_native_command6_target_state();
    let mut markers = 0;
    for _ in 0..12 {
        let planned = plan_native_command6_target_frame(state, s, points, 1).unwrap();
        if planned.numeric_marker {
            markers += 1;
        }
        state = planned.next;
    }
    assert_eq!(markers, 11, "numeric marker count across mode3 budget12");
}

// Go: TestNativeCommand6TargetSequenceConsumesOnlyFirstFiveMarkers
#[test]
fn native_command6_target_sequence_consumes_only_first_five_markers() {
    let s = schedule(1);
    let frames = build_native_command6_target_sequence(s, native_command6_coordinates(10, s.base_byte), 1).unwrap();
    assert_eq!(frames.len() as i64, NATIVE_COMMAND6_TARGET_FRAMES);
    let (mut markers, mut published) = (0, 0);
    for (index, frame) in frames.iter().enumerate() {
        if frame.numeric_marker {
            markers += 1;
        }
        if frame.hp_stage != 0 {
            published += 1;
            assert_eq!(frame.hp_stage, published, "frame {index} HP stage");
        }
    }
    assert!(markers == 11 && published == NATIVE_COMMAND6_DAMAGE_STAGES, "markers={markers} published={published}");
    for (index, frame) in frames.iter().enumerate().skip(6) {
        assert_eq!(frame.hp_stage, 0, "late frame {index} republished HP stage");
    }
}

// Go: TestNativeCommand6TransitionSequencePreservesNineFrameSlide
#[test]
fn native_command6_transition_sequence_preserves_nine_frame_slide() {
    let s = schedule(1);
    let points = native_command6_coordinates(10, s.base_byte);
    let targets = build_native_command6_target_sequence(s, points, 1).unwrap();
    let frames = build_native_command6_transition_sequence(targets[targets.len() - 1].next, s, points, 1).unwrap();
    let want_offsets = [-35, -70, -105, -140, -140, -105, -70, -35, 0];
    assert_eq!(frames.len(), want_offsets.len(), "transition frames");
    for (index, frame) in frames.iter().enumerate() {
        assert!(
            frame.target_offset_x == want_offsets[index] && frame.use_next_target == (index >= 4),
            "frame {index}={frame:?}"
        );
    }
    let zs = schedule(0);
    let zero = build_native_command6_transition_sequence(
        new_native_command6_target_state(),
        zs,
        native_command6_coordinates(10, zs.base_byte),
        0,
    )
    .unwrap();
    assert!(zero[0].target_offset_x == 35 && zero[8].target_offset_x == 0, "zero-side transition");
}

// Go: TestNativeCommand6ZeroSideDrawsOnlyMode5Main
#[test]
fn native_command6_zero_side_draws_only_mode5_main() {
    let s = schedule(0);
    let frame = plan_native_command6_target_frame(
        new_native_command6_target_state(),
        s,
        native_command6_coordinates(10, s.base_byte),
        0,
    )
    .unwrap();
    assert!(frame.mode4.is_empty() && frame.mode5.len() == 5, "zero-side layers");
}

// Go: TestOriginalFDOTHERCommand6ResourcesMatchRecoveredSignatures
#[test]
fn original_fdother_command6_resources_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(32, 1u8), (33, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command6_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
    }
    for sample in [1, 2, 3] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND6_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "resource 87 sample {sample}");
    }
}

// Go: TestNativeCommand6RNGRejectsInvalidInputBeforeResolve
#[test]
fn native_command6_rng_rejects_invalid_input_before_resolve() {
    let zero = schedule(0);
    let mut calls = 0;
    let unset = NativeCommand6PresentationSchedule::default();
    let wrong = NativeCommand6PresentationSchedule { effect_resource: 32, ..Default::default() };
    for (schedule, count, with_resolver) in [(unset, 1, true), (wrong, 1, true), (zero, 0, true), (zero, 1, false)] {
        let mut resolve = |_: i64, rng: u16| -> Resolved {
            calls += 1;
            Ok((rng, true))
        };
        let resolver: Resolver<'_> = if with_resolver { Some(&mut resolve) } else { None };
        let err = walk_native_command6_rng(14047, schedule, 0, count, resolver).unwrap_err();
        assert_eq!(err.state, 14047, "invalid preflight final err={err}");
    }
    assert_eq!(calls, 0, "invalid preflight called the resolver");
    let mut failing = |_: i64, _: u16| -> Resolved { Err("resolver failed".into()) };
    let err = walk_native_command6_rng(14047, zero, 0, 1, Some(&mut failing)).unwrap_err();
    assert!(
        matches!(&err.error, FiganiError::Resolver(e) if e.to_string() == "resolver failed") && err.state == 14047,
        "resolver failure final={} err={err}",
        err.state
    );
}

// Go: TestNativeCommand6CoordinatesMatchOriginalPlayerTargets
// #162 原版正常 cast 的五通道目的指標，包含兩個被 nearest-even 改錯的座標。第 0 通道 frame0 加表格
// (10,10)，其餘 frame4 偏移 0；不使用重製畫面猜期望值。
#[test]
fn native_command6_coordinates_match_original_player_targets() {
    let s = schedule(2);
    let points = native_command6_target_coordinates(native_command6_coordinates(36, s.base_byte), 42);
    let frame = plan_native_command6_target_frame(new_native_command6_target_state(), s, points, 2).unwrap();
    let layers: Vec<_> = frame.mode4.iter().chain(&frame.mode5).copied().collect();
    let want = [p(76, 40), p(41, 71), p(41, -11), p(0, 4), p(0, 42)];
    assert_eq!(layers.len(), 5, "layers={layers:?}");
    for layer in &layers {
        assert_eq!(p(layer.x, layer.y), want[layer.channel as usize], "channel{}", layer.channel);
    }
    // radius42 的負 X 截斷為 -3，正 Y 截斷為 0；兩側保留同一 __CHP 契約。
    for (base, x) in [(30u8, -3), (90, 56)] {
        let got = native_command6_coordinates(42, base);
        assert!(got[3] == p(x, 0) && got[4].y == -17, "base{base} radius42={got:?}");
    }
}

// Go: TestNativeCommand6TargetDisplayMatchesOriginalPlayerCast
// #161 正常原版 12 張 viewport 及 target wrapper 指標直接給出的 shade／pose／RNG。起始 RNG 59907 是
// 已證傷害後的狀態，這裡不重擲命中或傷害。
#[test]
fn native_command6_target_display_matches_original_player_cast() {
    let s = schedule(2);
    let points = native_command6_target_coordinates(native_command6_coordinates(36, s.base_byte), 42);
    let planned = build_native_command6_target_sequence(s, points, 2).unwrap();
    let draws = build_native_command6_target_display_frames(new_native_command6_display_state(), &planned, true, 59907)
        .unwrap();
    let shade = [8, 7, 6, 5, 4, 3, 2, 8, 7, 6, 5, 4];
    let x = [0, 0, -6, 6, 0, 0, 6, -6, 6, -6, -6, -6];
    let y = [0, 0, -3, -3, -3, -3, -3, -3, -3, -3, -3, -3];
    let rng = [59907u16, 53435, 1659, 46204, 9346, 42165, 42569, 45801, 6122, 16373, 32846, 33552];
    for (i, draw) in draws.iter().enumerate() {
        assert!(
            draw.shader == (shade[i] << 8 | 0xb0)
                && draw.offset_x == x[i]
                && draw.offset_y == y[i]
                && draw.rng_after == rng[i],
            "frame{i}={draw:?}"
        );
    }
    let next = draws[11].next;
    assert_eq!(next, NativeCommand6DisplayState { shade: 3, pose: 0, jitter: 1 });
    let missed = build_native_command6_target_display_frames(next, &planned, false, 1234).unwrap();
    for draw in &missed {
        assert!(
            draw.shader == -1
                && draw.offset_x == 0
                && draw.offset_y == 0
                && draw.rng_after == 1234
                && draw.next == next,
            "miss advanced={draw:?}"
        );
    }
    let follow = build_native_command6_target_display_frames(missed[11].next, &planned, true, 1234).unwrap();
    assert!(
        follow[0].shader == 0x3b0 && follow[0].offset_x == 6 && follow[0].offset_y == -3,
        "next hit reset caller={:?}",
        follow[0]
    );
    for state in [
        NativeCommand6DisplayState { shade: 1, ..Default::default() },
        NativeCommand6DisplayState { shade: 9, ..Default::default() },
        NativeCommand6DisplayState { shade: 2, pose: 4, jitter: 0 },
        NativeCommand6DisplayState { shade: 2, pose: 0, jitter: 2 },
    ] {
        assert!(
            build_native_command6_target_display_frames(state, &planned, true, 59907).is_err(),
            "invalid caller state accepted"
        );
    }
}
