//! 移植自 Go `battlepresent/native_command3_sequence_test.go`。

use super::*;
use crate::battlepresent::native_command3::native_command3_test::command3_compositor_effect;
use crate::battlepresent::test_util::{anim, px1, surface};

#[test]
fn test_build_native_command3_effect_sequence_prebuilds_every_phase() {
    let effect = command3_compositor_effect();
    let schedule = figani::build_native_command3_presentation_schedule(1, Some(&effect)).unwrap();
    let idle = |pixel: u8| anim(vec![px1(0, 0, pixel, 1)]);
    let base = surface();
    let stages = || vec![base.as_slice(); NATIVE_COMMAND3_DAMAGE_STAGES as usize + 1];
    let (actor, idle4, idle5) = (idle(3), idle(4), idle(5));
    let sequence = build_native_command3_effect_sequence(&NativeCommand3EffectInput {
        front_base: &base,
        tail_base: &base,
        target_bases: vec![stages(), stages()],
        transition_bases: vec![&base],
        actor_effect: Some(&actor),
        target_idle: vec![Some(&idle4), Some(&idle5)],
        effect: Some(&effect),
        schedule,
        raw_side: 1,
    })
    .unwrap();
    assert!(
        sequence.front.len() == 2
            && sequence.targets.len() == 2
            && sequence.targets[0].len() == 40
            && sequence.targets[1].len() == 40
            && sequence.transitions.len() == 1
            && sequence.transitions[0].len() == 9
            && sequence.tail.len() == 20,
        "shape front={} targets={}/{} tail={}",
        sequence.front.len(),
        sequence.targets[0].len(),
        sequence.targets[1].len(),
        sequence.tail.len()
    );
    for (target_index, target) in sequence.targets.iter().enumerate() {
        let stages = target.iter().filter(|f| f.hp_stage != 0).count() as i64;
        assert_eq!(stages, NATIVE_COMMAND3_DAMAGE_STAGES, "target {target_index} stages");
    }
}

#[test]
fn test_build_native_command3_effect_sequence_rejects_late_missing_base() {
    let effect = command3_compositor_effect();
    let schedule = figani::build_native_command3_presentation_schedule(1, Some(&effect)).unwrap_or_default();
    let idle = anim(vec![px1(0, 0, 1, 1)]);
    let base = surface();
    let mut stages = vec![base.as_slice(); NATIVE_COMMAND3_DAMAGE_STAGES as usize + 1];
    *stages.last_mut().unwrap() = &[];
    let got = build_native_command3_effect_sequence(&NativeCommand3EffectInput {
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
