//! 移植自 Go `battlepresent/native_command1_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, fig, px1, surface};

fn effect30(value: impl Fn(usize) -> u8) -> Animation {
    Animation { frames: (0..30).map(|i| px1(0, 0, value(i), 0)).collect(), header_byte2: 30, ..Default::default() }
}

#[test]
fn test_compose_native_command1_target_frame_preserves_mode_order() {
    let effect = effect30(|i| i as u8 + 1);
    let mut schedule = figani::build_native_command1_presentation_schedule(1, Some(&effect)).unwrap();
    schedule.anchor_x = 0;
    schedule.x_offsets = [0; 8];
    schedule.y_offsets = [0; 8];
    let target = px1(0, 0, 99, 0);
    let got = compose_native_command1_target_frame(&surface(), &target, Some(&effect), schedule, 8).unwrap();
    // mode5 是最後一層；step8時 slot4 counter0，最後畫 frame0。
    assert_eq!(got[0], 1, "composed pixel want final mode5 frame0");
}

#[test]
fn test_build_native_command1_target_sequence_preserves_thirty_one_frames_and_eight_markers() {
    let effect = effect30(|i| i as u8 + 1);
    let schedule = figani::build_native_command1_presentation_schedule(1, Some(&effect)).unwrap();
    let target = anim(vec![fig(0, 0, 1, 1, vec![99], vec![1], 2)]);
    let bases: Vec<Vec<u8>> = (0..9).map(|_| surface()).collect();
    let base_refs: Vec<&[u8]> = bases.iter().map(Vec::as_slice).collect();
    let sequence = build_native_command1_target_sequence(&base_refs, Some(&target), Some(&effect), schedule).unwrap();
    let (mut hp_steps, mut sample_steps) = (Vec::new(), Vec::new());
    for step in 0..sequence.frames.len() {
        if sequence.hp_stages[step] != 0 {
            hp_steps.push(step);
        }
        if sequence.sample_markers[step] {
            sample_steps.push(step);
        }
    }
    assert!(
        sequence.frames.len() == 31
            && hp_steps == [8, 10, 12, 14, 16, 18, 20, 22]
            && sample_steps == [4, 6, 8, 10, 12, 14, 16, 18],
        "frames={} hp={hp_steps:?} sample={sample_steps:?}",
        sequence.frames.len()
    );
    for (stage, &step) in hp_steps.iter().enumerate() {
        assert_eq!(sequence.hp_stages[step], stage as i64 + 1, "step{step} HP stage");
    }
    assert!(
        sequence.next_idle_frame == 0 && sequence.next_idle_repeat == 1,
        "next idle=({},{}) want=(0,1)",
        sequence.next_idle_frame,
        sequence.next_idle_repeat
    );
}

#[test]
fn test_build_native_command1_transition_frames_uses_nine_recovered_offsets() {
    let actor = anim(vec![px1(0, 0, 7, 0)]);
    let current = anim(vec![px1(0, 0, 8, 0)]);
    let next = anim(vec![px1(0, 0, 9, 0)]);
    let frames =
        build_native_command1_transition_frames(&surface(), Some(&actor), Some(&current), Some(&next), 0).unwrap();
    assert!(
        frames.len() == 9 && frames[0][35] == 8 && frames[3][140] == 8 && frames[4][140] == 9 && frames[8][0] == 9,
        "command1 transition shape/offsets changed"
    );
}

#[test]
fn test_compose_native_command1_target_frame_keeps_thirty_one_frame_tail() {
    let effect = effect30(|i| i as u8 + 1);
    let mut schedule = figani::build_native_command1_presentation_schedule(1, Some(&effect)).unwrap();
    schedule.anchor_x = 0;
    schedule.x_offsets = [0; 8];
    schedule.y_offsets = [0; 8];
    let target = px1(0, 0, 99, 0);
    let got = compose_native_command1_target_frame(&surface(), &target, Some(&effect), schedule, 30).unwrap();
    assert_eq!(got[0], 99, "frame30 pixel want target idle only");
}

#[test]
fn test_compose_native_command1_target_frame_rejects_partial_bank() {
    let mut effect = effect30(|_| 1);
    let schedule = figani::build_native_command1_presentation_schedule(1, Some(&effect)).unwrap_or_default();
    effect.frames.truncate(29);
    assert!(
        compose_native_command1_target_frame(&surface(), &px1(0, 0, 1, 0), Some(&effect), schedule, 0).is_err(),
        "partial command1 bank accepted"
    );
}
