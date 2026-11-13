//! Go `figani_test.go`。

use super::*;

const ONE_FRAME: [u8; 30] =
    [1, 0, 0, 0, 0x7e, 0, 12, 0, 12, 0, 0, 0, 2, 0, 3, 0, 4, 5, 2, 6, 0, 4, 0, 1, 0, 0x00, 7, 0x40, 9, 0xc0];

// Go: TestParsePreservesTransparentAndDitherPixels
#[test]
fn parse_preserves_transparent_and_dither_pixels() {
    // 一張 4x1 幀：run(7)、dither(9)，然後透明略過。
    let a = parse(&ONE_FRAME).unwrap();
    let f = &a.frames[0];
    assert!(
        f.x == 2
            && f.y == 3
            && f.width == 4
            && f.height == 1
            && f.delay == 2
            && f.raw_byte4 == 4
            && f.raw_byte5 == 5
            && f.raw_byte7 == 6,
        "frame={f:?}"
    );
    assert_eq!((a.header_byte1, a.header_byte2, a.header_byte4), (0, 0, 0x7e), "header byte1/byte2/byte4");
    let mut dst = vec![1u8; 50];
    f.blit_at(&mut dst, 10).unwrap();
    assert_eq!(&dst[32..36], &[7, 1, 9, 1], "blit");
}

// Go: TestParseTreatsPreludeFlagAsByteNotFrameCountHighByte
#[test]
fn parse_treats_prelude_flag_as_byte_not_frame_count_high_byte() {
    // 同一筆合法的一幀資源，原生各自消費的 byte1 前奏旗標與 byte2 前奏幀數都設為 1。
    let mut raw = ONE_FRAME;
    raw[1] = 1;
    raw[2] = 1;
    let a = parse(&raw).unwrap();
    assert!(a.frames.len() == 1 && a.header_byte1 == 1 && a.header_byte2 == 1, "animation={a:?}");
}

// Go: TestDecodeOriginalFIGANIResource
#[test]
fn decode_original_figani_resource() {
    let Some(path) = original("FIGANI.DAT") else { return };
    let a = decode_resource(path, 13).unwrap();
    let f = &a.frames[0];
    assert!(f.width > 0 && f.height > 0 && f.pixels.len() as i64 == f.width * f.height, "decoded resource 13");
}

// Go: TestDecodeOriginalFIGANIPreludeResource
#[test]
fn decode_original_figani_prelude_resource() {
    let Some(path) = original("FIGANI.DAT") else { return };
    let a = decode_resource(path, 25).unwrap();
    assert_eq!((a.frames.len(), a.header_byte1, a.header_byte2), (19, 1, 14), "prelude resource");
}

// Go: TestNativeCommand24ScheduleMatchesSelector32Resource98
#[test]
fn native_command24_schedule_matches_selector32_resource98() {
    let Some(path) = original("FIGANI.DAT") else { return };
    let animation = decode_resource(path, 98).unwrap();
    let schedule = build_native_command24_schedule(Some(&animation)).unwrap();
    assert_eq!(
        schedule,
        NativeCommand24Schedule {
            target_start: 9,
            actor_impact_frame: 4,
            target_impact_frame: 10,
            audio_resource: 53,
            actor_sample: 3,
            target_sample: 2,
            shake_offsets: [0, 4, 9, 14, 18, 14],
        }
    );
    let mut bad = animation.clone();
    bad.frames[10].raw_byte4 = 0;
    assert!(build_native_command24_schedule(Some(&bad)).is_err(), "command24 accepted missing target impact marker");
}

// Go: TestNativeCommand29ScheduleMatchesGrowthSelector34Resource104
#[test]
fn native_command29_schedule_matches_growth_selector34_resource104() {
    let Some(path) = original("FIGANI.DAT") else { return };
    let animation = decode_resource(path, 104).unwrap();
    let s = build_native_command_derived_strike_schedule(Some(&animation), 29).unwrap();
    assert!(
        s.command_id == 29
            && s.target_start == 11
            && s.actor_impact_frame == 6
            && s.target_impact_frame == 14
            && s.damage_denominator == 1
            && s.audio_resource == 50
            && s.actor_sample == 1
            && s.target_sample == 4
            && !s.uses_target_base
            && s.prelude_mode == 1
            && s.uses_bg_transition
            && s.frame_delays.len() == 18,
        "command29 schedule={s:?}"
    );
}

// Go: TestNativeCommand28ScheduleKeepsCallerSpecificPresentationBranches
#[test]
fn native_command28_schedule_keeps_caller_specific_presentation_branches() {
    let frame = |delay, raw_byte4, raw_byte5| Frame { delay, raw_byte4, raw_byte5, ..Frame::default() };
    let animation = Animation {
        header_byte1: 1,
        header_byte2: 2,
        header_byte4: 2,
        frames: vec![frame(1, 0, 0), frame(2, 1, 3), frame(4, 0, 0), frame(2, 1, 4)],
    };
    let s = build_native_command_derived_strike_schedule(Some(&animation), 28).unwrap();
    assert!(
        s.damage_denominator == 8
            && s.uses_target_base
            && s.prelude_mode == 0
            && !s.uses_bg_transition
            && s.audio_resource == 49
            && s.actor_sample == 3
            && s.target_sample == 4,
        "command28 schedule={s:?}"
    );
}

// Go: TestNativeCommand28Resource59HasEightImpactsAndOwnCues
#[test]
fn native_command28_resource59_has_eight_impacts_and_own_cues() {
    let Some(pack) = asset_pack() else { return };
    let animation = load_separated_resource(&format!("{pack}/animations"), 59).unwrap();
    let s = build_native_command_derived_strike_schedule(Some(&animation), 28).unwrap();
    assert!(
        animation.header_byte1 == 0
            && s.target_start == 7
            && s.actor_impact_frame == 6
            && animation.frames.len() == 19
            && s.target_impact_frames == [7, 8, 9, 10, 11, 12, 13, 14]
            && s.damage_denominator == 8
            && s.audio_resource == 50,
        "resource59 schedule differs: {s:?}"
    );
    assert!(
        s.actor_cues == [NativeCommandFrameCue { frame: 2, sample: 2 }, NativeCommandFrameCue { frame: 6, sample: 4 }]
            && s.target_cues.len() == 8,
        "resource59 actor or target cues differ"
    );
    for (i, cue) in s.target_cues.iter().enumerate() {
        assert!(cue.frame == 7 + i as i64 && cue.sample == 3 + (i % 2) as i64, "resource59 cue{i}={cue:?}");
    }
    assert!(
        build_native_command_derived_strike_schedule(Some(&animation), 29).is_err(),
        "header0 must not relax command29 contract"
    );
    let mut mutated = animation.clone();
    mutated.frames[15].raw_byte4 = 1;
    assert!(
        build_native_command_derived_strike_schedule(Some(&mutated), 28).is_err(),
        "more than eight impacts accepted"
    );
    let mut mutated = animation.clone();
    mutated.frames[0].raw_byte4 = 1;
    assert!(
        build_native_command_derived_strike_schedule(Some(&mutated), 28).is_err(),
        "duplicate actor MP marker accepted"
    );
}

// Go: TestDecodeOriginalPlayerClass19HeaderFlags
#[test]
fn decode_original_player_class19_header_flags() {
    let Some(path) = original("FIGANI.DAT") else { return };
    // 玩家可達的 class-19 來源用視覺組 4..7（可選轉職）或 20（莎拉的初始職業）。0x2b659 收到 group*3+1。
    for (resource, want) in [(13, 2), (16, 2), (19, 2), (22, 5), (61, 5)] {
        let a = decode_resource(&path, resource).unwrap();
        assert_eq!(a.header_byte4, want, "resource {resource} HeaderByte4");
    }
}

// Go: TestNativeSchedulerInitializesWithoutRenderingAndAdvancesAfterRender
#[test]
fn native_scheduler_initializes_without_rendering_and_advances_after_render() {
    let delays = |d: &[i64]| Animation {
        frames: d.iter().map(|&delay| Frame { delay, ..Frame::default() }).collect(),
        ..Animation::default()
    };
    let a = delays(&[1, 2, 1]);
    let mut s = NativeScheduler::default();
    assert_eq!(s.step(Some(&a), false).unwrap(), (0, false));
    assert_eq!(s.subframe, 0);
    assert_eq!(s.step(Some(&a), true).unwrap(), (0, true));
    assert_eq!((s.frame_index, s.subframe), (1, 0));
    assert_eq!(s.step(Some(&a), true).unwrap().0, 1);
    assert_eq!((s.frame_index, s.subframe), (1, 1));
    assert_eq!(s.step(Some(&a), true).unwrap().0, 1);
    assert_eq!((s.frame_index, s.subframe), (2, 0));
    assert_eq!(s.step(Some(&a), true).unwrap().0, 2);
    assert_eq!(s.frame_index, 0);
}

// Go: TestDisplaySchedulerHonoursNativeDelaysAndScale
#[test]
fn display_scheduler_honours_native_delays_and_scale() {
    let mut s = DisplayScheduler::new(&[1, 2, 1], 2).unwrap();
    assert_eq!(s.body_ticks(), 8);
    for (frame, want) in [0, 0, 1, 1, 1, 1, 2, 2].into_iter().enumerate() {
        let (got, presented, done) = s.step().unwrap();
        assert!(presented && got == want, "tick {frame} frame/present/done={got}/{presented}/{done}");
        assert!(!(frame < 7 && done), "tick {frame} ended the body early");
        assert!(!(frame == 7 && !done), "last body tick did not report done");
    }
    assert_eq!(s.step().unwrap(), (2, false, true), "tail step");
    for (frame, start) in [(0, 0), (1, 2), (2, 6)] {
        assert_eq!(s.frame_start(frame), Some(start), "frame {frame} start");
    }
}

// Go: TestDisplaySchedulerRejectsUnknownDelayState（子測試 empty、negative）
#[test]
fn display_scheduler_rejects_unknown_delay_state() {
    for (name, delays) in [("empty", &[][..]), ("negative", &[1, -1][..])] {
        assert!(DisplayScheduler::new(delays, 1).is_err(), "{name}: invalid delay state was accepted");
    }
    assert!(DisplayScheduler::new(&[1], 0).is_err(), "zero display scale was accepted");
}

// Go: TestDisplaySchedulerPresentsZeroDelayFrameOnce
#[test]
fn display_scheduler_presents_zero_delay_frame_once() {
    let mut s = DisplayScheduler::new(&[1, 0, 2], 1).unwrap();
    assert_eq!(s.body_ticks(), 4);
    for (tick, want) in [0, 1, 2, 2].into_iter().enumerate() {
        let (got, presented, _) = s.step().unwrap();
        assert!(presented && got == want, "tick {tick} frame={got} presented={presented}, want {want}");
    }
}

// Go: TestFrameBlitAtBaseShiftsNativeWorkSurface
#[test]
fn frame_blit_at_base_shifts_native_work_surface() {
    let f = Frame { x: 2, y: 3, ..unit_frame(9) };
    let mut dst = vec![0u8; 640 * 5];
    f.blit_at_base(&mut dst, 640, 80).unwrap();
    assert_eq!(dst[80 + 3 * 640 + 2], 9, "shifted pixel");
}

// Go: TestFrameBlitTranslatedClipsAndCanOverrideOpaquePixels
#[test]
fn frame_blit_translated_clips_and_can_override_opaque_pixels() {
    let f = Frame { width: 3, height: 1, pixels: vec![7, 8, 9], mask: vec![1, 0, 1], ..Frame::default() };
    let mut dst = vec![1u8, 1, 1, 1];
    f.blit_translated(&mut dst, 4, -1, 0, Some(33)).unwrap();
    // 來源 x0 被裁掉；透明的來源 x1 落在 x0 並保留目的地；不透明的來源 x2 落在 x1 並套用覆寫。
    assert_eq!(dst, [1, 33, 1, 1], "translated blit");
}
