//! Go `remake/cmd/fd2/rust_golden_frames_test.go` 的 opening 模式（`goldenFramesOpening`），R5 階段 2（`r5t`）。
//!
//! 「標題開場動畫 → 選單 → START → 序章 → 第一次進入 `battle_ch01`」，步驟與 Go 相同（`startFromNormalTitle`、
//! `settleTown`、`driveStory` 的同款迴圈），`loadGame` 之後與每個 `update` 之後各擷取一次。設
//! `FD2_GOLDEN_FRAMES_MODE=opening`、`FD2_GOLDEN_FRAMES_OUT=<目錄>`、`FD2_PARITY_SLOT=<建構槽>` 才執行，由
//! `tools/rust_r5_opening_frames.sh` 以 `--ignored` 呼叫；沒設時印「略過」後返回（Go `t.Skip`）。
//!
//! `Game::draw` 停在未移植的 `todo!` 時，[`FrameCapture::try_observe`] 記下位置，這裡就結束並照常寫
//! `frames.json`（`stopped` 欄位寫明停在哪一次擷取），比對工具只比停下之前的擷取。全域亂數同 chapter 模式以
//! [`pin_global_rand`] 接上（R5 階段 2 r5d）；標題段不用嘴型亂數。

use super::*;
use std::path::{Path, PathBuf};

/// Go `goldenFramesOpening` 每個 `step()`：`parityUpdateGame` 之後擷取。回傳 false 表示 `Game::draw` 停在
/// 未移植的函式。
fn opening_step(d: &mut Driver, capture: &mut FrameCapture) -> bool {
    if let Err(err) = parity_update_game(d) {
        panic!("Update：{err}");
    }
    if !d.g.load_err.is_empty() {
        panic!("執行期錯誤：{}", d.g.load_err);
    }
    capture.try_observe(&mut d.g)
}

/// Go `TestRustGoldenFrames` 的 opening 分支。父行程檢查環境變數，再以 Go `t.Setenv` 的那組變數在子行程執行。
#[test]
#[ignore = "R5 opening 黃金畫格：由 tools/rust_r5_opening_frames.sh 以 --ignored 執行"]
fn test_rust_golden_frames_opening() {
    let var = |name: &str| std::env::var(name).unwrap_or_default();
    let (out, mode, slot) = (var("FD2_GOLDEN_FRAMES_OUT"), var("FD2_GOLDEN_FRAMES_MODE"), var("FD2_PARITY_SLOT"));
    if out.is_empty() {
        eprintln!("略過：需要 FD2_GOLDEN_FRAMES_OUT（R5 opening 黃金畫格，只由 tools/rust_r5_opening_frames.sh 執行）");
        return;
    }
    assert_eq!(mode, "opening", "FD2_GOLDEN_FRAMES_MODE={mode:?}；本測試只跑 opening 模式");
    assert!(!slot.is_empty(), "opening 模式需要 FD2_PARITY_SLOT（與 chapter 模式同一個建構槽）");
    let out = PathBuf::from(out);
    let child = std::env::var_os("FD2_GAME_F_TEST_DIR").is_some();
    let slot_copy = out.join("FD2.SAV");
    if !child {
        // Go 把建構槽複製到 t.TempDir()；這裡放在輸出目錄，與章重播相同（建構槽是唯讀輸入）。
        std::fs::create_dir_all(&out).expect("建立輸出目錄");
        let raw = std::fs::read(&slot).unwrap_or_else(|e| panic!("{slot}：{e}"));
        std::fs::write(&slot_copy, raw).expect("寫 FD2.SAV 複本");
    }
    let slot_copy_env = slot_copy.clone();
    let Some(_data_home) = f_child_env(module_path!(), "test_rust_golden_frames_opening", move |dir| {
        vec![
            ("FD2_TITLE", "1".into()),
            ("FD2_SEED", "4".into()),
            ("FD2_MUTE", "1".into()),
            ("FD2_CAMPAIGN", CANONICAL_CAMPAIGN_REFERENCE.into()),
            ("FD2_SHOT_AI", "1".into()),
            ("FD2_NATIVE_SAVE", slot_copy_env.clone().into_os_string()),
            ("XDG_DATA_HOME", dir.as_os_str().to_os_string()),
        ]
    }) else {
        return;
    };
    run_golden_frames_opening(&out);
}

/// Go `goldenFramesOpening` 在 `t.Setenv` 之後的本體（子行程）。
fn run_golden_frames_opening(out: &Path) {
    let mut capture = FrameCapture::new("opening", out.to_path_buf());
    let g = load_game().expect("loadGame");
    // R5 r5f：Go 黃金載入了系統的 NotoSansCJK-Regular.ttc。
    require_golden_font(&g);
    if !g.load_err.is_empty() {
        panic!("{}", g.load_err);
    }
    if g.title_phase != "cutscene" {
        panic!("正常START未從開場動畫開始：{:?}", g.title_phase);
    }
    let mut d = Driver {
        g,
        clock: VirtualFrameClock::default(),
        frame_capture: None,
        map_layer_capture: None,
        dialogue_layer_capture: None,
        r5p_layer_capture: None,
        r5c_layer_capture: None,
        ttf_layer_capture: None,
    };
    // R5 階段 2 r5d：Go 黃金的 rand.Seed(4)。
    pin_global_rand(&mut d.g);
    let mut phases = serde_json::Map::new();
    let mut extra = serde_json::Map::new();
    extra.insert(
        "scope".into(),
        "標題開場動畫 → 選單 → START → 序章故事 → 第一次進入 battle_ch01；每個 Update 之後擷取（另含 loadGame 後一張）"
            .into(),
    );
    'run: {
        // loadGame 之後、第一個 Update 之前。
        if !capture.try_observe(&mut d.g) {
            break 'run;
        }
        // 1. 開場動畫到標題選單（同 startFromNormalTitle）。
        let mut frame = 0;
        while d.g.title_phase != "menu" {
            if frame >= JOURNEY_TITLE_FRAMES || d.g.title_phase.is_empty() {
                panic!("正常開場動畫未進標題選單");
            }
            ack_presents(&mut d.g);
            if !opening_step(&mut d, &mut capture) {
                break 'run;
            }
            frame += 1;
        }
        phases.insert("title_menu".into(), capture.captures().into());
        // 2. START（選單第 0 項），等標題結束（同 startFromNormalTitle 的 pump(600)）。
        if d.g.title_sel != 0 || !d.g.apply_title_menu_event(TITLE_MENU_CONFIRM) {
            panic!("正常START未由標題owner消費");
        }
        let mut frame = 0;
        while !d.g.title_phase.is_empty() {
            if frame >= 600 {
                panic!("START未正常離開標題");
            }
            ack_presents(&mut d.g);
            answer_native_treasure_prompt(&mut d.g);
            if !opening_step(&mut d, &mut capture) {
                break 'run;
            }
            frame += 1;
        }
        if d.g.camp.is_none() || node_id(&d.g) != JOURNEY_OPENING_NODE {
            panic!("START未正常進入序章");
        }
        phases.insert("story_ch00".into(), capture.captures().into());
        // 3. 同 parityReplay.settleTown。
        for frame in 0..JOURNEY_TOWN_SETTLE_MAX {
            let g = &d.g;
            if g.native_class_ui_job.is_none()
                && g.fade.is_none()
                && g.transition_reveal.is_none()
                && g.indexed_transition.is_none()
                && frame > 10
            {
                break;
            }
            ack_presents(&mut d.g);
            if !opening_step(&mut d, &mut capture) {
                break 'run;
            }
        }
        // 4. 同 driveStory（停止條件：進入 battle_ch01）。
        let mut j = new_journey_trace();
        let (mut last_sig, mut still) = (String::new(), 0);
        let mut frame = 0;
        while node_id(&d.g) != "battle_ch01" {
            if frame >= JOURNEY_STORY_FRAMES {
                panic!("故事節點 {} 在 {JOURNEY_STORY_FRAMES} 幀內沒有結束", node_id(&d.g));
            }
            j.observe(&d.g);
            if let Some(mut n) = node(&d.g)
                && (n.r#type == "story" || n.r#type == "cutscene")
                && story_enter_ready(&d.g)
            {
                let before = story_signature(&d.g);
                d.g.handle_native_story_input(Some(&mut n), NativeStoryInput { enter: true, escape: false });
                if story_signature(&d.g) != before {
                    j.press(&journey_phase(&d.g));
                }
            }
            ack_presents(&mut d.g);
            if !opening_step(&mut d, &mut capture) {
                break 'run;
            }
            let sig = progress_signature(&d.g);
            if sig == last_sig {
                still += 1;
                if still > JOURNEY_STALL_FRAMES {
                    panic!("故事節點 {} 卡住 {still} 幀", node_id(&d.g));
                }
            } else {
                (last_sig, still) = (sig, 0);
            }
            frame += 1;
        }
        phases.insert("battle_ch01".into(), capture.captures().into());
        extra.insert("story_nodes".into(), j.nodes.unwrap_or_default().join(" → ").into());
        extra.insert("story_presses".into(), j.presses.into());
    }
    extra.insert("phase_captures".into(), phases.into());
    capture.finish(extra);
}
