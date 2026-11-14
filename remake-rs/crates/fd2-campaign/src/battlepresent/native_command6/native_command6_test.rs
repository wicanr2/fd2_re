//! 移植自 Go `battlepresent/native_command6_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, px1, surface};
use crate::battlepresent::{NativeCommand6EffectInput, build_native_command6_effect_sequence};
use crate::test_support;
use fd2_formats::figani::{self, NATIVE_COMMAND6_DAMAGE_STAGES};

fn no_shader() -> NativeCommand6TargetDisplayFrame {
    NativeCommand6TargetDisplayFrame { shader: -1, ..Default::default() }
}

#[test]
fn test_command6_signed_prefix_requires_exact_resource_and_writer() {
    let Some(path) = test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT") else {
        return;
    };
    let effect = figani::decode_resource(&path, 32);
    assert!(
        matches!(&effect, Ok(e) if command6_bank32_fingerprint(Some(e))),
        "固定資源 typed 指紋不符：{:?}",
        effect.as_ref().err()
    );
    let effect = effect.unwrap();
    let Some(root) = test_support::input("FD2_ASSET_PACK", "animations") else {
        return;
    };
    let separated = figani::load_separated_archive_resource(root.to_str().unwrap(), "FDOTHER.DAT", 32);
    assert!(
        matches!(&separated, Ok(e) if command6_bank32_fingerprint(Some(e))),
        "分離資源 typed 指紋不符：{:?}",
        separated.as_ref().err()
    );
    for name in ["accepted", "pixel", "mask", "header", "delay", "x", "y", "mode", "channel", "primary", "actor"] {
        let mut changed = effect.clone();
        let mut layer = NativeCommand6Layer { mode: 5, channel: 2, frame: 9, x: -19, y: -31, secondary: true };
        match name {
            "pixel" => changed.frames[0].pixels[0] ^= 1,
            "mask" => changed.frames[0].mask[0] ^= 1,
            "header" => changed.header_byte4 ^= 1,
            "delay" => changed.frames[0].delay += 1,
            "x" => layer.x -= 1,
            "y" => layer.y -= 1,
            "mode" => layer.mode = 4,
            "channel" => layer.channel = 4,
            "primary" => layer.secondary = false,
            _ => {}
        }
        let mut work = vec![0u8; 640 * 270];
        let mut prefix = vec![0u8; 640];
        let result = if name == "actor" {
            blit_command0_work_frame(&mut work, &effect.frames[9], -19, -31)
        } else {
            blit_command6_effect_work_frame(&mut work, &mut prefix, &changed, layer)
        };
        if name != "accepted" {
            assert!(
                result.is_err() && work.iter().all(|&b| b == 0) && prefix.iter().all(|&b| b == 0),
                "{name}: 未知寫入獲准或修改了工作緩衝"
            );
            continue;
        }
        assert!(
            result.is_ok() && prefix[218] == effect.frames[9].pixels[77],
            "{name}: 已觀測 -422 寫入未保留：{result:?}"
        );
        prefix[218] = 0;
        assert!(prefix.iter().all(|&b| b == 0), "{name}: 前置列有未觀測寫入");
    }
}

#[test]
fn test_compose_native_command6_target_frame_preserves_mode_order() {
    let effect = Animation {
        frames: (0..10).map(|i| px1(0, 0, i as u8 + 1, 0)).collect(),
        header_byte2: 10,
        ..Default::default()
    };
    let target = px1(0, 0, 99, 0);
    let frame = NativeCommand6TargetFrame {
        mode4: vec![NativeCommand6Layer { mode: 4, channel: 0, frame: 4, ..Default::default() }],
        mode5: vec![NativeCommand6Layer { mode: 5, channel: 0, frame: 5, ..Default::default() }],
        ..Default::default()
    };
    let got =
        compose_native_command6_target_frame(&surface(), &target, &target, Some(&effect), &frame, no_shader()).unwrap();
    assert_eq!(got[0], 6, "composed pixel want final mode5 frame5");
}

#[test]
fn test_compose_native_command6_target_frame_rejects_invalid_layer() {
    let effect = Animation { frames: vec![Frame::default(); 10], header_byte2: 10, ..Default::default() };
    let frame = NativeCommand6TargetFrame {
        mode4: vec![NativeCommand6Layer { frame: 10, ..Default::default() }],
        ..Default::default()
    };
    assert!(
        compose_native_command6_target_frame(
            &surface(),
            &Frame::default(),
            &Frame::default(),
            Some(&effect),
            &frame,
            no_shader()
        )
        .is_err(),
        "out-of-range command6 layer accepted"
    );
}

#[test]
fn test_compose_native_command6_orbit_frame_preserves_layer_order() {
    let mut effect = Animation {
        frames: vec![Frame::default(); NATIVE_COMMAND6_EFFECT_FRAME_COUNT as usize],
        header_byte2: NATIVE_COMMAND6_EFFECT_FRAME_COUNT as u8,
        ..Default::default()
    };
    effect.frames[4] = px1(0, 0, 7, 0);
    let actor = anim(vec![px1(0, 0, 9, 0)]);
    let frame = NativeCommand6OrbitFrame {
        first: vec![NativeCommand6Layer { mode: 1, frame: 4, x: 1, y: 1, ..Default::default() }],
        second: vec![NativeCommand6Layer { mode: 2, frame: 4, x: 2, y: 1, ..Default::default() }],
        ..Default::default()
    };
    let got = compose_native_command6_orbit_frame(&surface(), Some(&actor), None, Some(&effect), &frame).unwrap();
    assert!(got[320 + 1] == 7 && got[320 + 2] == 7, "orbit pixels={},{}", got[320 + 1], got[320 + 2]);
    assert_eq!(got[0], 9, "orbit actor pixel");
}

#[test]
fn test_compose_native_command6_tail_orbit_requires_and_draws_target() {
    let effect = anim(vec![Frame::default(); NATIVE_COMMAND6_EFFECT_FRAME_COUNT as usize]);
    let actor = anim(vec![px1(0, 0, 7, 0)]);
    let target = px1(2, 0, 9, 0);
    let frame = NativeCommand6OrbitFrame { draw_target: true, ..Default::default() };
    assert!(
        compose_native_command6_orbit_frame(&surface(), Some(&actor), None, Some(&effect), &frame).is_err(),
        "tail without target accepted"
    );
    let got = compose_native_command6_orbit_frame(&surface(), Some(&actor), Some(&target), Some(&effect), &frame);
    assert!(matches!(&got, Ok(p) if p[2] == 9), "tail target err={:?}", got.as_ref().err());
}

#[test]
fn test_compose_native_command6_transition_frame_uses_last_actor_and_selected_target() {
    let effect = anim(vec![Frame::default(); NATIVE_COMMAND6_EFFECT_FRAME_COUNT as usize]);
    let actor = anim(vec![px1(0, 0, 1, 0), px1(0, 0, 7, 0)]);
    let target = anim(vec![px1(0, 0, 9, 0)]);
    let got = compose_native_command6_transition_frame(
        &surface(),
        Some(&actor),
        Some(&target),
        Some(&effect),
        &NativeCommand6TransitionFrame { target_offset_x: 2, ..Default::default() },
    )
    .unwrap();
    assert!(got[0] == 7 && got[2] == 9, "transition actor/target pixels={},{}", got[0], got[2]);
}

/// #151：直接解碼固定原始FDOTHER，對照原版0x2A300-byte work配置。
/// 這是資產／compositor回歸，不當作原版一般玩家逐幀收據。
#[test]
fn test_compose_native_command6_real_resources_match_native_work_allocation() {
    let Some(path) = test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT") else {
        return;
    };
    let raw = std::fs::read(&path).unwrap();
    let got: String = Sha256::digest(&raw).iter().map(|b| format!("{b:02x}")).collect();
    assert_eq!(got, "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce", "原版FDOTHER雜湊不符");
    for side in [0u8, 1] {
        let resource = if side == 0 { 33 } else { 32 };
        let effect = figani::decode_resource(&path, resource).unwrap();
        let schedule = figani::build_native_command6_presentation_schedule(side, Some(&effect)).unwrap();
        let points =
            figani::native_command6_target_coordinates(figani::native_command6_coordinates(36, schedule.base_byte), 42);
        let sequence = figani::build_native_command6_target_sequence(schedule, points, side).unwrap();
        let mut target = px1(0, 0, 99, 0);
        let actor = px1(300, 0, 42, 0);
        for (step, planned) in sequence.iter().enumerate() {
            // 獨立參照：用既有FIGANI嚴格blit寫原版大小，依原始viewport擷取。
            let mut work = vec![0u8; 640 + 0x2A300];
            let mut reference_err = None;
            let mut blit = |frame: &Frame, x: i64, y: i64| {
                let mut frame = frame.clone();
                frame.x += 160 + x;
                frame.y += 31 + y;
                if let Err(e) = frame.blit_at(&mut work, 640) {
                    reference_err = Some(e);
                }
            };
            for layer in &planned.mode4 {
                blit(&effect.frames[layer.frame as usize], layer.x, layer.y);
            }
            blit(&actor, 0, 0);
            blit(&target, 0, 0);
            for layer in &planned.mode5 {
                blit(&effect.frames[layer.frame as usize], layer.x, layer.y);
            }
            let mut expected = vec![0u8; 320 * 200];
            for y in 0..200 {
                let start = 640 + 0x4BA0 + y * 640;
                expected[y * 320..(y + 1) * 320].copy_from_slice(&work[start..start + 320]);
            }
            let got =
                compose_native_command6_target_frame(&surface(), &actor, &target, Some(&effect), planned, no_shader());
            if reference_err.is_some() {
                // 未知範圍仍零發布；固定#32的已證前置列由獨立BlitAt參照保留。
                assert!(got.is_err(), "原版resource{resource} step{step}未知邊界被發布");
                continue;
            }
            let got = got.unwrap_or_else(|e| panic!("原版resource{resource} side{side} target step{step}：{e}"));
            for (i, &value) in expected.iter().enumerate() {
                assert_eq!(got[i], value, "resource{resource} step{step} viewport pixel{i}");
            }
        }
        // 完整兩目標 owner：兩個固定原始資源均須可預建。
        let base = surface();
        let stages = vec![base.as_slice(); NATIVE_COMMAND6_DAMAGE_STAGES as usize + 1];
        target.delay = 1;
        let idle = anim(vec![target.clone()]);
        let all = build_native_command6_effect_sequence(&NativeCommand6EffectInput {
            front_base: &base,
            tail_base: &base,
            target_bases: vec![stages.clone(), stages],
            transition_bases: vec![&base],
            actor_effect: Some(&idle),
            target_idle: vec![Some(&idle), Some(&idle)],
            effect: Some(&effect),
            schedule,
            raw_side: side,
            target_hits: vec![false, false],
            target_numeric_rng: vec![1, 1],
        });
        assert!(
            matches!(&all, Ok(a) if a.targets.len() == 2 && a.targets[1].frames.len() == 12 && a.transitions.len() == 1),
            "原始#{resource}完整兩目標預建失敗：{:?}",
            all.as_ref().err()
        );
        assert!(base.iter().all(|&p| p == 0), "原始資產預建改寫caller base");
    }
}

#[test]
fn test_native_command_work_bounds_match_allocation() {
    let mut work = vec![0u8; 0x2A300];
    let last = px1(479, 239, 7, 0);
    blit_command0_work_frame(&mut work, &last, 0, 0).unwrap();
    assert_eq!(*work.last().unwrap(), 7, "原版work最後一byte未寫入");
    for frame in [px1(479, 240, 9, 0), px1(480, 239, 9, 0), Frame { mask: vec![], ..px1(479, 239, 9, 0) }] {
        assert!(blit_command0_work_frame(&mut work, &frame, 0, 0).is_err(), "真正越界或malformed影格未拒收");
        assert_eq!(*work.last().unwrap(), 7, "拒收影格發布部分像素");
    }
}
