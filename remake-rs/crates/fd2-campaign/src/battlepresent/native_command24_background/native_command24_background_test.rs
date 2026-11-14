//! 移植自 Go `battlepresent/native_command24_background_test.go`。

use super::*;
use crate::battlepresent::test_util::{px1, raw_frame};

pub(in crate::battlepresent) fn solid_native_command24_layer(value: u8) -> fdother::Frame {
    let mut pixels = Vec::with_capacity(4 + 100 * 10);
    pixels.extend_from_slice(&320u16.to_le_bytes());
    pixels.extend_from_slice(&100u16.to_le_bytes());
    for _ in 0..100 {
        for _ in 0..5 {
            pixels.extend_from_slice(&[63, value]);
        }
    }
    raw_frame(0, 0, 320, 100, pixels)
}

fn layers3(a: u8, b: u8, c: u8) -> [fdother::Frame; 3] {
    [solid_native_command24_layer(a), solid_native_command24_layer(b), solid_native_command24_layer(c)]
}

#[test]
fn test_native_background_presents_keep_metadata_and_immutable_history() {
    let (source, target) = (vec![0u8; 64000], vec![0u8; 64000]);
    let in_ = NativeCommand24BackgroundInputs {
        layers: layers3(10, 20, 30),
        source: &source,
        target: &target,
        target_idle: px1(0, 0, 99, 0),
    };
    for right in [false, true] {
        type Build = fn(&NativeCommand24BackgroundInputs) -> Result<Vec<NativeCommand24BackgroundPresent>>;
        type Legacy = fn(&NativeCommand24BackgroundInputs) -> Result<Vec<Vec<u8>>>;
        let (build, legacy, first, content): (Build, Legacy, i64, i64) = if right {
            (build_native_physical_right_background_presents, build_native_physical_right_background_frames, 9, 0)
        } else {
            (build_native_command24_background_presents, build_native_command24_background_frames, 10, 320)
        };
        let mut presents = build(&in_).unwrap();
        let frames = legacy(&in_);
        assert!(
            matches!(&frames, Ok(f) if f.len() == presents.len()) && presents.len() as i64 == first + 10,
            "counts: {:?}",
            frames.as_ref().err()
        );
        let frames = frames.unwrap();
        for (i, present) in presents.iter().enumerate() {
            let i = i as i64;
            let incoming = i >= first;
            let (mut step, mut cx, mut viewport, mut layer) = (i, content, 32 * (9 - i), (9 - i) % 3);
            if right {
                (viewport, layer) = (32 * (i + 1), (i + 1) % 3);
            }
            if incoming {
                (step, cx) = (i - first, 320 - content);
                (viewport, layer) = (32 * (9 - step), (11 - step) % 3);
                if right {
                    (viewport, layer) = (32 * (step + 1), (step + 2) % 3);
                }
            }
            assert!(
                present.incoming == incoming
                    && present.content_x == cx
                    && present.bgx == 320 - cx
                    && present.viewport_x == viewport
                    && present.bg_layers.len() as i64 == step + 1
                    && present.bg_layers[step as usize] == layer
                    && frames[i as usize] == present.pixels,
                "right{right} present{i} metadata differs"
            );
        }
        presents[0].bg_layers[0] = 99;
        presents[0].pixels[0] = 99;
        assert!(presents[1].bg_layers[0] != 99 && presents[1].pixels[0] != 99, "present history or pixels alias");
    }
}

#[test]
fn test_build_native_command24_background_frames_matches29_c90_viewports() {
    let (source, target) = (vec![40u8; 320 * 200], vec![50u8; 320 * 200]);
    let frames = build_native_command24_background_frames(&NativeCommand24BackgroundInputs {
        layers: layers3(10, 20, 30),
        source: &source,
        target: &target,
        target_idle: px1(10, 10, 99, 0),
    })
    .unwrap();
    assert_eq!(frames.len(), 20, "frame count want 20");
    let row = 50 * 320;
    assert!(
        frames[0][row] == 10 && frames[0][row + 31] == 10 && frames[0][row + 32] == 40,
        "first slide boundary={:?}",
        &frames[0][row..row + 34]
    );
    assert!(frames[9][row] == 10 && frames[9][row + 319] == 10, "first slide did not end on BG resource0");
    assert!(
        frames[10][row] == 50 && frames[10][row + 31] == 50 && frames[10][row + 32] == 30,
        "second slide boundary={:?}",
        &frames[10][row..row + 34]
    );
    assert!(
        frames[19][10 + 10 * 320] == 99 && frames[19][319] == 50,
        "second slide did not end on target base plus idle frame"
    );
}

#[test]
fn test_build_native_command24_background_frames_rejects_before_partial_output() {
    let (source, target) = (vec![0u8; 320 * 200], vec![0u8; 320 * 200]);
    let r = build_native_command24_background_frames(&NativeCommand24BackgroundInputs {
        layers: [solid_native_command24_layer(1), solid_native_command24_layer(2), fdother::Frame::default()],
        source: &source,
        target: &target,
        target_idle: px1(0, 0, 1, 0),
    });
    assert!(r.is_err(), "malformed BG layer was accepted");
}

#[test]
fn test_native_physical_right_background_keeps29_ded_asymmetry() {
    let (source, target) = (vec![40u8; 320 * 200], vec![50u8; 320 * 200]);
    let mut in_ = NativeCommand24BackgroundInputs {
        layers: layers3(10, 20, 30),
        source: &source,
        target: &target,
        target_idle: px1(10, 10, 99, 0),
    };
    let frames = build_native_physical_right_background_frames(&in_).unwrap();
    assert_eq!(frames.len(), 19, "29DED presents; original is 9+10");
    let row = 50 * 320;
    // 第一段 i=1：offset32，因此前288 pixels仍是source，末32為BG1。
    assert!(
        frames[0][row + 287] == 40 && frames[0][row + 288] == 20 && frames[0][row + 319] == 20,
        "29DED first source boundary or BG1 changed"
    );
    // 第一段沒有offset320；最後仍有32 pixels的source。
    assert!(frames[8][row + 31] == 40 && frames[8][row + 32] == 10, "29DED unexpectedly gained a tenth first slide");
    assert!(
        frames[9][row + 287] == 30
            && frames[9][row + 288] == 50
            && frames[18][10 + 10 * 320] == 99
            && frames[18][319] == 50,
        "29DED target viewport or idle origin changed"
    );
    in_.layers[2] = fdother::Frame::default();
    assert!(
        build_native_physical_right_background_frames(&in_).is_err(),
        "malformed physical scroll returned a partial schedule"
    );
}
