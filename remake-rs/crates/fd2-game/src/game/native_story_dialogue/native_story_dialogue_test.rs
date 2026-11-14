//! Go 來源：`remake/cmd/fd2/native_story_dialogue_progress_test.go`（R4 B 組，期望值不改）。
//!
//! `TestNativeStoryOpeningFrameIndexPreservesFiveStageOrder` 測的是繪圖輔助 `nativeStoryOpeningFrameIndex`
//! （`r8`，`drawNativeStoryDialogue` 才用），不在這裡移植。

use super::*;

fn native_line(pages: &[&[&str]]) -> battle::DialogLine {
    battle::DialogLine {
        native_dialogue: Some(battle::NativeDialogueLayout {
            pages: pages.iter().map(|p| p.iter().map(|s| s.to_string()).collect()).collect(),
            ..Default::default()
        }),
        ..Default::default()
    }
}

/// Go：`native_story_dialogue_progress_test.go:10` `TestNativeStoryDialogueProgressBlocksInputUntilPageComplete`
#[test]
fn test_native_story_dialogue_progress_blocks_input_until_page_complete() {
    let mut line = native_line(&[&["甲乙"]]);
    (line.speaker, line.upper) = (26, Some(true));
    let mut g = Game {
        dialog: vec![line],
        dlg_shown: DLG_NONE,
        dlg_page: 0,
        native_dialogue_progress: -1,
        native_dialogue_progressive: vec![vec![vec![0], vec![1], vec![2]]],
        ..Default::default()
    };
    assert!(!g.dlg_advance() && g.dialog.len() == 1, "native story input advanced before the first progressive frame");
    g.step_native_story_dialogue_progress();
    assert!(
        g.native_dialogue_progress == 0 && !g.dlg_advance(),
        "frame0 progression/input = {}/{}",
        g.native_dialogue_progress,
        g.dialog.is_empty()
    );
    g.step_native_story_dialogue_progress();
    assert!(
        g.native_dialogue_progress == 1 && !g.dlg_advance(),
        "first glyph progression/input = {}/{}",
        g.native_dialogue_progress,
        g.dialog.is_empty()
    );
    g.step_native_story_dialogue_progress();
    assert_eq!(g.native_dialogue_progress, 2, "final glyph progression");
    assert!(
        g.dlg_advance() && g.dialog.is_empty() && g.native_dialogue_progress == -1,
        "completed page did not advance/reset: progress={} dialog={}",
        g.native_dialogue_progress,
        g.dialog.len()
    );
}

fn record(byte8: u8, byte5: u8) -> battle::Unit {
    let mut u = battle::Unit::default();
    (u.native_record_byte8, u.has_native_record_byte8, u.native_record_byte5, u.has_native_record_byte5) =
        (byte8, true, byte5, true);
    u
}

/// Go：`native_story_dialogue_progress_test.go:48` `TestNativeStoryDialogueMotionTargetReplays12C60ActiveRawLookup`
#[test]
fn test_native_story_dialogue_motion_target_replays12_c60_active_raw_lookup() {
    let layout = campaign::NativeDialogueLayout { control: String::from("FFEE"), operand: 21, ..Default::default() };
    let mut g = Game { story_actors: vec![record(21, 1), record(21, 0)], ..Default::default() };
    let got = g.resolve_native_story_dialogue_motion_target(&layout);
    assert!(matches!(got, Ok(112)), "FFEE active lookup target={got:?}, want 112");
    g.story_actors[1].native_record_byte5 = 1;
    let got = g.resolve_native_story_dialogue_motion_target(&layout);
    assert!(matches!(got, Ok(0)), "FFEE inactive-only lookup target={got:?}, want 0");
    g.story_actors[1].has_native_record_byte5 = false;
    assert!(
        g.resolve_native_story_dialogue_motion_target(&layout).is_err(),
        "missing raw +5 provenance did not fail closed"
    );
    let ffed = campaign::NativeDialogueLayout { control: String::from("FFED"), ..Default::default() };
    let got = g.resolve_native_story_dialogue_motion_target(&ffed);
    assert!(matches!(got, Ok(2)), "FFED direct target={got:?}, want 2");
}

/// Go：`native_story_dialogue_progress_test.go:70` `TestNativeStoryDialogueSpeakerUsesRawControlOwner`
#[test]
fn test_native_story_dialogue_speaker_uses_raw_control_owner() {
    let mut a = record(7, 0);
    (a.battle_fig, a.has_battle_fig) = (48, true);
    let mut b = record(30, 0);
    (b.battle_fig, b.has_battle_fig) = (30, true);
    let mut g = Game { story_actors: vec![a, b], ..Default::default() };
    let layout = |control: &str, operand: i64| campaign::NativeDialogueLayout {
        control: control.to_string(),
        operand,
        ..Default::default()
    };
    let got = g.resolve_native_story_dialogue_speaker(&layout("FFEF", 30));
    assert!(matches!(got, Ok(30)), "identity speaker={got:?}, want raw +7 30");
    let got = g.resolve_native_story_dialogue_speaker(&layout("FFEC", 0));
    assert!(matches!(got, Ok(48)), "slot speaker={got:?}, want raw +7 48");
    let got = g.resolve_native_story_dialogue_speaker(&layout("FFEE", 39));
    assert!(matches!(got, Ok(39)), "fallback speaker={got:?}, want direct DATO 39");
    g.story_actors[1].native_record_byte5 = 1;
    let got = g.resolve_native_story_dialogue_speaker(&layout("FFEF", 30));
    assert!(matches!(got, Ok(30)), "inactive fallback speaker={got:?}, want direct DATO 30");
}

/// Go：`native_story_dialogue_progress_test.go:90` `TestNativeStoryDialogueClosingKeepsOldLineUntilAllFramesPublished`
#[test]
fn test_native_story_dialogue_closing_keeps_old_line_until_all_frames_published() {
    let mut g = Game {
        dialog: vec![native_line(&[&["甲"]])],
        native_dialogue_closing: vec![vec![1], vec![2], vec![3]],
        dlg_shown: 1,
        ..Default::default()
    };
    assert!(
        g.begin_native_story_dialogue_closing() && g.dialog.len() == 1 && g.native_dialogue_closing_t == 0,
        "closing did not retain its caller-owned old line"
    );
    for want in 1..3 {
        g.step_native_story_dialogue_progress();
        assert!(
            g.native_dialogue_closing_live && g.native_dialogue_closing_t == want && g.dialog.len() == 1,
            "closing stage{want} state live={} tick={} dialog={}",
            g.native_dialogue_closing_live,
            g.native_dialogue_closing_t,
            g.dialog.len()
        );
    }
    g.step_native_story_dialogue_progress();
    assert!(
        !g.native_dialogue_closing_live && g.dialog.is_empty() && g.dlg_shown == DLG_NONE,
        "closing completion live={} dialog={} shown={}",
        g.native_dialogue_closing_live,
        g.dialog.len(),
        g.dlg_shown
    );
}

/// Go：`native_story_dialogue_progress_test.go:111` `TestNativeStoryDialogueProgressWaitsForStableDialoguePhase`
#[test]
fn test_native_story_dialogue_progress_waits_for_stable_dialogue_phase() {
    let mut g = Game {
        dialog: vec![native_line(&[&["甲"]])],
        dlg_phase: 2,
        native_dialogue_progress: -1,
        native_dialogue_progressive: vec![vec![vec![0], vec![1]]],
        ..Default::default()
    };
    g.step_native_story_dialogue_progress();
    assert_eq!(g.native_dialogue_progress, -1, "opening phase leaked progressive glyph");
    g.dlg_phase = 0;
    g.dlg_scroll_t = 1;
    g.step_native_story_dialogue_progress();
    assert_eq!(g.native_dialogue_progress, -1, "page scroll leaked progressive glyph");
}

/// Go：`native_story_dialogue_progress_test.go:130` `TestNativeStoryDialogueMouthRunsOnlyWhileCompletePageWaitsForInput`
#[test]
fn test_native_story_dialogue_mouth_runs_only_while_complete_page_waits_for_input() {
    let mut g = Game {
        dialog: vec![native_line(&[&["甲"]])],
        dlg_phase: 0,
        native_dialogue_progress: 0,
        native_dialogue_progressive: vec![vec![vec![0; 1], vec![0; 1]]],
        native_dialogue_mouth_open: vec![vec![0; 320 * 200]],
        ..Default::default()
    };
    g.step_dialogue_mouth();
    assert!(
        !g.native_dialogue_mouth_ready && !g.mouth_open,
        "progressive glyph phase started the stable-page mouth owner"
    );
    g.native_dialogue_progress = 1;
    g.step_dialogue_mouth();
    assert!(
        g.native_dialogue_mouth_ready && !g.mouth_open && (2..=31).contains(&g.mouth_state.countdown),
        "initial wait state = ready:{} open:{} countdown:{}",
        g.native_dialogue_mouth_ready,
        g.mouth_open,
        g.mouth_state.countdown
    );
    g.mouth_state.countdown = 0;
    g.frame = 2;
    g.step_dialogue_mouth();
    assert!(
        g.mouth_open && g.mouth_state.frame_index() == 3,
        "zero post-decrement did not select frame3: {:?}",
        g.mouth_state
    );
    g.frame = 4;
    g.step_dialogue_mouth();
    assert!(
        !g.mouth_open && (2..=31).contains(&g.mouth_state.countdown),
        "one-tick mouth did not close/resample: {:?}",
        g.mouth_state
    );
    g.dlg_scroll_t = 1;
    g.step_dialogue_mouth();
    assert!(
        !g.native_dialogue_mouth_ready && !g.mouth_open && g.mouth_state.countdown == 0,
        "page scroll did not reset mouth owner: {:?}",
        g.mouth_state
    );
}

/// Go：`native_story_dialogue_progress_test.go:165` `TestNativeStoryDialogueInputFailsClosedWithoutProgressiveFrames`
#[test]
fn test_native_story_dialogue_input_fails_closed_without_progressive_frames() {
    let mut g = Game { dialog: vec![native_line(&[&["甲"]])], ..Default::default() };
    assert!(
        !g.dlg_advance() && g.dialog.len() == 1,
        "native story dialogue advanced without its indexed progressive frames"
    );
}

/// R4 B 組補充：嘴型亂數改由 `story_mouth_rng` 以固定種子惰性建立，同一輸入序列重跑結果相同
/// （Go 用全域 `rand`，每次啟動不同；KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`）。
#[test]
fn test_story_mouth_rng_is_deterministic_and_separate_from_game_rng() {
    let run = || {
        let mut g = Game {
            dialog: vec![native_line(&[&["甲"]])],
            native_dialogue_progress: 1,
            native_dialogue_progressive: vec![vec![vec![0; 1], vec![0; 1]]],
            native_dialogue_mouth_open: vec![vec![0; 320 * 200]],
            ..Default::default()
        };
        let mut trace = Vec::new();
        for frame in 0..200 {
            g.frame = frame;
            g.step_dialogue_mouth();
            trace.push((g.mouth_open, g.mouth_state.countdown));
        }
        assert!(g.rng.is_none(), "嘴型不得建立或消耗遊戲亂數 rng");
        trace
    };
    let first = run();
    assert_eq!(first, run(), "同一輸入序列的嘴型軌跡不同");
    assert!(first.iter().any(|&(open, _)| open), "200 幀內嘴型一次都沒張開");
}
