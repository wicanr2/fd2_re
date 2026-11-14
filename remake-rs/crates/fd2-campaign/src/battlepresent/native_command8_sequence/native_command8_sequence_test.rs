//! 移植自 Go `battlepresent/native_command8_sequence_test.go`。

use super::*;
use crate::battlepresent::native_command8::native_command8_test::command8_compositor_effect;
use crate::battlepresent::test_util::{anim, px1, surface};

#[test]
fn test_build_native_command8_effect_sequence_prebuilds_every_phase() {
    let effect = command8_compositor_effect();
    let schedule = figani::build_native_command8_presentation_schedule(1, Some(&effect)).unwrap();
    let idle = |pixel: u8| anim(vec![px1(0, 0, pixel, 1)]);
    let base = surface();
    let stages = || vec![base.as_slice(); NATIVE_COMMAND8_DAMAGE_STAGES as usize + 1];
    let (actor, idle4) = (idle(3), idle(4));
    let sequence = build_native_command8_effect_sequence(&NativeCommand8EffectInput {
        front_base: &base,
        tail_base: &base,
        target_bases: vec![stages()],
        actor_effect: Some(&actor),
        target_idle: vec![Some(&idle4)],
        effect: Some(&effect),
        schedule,
        raw_side: 1,
        ..Default::default()
    })
    .unwrap();
    assert!(
        sequence.front.len() == 3
            && sequence.targets.len() == 1
            && sequence.targets[0].len() == 34
            && sequence.transitions.is_empty()
            && sequence.tail.len() == 2,
        "shape front={} targets={} transition={} tail={}",
        sequence.front.len(),
        sequence.targets[0].len(),
        sequence.transitions.len(),
        sequence.tail.len()
    );
    for (target_index, target) in sequence.targets.iter().enumerate() {
        let stages = target.iter().filter(|f| f.hp_stage != 0).count() as i64;
        assert_eq!(stages, 16, "target {target_index} stages");
    }
}

#[test]
fn test_build_native_command8_effect_sequence_rejects_late_missing_base() {
    let effect = command8_compositor_effect();
    let schedule = figani::build_native_command8_presentation_schedule(1, Some(&effect)).unwrap_or_default();
    let idle = anim(vec![px1(0, 0, 1, 1)]);
    let base = surface();
    let mut stages = vec![base.as_slice(); NATIVE_COMMAND8_DAMAGE_STAGES as usize + 1];
    *stages.last_mut().unwrap() = &[];
    let got = build_native_command8_effect_sequence(&NativeCommand8EffectInput {
        front_base: &base,
        tail_base: &base,
        target_bases: vec![stages],
        actor_effect: Some(&idle),
        target_idle: vec![Some(&idle)],
        effect: Some(&effect),
        schedule,
        raw_side: 1,
        ..Default::default()
    });
    assert!(got.is_err(), "partial sequence accepted: {:?}", got.map(|s| s.front.len()));
}
