//! B 組在 `main.go` 名下函式的測試（R4 B 組，期望值不改）。Go 來源：`beatrunner_test.go`（BeatRunner 與
//! acting／focus／walk 的逐幀狀態機）、`battleevent_test.go` 的 `TestNativeBattleIntroCallRequiresExactEventCallerProvenance`，
//! 以及 Rust 新增的補充測試（標「B 組補充」）。
//!
//! Go 的 `newBeatTestGame` 呼叫 `attachOfficialLocale`；這些測試的 `campLines` 都沒有 `line_id`，`localizedStoryText`
//! 在 `zh-Hant` 下直接回傳原文，所以 Rust 的 helper 只設 `locale_id`（官方語系目錄要等 C 組的語系契約）。
//! Go 以 `assetPath("assets/...")` 讀受版控素材，Rust 改用 `test_support::remake_asset`（同一個檔）。
//! 沒有移植的 `beatrunner_test.go` 測試見 `PORTING.md`「B 組測試」。

use super::*;
use crate::test_support;

/// Go `newBeatTestGame`（`beatrunner_test.go:18`）。
fn new_beat_test_game(beats: Vec<campaign::Beat>) -> Game {
    let c = campaign::Campaign {
        start: String::from("cs"),
        nodes: BTreeMap::from([
            (
                String::from("cs"),
                campaign::Node {
                    r#type: String::from("cutscene"),
                    beats: beats.clone(),
                    actors: vec![
                        campaign::Actor { fig: 0, x: 1, y: 1, ..Default::default() },
                        campaign::Actor { fig: 4, x: 2, y: 2, ..Default::default() },
                    ],
                    next: String::from("end"),
                    ..Default::default()
                },
            ),
            (
                String::from("end"),
                campaign::Node { r#type: String::from("ending"), text: String::from("完"), ..Default::default() },
            ),
        ]),
        ..Default::default()
    };
    let actors: Vec<battle::Unit> = c.nodes["cs"]
        .actors
        .iter()
        .map(|a| {
            let mut u = battle::Unit::default();
            (u.fig, u.x, u.y, u.on_field) = (a.fig, a.x, a.y, true);
            u
        })
        .collect();
    let line = |speaker: i64, text: &str| campaign::Line { speaker, text: text.to_string(), ..Default::default() };
    Game {
        m: Some(MapData { w: 20, h: 20, tile_w: 24, tile_h: 24, cols: 8, tiles: vec![0; 400], ..Default::default() }),
        camp: Some(campaign::new_runner(&c)),
        camp_lines: vec![line(0, "第一句"), line(4, "第二句"), line(0, "第三句")],
        story_actors: actors,
        story_bg: true,
        beats,
        beat_idx: -1,
        locale_id: String::from("zh-Hant"),
        ..Default::default()
    }
}

impl Game {
    /// Go 測試 helper `(*Game).tick`（`beatrunner_test.go:57`）：手動跑 `Update` 裡 BeatRunner 相關的步驟。
    /// `stepNativeCh20SkyKey` 是 `r8`（第一章沒有天空之鑰 job），沒有 job 時 Go 也不做事，這裡省略。
    pub(crate) fn tick(&mut self, n: i64) {
        for _ in 0..n {
            self.step_native_spawn_intro();
            self.step_story_walks();
            self.step_act_job();
            self.step_focus_unit();
            self.step_fade();
            self.step_native_palette_ramp();
            self.step_cam_pan();
            if self.beat_delay > 0 {
                self.beat_delay -= 1;
                if self.beat_delay == 0 {
                    self.beat_advance();
                }
            }
        }
    }
}

fn beat(op: &str) -> campaign::Beat {
    campaign::Beat { op: op.to_string(), ..Default::default() }
}

fn acting_unit(fig: i64, slot: Option<i64>, pose: i64) -> campaign::ActingUnit {
    campaign::ActingUnit { slot, fig, pose }
}

fn unit_at(fig: i64, x: i64, y: i64) -> battle::Unit {
    let mut u = battle::Unit::default();
    (u.fig, u.x, u.y, u.on_field) = (fig, x, y, true);
    u
}

fn map32_acting() -> Option<BTreeMap<i64, Vec<campaign::ActingFrame>>> {
    let path = test_support::remake_asset("cutscenes/acting/map32.json")?;
    Some(campaign::load_acting_resource_set(&path).expect("map32 acting 資源"))
}

/// Go：`beatrunner_test.go:78` `TestBeatPaletteUpdatePresentsRecoveredFullDACWhiteFlash`
#[test]
fn test_beat_palette_update_presents_recovered_full_dac_white_flash() {
    let pu = |start, end, delta| campaign::Beat {
        palette_start: start,
        palette_end: end,
        palette_delta: delta,
        ..beat("palette_update")
    };
    let mut g = new_beat_test_game(vec![pu(0, 255, 255), campaign::Beat { ms: 200, ..beat("delay") }, pu(0, 255, 0)]);
    g.beat_advance();
    assert!(
        g.native_full_dac_white && g.beat_delay == 12 && g.load_err.is_empty(),
        "white flash start white={} delay={} err={:?}",
        g.native_full_dac_white,
        g.beat_delay,
        g.load_err
    );
    g.tick(12);
    assert!(!g.native_full_dac_white && g.load_err.is_empty(), "white flash restore err={:?}", g.load_err);
    let mut rejected = new_beat_test_game(vec![pu(1, 255, 255)]);
    rejected.beat_advance();
    assert!(!rejected.load_err.is_empty() && !rejected.native_full_dac_white, "partial non-indexed update");
}

/// Go：`beatrunner_test.go:102` `TestBeatPanMovesCamera`
#[test]
fn test_beat_pan_moves_camera() {
    let mut g = new_beat_test_game(vec![campaign::Beat { x: 100, y: 200, frames: 10, ..beat("pan") }]);
    g.beat_advance(); // 啟動第 0 拍
    g.cam_max_y = 50.0; // follow-only cap must not corrupt an explicit original PAN target.
    assert!(g.cam_pan.is_some(), "pan 拍應設定 camPan");
    g.tick(10);
    assert!(g.cam_x == 100.0 && g.cam_y == 200.0, "pan 走完應到 (100,200),得 ({},{})", g.cam_x, g.cam_y);
    assert!(g.cam_pan.is_none(), "pan 走完應清除 camPan");
}

/// Go：`beatrunner_test.go:136` `TestBeatPanTileStepMatchesOriginalXFirstRedrawOrder`
#[test]
fn test_beat_pan_tile_step_matches_original_x_first_redraw_order() {
    let mut g = new_beat_test_game(vec![campaign::Beat { x: 72, y: 48, tile_step: true, ..beat("pan") }]);
    (g.cam_x, g.cam_y) = (0.0, 0.0);
    g.beat_advance();
    g.tick(2);
    assert!(
        g.cam_x == 48.0 && g.cam_y == 0.0 && g.cam_pan.is_some(),
        "0x135dd first two redraws ({},{})",
        g.cam_x,
        g.cam_y
    );
    g.tick(1);
    assert!(g.cam_x == 72.0 && g.cam_y == 0.0, "0x135dd must finish X before Y: ({},{})", g.cam_x, g.cam_y);
    g.tick(1);
    assert!(g.cam_x == 72.0 && g.cam_y == 24.0, "first Y redraw = ({},{}), want (72,24)", g.cam_x, g.cam_y);
    g.tick(1);
    assert!(
        g.cam_x == 72.0 && g.cam_y == 48.0 && g.cam_pan.is_none(),
        "tile-step pan finish = ({},{})",
        g.cam_x,
        g.cam_y
    );
}

/// Go：`beatrunner_test.go:158` `TestBeatWalkMovesActorAndAdvances`
#[test]
fn test_beat_walk_moves_actor_and_advances() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { fig: 0, x: 5, y: 1, frames: 5, follow: true, ..beat("walk") },
        campaign::Beat { line: 0, count: 1, ..beat("dialog") },
    ]);
    g.beat_advance();
    assert_eq!(g.story_walks.len(), 1, "walk 拍應建立 1 個 storyWalks job");
    assert!(g.follow_walk, "walk 拍 Follow=true 應設定 g.followWalk");
    g.tick(5);
    assert!(g.story_walks.is_empty(), "走完應清空 storyWalks");
    let u = &g.story_actors[0];
    assert!(u.x == 5 && u.y == 1, "索爾應走到 (5,1),得 ({},{})", u.x, u.y);
    assert_eq!(g.dialog.len(), 1, "walk 完成應接 dialog 拍,g.dialog 應有 1 句");
}

/// Go：`beatrunner_test.go:183` `TestBeatDialogCountConsecutiveLines`
#[test]
fn test_beat_dialog_count_consecutive_lines() {
    let mut g = new_beat_test_game(vec![campaign::Beat { line: 0, count: 2, ..beat("dialog") }]);
    g.beat_advance();
    assert_eq!(g.dialog.len(), 2, "count=2 應推入 2 句");
    assert_eq!(g.dialog.last().unwrap().text, "第一句", "堆疊末端應是第一句");
    g.dialog.pop();
    assert_eq!(g.dialog.len(), 1, "pop 一次應剩 1 句");
    g.dialog.pop();
    assert!(g.dialog.is_empty(), "pop 兩次應清空");
    g.beat_advance(); // 對白播完,序列只有 1 拍,應跑到收尾(進入淡出)
    assert!(g.fade.is_some(), "beats 跑完應觸發 advanceStoryNode 的淡出轉場");
}

/// Go：`beatrunner_test.go:209` `TestBeatDialogResolvesOriginalRuntimeSpeakerSlot`
#[test]
fn test_beat_dialog_resolves_original_runtime_speaker_slot() {
    let slot = 6;
    let dialog0 = || vec![campaign::Beat { line: 0, ..beat("dialog") }];
    let mut g = new_beat_test_game(dialog0());
    g.camp_lines = vec![campaign::Line {
        speaker: 133,
        speaker_slot: Some(slot),
        upper: Some(true),
        text: String::from("謝謝各位的幫助"),
        ..Default::default()
    }];
    let mut st = battle::State::default();
    st.units = vec![battle::Unit::default(); 7];
    st.units[6].portrait = 134;
    g.st = Some(st);
    g.beat_advance();
    assert!(
        g.load_err.is_empty() && g.dialog.len() == 1 && g.dialog[0].speaker == 134 && g.dialog[0].upper == Some(true),
        "runtime speaker slot did not resolve unit portrait/box: err={:?} dialog={:?}",
        g.load_err,
        g.dialog
    );
    let mut cutscene = new_beat_test_game(dialog0());
    cutscene.camp_lines = vec![campaign::Line {
        speaker: 96,
        speaker_slot: Some(1),
        text: String::from("場景單位"),
        ..Default::default()
    }];
    cutscene.story_actors[1].portrait = 133;
    cutscene.beat_advance();
    assert!(
        cutscene.load_err.is_empty() && cutscene.dialog.len() == 1 && cutscene.dialog[0].speaker == 133,
        "cutscene speaker slot did not resolve materialized actor: err={:?} dialog={:?}",
        cutscene.load_err,
        cutscene.dialog
    );
    let mut bad = new_beat_test_game(dialog0());
    bad.camp_lines = vec![campaign::Line {
        speaker: 133,
        speaker_slot: Some(slot),
        text: String::from("不可猜頭像"),
        ..Default::default()
    }];
    let mut st = battle::State::default();
    st.units = vec![battle::Unit::default(); 6];
    bad.st = Some(st);
    bad.beat_advance();
    assert!(!bad.load_err.is_empty() && bad.dialog.is_empty(), "missing direct speaker slot must fail closed");
}

/// Go：`beatrunner_test.go:263` `TestBeatActCyclesPosesThenAdvances`
#[test]
fn test_beat_act_cycles_poses_then_advances() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { fig: 4, poses: vec![1, 2, 3], pose_frames: 3, ..beat("act") },
        campaign::Beat { line: 2, count: 1, ..beat("dialog") },
    ]);
    g.beat_advance();
    assert!(g.act_job.is_some(), "act 拍應建立 actJob");
    g.tick(3 * 3); // 3 個 pose × 3 幀
    assert!(g.act_job.is_none(), "pose 序列跑完應清除 actJob");
    assert_eq!(g.story_actors[1].dir, 3, "最後一個 pose 應停在 3");
    assert!(g.dialog.len() == 1 && g.dialog[0].text == "第三句", "act 完成應接下一拍(dialog line=2),得 {:?}", g.dialog);
}

/// Go：`beatrunner_test.go:285` `TestBeatActingNormalFrameMovesEachBeat`
#[test]
fn test_beat_acting_normal_frame_moves_each_beat() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat {
            acting: vec![campaign::ActingFrame {
                beats: 3,
                units: vec![acting_unit(0, None, 3)],
                ..Default::default()
            }],
            ..beat("act")
        },
        campaign::Beat { line: 1, count: 1, ..beat("dialog") },
    ]);
    g.beat_advance();
    assert!(g.act_job.is_some(), "acting frame 應建立 actJob");
    g.tick(6);
    let u = &g.story_actors[0];
    assert!(
        u.x == 1 && u.off_x > 0.0 && u.dir == 3,
        "第 6 tick 應仍在第一格內插，得 X={} OffX={} Dir={}",
        u.x,
        u.off_x,
        u.dir
    );
    g.tick(15); // 3 格 × 每格 7 tick，合計 21
    assert!(g.act_job.is_none(), "正常 acting 的全部 beat 後應結束");
    let u = &g.story_actors[0];
    assert!(
        u.x == 4 && u.y == 1 && u.off_x == 0.0 && u.off_y == 0.0 && u.dir == 3,
        "右三格後應為 (4,1) 且定格，得 ({},{}) off=({},{}) dir={}",
        u.x,
        u.y,
        u.off_x,
        u.off_y,
        u.dir
    );
    assert!(g.dialog.len() == 1 && g.dialog[0].text == "第二句", "acting 結束應接下一 dialog，得 {:?}", g.dialog);
}

/// Go：`beatrunner_test.go:314` `TestBeatActingSpecialFrameStaysPut`
#[test]
fn test_beat_acting_special_frame_stays_put() {
    let mut g = new_beat_test_game(vec![campaign::Beat {
        acting: vec![campaign::ActingFrame { beats: 5, special: true, units: vec![acting_unit(4, None, 1)] }],
        ..beat("act")
    }]);
    g.beat_advance();
    g.tick(5);
    let u = &g.story_actors[1];
    assert!(
        u.x == 2 && u.y == 2 && u.off_x == 0.0 && u.off_y == 0.0 && u.dir == 1,
        "special acting 必須原地面左，得 ({},{}) off=({},{}) dir={}",
        u.x,
        u.y,
        u.off_x,
        u.off_y,
        u.dir
    );
}

/// Go：`beatrunner_test.go:330` `TestBeatActingUsesOriginalSlotBeforeDuplicateFig`
#[test]
fn test_beat_acting_uses_original_slot_before_duplicate_fig() {
    // Synthetic duplicate-Fig regression: decoded resources are slot-addressed,
    // so a Fig-only lookup must never redirect either target to the first guard.
    let frames = vec![campaign::ActingFrame {
        beats: 3,
        special: true,
        units: vec![acting_unit(0, Some(1), 1), acting_unit(0, Some(2), 2)],
    }];
    let mut g = new_beat_test_game(vec![campaign::Beat { acting: frames, ..beat("act") }]);
    g.story_actors = (0..3)
        .map(|_| {
            let mut u = battle::Unit::default();
            (u.fig, u.on_field, u.dir) = (69, true, 3);
            u
        })
        .collect();
    g.beat_advance();
    g.tick(3);
    assert_eq!(g.story_actors[0].dir, 3, "duplicate fig fallback touched slot0");
    assert!(
        g.story_actors[1].dir == 1 && g.story_actors[2].dir == 2,
        "decoded slots 1/2 were not targeted: dirs=({},{})",
        g.story_actors[1].dir,
        g.story_actors[2].dir
    );
}

/// Go：`beatrunner_test.go:353` `TestBeatActingZeroSpecialPreservesOriginalThreeTickTransition`
#[test]
fn test_beat_acting_zero_special_preserves_original_three_tick_transition() {
    let mut g = new_beat_test_game(vec![campaign::Beat {
        acting: vec![campaign::ActingFrame { beats: 0, special: true, units: vec![acting_unit(0, None, 2)] }],
        ..beat("act")
    }]);
    g.beat_advance();
    assert_eq!(g.story_actors[0].dir, 2, "zero-special frame must apply native pose before delay");
    g.tick(2);
    assert!(g.act_job.is_some(), "zero-special frame advanced before original delay(1)+delay(2)");
    g.tick(1);
    assert!(g.act_job.is_none(), "zero-special frame did not advance after three ticks");
}

/// Go：`beatrunner_test.go:372` `TestBeatNativePaletteFadeOutFailsClosedWithoutIndexedInput`
#[test]
fn test_beat_native_palette_fade_out_fails_closed_without_indexed_input() {
    let mut g = new_beat_test_game(vec![campaign::Beat {
        native_palette_fade: Some(campaign::NativePaletteFadeOut { start: 0, end: 63, delay_ms: 2 }),
        ..beat("native_palette_fade_out")
    }]);
    g.beat_advance();
    assert_eq!(
        g.load_err, "beat native_palette_fade_out: native indexed framebuffer or DAC baseline unavailable",
        "native palette fade error"
    );
}

/// Go：`beatrunner_test.go:394` `TestBeatCh07PostBlackoutCoversWholeSurfaceAndRejectsOtherShapes`
#[test]
fn test_beat_ch07_post_blackout_covers_whole_surface_and_rejects_other_shapes() {
    let payload = campaign::NativePaletteBlackout { start: 0, end: 255, delta: 64, clear_bytes: 0xFA00 };
    let blackout = |source: &str| campaign::Beat {
        source: source.to_string(),
        native_palette_blackout: Some(payload),
        ..beat("native_palette_blackout")
    };
    let mut g = new_beat_test_game(vec![blackout("0x23599"), campaign::Beat { frames: 1, ..beat("delay") }]);
    g.native_map_vga = vec![0x7f; 0xFA00];
    g.beat_advance();
    assert!(
        g.native_full_dac_black && g.beat_delay == 1 && g.load_err.is_empty(),
        "blackout black={} delay={} err={:?}",
        g.native_full_dac_black,
        g.beat_delay,
        g.load_err
    );
    if let Some(i) = g.native_map_vga.iter().position(|&v| v != 0) {
        panic!("blackout framebuffer[{i}]={}", g.native_map_vga[i]);
    }
    let mut rejected = new_beat_test_game(vec![blackout("0x23598")]);
    rejected.beat_advance();
    assert!(!rejected.load_err.is_empty() && !rejected.native_full_dac_black, "non-proven blackout");
    let mut short = new_beat_test_game(vec![blackout("0x23599")]);
    short.native_map_vga = vec![0; 0xF9FF];
    short.beat_advance();
    assert!(!short.load_err.is_empty() && !short.native_full_dac_black, "short framebuffer");
}

/// Go：`beatrunner_test.go:432` `TestBeatClearNativeRecordBit7PreservesOtherBits`
#[test]
fn test_beat_clear_native_record_bit7_preserves_other_bits() {
    let mut g = new_beat_test_game(vec![beat("clear_native_record_bit7")]);
    let raw = |b5: u8| {
        let mut u = battle::Unit::default();
        (u.has_native_record_byte5, u.native_record_byte5) = (true, b5);
        u
    };
    let mut st = battle::State::default();
    st.units = vec![raw(0xff), raw(0x81)];
    g.st = Some(st);
    g.beat_advance();
    let st = g.st.as_ref().unwrap();
    assert!(
        g.load_err.is_empty() && st.units[0].native_record_byte5 == 0x7f && st.units[1].native_record_byte5 == 0x01,
        "raw bit7 clear mutated wrong bytes: err={:?} bytes={:#x}/{:#x}",
        g.load_err,
        st.units[0].native_record_byte5,
        st.units[1].native_record_byte5
    );
}

/// Go：`beatrunner_test.go:460` `TestBeatActingDecodedNormalSlotMovement`
#[test]
fn test_beat_acting_decoded_normal_slot_movement() {
    // Direct resource102 at ch00 source 0x32461: slot4 left×2, up×1,
    // left×1. A duplicate Fig at slot0 must remain untouched.
    let Some(resources) = map32_acting() else {
        return;
    };
    let mut g = new_beat_test_game(vec![campaign::Beat { acting: resources[&102].clone(), ..beat("act") }]);
    g.story_actors = vec![battle::Unit::default(); 5];
    g.story_actors[0] = unit_at(4, 9, 5);
    g.story_actors[4] = unit_at(4, 10, 10);
    g.beat_advance();
    g.tick(28); // four normal grid beats × seven ticks
    let got = &g.story_actors[0];
    assert!(got.x == 9 && got.y == 5, "same-Fig slot0 moved instead of slot4: ({},{})", got.x, got.y);
    let got = &g.story_actors[4];
    assert!(got.x == 7 && got.y == 9 && got.dir == 1, "slot4 decoded movement = ({},{}) dir={}", got.x, got.y, got.dir);
}

/// Go：`beatrunner_test.go:530` `TestBeatScrollStepSlot2MatchesCh00ACT99Followup`
#[test]
fn test_beat_scroll_step_slot2_matches_ch00_act99_followup() {
    // ch00 handler 0x32351 calls 0x13185(slot2) 15 times immediately after
    // direct ACT99 has moved Sol from Y42 to Y36.  One original grid step is
    // six presented frames, so the complete scroll is exactly 90 ticks.
    let mut g = new_beat_test_game(vec![campaign::Beat {
        slot: Some(2),
        steps: 15,
        frames: 90,
        follow: true,
        ..beat("scroll_step")
    }]);
    g.m = Some(MapData { w: 20, h: 60, tile_w: 24, tile_h: 24, cols: 8, tiles: vec![0; 1200], ..Default::default() });
    g.story_actors = vec![battle::Unit::default(); 3];
    g.story_actors[2] = unit_at(0, 8, 36);
    g.story_actors[2].materialize_native_map_presentation().unwrap();
    g.cam_y = (34 * 24) as f64;
    g.has_story_native_map_view = true;
    g.story_native_map_view =
        battle::NativeMapViewState { camera_y: 34, cursor_y: 34, visible_cursor_y: 0, ..Default::default() };
    g.beat_advance();
    assert!(
        g.story_walks.len() == 1 && !g.follow_walk,
        "scroll_step should use its original safe-band follow rather than centering, walks={} follow={}",
        g.story_walks.len(),
        g.follow_walk
    );
    g.tick(89);
    let got = &g.story_actors[2];
    assert!(
        got.y == 21 && got.dir == 2 && got.off_y != 0.0,
        "after 89/90 ticks slot2 should still interpolate: {got:?}"
    );
    g.tick(1);
    let got = &g.story_actors[2];
    assert!(
        got.x == 8 && got.y == 21 && got.dir == 2 && got.off_x == 0.0 && got.off_y == 0.0,
        "15-step scroll should finish slot2 at (8,21), pose2, without offset: {got:?}"
    );
    let p = got.native_map_presentation;
    assert!(
        p.x == 8 && p.y == 21 && p.pose == 2 && p.motion == 0,
        "completed scroll left stale native presentation: {p:?}"
    );
    assert_eq!(g.cam_y, (20 * 24) as f64, "0x13185 safe-band camera");
    let want_view =
        battle::NativeMapViewState { camera_y: 20, cursor_y: 19, visible_cursor_y: -1, ..Default::default() };
    assert_eq!(g.story_native_map_view, want_view, "0x13185 published native story view");
}

/// Go：`beatrunner_test.go:581` `TestBeatDirectACT100MovesSlot2DownTenCells`
#[test]
fn test_beat_direct_act100_moves_slot2_down_ten_cells() {
    let Some(resources) = map32_acting() else {
        return;
    };
    let frames = resources[&100].clone();
    assert!(
        frames.len() == 1 && frames[0].beats == 10 && !frames[0].special && frames[0].units.len() == 1,
        "ACT100 must retain its direct decoded one-frame down×10 data: {frames:?}"
    );
    assert!(
        frames[0].units[0].slot == Some(2) && frames[0].units[0].pose == 0,
        "ACT100 target must be original slot2 pose0: {:?}",
        frames[0]
    );
    let mut g =
        new_beat_test_game(vec![campaign::Beat { source: String::from("0x323f5"), acting: frames, ..beat("act") }]);
    g.story_actors = vec![battle::Unit::default(); 3];
    g.story_actors[2] = unit_at(0, 8, 8);
    g.beat_advance();
    g.tick(69);
    let got = &g.story_actors[2];
    assert!(got.y == 17 && got.dir == 0, "ACT100 before its 70th tick should have completed nine down cells: {got:?}");
    g.tick(1);
    let got = &g.story_actors[2];
    assert!(
        got.x == 8 && got.y == 18 && got.dir == 0 && got.off_x == 0.0 && got.off_y == 0.0,
        "ACT100 direct frame should move slot2 Y8→18 in 70 ticks, pose0: {got:?}"
    );
}

/// Go：`beatrunner_test.go:608` `TestBeatFocusUnitWalksCursorAndScrollsAtOriginalSafeBand`
#[test]
fn test_beat_focus_unit_walks_cursor_and_scrolls_at_original_safe_band() {
    let mut g = new_beat_test_game(vec![campaign::Beat { slot: Some(2), ..beat("focus_unit") }]);
    // Original 0x12cea uses a 13×8 viewport. It walks X first, then Y, and only
    // scrolls the map origin after screen cursor X>10 / Y>5; it never centers.
    g.m = Some(MapData { w: 40, h: 30, tile_w: 24, tile_h: 24, cols: 8, tiles: vec![0; 1200], ..Default::default() });
    g.story_actors = vec![battle::Unit::default(); 3];
    g.story_actors[2] = unit_at(0, 20, 15);
    g.beat_advance();
    assert!(g.focus_job.is_some() && g.cur_x == 0 && g.cur_y == 0, "focus_unit must start a blocking grid walk");
    g.tick(20);
    assert!(
        g.cur_x == 20 && g.cur_y == 0 && g.cam_x == 216.0 && g.cam_y == 0.0 && g.focus_job.is_some(),
        "after X phase cursor=({},{}) camera=({},{})",
        g.cur_x,
        g.cur_y,
        g.cam_x,
        g.cam_y
    );
    g.tick(14);
    assert!(
        g.cur_x == 20 && g.cur_y == 14 && g.cam_x == 216.0 && g.cam_y == 192.0 && g.focus_job.is_some(),
        "before final Y step cursor=({},{}) camera=({},{})",
        g.cur_x,
        g.cur_y,
        g.cam_x,
        g.cam_y
    );
    g.tick(1);
    assert!(
        g.cur_x == 20 && g.cur_y == 15 && g.cam_x == 216.0 && g.cam_y == 216.0 && g.focus_job.is_none(),
        "focus_unit finish cursor=({},{}) camera=({},{})",
        g.cur_x,
        g.cur_y,
        g.cam_x,
        g.cam_y
    );
}

/// Go：`beatrunner_test.go:634` `TestBeatFocusUnitSynchronizesNativeStoryView`
#[test]
fn test_beat_focus_unit_synchronizes_native_story_view() {
    let mut g = new_beat_test_game(vec![campaign::Beat { slot: Some(0), ..beat("focus_unit") }]);
    g.m =
        Some(MapData { w: 31, h: 64, tile_w: 24, tile_h: 24, cols: 8, tiles: vec![0; 31 * 64], ..Default::default() });
    g.story_actors = vec![unit_at(0, 15, 63)];
    (g.cam_x, g.cam_y) = ((9 * 24) as f64, (19 * 24) as f64);
    (g.cur_x, g.cur_y) = (9, 19);
    g.story_native_map_view =
        battle::NativeMapViewState { camera_x: 9, camera_y: 19, cursor_x: 9, cursor_y: 19, ..Default::default() };
    g.has_story_native_map_view = true;
    g.handler_chapter = 28;
    g.beat_advance();
    // This unit fixture exercises the handler-chapter owner without a campaign
    // node. newBeatTestGame installs an unrelated node by default, so remove it
    // only after focus_unit has materialized its job.
    g.camp = None;
    let mut ticks = 0;
    while g.focus_job.is_some() && ticks < 100 {
        g.step_focus_unit();
        ticks += 1;
    }
    assert!(g.load_err.is_empty() && g.focus_job.is_none(), "ch28 focus did not finish: err={:?}", g.load_err);
    let want = battle::NativeMapViewState {
        camera_x: 9,
        camera_y: 56,
        cursor_x: 15,
        cursor_y: 63,
        visible_cursor_x: 6,
        visible_cursor_y: 7,
    };
    assert!(
        g.story_native_map_view == want
            && g.cur_x == 15
            && g.cur_y == 63
            && g.cam_x == (9 * 24) as f64
            && g.cam_y == (56 * 24) as f64,
        "ch28 focus view={:?} cursor=({},{}) camera=({},{}), want {want:?}",
        g.story_native_map_view,
        g.cur_x,
        g.cur_y,
        g.cam_x,
        g.cam_y
    );
}

/// Go：`beatrunner_test.go:669` `TestBeatActingSkipsUnmaterializedSlotWithinAllocation`
#[test]
fn test_beat_acting_skips_unmaterialized_slot_within_allocation() {
    let act = |slot: i64| {
        vec![campaign::Beat {
            source: String::from("0x32657"),
            acting: vec![campaign::ActingFrame { beats: 1, special: true, units: vec![acting_unit(0, Some(slot), 2)] }],
            ..beat("act")
        }]
    };
    let mut g = new_beat_test_game(act(8));
    g.story_actors = vec![battle::Unit::default(); 5]; // map31 after groups 1+3+5
    g.beat_advance();
    assert!(
        g.load_err.is_empty(),
        "act on an unmaterialized slot inside the 96-record allocation must be a no-op, got {:?}",
        g.load_err
    );
    let mut g = new_beat_test_game(act(NATIVE_RUNTIME_RECORD_CAPACITY));
    g.story_actors = vec![battle::Unit::default(); 5];
    g.beat_advance();
    assert!(!g.load_err.is_empty(), "act targeting a slot outside the 96-record allocation must fail closed");
}

/// Go：`beatrunner_test.go:690` `TestBeatFadeBothDirectionsCallThen`
#[test]
fn test_beat_fade_both_directions_call_then() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { out: true, frames: 4, ..beat("fade") },
        campaign::Beat { out: false, frames: 4, ..beat("fade") },
        campaign::Beat { line: 0, count: 1, ..beat("dialog") },
    ]);
    g.beat_advance();
    assert!(g.fade.as_ref().is_some_and(|f| f.out), "第一拍應是淡出中");
    g.tick(4);
    assert!(g.fade.as_ref().is_some_and(|f| !f.out), "淡出走完應接淡入拍");
    g.tick(4);
    assert!(g.fade.is_none(), "淡入走完應清除 fade");
    assert_eq!(g.dialog.len(), 1, "淡入完成應接 dialog 拍");
}

/// Go：`beatrunner_test.go:713` `TestBeatDelayCountsDownThenAdvances`
#[test]
fn test_beat_delay_counts_down_then_advances() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { frames: 3, ..beat("delay") },
        campaign::Beat { line: 0, count: 1, ..beat("dialog") },
    ]);
    g.beat_advance();
    assert_eq!(g.beat_delay, 3, "delay 拍應設 beatDelay=3");
    for _ in 0..3 {
        // 模擬 Update 裡 beatDelay 倒數(見 Update 內該區塊)
        g.beat_delay -= 1;
        if g.beat_delay == 0 {
            g.beat_advance();
        }
    }
    assert_eq!(g.dialog.len(), 1, "delay 倒數完應接下一拍");
}

/// Go：`beatrunner_test.go:733` `TestBeatSequenceEndTriggersNodeTransition`
#[test]
#[ignore = "等 D 合併（cancel_native_command32_presentation）"]
fn test_beat_sequence_end_triggers_node_transition() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { track: String::from("FDMUS_999"), ..beat("bgm") }, // 非阻塞拍:應立即連呼下一拍
        campaign::Beat { group: 1, ..beat("spawn") },                       // 非阻塞：啟用既有 roster group
        campaign::Beat { char_id: 0, ..beat("join") },                      // 非阻塞：寫入永久 party membership
    ]);
    g.beat_advance(); // 三個非阻塞拍應在同一次呼叫內全部跑完,直接進入收尾
    assert!(g.fade.is_some(), "非阻塞拍序列跑完應觸發收尾淡出");
    g.tick(STORY_FADE_FRAMES); // 淡出走完 → camp.Advance("cs"→"end") + enterNode("end")
    assert_eq!(g.camp.as_ref().unwrap().cur, "end", "應轉場到 end");
}

/// Go：`beatrunner_test.go:749` `TestBeatJoinPersistsOnlyPlayerCharacterIDs`
#[test]
fn test_beat_join_persists_only_player_character_ids() {
    let join = |id: i64| campaign::Beat { char_id: id, ..beat("join") };
    let mut g = new_beat_test_game(vec![join(0), join(9), join(4), join(9)]);
    g.beat_advance();
    assert!(
        g.party_members.get(&9).copied().unwrap_or(false) && g.party_members.len() == 3,
        "join did not persist party membership: {:?}",
        g.party_members
    );
    assert_eq!(g.party_join_order, vec![0, 9, 4], "JOIN chronology lost or duplicated");
    let mut bad = new_beat_test_game(vec![join(75)]);
    bad.beat_advance();
    assert!(!bad.load_err.is_empty() && bad.party_members.is_empty(), "scene portrait join must fail closed");
}

/// Go：`beatrunner_test.go:2283` `TestBeatSpawnActivatesOnlyItsRosterGroup`
#[test]
fn test_beat_spawn_activates_only_its_roster_group() {
    let mut g = new_beat_test_game(vec![campaign::Beat { group: 3, ..beat("spawn") }]);
    g.story_actors = [1, 3, 3, 5]
        .into_iter()
        .map(|group| {
            let mut u = battle::Unit::default();
            u.group = group;
            u
        })
        .collect();
    g.beat_advance();
    let on: Vec<bool> = g.story_actors.iter().map(|u| u.on_field).collect();
    assert_eq!(on, vec![false, true, true, false], "spawn group=3 activated wrong story slots");
}

/// Go：`beatrunner_test.go:2389` `TestBeatSpawnAppendsFDFIELDGroupInOriginalOrder`
#[test]
fn test_beat_spawn_appends_fdfield_group_in_original_order() {
    // Original 0x10b4e does not reveal preallocated units: it constructs every
    // matching FDFIELD record at unit_count, so the runtime slot identity is
    // the order groups were spawned. This is the map31 pattern (1, then 3, 5).
    let spawn = |group: i64| campaign::Beat { group, ..beat("spawn") };
    let mut g = new_beat_test_game(vec![spawn(1), spawn(3), spawn(1)]);
    g.story_actors = Vec::new(); // LOADCH map31 has no group0 records.
    g.story_roster = [(1, 10), (3, 30), (1, 11), (3, 9)]
        .into_iter()
        .map(|(group, fig)| {
            let mut u = battle::Unit::default();
            (u.group, u.fig) = (group, fig);
            u
        })
        .collect();
    g.story_spawned = BTreeMap::from([(0, true)]);
    g.beat_advance();
    assert_eq!(g.story_actors.len(), 4, "spawn constructed runtime units");
    for (i, fig) in [10, 11, 30, 9].into_iter().enumerate() {
        assert!(
            g.story_actors[i].fig == fig && g.story_actors[i].on_field,
            "runtime slot {i} = {:?}, want on-field fig={fig}",
            g.story_actors[i]
        );
    }
}

/// Go：`beatrunner_test.go:2412` `TestDeactivateResetAndRedrawPreserveHandlerBoundaries`
#[test]
fn test_deactivate_reset_and_redraw_preserve_handler_boundaries() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { slot: Some(0), ..beat("deactivate_unit") },
        campaign::Beat { ms: 20, ..beat("reset_pose") },
        campaign::Beat { frames: 1, ..beat("redraw") },
    ]);
    g.story_actors[0].on_field = true;
    g.story_actors[0].dir = 3;
    g.beat_advance();
    assert!(
        !g.story_actors[0].on_field && g.story_actors[0].dir == 0 && g.beat_delay == 1,
        "deactivate/reset state = onField:{} dir:{} delay:{}",
        g.story_actors[0].on_field,
        g.story_actors[0].dir,
        g.beat_delay
    );
    g.tick(2);
    assert!(
        g.beat_delay == 0 && g.beat_idx >= 3,
        "reset/redraw boundaries did not advance: beat={} delay={}",
        g.beat_idx,
        g.beat_delay
    );
}

/// Go：`beatrunner_test.go:2431` `TestCh23PostLayoutResetsSharedViewBeforeNativeFocus`
#[test]
fn test_ch23_post_layout_resets_shared_view_before_native_focus() {
    // 0x23465..0x23493 resets camera, absolute cursor and visible cursor.
    // A stale battle cursor previously drove the story camera to Y=0 while
    // the 0x2189A compositor retained Y=30 (Issue #140).
    let mut g = new_beat_test_game(vec![
        campaign::Beat {
            source: String::from("0x247b4"),
            layout: Some(campaign::HandlerLayout {
                units: vec![campaign::HandlerUnitLayout { slot: 0, x: 20, y: 19, pose: 0 }],
                cam_x: 336,
                cam_y: 336,
            }),
            ..beat("layout_units")
        },
        campaign::Beat { frames: 100, ..beat("delay") },
    ]);
    g.camp = Some(campaign::new_runner(&campaign::Campaign {
        start: String::from("postbattle_ch23_persist"),
        nodes: BTreeMap::from([(
            String::from("postbattle_ch23_persist"),
            campaign::Node { r#type: String::from("cutscene"), ..Default::default() },
        )]),
        ..Default::default()
    }));
    g.m = Some(MapData { w: 32, h: 48, tile_w: 24, tile_h: 24, ..Default::default() });
    let stale = battle::NativeMapViewState {
        camera_x: 14,
        camera_y: 30,
        cursor_x: 20,
        cursor_y: 35,
        visible_cursor_x: 6,
        visible_cursor_y: 5,
    };
    let mut st = battle::State::default();
    (st.w, st.h) = (32, 48);
    let mut u = battle::Unit::default();
    (u.x, u.y) = (20, 35);
    st.units = vec![u];
    st.materialize_native_map_view_state(stale).unwrap();
    g.st = Some(st);
    (g.story_native_map_view, g.has_story_native_map_view) = (stale, true);
    (g.cur_x, g.cur_y, g.cam_x, g.cam_y) = (20, 35, 336.0, 720.0);
    g.beat_advance();
    let mut want =
        battle::NativeMapViewState { camera_x: 14, camera_y: 14, cursor_x: 14, cursor_y: 14, ..Default::default() };
    let st = g.st.as_ref().unwrap();
    assert!(
        g.load_err.is_empty()
            && g.story_native_map_view == want
            && st.native_map_view_state == want
            && g.cur_x == 14
            && g.cur_y == 14
            && g.cam_x == 336.0
            && g.cam_y == 336.0
            && st.has_native_map_range_mode_state
            && st.native_map_range_mode == 0,
        "layout reset: story={:?} battle={:?} error={:?}",
        g.story_native_map_view,
        st.native_map_view_state,
        g.load_err
    );
    g.focus_job = Some(FocusUnitJob {
        target_x: 20,
        target_y: 17,
        native_view: true,
        then: Some(Box::new(|_: &mut Game| {})),
        ..Default::default()
    });
    let mut step = 0;
    while step < 20 && g.focus_job.is_some() {
        g.step_focus_unit();
        step += 1;
    }
    (want.cursor_x, want.cursor_y, want.visible_cursor_x, want.visible_cursor_y) = (20, 17, 6, 3);
    let st = g.st.as_ref().unwrap();
    assert!(
        g.focus_job.is_none()
            && g.load_err.is_empty()
            && g.story_native_map_view == want
            && st.native_map_view_state == want,
        "focus shared view: story={:?} battle={:?} error={:?}",
        g.story_native_map_view,
        st.native_map_view_state,
        g.load_err
    );
    g.cam_pan = Some(CamPanJob { to_x: 336.0, to_y: 360.0, tile_step: true, ..Default::default() });
    g.step_cam_pan();
    (want.camera_y, want.cursor_y) = (15, 18);
    let st = g.st.as_ref().unwrap();
    assert!(
        g.load_err.is_empty() && g.story_native_map_view == want && st.native_map_view_state == want,
        "pan shared view: story={:?} battle={:?} error={:?}",
        g.story_native_map_view,
        st.native_map_view_state,
        g.load_err
    );
}

/// Go：`beatrunner_test.go:2477` `TestLayoutUnitsUsesCanonicalRuntimeSlotsAndCamera`
#[test]
fn test_layout_units_uses_canonical_runtime_slots_and_camera() {
    let layout = campaign::HandlerLayout {
        units: vec![
            campaign::HandlerUnitLayout { slot: 0, x: 8, y: 3, pose: 2 },
            campaign::HandlerUnitLayout { slot: 6, x: 8, y: 1, pose: 0 },
        ],
        cam_x: 48,
        cam_y: 0,
    };
    let mut g = new_beat_test_game(vec![campaign::Beat { layout: Some(layout), ..beat("layout_units") }]);
    let mut st = battle::State::default();
    st.units = (0..7)
        .map(|i| {
            let mut u = battle::Unit::default();
            (u.x, u.y, u.dir, u.off_x, u.off_y) = (20 + i, 30 + i, 3, 4.0, 5.0);
            u
        })
        .collect();
    g.st = Some(st);
    g.beat_advance();
    assert!(g.load_err.is_empty(), "{}", g.load_err);
    let st = g.st.as_ref().unwrap();
    let got = &st.units[0];
    assert!(got.x == 8 && got.y == 3 && got.dir == 2 && got.off_x == 0.0 && got.off_y == 0.0, "layout slot0 = {got:?}");
    let got = &st.units[6];
    assert!(got.x == 8 && got.y == 1 && got.dir == 0, "layout Tino slot6 = {got:?}");
    assert!(g.cam_x == 48.0 && g.cam_y == 0.0, "layout camera = ({},{}), want (48,0)", g.cam_x, g.cam_y);
}

/// Go：`beatrunner_test.go:2506` `TestRuntimeContextAcceptsOnlyDeclaredPostBattleFrontiers`
#[test]
fn test_runtime_context_accepts_only_declared_post_battle_frontiers() {
    let context =
        campaign::HandlerRuntimeContext { slot_counts: vec![15, 27], story_viewport: true, ..Default::default() };
    let with_units = |count: usize| {
        let mut g = new_beat_test_game(vec![campaign::Beat {
            runtime_context: Some(context.clone()),
            ..beat("runtime_context")
        }]);
        let mut st = battle::State::default();
        st.units = vec![battle::Unit::default(); count];
        g.st = Some(st);
        g.beat_advance();
        g
    };
    for count in [15, 27] {
        let g = with_units(count);
        assert!(
            g.load_err.is_empty() && g.story_bg,
            "runtime count{count} rejected: err={:?} storyBG={}",
            g.load_err,
            g.story_bg
        );
    }
    let rejected = with_units(16);
    assert!(!rejected.load_err.is_empty(), "undeclared 16-slot post-battle frontier was accepted");
}

/// Go：`battleevent_test.go:311` `TestNativeBattleIntroCallRequiresExactEventCallerProvenance`
#[test]
fn test_native_battle_intro_call_requires_exact_event_caller_provenance() {
    let mut action = battle::Action {
        r#type: String::from("spawn_group"),
        groups: vec![4],
        camp: String::from("enemy"),
        native_event_id: Some(1),
        native_spawns: vec![battle::NativeSpawnCall {
            group: 4,
            via: String::from("spawn_group_with_intro"),
            source: String::from("0x342ce"),
            raw_placement_gate: Some(0),
            following_acting: Some(battle::NativeFollowingActing { resource: 3, source: String::from("0x342e7") }),
            ..Default::default()
        }],
        ..Default::default()
    };
    let got = native_battle_intro_call(&action);
    assert!(matches!(&got, Ok((call, true)) if call.group == 4), "exact event1 provenance rejected: {got:?}");
    action.native_spawns[0].source = String::from("0x342cf");
    assert!(native_battle_intro_call(&action).is_err(), "changed spawn caller address was accepted");
    action.native_spawns[0].source = String::from("0x342ce");
    action.native_spawns[0].following_acting.as_mut().unwrap().resource = 4;
    assert!(native_battle_intro_call(&action).is_err(), "changed following ACTING resource was accepted");
}

/// B 組補充：`dirToward`（Go `main.go:7407`）與 `actingDelta`（`main.go:1467`）的方向對應。
#[test]
fn test_dir_toward_and_acting_delta_are_inverse_for_unit_steps() {
    for pose in 0..4 {
        let (dx, dy) = acting_delta(pose);
        assert_eq!(dir_toward(5, 5, 5 + dx, 5 + dy), pose, "pose {pose}");
    }
    assert_eq!(acting_delta(7), (0, 0));
    // |dx| == |dy| 時走 Y 分支（Go 用 `>`）。
    assert_eq!(dir_toward(0, 0, 2, 2), 0);
    assert_eq!(dir_toward(0, 0, -2, -2), 2);
}

/// B 組補充：`nativeMapFocusVisibleSeed`（Go `main.go:1248`）把差值夾回 13×8 視窗。
#[test]
fn test_native_map_focus_visible_seed_clamps_to_viewport() {
    assert_eq!(native_map_focus_visible_seed(-3, -1), (0, 0));
    assert_eq!(native_map_focus_visible_seed(5, 4), (5, 4));
    assert_eq!(native_map_focus_visible_seed(13, 8), (12, 7));
}

/// B 組補充：`handlerStoryPath`（Go `main.go:7240`）只替相對檔名補 `assets/story`。
#[test]
fn test_handler_story_path_keeps_absolute_and_asset_paths() {
    assert_eq!(handler_story_path("ch01.json"), "assets/story/ch01.json");
    assert_eq!(handler_story_path("assets/story/ch02.json"), "assets/story/ch02.json");
    assert_eq!(handler_story_path("/tmp/x.json"), "/tmp/x.json");
}

/// B 組補充：`stepActJob` 收尾後延續裝上的新 act job 不會被舊 job 蓋掉（Rust 先取出 job 再推進）。
#[test]
fn test_step_act_job_keeps_job_installed_by_continuation() {
    let mut g = new_beat_test_game(vec![
        campaign::Beat { fig: 0, poses: vec![1], pose_frames: 1, ..beat("act") },
        campaign::Beat { fig: 4, poses: vec![2, 3], pose_frames: 1, ..beat("act") },
    ]);
    g.beat_advance();
    g.step_act_job(); // 第一個 job 收尾，延續 beat_advance 裝上第二個
    let j = g.act_job.as_ref().expect("第二拍的 act job 應已裝上");
    assert_eq!((j.actor, j.poses.clone()), (1, vec![2, 3]));
    g.tick(2);
    assert!(g.act_job.is_none());
    assert_eq!(g.story_actors[1].dir, 3);
}

// ---------------- native_map_presentation_test.go、native_hud_redraw_test.go ----------------

fn presented_unit(x: i64, y: i64) -> battle::Unit {
    let mut u = battle::Unit::default();
    (u.x, u.y) = (x, y);
    u.materialize_native_map_presentation().unwrap();
    u
}

fn walk_game(st: battle::State, from: (i64, i64), to: (i64, i64)) -> Game {
    Game {
        m: Some(MapData { tile_w: 24, tile_h: 24, ..Default::default() }),
        st: Some(st),
        walk: Some(WalkAnim {
            u: Some(0),
            path: vec![battle::Cell { x: from.0, y: from.1 }, battle::Cell { x: to.0, y: to.1 }],
            ..Default::default()
        }),
        ..Default::default()
    }
}

/// Go：`native_map_presentation_test.go:10` `TestBattleWalkPreservesNativeSevenTickRecordLifecycle`
///
/// Go 的單位不在 `g.st` 裡（`g.st` 為 nil）；Rust 的走行單位是 `st.units` 的索引，所以放進一個沒有視圖、
/// 沒有格子事件的 `State`，`noteNativeFieldEventStep` 與 `followNativeMapCursorStep` 一樣不做事。
#[test]
fn test_battle_walk_preserves_native_seven_tick_record_lifecycle() {
    let mut st = battle::State::default();
    st.units = vec![presented_unit(2, 3)];
    let mut g = walk_game(st, (2, 3), (3, 3));
    for tick in 1..=6u8 {
        g.step_battle_walk();
        let u = &g.st.as_ref().unwrap().units[0];
        let raw = u.native_map_presentation;
        assert!(raw.x == 2 && raw.y == 3 && raw.pose == 3 && raw.motion == tick, "tick {tick} raw={raw:?}");
        assert!(u.x == 2 && u.y == 3 && u.off_x > 0.0, "tick {tick} normalized=({},{} off={})", u.x, u.y, u.off_x);
    }
    g.step_battle_walk();
    assert!(g.walk.is_none() && g.moved && g.ring, "seventh tick did not finish player walk");
    let u = &g.st.as_ref().unwrap().units[0];
    let want =
        battle::NativeMapPresentationState { x: 3, y: 3, pose: NATIVE_MAP_REST_POSE as u8, ..Default::default() };
    assert!(
        u.native_map_presentation == want && u.x == 3 && u.y == 3 && u.off_x == 0.0 && u.off_y == 0.0,
        "seventh tick raw={:?} normalized=({},{},{},{})",
        u.native_map_presentation,
        u.x,
        u.y,
        u.off_x,
        u.off_y
    );
}

fn map25_walk(from: (i64, i64), dir: i64, to: (i64, i64)) -> Option<Game> {
    let path = test_support::remake_asset("maps/map25/map25_units.json")?;
    let mut st = battle::load(&path).expect("map25 units");
    let trigger = &mut st.units[0];
    (trigger.native_record_byte6, trigger.has_native_record_byte6) = (1, true);
    trigger.set_map_placement(from.0, from.1, dir);
    trigger.materialize_native_map_presentation().unwrap();
    for index in 39..=44 {
        (st.units[index].native_record_byte34, st.units[index].has_native_record_byte34) = (0xA7, true);
    }
    Some(walk_game(st, from, to))
}

/// Go：`native_map_presentation_test.go:49` `TestBattleWalkNotesSelector0DuringWalkAndDispatchesAtActionEnd`
#[test]
fn test_battle_walk_notes_selector0_during_walk_and_dispatches_at_action_end() {
    let Some(mut g) = map25_walk((11, 36), 1, (10, 36)) else {
        return;
    };
    let byte34 = |g: &Game, index: usize| g.st.as_ref().unwrap().units[index].native_record_byte34;
    for tick in 1..=6 {
        g.step_battle_walk();
        for index in 39..=44 {
            assert_eq!(byte34(&g, index), 0xA7, "tick {tick} unit{index}: selector0 ran before commit");
        }
    }
    g.step_battle_walk();
    for index in 39..=44 {
        assert_eq!(byte34(&g, index), 0xA7, "unit{index}: selector0 ran before the action ended");
    }
    let pending = g.native_field_event_pending;
    assert!(pending.is_some_and(|p| p.x == 10 && p.y == 36), "pending={pending:?}, want the event at (10,36) noted");
    g.dispatch_native_field_event_pending(Some(0));
    for index in 39..=44 {
        assert_eq!(byte34(&g, index), 0xA0, "unit{index}: want 0xa0 after the action-end dispatch");
    }
    assert!(g.native_field_event_pending.is_none(), "dispatch must reset [0x51A8F] to 0xff");
}

/// Go：`native_map_presentation_test.go:101` `TestBattleWalkRightStepAlsoNotesSelector0`
#[test]
fn test_battle_walk_right_step_also_notes_selector0() {
    let Some(mut g) = map25_walk((9, 36), 3, (10, 36)) else {
        return;
    };
    for _ in 1..=7 {
        g.step_battle_walk();
    }
    g.dispatch_native_field_event_pending(Some(0));
    for index in 39..=44 {
        let got = g.st.as_ref().unwrap().units[index].native_record_byte34;
        assert_eq!(got, 0xA0, "unit{index}: a right step into the cell must note selector0 too");
    }
}

/// Go：`native_map_presentation_test.go:187` `TestOriginalActingUpdatesMaterializedRawPresentation`
#[test]
fn test_original_acting_updates_materialized_raw_presentation() {
    let mut st = battle::State::default();
    st.units = vec![presented_unit(4, 5)];
    let mut g =
        Game { m: Some(MapData { tile_w: 24, tile_h: 24, ..Default::default() }), st: Some(st), ..Default::default() };
    let mut j = ActPoseJob {
        acting: vec![campaign::ActingFrame { beats: 1, units: vec![acting_unit(0, Some(0), 2)], ..Default::default() }],
        ..Default::default()
    };
    g.begin_acting_frame(&mut j);
    for tick in 1..=6u8 {
        g.step_original_acting(&mut j);
        let raw = g.st.as_ref().unwrap().units[0].native_map_presentation;
        assert!(raw.x == 4 && raw.y == 5 && raw.pose == 2 && raw.motion == tick, "acting tick {tick} raw={raw:?}");
    }
    g.step_original_acting(&mut j);
    let u = &g.st.as_ref().unwrap().units[0];
    let want = battle::NativeMapPresentationState { x: 4, y: 4, pose: 2, ..Default::default() };
    assert!(
        u.native_map_presentation == want && u.x == 4 && u.y == 4,
        "acting completion raw={:?} normalized=({},{})",
        u.native_map_presentation,
        u.x,
        u.y
    );
}

fn view(cx: i64, cy: i64, x: i64, y: i64, vx: i64, vy: i64) -> battle::NativeMapViewState {
    battle::NativeMapViewState {
        camera_x: cx,
        camera_y: cy,
        cursor_x: x,
        cursor_y: y,
        visible_cursor_x: vx,
        visible_cursor_y: vy,
    }
}

fn view_state(v: battle::NativeMapViewState, gate_a: u8, gate_b: u8, anchor: i64) -> battle::State {
    let mut st = battle::State::default();
    (st.w, st.h) = (30, 30);
    st.materialize_native_map_view_state(v).unwrap();
    assert!(st.materialize_native_map_hud_state(gate_a, gate_b, anchor), "HUD state rejected");
    st
}

/// Go：`native_map_presentation_test.go:217` `TestGameCursorUsesMaterializedNativeMapView`
#[test]
fn test_game_cursor_uses_materialized_native_map_view() {
    let st = view_state(view(4, 5, 15, 11, 11, 6), 1, 1, 0xf2);
    let mut g = Game {
        m: Some(MapData { w: 30, h: 30, tile_w: 24, tile_h: 24, ..Default::default() }),
        st: Some(st),
        ..Default::default()
    };
    g.move_map_cursor(1, 0);
    assert!(
        g.cur_x == 16 && g.cur_y == 11 && g.cam_x == (5 * 24) as f64 && g.cam_y == (5 * 24) as f64,
        "game view cursor=({},{}) camera=({},{})",
        g.cur_x,
        g.cur_y,
        g.cam_x,
        g.cam_y
    );
    let hud = g.st.as_ref().unwrap().native_map_hud_state;
    assert_eq!(hud.anchor_x, 1, "HUD anchor did not follow visible cursor: {hud:?}");
}

/// Go：`native_map_presentation_test.go:239` `TestScreenshotCursorUsesNativeViewAndHUDStateMachine`
#[test]
fn test_screenshot_cursor_uses_native_view_and_hud_state_machine() {
    let st = view_state(view(1, 13, 8, 17, 7, 4), 1, 1, 1);
    let mut g = Game {
        m: Some(MapData { w: 30, h: 30, tile_w: 24, tile_h: 24, ..Default::default() }),
        st: Some(st),
        ..Default::default()
    };
    g.sync_native_map_view();
    assert!(g.position_screenshot_cursor(8, 15), "screenshot cursor rejected");
    let st = g.st.as_ref().unwrap();
    assert!(
        st.native_map_view_state == view(1, 13, 8, 15, 7, 2) && g.cur_x == 8 && g.cur_y == 15,
        "screenshot view={:?} game=({},{})",
        st.native_map_view_state,
        g.cur_x,
        g.cur_y
    );
    assert_eq!(st.native_map_hud_state.anchor_x, 1, "screenshot HUD anchor={:?}", st.native_map_hud_state);
}

/// Go `newNativeHUDRedrawGame`（`native_hud_redraw_test.go:100`）。
fn new_native_hud_redraw_game(gate_b: u8) -> Game {
    let mut st = battle::State::default();
    (st.w, st.h) = (30, 30);
    st.materialize_native_map_view_state(view(5, 10, 7, 16, 2, 6)).unwrap();
    assert!(
        st.materialize_native_map_range_mode(0) && st.materialize_native_map_hud_state(1, gate_b, 1),
        "HUD 狀態建不起來"
    );
    let mut g = Game {
        st: Some(st),
        m: Some(MapData { w: 30, h: 30, tile_w: 24, tile_h: 24, ..Default::default() }),
        ..Default::default()
    };
    g.sync_native_map_view();
    g
}

/// Go：`native_hud_redraw_test.go:115` `TestNativeFocusRedrawsHUDBeforeFirstStep`
#[test]
fn test_native_focus_redraws_hud_before_first_step() {
    // 0x12CEA 開頭 0x12D01 無條件 0x11CAC(0)：目標就是目前格也要評估一次 anchor。
    let mut g = new_native_hud_redraw_game(1);
    g.ai_focus_cursor(7, 16);
    let got = g.st.as_ref().unwrap().native_map_hud_state.anchor_x;
    assert_eq!(got, 0xf2, "聚焦開頭沒有重繪評估 anchor：{got:#x}");
}

/// Go：`native_hud_redraw_test.go:124` `TestNativeFocusSkipsHUDWhenGateBClosed`
#[test]
fn test_native_focus_skips_hud_when_gate_b_closed() {
    // 0x1ACF3 閘 B 為 0 直接返回，0x1AD2A 不跑（敵方回合、0x1A30B 內的回合開頭聚焦）。
    let mut g = new_native_hud_redraw_game(0);
    g.ai_focus_cursor(7, 16);
    let got = g.st.as_ref().unwrap().native_map_hud_state.anchor_x;
    assert_eq!(got, 1, "閘 B 關著卻評估了 anchor：{got:#x}");
}

fn native_focus_job(x: i64, y: i64) -> FocusUnitJob {
    FocusUnitJob {
        target_x: x,
        target_y: y,
        native_view: true,
        then: Some(Box::new(|_: &mut Game| {})),
        ..Default::default()
    }
}

/// Go：`native_hud_redraw_test.go:133` `TestFocusUnitJobOnBattleViewUsesRedrawEntry`
#[test]
fn test_focus_unit_job_on_battle_view_uses_redraw_entry() {
    // 0x13FD4 原地回復的 0x12D7B 走 focusUnitJob：同一支 0x12CEA，開頭重繪一次。
    let mut g = new_native_hud_redraw_game(1);
    g.focus_job = Some(native_focus_job(7, 16));
    let mut step = 0;
    while g.focus_job.is_some() && step < 10 {
        g.step_focus_unit();
        step += 1;
    }
    assert!(g.focus_job.is_none() && g.load_err.is_empty(), "聚焦沒有結束：{}", g.load_err);
    let got = g.st.as_ref().unwrap().native_map_hud_state.anchor_x;
    assert_eq!(got, 0xf2, "focusUnitJob 沒有經重繪入口評估 anchor：{got:#x}");
}

/// Go：`native_hud_redraw_test.go:148` `TestFocusUnitJobStepWithoutRedrawKeepsAnchor`
#[test]
fn test_focus_unit_job_step_without_redraw_keeps_anchor() {
    // 往右走一格只動可見游標、[0x51A83]==0：0x11BFA 不重繪，anchor 停在開頭評估的結果。
    let mut g = new_native_hud_redraw_game(1);
    {
        let st = g.st.as_mut().unwrap();
        st.native_map_hud_state.anchor_x = 1;
        st.native_map_view_state.visible_cursor_y = 4;
        st.native_map_view_state.cursor_y = 14;
    }
    g.sync_native_map_view();
    g.focus_job = Some(native_focus_job(7, 16));
    let mut step = 0;
    while g.focus_job.is_some() && step < 10 {
        g.step_focus_unit();
        step += 1;
    }
    let st = g.st.as_ref().unwrap();
    let v = st.native_map_view_state;
    assert!(v.cursor_y == 16 && v.visible_cursor_y == 6, "聚焦終點 {v:?}");
    assert_eq!(st.native_map_hud_state.anchor_x, 1, "沒重繪的步進評估了 anchor");
}
