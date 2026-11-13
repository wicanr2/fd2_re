//! Go `native_command2_test.go`。

use super::*;
use crate::archive::read_nested_resource;
use crate::native_rng::native_rng_step;

// Go: TestNativeCommand2RNGWalkMatchesChapter18FourTargetBoundary
#[test]
fn native_command2_rng_walk_matches_chapter18_four_target_boundary() {
    // 依已閉合數值轉移與 marker 推導四組 RNG；原版 r2 的 HP 與下一筆 record51 的 0x13A9F 入口
    // 43037 核對推導結果。
    let before = [42897u16, 42415, 13218, 47507];
    let after = [27114u16, 61801, 28245, 60022];
    let mut calls = 0;
    let mut resolve = |index: i64, rng: u16| -> Resolved {
        assert!(index == calls && rng == before[index as usize], "目標{index}的數值入口RNG={rng}");
        calls += 1;
        Ok((after[index as usize], true))
    };
    let rng = walk_native_command2_rng(before[0], 4, Some(&mut resolve)).unwrap();
    assert!(calls == 4 && rng == 43037, "指令2目標間亂數不符：calls={calls} rng={rng}");
}

// Go: TestNativeCommand2RNGWalkMissSkipsHitMarkers
#[test]
fn native_command2_rng_walk_miss_skips_hit_markers() {
    let miss_after = native_rng_step(42897);
    let mut calls = 0;
    let mut resolve = |index: i64, rng: u16| -> Resolved {
        calls += 1;
        if index == 0 {
            return Ok((miss_after, false));
        }
        assert_eq!(rng, miss_after, "未命中目標不得消耗命中HP marker亂數");
        Ok((rng, false))
    };
    let rng = walk_native_command2_rng(42897, 2, Some(&mut resolve)).unwrap();
    assert!(calls == 2 && rng == miss_after, "miss亂數不符：calls={calls} rng={rng}");
}

fn command2_test_animation() -> Animation {
    test_animation(NATIVE_COMMAND2_EFFECT_FRAME_COUNT as usize, 0)
}

fn command2_sequence_animation(delay: i64) -> Animation {
    let mut a = command2_test_animation();
    for frame in &mut a.frames {
        frame.delay = delay;
    }
    a
}

// Go: TestNativeCommand2SequencesPreservePhaseBudgetsAndMarkers
#[test]
fn native_command2_sequences_preserve_phase_budgets_and_markers() {
    let (front, front_end) = build_native_command2_front_sequence(0, Some(&command2_sequence_animation(4))).unwrap();
    assert!(
        front.len() as i64 == NATIVE_COMMAND2_FRONT_FRAMES && front_end.frame == 7 && front_end.repeat == 1,
        "front len={} end={front_end:?}",
        front.len()
    );
    assert!(front[28].sample1, "raw-side-zero front omitted the pre-draw frame7 sample1 marker");
    let (target, target_end) = build_native_command2_target_sequence().unwrap();
    assert!(
        target.len() as i64 == NATIVE_COMMAND2_TARGET_FRAMES && target_end.frame == 16,
        "target end={target_end:?}"
    );
    let mut hp_steps = Vec::new();
    for (step, frame) in target.iter().enumerate() {
        if frame.hp_stage != 0 {
            hp_steps.push(step);
            assert!(frame.sample2 && frame.hp_stage == hp_steps.len() as i64, "target step{step} frame={frame:?}");
        }
    }
    assert_eq!(hp_steps, [0, 2, 4, 6, 8, 10], "HP steps");
    let transition = build_native_command2_transition_sequence(target_end);
    let mut sample_steps = Vec::new();
    for (step, frame) in transition.iter().enumerate() {
        if frame.sample2 {
            sample_steps.push(step);
        }
        assert!(!frame.numeric_marker && frame.hp_stage == 0, "boundary step{step} leaked numeric marker");
    }
    assert_eq!(sample_steps, [0, 2, 4, 6, 8], "boundary sample steps");
    let tail = build_native_command2_tail_sequence(1, front_end.repeat, Some(&command2_sequence_animation(4))).unwrap();
    assert!(tail.len() as i64 == NATIVE_COMMAND2_TAIL_FRAMES && tail[0].effect_frame == 10, "tail first={:?}", tail[0]);
}

// Go: TestNativeCommand2SchedulePreservesRawResources
#[test]
fn native_command2_schedule_preserves_raw_resources() {
    for (side, resource) in [(0u8, 27), (1, 26), (2, 26)] {
        let s = build_native_command2_presentation_schedule(side, Some(&command2_test_animation())).unwrap();
        assert!(
            s.effect_resource == resource && s.sound_resource == 83 && s.sound_indices == [1, 2, 3],
            "side={side} schedule={s:?}"
        );
    }
}

// Go: TestOriginalFDOTHERCommand2EffectBanksMatchRecoveredSignatures
#[test]
fn original_fdother_command2_effect_banks_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (resource, side) in [(26, 1u8), (27, 0)] {
        let animation = decode_resource(&path, resource).unwrap();
        let schedule = build_native_command2_presentation_schedule(side, Some(&animation)).unwrap();
        assert_eq!(schedule.effect_resource, resource);
        let (front, end) = build_native_command2_front_sequence(side, Some(&animation)).unwrap();
        assert_eq!(front.len() as i64, NATIVE_COMMAND2_FRONT_FRAMES, "resource {resource} front");
        let tail = build_native_command2_tail_sequence(side, end.repeat, Some(&animation)).unwrap();
        assert_eq!(tail.len() as i64, NATIVE_COMMAND2_TAIL_FRAMES, "resource {resource} tail");
    }
    for sample in [1, 2, 3] {
        let raw = read_nested_resource(&path, NATIVE_COMMAND2_SOUND_RESOURCE as usize, sample).unwrap();
        assert!(!raw.is_empty(), "resource 83 sample {sample}");
    }
}
