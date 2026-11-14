//! 移植自 Go `battlepresent/native_command24_prelude_test.go`。

use super::*;
use crate::battlepresent::test_util::{px1, raw_frame};

pub(in crate::battlepresent) fn command24_prelude_dac(value: u8) -> Vec<u8> {
    vec![value; 256 * 3]
}

const SIZE: usize = (NATIVE_COMMAND24_BACKGROUND_WIDTH * NATIVE_COMMAND24_BACKGROUND_HEIGHT) as usize;

fn platform() -> fdother::Frame {
    raw_frame(0, 0, 1, 1, vec![1, 0, 1, 0, 0, 7])
}

#[test]
fn test_build_native_command24_prelude_frames_matches_both29164_branches() {
    let base = vec![1u8; SIZE];
    let idle = px1(100, 10, 9, 0);
    let platform = platform();
    let dac = command24_prelude_dac(60);

    let right = build_native_command24_prelude_frames(&NativeCommand24PreludeInput {
        base: &base,
        actor_idle: idle.clone(),
        raw_side: 0,
        baseline_dac: &dac,
        ..Default::default()
    })
    .unwrap();
    assert!(
        right.len() == 9
            && right[0].stage == 8
            && right[8].stage == 0
            && right[0].pixels[10 * 320 + 20] == 9
            && right[8].pixels[10 * 320 + 100] == 9
            && right[0].dac[0] == 12
            && right[8].dac[0] == 60,
        "right branch does not match 0x29164"
    );

    let left = build_native_command24_prelude_frames(&NativeCommand24PreludeInput {
        base: &base,
        actor_idle: idle,
        platform: Some(&platform),
        raw_side: 2,
        baseline_dac: &dac,
    })
    .unwrap();
    assert!(
        left.len() == 9
            && left[0].pixels[10 * 320 + 180] == 9
            && left[8].pixels[10 * 320 + 100] == 9
            && left[0].pixels[157 * 320 + 244] == 7
            && left[8].pixels[157 * 320 + 164] == 7
            && left[0].dac[0] == 12
            && left[8].dac[0] == 60,
        "left branch does not match 0x29164"
    );
}

#[test]
fn test_build_native_command24_prelude_frames_fails_closed() {
    let base = vec![0u8; SIZE];
    let idle = px1(100, 10, 9, 0);
    let dac = command24_prelude_dac(60);
    let r = build_native_command24_prelude_frames(&NativeCommand24PreludeInput {
        base: &base,
        actor_idle: idle.clone(),
        raw_side: 1,
        baseline_dac: &dac,
        ..Default::default()
    });
    assert!(r.is_err(), "missing nonzero-side TAI accepted: {r:?}");
    let mut bad_dac = command24_prelude_dac(60);
    bad_dac[0] = 64;
    let r = build_native_command24_prelude_frames(&NativeCommand24PreludeInput {
        base: &base,
        actor_idle: idle,
        raw_side: 0,
        baseline_dac: &bad_dac,
        ..Default::default()
    });
    assert!(r.is_err(), "invalid native DAC accepted: {r:?}");
}

#[test]
fn test_build_native_command_prelude_frames_matches29164_mode_zero_both_sides() {
    let base = vec![0u8; SIZE];
    let actor = px1(100, 10, 9, 0);
    let target = px1(40, 20, 8, 0);
    let platform = platform();
    let dac = command24_prelude_dac(60);

    let right = build_native_command_prelude_frames(&NativeCommandPreludeInput {
        base: &base,
        actor_idle: actor.clone(),
        first_target_idle: target.clone(),
        platform: Some(&platform),
        raw_side: 0,
        mode: 0,
        baseline_dac: &dac,
    })
    .unwrap();
    assert!(
        right.len() == 9
            && right[0].pixels[10 * 320 + 20] == 9
            && right[8].pixels[10 * 320 + 100] == 9
            && right[0].pixels[20 * 320 + 40] == 8
            && right[8].pixels[157 * 320 + 164] == 7,
        "mode0 right branch mismatch"
    );

    let left = build_native_command_prelude_frames(&NativeCommandPreludeInput {
        base: &base,
        actor_idle: actor,
        first_target_idle: target,
        platform: Some(&platform),
        raw_side: 2,
        mode: 0,
        baseline_dac: &dac,
    })
    .unwrap();
    assert!(
        left.len() == 9
            && left[0].pixels[10 * 320 + 180] == 9
            && left[8].pixels[10 * 320 + 100] == 9
            && left[0].pixels[20 * 320 + 40] == 8
            && left[0].pixels[157 * 320 + 244] == 7,
        "mode0 left branch mismatch"
    );
}

#[test]
fn test_build_native_command_prelude_frames_mode_zero_fails_without_target_or_tai() {
    let base = vec![0u8; SIZE];
    let actor = px1(1, 1, 9, 0);
    let dac = command24_prelude_dac(60);
    let r = build_native_command_prelude_frames(&NativeCommandPreludeInput {
        base: &base,
        actor_idle: actor.clone(),
        raw_side: 0,
        mode: 0,
        baseline_dac: &dac,
        ..Default::default()
    });
    assert!(r.is_err(), "mode0 accepted missing TAI/target: {r:?}");
    let platform = platform();
    let r = build_native_command_prelude_frames(&NativeCommandPreludeInput {
        base: &base,
        actor_idle: actor,
        platform: Some(&platform),
        raw_side: 0,
        mode: 0,
        baseline_dac: &dac,
        ..Default::default()
    });
    assert!(r.is_err(), "mode0 accepted missing target: {r:?}");
}

#[test]
fn test_command_prelude_original_bottom_row_both_sides() {
    // Go 版以原版 FIGANI.DAT 是否存在決定略過，再讀分離素材包的 animations。
    if crate::test_support::input("FD2_FLAME2_DIR", "FIGANI.DAT").is_none() {
        return;
    }
    let Some(root) = crate::test_support::input("FD2_ASSET_PACK", "animations") else {
        return;
    };
    let tall = figani::load_separated_resource(root.to_str().unwrap(), 51).unwrap();
    let short = figani::load_separated_resource(root.to_str().unwrap(), 309).unwrap();
    for (name, side, actor, target) in [
        ("enemy-target", 0u8, &short.frames[0], &tall.frames[0]),
        ("enemy-actor", 0, &tall.frames[0], &short.frames[0]),
        ("own-target", 2, &short.frames[0], &tall.frames[0]),
        ("own-actor", 2, &tall.frames[0], &short.frames[0]),
    ] {
        let base = vec![9u8; 320 * 200];
        let before = base.clone();
        let mut platform = raw_frame(0, 0, 1, 1, vec![1, 0, 1, 0, 0, 7]);
        let dac = command24_prelude_dac(60);
        let frames = build_native_command_prelude_frames(&NativeCommandPreludeInput {
            base: &base,
            actor_idle: actor.clone(),
            first_target_idle: target.clone(),
            platform: Some(&platform),
            raw_side: side,
            mode: 0,
            baseline_dac: &dac,
        });
        let frames = match frames {
            Ok(f) if f.len() == 9 && f[8].stage == 0 => f,
            other => panic!("{name}: 完整九段未建立：{:?}", other.map(|f| f.len())),
        };
        let mut want = base.clone();
        (platform.x, platform.y) = (164, 157);
        if side != 0 {
            target.blit_at(&mut want, 320).unwrap();
        }
        platform.blit(&mut want, 320, -1).unwrap();
        actor.blit_at(&mut want, 320).unwrap();
        if side == 0 {
            target.blit_at(&mut want, 320).unwrap();
        }
        assert!(
            frames[8].pixels == want && base == before,
            "{name}: 末段640-stride viewport與320-wide完整原生角色畫面不相同，或輸入被改寫"
        );
    }
}
