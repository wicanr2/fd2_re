//! 移植自 Go `battlepresent/native_command6_sequence_test.go`。

use super::*;
use crate::battlepresent::test_util::{px1, surface};
use fd2_formats::figani::NATIVE_COMMAND6_EFFECT_FRAME_COUNT;

fn command6_sequence_animation(frames: usize) -> Animation {
    Animation {
        frames: (0..frames).map(|i| px1(0, 0, i as u8 + 1, 1)).collect(),
        header_byte2: frames as u8,
        ..Default::default()
    }
}

#[test]
fn test_build_native_command6_effect_sequence_prebuilds_every_target() {
    let mut effect = command6_sequence_animation(NATIVE_COMMAND6_EFFECT_FRAME_COUNT as usize);
    for frame in &mut effect.frames {
        frame.y = 20;
    }
    let schedule = figani::build_native_command6_presentation_schedule(1, Some(&effect)).unwrap();
    let base = surface();
    let target_bases = vec![vec![base.as_slice(); NATIVE_COMMAND6_DAMAGE_STAGES as usize + 1]; 2];
    let (actor, idle_a, idle_b) =
        (command6_sequence_animation(2), command6_sequence_animation(2), command6_sequence_animation(2));
    let sequence = build_native_command6_effect_sequence(&NativeCommand6EffectInput {
        front_base: &base,
        tail_base: &base,
        target_bases,
        transition_bases: vec![&base],
        actor_effect: Some(&actor),
        target_idle: vec![Some(&idle_a), Some(&idle_b)],
        effect: Some(&effect),
        schedule,
        raw_side: 1,
        target_hits: vec![false, false],
        target_numeric_rng: vec![1, 1],
    })
    .unwrap();
    assert!(
        sequence.front.len() == 7
            && sequence.targets.len() == 2
            && sequence.targets[0].frames.len() == 12
            && sequence.transitions.len() == 1
            && sequence.transitions[0].len() == 9
            && sequence.tail.len() == 7,
        "sequence shape front={} targets={} tail={}",
        sequence.front.len(),
        sequence.targets.len(),
        sequence.tail.len()
    );
    // #156：依原始 mode3/mode5 及 12+9 次 counter 推進手算。
    // #162：__CHP向零截斷，首目標channel4=(0,42)；後續frame2加(3,3)。
    for (target, x, y, pixel) in [(0usize, 0usize, 62usize, 5u8), (1, 3, 65, 3), (1, 74, 58, 2)] {
        let got = sequence.targets[target].frames[0][y * 320 + x];
        assert_eq!(got, pixel, "target{target} pixel({x},{y})");
    }
    for (target, frames) in sequence.targets.iter().enumerate() {
        let published = frames.hp_stages.iter().filter(|&&s| s != 0).count() as i64;
        assert_eq!(published, NATIVE_COMMAND6_DAMAGE_STAGES, "target {target} published stages");
    }
}

#[test]
fn test_build_native_command6_effect_sequence_fails_before_partial_output() {
    let effect = command6_sequence_animation(NATIVE_COMMAND6_EFFECT_FRAME_COUNT as usize);
    let schedule = figani::build_native_command6_presentation_schedule(1, Some(&effect)).unwrap_or_default();
    let base = surface();
    let (actor, idle) = (command6_sequence_animation(1), command6_sequence_animation(1));
    let got = build_native_command6_effect_sequence(&NativeCommand6EffectInput {
        front_base: &base,
        tail_base: &base,
        target_bases: vec![vec![&base]],
        actor_effect: Some(&actor),
        target_idle: vec![Some(&idle)],
        effect: Some(&effect),
        schedule,
        raw_side: 1,
        target_hits: vec![false],
        target_numeric_rng: vec![1],
        ..Default::default()
    });
    assert!(got.is_err(), "malformed stage bases accepted: {:?}", got.map(|s| s.front.len()));
}

/// #161 stage marker更新持續base，下一張viewport才顯示新HP背景。
/// 對第三方原版槽的同狀態全圖另由cmd/fd2探針驗，不以此fixture宣稱parity。
#[test]
fn test_native_command6_sequence_retains_previous_hp_base_for_marker_frame() {
    let mut effect = command6_sequence_animation(10);
    for frame in &mut effect.frames {
        frame.y = 20;
    }
    let schedule = figani::build_native_command6_presentation_schedule(0, Some(&effect)).unwrap();
    let base = surface();
    let stages: Vec<Vec<u8>> = (0..6)
        .map(|stage| {
            let mut s = base.clone();
            s[319 + 199 * 320] = 10 + stage as u8;
            s
        })
        .collect();
    let (actor, idle) = (command6_sequence_animation(1), command6_sequence_animation(1));
    let mut in_ = NativeCommand6EffectInput {
        front_base: &base,
        tail_base: &stages[5],
        target_bases: vec![stages.iter().map(Vec::as_slice).collect()],
        actor_effect: Some(&actor),
        target_idle: vec![Some(&idle)],
        effect: Some(&effect),
        schedule,
        raw_side: 0,
        target_hits: vec![true],
        target_numeric_rng: vec![59907],
        ..Default::default()
    };
    let out = build_native_command6_effect_sequence(&in_).unwrap();
    let want = [10u8, 10, 11, 12, 13, 14, 15, 15, 15, 15, 15, 15];
    for (frame, &value) in want.iter().enumerate() {
        assert_eq!(out.targets[0].frames[frame][319 + 199 * 320], value, "frame{frame} base");
    }
    assert_eq!(out.rng_after, 33552, "RNG want33552");
    in_.target_numeric_rng = Vec::new();
    assert!(build_native_command6_effect_sequence(&in_).is_err(), "missing numeric provenance published output");
}
