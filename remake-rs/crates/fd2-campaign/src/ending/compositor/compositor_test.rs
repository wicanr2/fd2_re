//! 移植自 `remake/internal/ending/compositor_test.go`（含 Player 與 figure fade 的測試）。
//!
//! `fdother::Frame` 有 fd2-formats 私有欄位，crate 外不能用結構字面值，測試以 [`frame`] 逐欄設定。
//! Go 的 `*IndexedCompositor` 在 Rust 是 [`SharedCompositor`]，讀狀態前先 `lock()`。

use super::*;
use crate::test_support::{input, remake_asset};

/// `fdother.Frame{X, Y, Width, Height, Pixels}`。
pub(crate) fn frame(x: i64, y: i64, width: i64, height: i64, pixels: &[u8]) -> fdother::Frame {
    let mut f = fdother::Frame::default();
    (f.x, f.y, f.width, f.height) = (x, y, width, height);
    f.pixels = pixels.to_vec();
    f
}

/// Go 測試常用的 9 格合成影格：`frames[i] = {Width:1, Height:1, Pixels:{1,0,1,0,0,i}}`。
pub(crate) fn composite_frames() -> Vec<fdother::Frame> {
    let mut frames = vec![fdother::Frame::default(); 9];
    for (i, f) in frames.iter_mut().enumerate().skip(1) {
        *f = frame(0, 0, 1, 1, &[1, 0, 1, 0, 0, i as u8]);
    }
    frames
}

pub(crate) fn shared(c: IndexedCompositor) -> SharedCompositor {
    std::sync::Arc::new(std::sync::Mutex::new(c))
}

fn clip(frames: usize) -> afm::Clip {
    // `afm::Clip` 沒有 `Default`；Go 測試只設兩個欄位，其餘為零值。
    afm::Clip {
        title: Vec::new(),
        header_frames: 0,
        frames: Vec::new(),
        indexed_frames: vec![vec![0; BYTES as usize]; frames],
        palettes: vec![vec![0; 768]; frames],
    }
}

fn seg(op: &str, source: &str) -> Segment {
    Segment { op: op.into(), source: source.into(), ..Default::default() }
}

pub(crate) fn transparent_tai003_fixture() -> fdother::Frame {
    let mut f = fdother::Frame::default();
    (f.width, f.height) = (10, 3);
    (f.indexed, f.mask) = (vec![0; 30], vec![0; 30]);
    f
}

#[test]
fn test_indexed_compositor_copies_blits_and_clamps_palette() {
    let mut c = new_indexed_compositor();
    let mut offscreen = std::mem::take(&mut c.offscreen);
    c.blit(&frame(1, 1, 2, 1, &[2, 0, 1, 0, 1, 9]), &mut offscreen, WIDTH).unwrap();
    c.copy_to_vga(&offscreen).unwrap();
    c.offscreen = offscreen;
    let w = WIDTH as usize;
    assert!(c.vga[w + 1] == 9 && c.vga[w + 2] == 9, "blit/copy={:?}", &c.vga[w..w + 4]);
    (c.baseline[3], c.baseline[4], c.baseline[5]) = (2, 62, 63);
    c.baseline_known = true;
    c.palette_delta(1, 1, 4).unwrap();
    assert_eq!(c.palette[3..6], [6, 63, 63], "palette");
}

#[test]
fn test_composite40_uses_native_viewport_and_offsets() {
    let mut c = new_indexed_compositor();
    c.offscreen[0] = 3;
    c.composite40(&composite_frames(), 0).unwrap();
    assert!(
        c.vga[0] == 3 && c.work[290] == 1 && c.work[80] == 5,
        "viewport={} primary={} secondary={}",
        c.vga[0],
        c.work[290],
        c.work[80]
    );
}

#[test]
fn test_composite200_uses_baseline_palette() {
    let mut c = new_indexed_compositor();
    c.offscreen[1] = 3;
    c.baseline[0] = 1;
    c.composite200(&composite_frames(), 136).unwrap();
    assert!(c.vga[1] == 3 && c.palette[0] == 0, "vga={} palette={}", c.vga[1], c.palette[0]);
}

#[test]
fn test_recovered_prefix_stops_at_native_only_gate() {
    let mut c = new_indexed_compositor();
    let timeline = Timeline {
        segments: vec![
            Segment {
                frame: Some(0),
                target: "offscreen".into(),
                stride: WIDTH,
                transparent: Some(-1),
                ..seg("blit_frame", "test")
            },
            Segment { bytes: BYTES, from: "offscreen".into(), to: "vga".into(), ..seg("copy_buffer", "test") },
            seg("native_call_opaque", "gate"),
        ],
        ..Default::default()
    };
    let frames = [frame(0, 0, 1, 1, &[1, 0, 1, 0, 0, 7])];
    // Go 同時回傳停下的段索引 2；Rust 的錯誤不帶索引，改核對錯誤訊息指向第 2 段（gate）。
    let result = c.run_recovered_prefix(&timeline, &frames);
    assert!(
        result.as_ref().is_err_and(|err| err.message() == "ending: native-only segment native_call_opaque at gate")
            && c.vga[0] == 7,
        "result={result:?} pixel={}",
        c.vga[0]
    );
}

#[test]
fn test_present_ani_replaces_indexed_vga_and_palette() {
    let mut c = new_indexed_compositor();
    let (mut f, mut palette) = (vec![0u8; BYTES as usize], vec![0u8; 768]);
    (f[7], palette[9]) = (42, 63);
    c.present_ani(&f, &palette).unwrap();
    assert!(c.vga[7] == 42 && c.palette[9] == 63, "ANI state not presented");
}

#[test]
fn test_rgba_converts_six_bit_dac() {
    let mut c = new_indexed_compositor();
    c.vga[0] = 1;
    (c.palette[3], c.palette[4], c.palette[5]) = (63, 32, 0);
    let p = c.rgba().pix;
    assert_eq!(p[..4], [255, 130, 0, 255], "rgba");
}

#[test]
fn test_present_ani_frame_keeps_frame_palette_pair() {
    let mut c = new_indexed_compositor();
    let mut clip = clip(1);
    (clip.indexed_frames[0][3], clip.palettes[0][4]) = (11, 12);
    let err = c.present_ani_frame(Some(&clip), 0);
    assert!(err.is_ok() && c.vga[3] == 11 && c.palette[4] == 12, "err={err:?}");
}

#[test]
fn test_ani_player_uses_exact_millisecond_cadence() {
    let mut c = new_indexed_compositor();
    let mut clip = clip(2);
    (clip.indexed_frames[0][0], clip.indexed_frames[1][0]) = (1, 2);
    let mut p = ANIPlayer { clip: Some(clip), delay_ms: 100, ..Default::default() };
    assert!(matches!(p.advance(&mut c, 0), Ok(false)) && c.vga[0] == 1, "first frame must present immediately");
    assert!(matches!(p.advance(&mut c, 99), Ok(false)) && c.vga[0] == 1, "advanced too early");
    assert!(matches!(p.advance(&mut c, 1), Ok(true)) && c.vga[0] == 2, "last frame");
}

fn advance(p: &mut Player, ms: i64) -> (PlaybackState, String) {
    match p.advance(ms) {
        Ok(state) => (state, String::new()),
        Err(err) => (p.state, err.to_string()),
    }
}

#[test]
fn test_player_plays_recovered_prefix_then_blocks_at_palette_ramp() {
    let timeline = Timeline {
        segments: vec![
            Segment {
                frame: Some(0),
                target: "offscreen".into(),
                stride: WIDTH,
                transparent: Some(-1),
                ..seg("blit_frame", "blit")
            },
            Segment { bytes: BYTES, from: "offscreen".into(), to: "vga".into(), ..seg("copy_buffer", "copy") },
            Segment { ms: 1000, ..seg("delay_ms", "wait") },
            Segment { ani_resource: Some(2), frame_delay_ms: 100, skippable: Some(false), ..seg("ani_play", "ani") },
            Segment {
                palette_start: Some(0),
                palette_end: Some(0),
                palette_value: Some(2),
                ..seg("palette_update", "palette")
            },
            Segment {
                palette_start: Some(2),
                palette_end: Some(0),
                palette_step: -1,
                palette_delay: 4,
                ..seg("palette_ramp", "ramp")
            },
            seg("native_text_branch_opaque", "unrecovered"),
        ],
        ..Default::default()
    };
    let frames = [frame(0, 0, 1, 1, &[1, 0, 1, 0, 0, 7])];
    let mut clip = clip(2);
    (clip.indexed_frames[0][0], clip.indexed_frames[1][0]) = (8, 9);
    let c = shared(new_indexed_compositor());
    let mut p = new_player(timeline, &frames, Some(&clip), Some(c.clone())).unwrap();
    let vga0 = |c: &SharedCompositor| c.lock().unwrap().vga[0];
    let ok = |r: (PlaybackState, String), want: PlaybackState| r.1.is_empty() && r.0 == want;
    let r = advance(&mut p, 0);
    assert!(ok(r.clone(), PLAYBACK_RUNNING) && vga0(&c) == 7, "initial state={r:?} pixel={}", vga0(&c));
    let r = advance(&mut p, 999);
    assert!(ok(r.clone(), PLAYBACK_RUNNING) && vga0(&c) == 7, "early delay state={r:?} pixel={}", vga0(&c));
    let r = advance(&mut p, 1);
    assert!(ok(r.clone(), PLAYBACK_RUNNING) && vga0(&c) == 8, "ANI first state={r:?} pixel={}", vga0(&c));
    let r = advance(&mut p, 100);
    assert!(ok(r.clone(), PLAYBACK_RUNNING) && vga0(&c) == 9, "ANI final state={r:?} pixel={}", vga0(&c));
    let r = advance(&mut p, 0);
    assert!(ok(r.clone(), PLAYBACK_RUNNING), "ramp begin state={r:?}");
    let r = advance(&mut p, 4);
    assert!(ok(r.clone(), PLAYBACK_RUNNING), "ramp second state={r:?}");
    let r = advance(&mut p, 4);
    assert!(ok(r.clone(), PLAYBACK_RUNNING), "ramp third state={r:?}");
    let r = advance(&mut p, 4);
    assert!(
        ok(r.clone(), PLAYBACK_BLOCKED) && p.blocked.as_ref().is_some_and(|b| b.source == "unrecovered"),
        "blocked state={r:?} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_player_runs_recovered_native_prefix_with_player_assets() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(ani_path) = input("FD2_FLAME2_DIR", "ANI.DAT") else { return };
    let Some(timeline_path) = remake_asset("endings/native_2bce5.json") else { return };
    let timeline = load_timeline(&timeline_path).unwrap();
    let frames = fdother::decode_resource(&fdother_path, timeline.resource.index).unwrap();
    let clip = afm::decode_resource(&ani_path, 2).unwrap();
    assert!(
        frames.len() == 111 && clip.indexed_frames.len() == 26,
        "ending resources frames={} ani={}",
        frames.len(),
        clip.indexed_frames.len()
    );
    let mut p = new_player(timeline, &frames, Some(&clip), Some(shared(new_indexed_compositor()))).unwrap();
    for (ms, label) in [(0, "setup"), (1000, "first wait"), (2500, "ANI"), (0, "palette setup"), (256, "palette ramp")]
    {
        let r = advance(&mut p, ms);
        assert!(r.1.is_empty() && r.0 == PLAYBACK_RUNNING, "{label} state={r:?}");
    }
    let r = advance(&mut p, 2000);
    assert!(
        r.1.is_empty()
            && r.0 == PLAYBACK_BLOCKED
            && p.blocked.as_ref().is_some_and(|b| b.op == "native_text_branch_opaque"),
        "prefix gate state={r:?} blocked={:?}",
        p.blocked
    );
    assert!(
        p.blocked.as_ref().is_some_and(|b| b.source == "0x2be44") && p.resume_blocked_dialogue(),
        "first native text gate did not resume: {:?}",
        p.blocked
    );
    let r = advance(&mut p, 5000);
    assert!(
        r.1.is_empty()
            && r.0 == PLAYBACK_BLOCKED
            && p.blocked.as_ref().is_some_and(|b| b.op == "native_text_branch_opaque" && b.source == "0x2bf1c"),
        "second text gate state={r:?} blocked={:?}",
        p.blocked
    );
    assert!(p.resume_blocked_dialogue(), "second native text gate did not resume");
    let r = advance(&mut p, 7000);
    assert!(
        r.1.is_empty()
            && r.0 == PLAYBACK_BLOCKED
            && p.blocked.as_ref().is_some_and(|b| b.op == "native_post_composite_opaque" && b.source == "0x2c172"),
        "post-composite gate state={r:?} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_ending_timeline_preserves_verified_audio_cue_order() {
    let Some(path) = remake_asset("endings/native_2bce5.json") else { return };
    let timeline = load_timeline(&path).unwrap();
    assert_eq!(timeline.audio_cues.len(), 3, "audio cues={:?}", timeline.audio_cues);
    let cue = |source: &str, track: i64, driver_arg: i64, stage: &str, gate: &str, trigger: &str| AudioCue {
        source: source.into(),
        track,
        driver_arg,
        runtime_stage: stage.into(),
        after_gate: gate.into(),
        trigger: trigger.into(),
    };
    let want = [
        cue("0x2c5cf", 4, 0, "party_cycle_start", "0x2c548", "0x2c405 after phase0 before party cycle"),
        cue("0x2c1ac", -1, 1, "tail_stop", "", "0x2bce5 after 0x2c405 returns before FDOTHER#60"),
        cue("0x2c1f5", 18, 0, "tail_start", "", "0x2bce5 after FDOTHER#60 frame before post-montage loop"),
    ];
    for (i, want) in want.iter().enumerate() {
        assert_eq!(timeline.audio_cues[i], *want, "audio cue {i}");
    }
    let mut p = new_player(timeline, &[], None, Some(shared(new_indexed_compositor()))).unwrap();
    let mut got = p.verified_audio_cues();
    got[0].track = 99;
    assert_eq!(p.verified_audio_cues()[0].track, 4, "audio cue accessor exposed mutable timeline state");
    let (c, ok) = p.audio_cue_for_runtime_stage("tail_stop");
    assert!(ok && c == want[1], "tail stop cue={c:?} ok={ok}");
    let (c, ok) = p.audio_cue_for_runtime_stage("tail_start");
    assert!(ok && c == want[2], "tail start cue={c:?} ok={ok}");
    let (c, ok) = p.audio_cue_for_runtime_stage("unknown");
    assert!(!ok && c == AudioCue::default(), "unknown runtime stage leaked cue={c:?} ok={ok}");
    p.state = PLAYBACK_BLOCKED;
    p.blocked = Some(seg("native_finale_montage_opaque", "0x2c548"));
    let (c, ok) = p.audio_cue_at_blocked_boundary();
    assert!(ok && c.track == 4 && c.driver_arg == 0 && c.after_gate == "0x2c548", "blocked cue={c:?} ok={ok}");
    p.blocked.as_mut().unwrap().source = "0x2c172".into();
    let (c, ok) = p.audio_cue_at_blocked_boundary();
    assert!(!ok && c == AudioCue::default(), "unrecovered boundary leaked cue={c:?} ok={ok}");
}

#[test]
fn test_blocked_dialogue_selects_only_native_text_branch() {
    let block = |portrait_id: i64| DialogueBlock { portrait_id, ..Default::default() };
    let mut p = Player {
        state: PLAYBACK_BLOCKED,
        blocked: Some(Segment {
            then_dialogue: vec![block(4)],
            else_dialogue: vec![block(37)],
            ..seg("native_text_branch_opaque", "")
        }),
        ..Default::default()
    };
    let (blocks, ok) = p.blocked_dialogue(26);
    assert!(ok && blocks.len() == 1 && blocks[0].portrait_id == 4, "chapter26 blocks={blocks:?} ok={ok}");
    let (blocks, ok) = p.blocked_dialogue(29);
    assert!(ok && blocks.len() == 1 && blocks[0].portrait_id == 37, "final blocks={blocks:?} ok={ok}");
    p.blocked.as_mut().unwrap().op = "native_composite_loop_opaque".into();
    let (blocks, ok) = p.blocked_dialogue(29);
    assert!(!ok && blocks.is_empty(), "non-text opaque block leaked dialogue: {blocks:?}");
    assert!(!p.resume_blocked_dialogue(), "non-text opaque block resumed");
    p.blocked.as_mut().unwrap().op = "native_text_branch_opaque".into();
    assert!(
        p.resume_blocked_dialogue() && p.state == PLAYBACK_RUNNING && p.blocked.is_none() && p.segment == 1,
        "text resume state={:?} blocked={:?} segment={}",
        p.state,
        p.blocked,
        p.segment
    );
}

#[test]
fn test_player_can_resume_two_recovered_native_text_gates() {
    let block = |portrait_id: i64| DialogueBlock { portrait_id, ..Default::default() };
    let timeline = Timeline {
        segments: vec![
            Segment { else_dialogue: vec![block(37)], ..seg("native_text_branch_opaque", "first") },
            Segment { else_dialogue: vec![block(45)], ..seg("native_text_branch_opaque", "second") },
            seg("still_opaque", "stop"),
        ],
        ..Default::default()
    };
    let mut p = new_player(timeline, &[], None, Some(shared(new_indexed_compositor()))).unwrap();
    for (source, resumes) in [("first", true), ("second", true), ("stop", false)] {
        let r = advance(&mut p, 0);
        assert!(
            r.1.is_empty() && r.0 == PLAYBACK_BLOCKED && p.blocked.as_ref().is_some_and(|b| b.source == source),
            "{source} gate state={r:?} blocked={:?}",
            p.blocked
        );
        assert_eq!(p.resume_blocked_dialogue(), resumes, "{source} gate resume");
    }
}

#[test]
fn test_player_expands_repeated_palette_ramp_before_next_gate() {
    let timeline = Timeline {
        segments: vec![
            Segment {
                palette_start: Some(1),
                palette_end: Some(0),
                palette_step: -1,
                palette_delay: 4,
                repeat: 2,
                tail_delay: 3,
                ..seg("palette_ramp_repeat", "repeat")
            },
            seg("native_text_branch_opaque", "gate"),
        ],
        ..Default::default()
    };
    let mut compositor = new_indexed_compositor();
    compositor.baseline_known = true;
    let mut p = new_player(timeline, &[], None, Some(shared(compositor))).unwrap();
    let r = advance(&mut p, 0);
    assert!(r.1.is_empty() && r.0 == PLAYBACK_RUNNING, "{r:?}");
    // (1,0)×2 at 4ms plus two 3ms tails.
    let r = advance(&mut p, 22);
    assert!(
        r.1.is_empty() && r.0 == PLAYBACK_BLOCKED && p.blocked.as_ref().is_some_and(|b| b.source == "gate"),
        "state={r:?} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_player_expands_timed_frame_sequence_before_next_gate() {
    let frames = [frame(0, 0, 1, 1, &[1, 0, 1, 0, 0, 5]), frame(0, 0, 1, 1, &[1, 0, 1, 0, 0, 6])];
    let timeline = Timeline {
        segments: vec![
            Segment {
                first_frame: Some(0),
                last_frame: Some(1),
                target: "vga".into(),
                stride: WIDTH,
                transparent: Some(-1),
                palette_delay: 20,
                ..seg("blit_frame_sequence", "sequence")
            },
            seg("native_text_branch_opaque", "gate"),
        ],
        ..Default::default()
    };
    let c = shared(new_indexed_compositor());
    let mut p = new_player(timeline, &frames, None, Some(c.clone())).unwrap();
    let r = advance(&mut p, 0);
    let pixel = c.lock().unwrap().vga[0];
    assert!(r.1.is_empty() && r.0 == PLAYBACK_RUNNING && pixel == 5, "first frame state={r:?} pixel={pixel}");
    let r = advance(&mut p, 40);
    let pixel = c.lock().unwrap().vga[0];
    assert!(
        r.1.is_empty() && r.0 == PLAYBACK_BLOCKED && pixel == 6 && p.blocked.is_some(),
        "sequence state={r:?} pixel={pixel} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_player_runs_recovered_composite40_before_text_gate() {
    let timeline = Timeline {
        segments: vec![seg("native_composite_loop_opaque", "0x2bf60"), seg("native_text_branch_opaque", "gate")],
        ..Default::default()
    };
    let mut p = new_player(timeline, &composite_frames(), None, Some(shared(new_indexed_compositor()))).unwrap();
    let r = advance(&mut p, 0);
    assert!(r.1.is_empty() && r.0 == PLAYBACK_RUNNING, "{r:?}");
    let r = advance(&mut p, 780);
    assert!(
        r.1.is_empty() && r.0 == PLAYBACK_BLOCKED && p.blocked.as_ref().is_some_and(|b| b.source == "gate"),
        "state={r:?} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_player_runs_recovered_composite200_then_stops_at_post_composite_gate() {
    let mut compositor = new_indexed_compositor();
    compositor.baseline[0] = 3;
    let c = shared(compositor);
    let timeline = Timeline {
        segments: vec![
            seg("native_composite_loop_baseline", "0x2c0c5"),
            seg("native_post_composite_opaque", "0x2c172"),
        ],
        ..Default::default()
    };
    let mut p = new_player(timeline, &composite_frames(), None, Some(c.clone())).unwrap();
    let r = advance(&mut p, 0);
    assert!(r.1.is_empty() && r.0 == PLAYBACK_RUNNING, "initial state={r:?}");
    // The initial pass is immediate; the remaining 199 passes take 20ms each.
    let r = advance(&mut p, 3980);
    assert!(
        r.1.is_empty() && r.0 == PLAYBACK_BLOCKED && p.blocked.as_ref().is_some_and(|b| b.source == "0x2c172"),
        "state={r:?} blocked={:?}",
        p.blocked
    );
    assert_eq!(c.lock().unwrap().palette[0], 2, "final baseline delta palette");
}

#[test]
fn test_player_phase0_bridge_needs_native_palette_and_stops_at_montage_gate() {
    let Some(text_path) = input("FD2_EXTRACTED_RAW_DIR", "FDTXT/FDTXT_031.bin") else { return };
    let Some(font_path) = input("FD2_EXTRACTED_RAW_DIR", "FDOTHER/FDOTHER_004.bin") else { return };
    let Some(phase_path) = remake_asset("endings/native_2c405.json") else { return };
    let text = std::fs::read(text_path).unwrap();
    let font = std::fs::read(font_path).unwrap();
    let phase = load_finale_phase(&phase_path).unwrap();
    let mut compositor = new_indexed_compositor();
    let mut palette = [0u8; 768];
    palette[0] = 50;
    compositor.present_ani(&vec![0u8; BYTES as usize], &palette).unwrap();
    let c = shared(compositor);
    let timeline = Timeline { segments: vec![seg("native_post_composite_opaque", "0x2c172")], ..Default::default() };
    let mut p = new_player(timeline, &[], None, Some(c.clone())).unwrap();
    p.enable_recovered_phase0(&phase, &Phase0Assets { text_resource: text, font_resource: font, ..Default::default() })
        .unwrap();
    let r = advance(&mut p, 0);
    let pal = c.lock().unwrap().palette[0];
    assert!(
        r.1.is_empty() && r.0 == PLAYBACK_RUNNING && p.phase0.is_some() && pal == 10,
        "initial phase bridge state={r:?} phase={} palette={pal}",
        p.phase0.is_some()
    );
    let r = advance(&mut p, 500);
    assert!(
        r.1.is_empty()
            && r.0 == PLAYBACK_BLOCKED
            && p.blocked.as_ref().is_some_and(|b| b.op == "native_finale_montage_opaque" && b.source == "0x2c548"),
        "phase bridge state={r:?} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_player_phase0_bridge_refuses_unproven_palette() {
    let Some(phase_path) = remake_asset("endings/native_2c405.json") else { return };
    let phase = load_finale_phase(&phase_path).unwrap();
    let timeline = Timeline { segments: vec![seg("native_post_composite_opaque", "0x2c172")], ..Default::default() };
    let mut p = new_player(timeline, &[], None, Some(shared(new_indexed_compositor()))).unwrap();
    p.enable_recovered_phase0(
        &phase,
        &Phase0Assets { text_resource: vec![1], font_resource: vec![1], ..Default::default() },
    )
    .unwrap();
    let r = advance(&mut p, 0);
    assert!(
        r.1.is_empty() && r.0 == PLAYBACK_BLOCKED && p.blocked.as_ref().is_some_and(|b| b.source == "0x2c172"),
        "state={r:?} blocked={:?}",
        p.blocked
    );
}

#[test]
fn test_render_figure_fade_pass_restores_backdrop_then_shifts_secondary_frame() {
    let mut c = new_indexed_compositor();
    c.baseline[0] = 50;
    let mut work = vec![0u8; (WIDTH * HEIGHT * 2) as usize];
    let restore = vec![1u8; BYTES as usize];
    let f = figani::Frame { x: 2, y: 3, width: 1, height: 1, pixels: vec![9], mask: vec![1], ..Default::default() };
    render_figure_fade_pass(
        Some(&mut c),
        &mut work,
        &restore,
        &transparent_tai003_fixture(),
        &f,
        FigureFadePass { stage: 8, source_offset: 80, palette_delta: 48 },
    )
    .unwrap();
    let w = WIDTH as usize;
    assert!(
        c.vga[0] == 1 && c.vga[3 * w + 82] == 9 && c.palette[0] == 2,
        "vga/palette={}/{}/{}",
        c.vga[0],
        c.vga[3 * w + 82],
        c.palette[0]
    );
}

#[test]
fn test_render_mirror_figure_fade_pass_uses_right_viewport_and_arg4_gate() {
    let mut c = new_indexed_compositor();
    let (w, h) = (WIDTH as usize, HEIGHT as usize);
    let mut work = vec![0u8; w * h * 2];
    for y in 0..h {
        work[y * w * 2 + w] = 1;
    }
    let primary =
        figani::Frame { x: 100, y: 3, width: 1, height: 1, pixels: vec![9], mask: vec![1], ..Default::default() };
    let secondary =
        figani::Frame { x: 120, y: 5, width: 1, height: 1, pixels: vec![7], mask: vec![1], ..Default::default() };
    let mut pass = MirrorFigureFadePass {
        stage: 8,
        primary_source_offset: 0xf0,
        palette_delta: 48,
        draw_secondary: true,
        draw_platform: true,
    };
    render_mirror_figure_fade_pass(Some(&mut c), &mut work, &transparent_tai003_fixture(), &primary, &secondary, pass)
        .unwrap();
    assert!(
        c.vga[0] == 1 && c.vga[3 * w + 20] == 9 && c.vga[5 * w + 120] == 7 && c.palette[0] == 0,
        "vga/palette={}/{}/{}/{}",
        c.vga[0],
        c.vga[3 * w + 20],
        c.vga[5 * w + 120],
        c.palette[0]
    );
    (pass.draw_secondary, pass.draw_platform) = (false, false);
    render_mirror_figure_fade_pass(Some(&mut c), &mut work, &fdother::Frame::default(), &primary, &secondary, pass)
        .unwrap();
    pass.draw_platform = true;
    let mut bad_tai = transparent_tai003_fixture();
    bad_tai.mask[0] = 255;
    assert!(
        render_mirror_figure_fade_pass(Some(&mut c), &mut work, &bad_tai, &primary, &secondary, pass).is_err(),
        "mirror platform accepted non-transparent TAI#3"
    );
}
