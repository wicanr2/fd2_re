//! Go 來源：`remake/cmd/fd2/native_story_dialogue_progress_test.go`（R4 B 組，期望值不改）。
//!
//! R5 階段 2 r5d：`TestNativeStoryOpeningFrameIndexPreservesFiveStageOrder`（繪圖輔助 `nativeStoryOpeningFrameIndex`）
//! 移植到這裡，另加 `drawNativeStoryDialogue` 的畫格測試（Go 沒有直接畫這個函式的測試）。

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

/// Go：`native_story_dialogue_progress_test.go:176` `TestNativeStoryOpeningFrameIndexPreservesFiveStageOrder`
#[test]
fn test_native_story_opening_frame_index_preserves_five_stage_order() {
    let want = [0, 0, 1, 2, 3, 4, 4, 4];
    for (tick, expected) in want.into_iter().enumerate() {
        let got = native_story_opening_frame_index(tick as i64, 5);
        assert_eq!(got, expected, "opening tick{tick}={got}, want {expected}");
    }
    let got = native_story_opening_frame_index(0, 0);
    assert_eq!(got, -1, "missing opening frames returned index {got}");
}

/// R5 階段 2 r5d 夾具：每個索引一個可辨認的不透明色（預乘＝原值）。
fn r5d_palette() -> ColorPalette {
    (0..256).map(|i| ColorRgba { r: i as u8, g: 255 - i as u8, b: (i / 2) as u8, a: 0xff }).collect()
}

fn r5d_frame(seed: u8) -> Vec<u8> {
    (0..320 * 200).map(|i: usize| ((i / 7) as u8).wrapping_add(seed)).collect()
}

fn r5d_dialogue_game() -> Game {
    Game {
        dialog: vec![native_line(&[&["甲"]])],
        native_class_ui: Some(NativeClassUIAssets { palette: r5d_palette(), ..Default::default() }),
        locale_id: String::from("zh-Hant"),
        // 只有一頁：最後一張等鍵時不疊箭頭（箭頭要原版 FDOTHER 對話框格，由局部黃金涵蓋）。
        native_dialogue_progressive: vec![vec![r5d_frame(1), r5d_frame(2), r5d_frame(3)]],
        native_dialogue_opening: (10..15).map(r5d_frame).collect(),
        native_dialogue_closing: (20..23).map(r5d_frame).collect(),
        native_dialogue_mouth_open: vec![r5d_frame(30)],
        native_map_presented: true,
        native_interface_hud_visible: true,
        ..Default::default()
    }
}

/// `presentNativeClassFrame` 的期望：320×200 索引畫格配調色盤，`GeoM.Scale(2, 2)` 最近鄰貼滿 640×400。
fn r5d_expect_2x(frame: &[u8]) -> Vec<u8> {
    let palette = r5d_palette();
    let mut out = vec![0u8; 640 * 400 * 4];
    for y in 0..400 {
        for x in 0..640 {
            let c = palette[frame[(y / 2) * 320 + x / 2] as usize];
            out[(y * 640 + x) * 4..(y * 640 + x) * 4 + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
        }
    }
    out
}

fn r5d_draw(g: &mut Game) -> (bool, Vec<u8>) {
    let mut screen = Picture::new_image(LOGICAL_W, LOGICAL_H);
    screen.fill(ColorRgba { r: 1, g: 2, b: 3, a: 0xff });
    let drawn = g.draw_native_story_dialogue(&mut screen);
    (drawn, screen.read_pixels().to_vec())
}

#[test]
fn modern_story_portrait_uses_native_anchor_and_only_the_steady_page() {
    let mut game = r5d_dialogue_game();
    let rgba = [220, 10, 60, 255];
    let base = Arc::new(Picture::from_nrgba(80, 80, &rgba.repeat(80 * 80)));
    let hd = Arc::new(Picture::new_image(480, 480));
    game.native_dialogue_modern_portrait =
        Some(ModernStoryPortraitFrame { image: Some(base), hd_image: Some(hd.clone()) });
    for control in ["FFEF", "FFEC"] {
        game.dialog[0].native_dialogue.as_mut().unwrap().control = control.into();
        game.native_dialogue_progress = 0;
        let Rectangle { min, max } = campaign::native_story_dialogue_portrait_rect(control).unwrap();
        let region = rect(min.x * 2, min.y * 2, max.x * 2, max.y * 2);
        let mut expected = r5d_expect_2x(&r5d_frame(1));
        for y in region.min.y..region.max.y {
            for x in region.min.x..region.max.x {
                let index = ((y * 640 + x) * 4) as usize;
                expected[index..index + 4].copy_from_slice(&rgba);
            }
        }
        let (drawn, actual) = r5d_draw(&mut game);
        assert!(drawn && actual == expected, "{control}");
        let overlay = game.modern_hd_overlay.as_ref().unwrap();
        assert_eq!(overlay.rect, region);
        let image = overlay.portrait.as_ref().unwrap().hd_image.as_ref().unwrap();
        assert!(Arc::ptr_eq(image, &hd));
    }
    game.begin_present_frame();
    game.dlg_phase = 2;
    assert!(r5d_draw(&mut game).1 == r5d_expect_2x(&r5d_frame(10)));
    assert!(game.modern_hd_overlay.is_none());
    game.native_dialogue_closing_live = true;
    assert!(r5d_draw(&mut game).1 == r5d_expect_2x(&r5d_frame(20)));
    assert!(game.modern_hd_overlay.is_none());
}

/// R5 階段 2 r5d：沒有原生對白時回 false、不畫；穩定頁以進度選預建畫格，經 `presentNativeClassFrame` 放大兩倍，
/// 並清掉本幀的 HUD／原生地圖旗標。
#[test]
fn test_draw_native_story_dialogue_presents_progressive_frame_2x() {
    let mut g = Game::default();
    let (drawn, pix) = r5d_draw(&mut g);
    assert!(!drawn, "沒有對白卻回 true");
    assert_eq!(&pix[..4], &[1, 2, 3, 0xff], "沒有對白卻改了畫面");

    let mut g = r5d_dialogue_game();
    g.native_dialogue_progress = -5; // 夾到第 0 張
    let (drawn, pix) = r5d_draw(&mut g);
    assert!(drawn && pix == r5d_expect_2x(&r5d_frame(1)), "進度 <0 沒有呈現第 0 張");
    assert!(!g.native_map_presented && !g.native_interface_hud_visible, "presentNativeClassFrame 沒有清本幀旗標");
    g.native_dialogue_progress = 99; // 夾到最後一張
    let (_, pix) = r5d_draw(&mut g);
    assert!(pix == r5d_expect_2x(&r5d_frame(3)), "進度超出沒有呈現最後一張");
    g.mouth_open = true; // 最後一張且張嘴：改用等鍵張嘴畫格
    let (_, pix) = r5d_draw(&mut g);
    assert!(pix == r5d_expect_2x(&r5d_frame(30)), "等鍵張嘴沒有換成 nativeDialogueMouthOpen");
    g.native_dialogue_progress = 1; // 不是最後一張：張嘴不換
    let (_, pix) = r5d_draw(&mut g);
    assert!(pix == r5d_expect_2x(&r5d_frame(2)), "中間進度不該換成張嘴畫格");
    assert!(g.load_err.is_empty(), "loadErr={}", g.load_err);
}

/// R5 階段 2 r5d：開框（`dlgPhase == 2`）以 `nativeStoryOpeningFrameIndex(dlgT)` 選幀，收框以 `closingT` 選幀並夾在
/// 最後一張；其他非穩定階段回 true 而不畫。
#[test]
fn test_draw_native_story_dialogue_opening_closing_and_other_phases() {
    let mut g = r5d_dialogue_game();
    g.dlg_phase = 2;
    for (tick, index) in [(0, 0), (1, 0), (3, 2), (9, 4)] {
        g.dlg_t = tick;
        let (drawn, pix) = r5d_draw(&mut g);
        assert!(drawn && pix == r5d_expect_2x(&r5d_frame(10 + index)), "開框 tick{tick} 不是第 {index} 張");
    }
    g.dlg_phase = 1;
    let (drawn, pix) = r5d_draw(&mut g);
    assert!(drawn && pix[..4] == [1, 2, 3, 0xff], "dlgPhase=1 應回 true 且不畫");
    g.native_dialogue_closing_live = true;
    for (t, index) in [(0, 0), (2, 2), (7, 2)] {
        g.native_dialogue_closing_t = t;
        let (drawn, pix) = r5d_draw(&mut g);
        assert!(drawn && pix == r5d_expect_2x(&r5d_frame(20 + index)), "收框 t{t} 不是第 {index} 張");
    }
    g.native_dialogue_closing.clear();
    let (drawn, pix) = r5d_draw(&mut g);
    assert!(drawn && pix[..4] == [1, 2, 3, 0xff], "沒有收框畫格時應回 true 且不畫");
}

/// R5 階段 2 r5d：繁中說話嘴型以 `ComposeNativeStoryDialogueMouthFrame` 合成；沒有版面時照 Go 的 nil `Validate`
/// 失敗，寫 `loadErr` 並回 true（不回退到其他介面）。
#[test]
fn test_draw_native_story_dialogue_speaking_mouth_composes_or_fails_closed() {
    let mut g = r5d_dialogue_game();
    g.native_dialogue_progress = 1;
    g.native_dialogue_speaking_frame = 2;
    g.native_dialogue_portraits =
        (0..4).map(|i| dato::Frame { width: 80, height: 80, pixels: vec![0x40 + i as u8; 80 * 80] }).collect();
    let layout = campaign::NativeDialogueLayout {
        source_dat: String::from("FDTXT.DAT"),
        control: String::from("FFEE"),
        pages: vec![vec![String::from("甲")]],
        ..Default::default()
    };
    g.native_dialogue_layout = Some(layout.clone());
    let want =
        campaign::compose_native_story_dialogue_mouth_frame(&r5d_frame(2), &g.native_dialogue_portraits[2], &layout)
            .expect("合成說話嘴型");
    let (drawn, pix) = r5d_draw(&mut g);
    assert!(drawn && pix == r5d_expect_2x(&want), "說話嘴型畫格不同（loadErr={}）", g.load_err);
    g.native_dialogue_layout = None;
    let (drawn, pix) = r5d_draw(&mut g);
    assert!(drawn && pix[..4] == [1, 2, 3, 0xff], "沒有版面仍畫了畫面");
    assert_eq!(
        g.load_err, "native story dialogue: campaign: native dialogue layout lacks source provenance",
        "沒有版面的錯誤訊息"
    );
}
