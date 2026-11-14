//! 移植自 Go `battlepresent/native_command9_test.go`。

use super::*;
use crate::battlepresent::test_util::{fig, surface};

fn command9_present_animation(count: usize, delay: i64) -> Animation {
    Animation {
        frames: (0..count).map(|i| fig(1, 1, 1, 1, vec![i as u8 + 1], vec![1], delay)).collect(),
        header_byte2: count as u8,
        ..Default::default()
    }
}

/// 用獨立320×200遮罩覆寫器重組來源，避免以同一work-buffer helper自證。
#[test]
fn test_native_command9_sources_recompose119_frames() {
    let (mut actor, mut target, mut effect) =
        (command9_present_animation(1, 1), command9_present_animation(3, 2), command9_present_animation(31, 0));
    for animation in [&mut actor, &mut target, &mut effect] {
        for (i, f) in animation.frames.iter_mut().enumerate() {
            (f.x, f.y, f.width, f.height) = (110, 55, 4, 2);
            (f.pixels, f.mask) = (vec![i as u8 + 40; 8], vec![1, 0, 1, 1, 1, 1, 0, 1]);
        }
    }
    actor.frames[0].pixels = vec![200; 8];
    for (i, f) in target.frames.iter_mut().enumerate() {
        f.pixels = vec![i as u8 + 100; 8];
    }
    effect.frames[30].mask = vec![0; 8];
    let bases: Vec<Vec<u8>> = (0..21).map(|i| vec![i as u8 + 1; 64000]).collect();
    let schedule = figani::build_native_command9_ai_schedule(0, Some(&effect)).unwrap();
    let seq = build_native_command9_ai_sequence(&NativeCommand9AIInput {
        before_base: &bases[0],
        after_base: &bases[20],
        target_bases: bases.iter().map(Vec::as_slice).collect(),
        actor: Some(&actor),
        target: Some(&target),
        effect: Some(&effect),
        schedule,
    })
    .unwrap();
    let (mut count, mut stages, mut sub1, mut sub2, mut blank) = (0, 0, 0, 0, 0);
    for group in [&seq.slide_in, &seq.front, &seq.target, &seq.tail, &seq.slide_out] {
        for frame in group {
            let s = &frame.sources;
            let mut out = s.base.clone();
            let mut blit = |f: &Frame, dx: i64| {
                for y in 0..f.height {
                    for x in 0..f.width {
                        let (at, px, py) = ((y * f.width + x) as usize, f.x + x + dx, f.y + y);
                        if f.mask[at] != 0 && (0..320).contains(&px) && (0..200).contains(&py) {
                            out[(py * 320 + px) as usize] = f.pixels[at];
                        }
                    }
                }
            };
            for e in &s.before {
                blit(&effect.frames[e.frame as usize], e.x);
            }
            if s.actor_frame >= 0 {
                blit(&actor.frames[s.actor_frame as usize], s.actor_offset_x);
            }
            blit(&target.frames[s.target_frame as usize], s.target_offset_x);
            for e in &s.after {
                blit(&effect.frames[e.frame as usize], e.x);
                if e.frame == 30 {
                    blank += 1;
                }
            }
            assert!(out == frame.pixels && !s.phase.is_empty(), "frame {count} indexed sources differ");
            if frame.hp_stage != 0 {
                stages += 1;
            }
            if frame.play_sub1 {
                sub1 += 1;
            }
            if frame.play_sub2 {
                sub2 += 1;
            }
            count += 1;
        }
    }
    assert!(
        count == 119
            && stages == 20
            && sub1 == 1
            && sub2 == 1
            && blank == 1
            && seq.slide_in[10].sources.actor_frame == -1
            && seq.slide_in[9].sources.actor_offset_x == -90
            && seq.slide_out[0].sources.actor_offset_x == -70
            && seq.slide_out[7].sources.actor_offset_x == 0,
        "sources changed phase/stage/sound/blank/slide: {count}/{stages}/{sub1}/{sub2}/{blank}"
    );
}

#[test]
fn test_build_native_command9_ai_sequence_preserves_all_phases() {
    let effect = command9_present_animation(31, 1);
    let schedule = figani::build_native_command9_ai_schedule(0, Some(&effect)).unwrap();
    let bases: Vec<Vec<u8>> = (0..=NATIVE_COMMAND9_AI_DAMAGE_STAGES).map(|_| surface()).collect();
    let (actor, target) = (command9_present_animation(1, 1), command9_present_animation(2, 1));
    let seq = build_native_command9_ai_sequence(&NativeCommand9AIInput {
        before_base: &bases[0],
        after_base: &bases[bases.len() - 1],
        target_bases: bases.iter().map(Vec::as_slice).collect(),
        actor: Some(&actor),
        target: Some(&target),
        effect: Some(&effect),
        schedule,
    })
    .unwrap();
    assert!(
        seq.slide_in.len() == 11
            && seq.front.len() == 20
            && seq.target.len() == 60
            && seq.tail.len() == 20
            && seq.slide_out.len() == 8,
        "sequence lengths={}/{}/{}/{}/{}",
        seq.slide_in.len(),
        seq.front.len(),
        seq.target.len(),
        seq.tail.len(),
        seq.slide_out.len()
    );
    let stages = seq.target.iter().filter(|f| f.hp_stage != 0).count();
    assert_eq!(stages, 20, "HP stages");
}

#[test]
fn test_build_native_command9_ai_sequence_rejects_missing_stage() {
    let effect = command9_present_animation(31, 1);
    let schedule = figani::build_native_command9_ai_schedule(0, Some(&effect)).unwrap_or_default();
    let (before, after) = (surface(), surface());
    let (actor, target) = (command9_present_animation(1, 1), command9_present_animation(1, 1));
    let empty: &[u8] = &[];
    assert!(
        build_native_command9_ai_sequence(&NativeCommand9AIInput {
            before_base: &before,
            after_base: &after,
            target_bases: vec![empty; 20],
            actor: Some(&actor),
            target: Some(&target),
            effect: Some(&effect),
            schedule,
        })
        .is_err(),
        "missing HP stage was accepted"
    );
}
