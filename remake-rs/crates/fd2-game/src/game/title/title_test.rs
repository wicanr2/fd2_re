//! Go 來源：`remake/cmd/fd2/title_menu_state_test.go`。
//!
//! 未移植的 Go 測試（理由見 `PORTING.md`「A 組測試」）：`TestTitleScrollPalettePulseMatchesNativeRowWindows`、
//! `TestTitlePublisherScheduleAndBrightness`（`titleScrollUsesPulsePalette`／`titlePublisherBrightness`
//! 是繪圖用的 `r8` 函式）、`TestNativeTitlePaletteTransitionsUseIndexedFDOTHERFrames`、
//! `TestNativeTitlePublisherUsesFixedFDOTHERResources`（`decode…` 是 `r8`）、`TestTitleMenuRowsMatchStableDOSBoxOracle`
//! （`titleMenuY`，`r8`）、`TestTitleSlotTraceIsBoundedAndCancelable`（`TitleSlotState.Step`，`r8`），以及三個
//! 經 `loadGame` 改寫環境變數的測試。

use super::*;
use crate::test_support;

/// Go：`title_menu_state_test.go:15` `TestTitleCutScriptMatchesNativeInterleavedSchedule`
#[test]
fn test_title_cut_script_matches_native_interleaved_schedule() {
    let want_kinds = [
        "publisher",
        "afm",
        "scroll",
        "static",
        "scroll",
        "afm",
        "afm",
        "scroll",
        "afm",
        "afm",
        "scroll",
        "afm",
        "scroll",
        "afm",
        "scroll",
        "static",
        "scroll",
        "hold",
        "redfade",
        "redhold",
        "afm",
        "titlefade",
    ];
    let kinds: Vec<&str> = CUT_SCRIPT.iter().map(|s| s.kind.as_str()).collect();
    assert_eq!(kinds, want_kinds);
    let want_scroll =
        [[535, 450, 153], [450, 330, 216], [330, 210, 216], [210, 110, 180], [110, 25, 153], [25, 10, 27], [10, 0, 18]];
    let got_scroll: Vec<[i64; 3]> =
        CUT_SCRIPT.iter().filter(|s| s.kind == "scroll").map(|s| [s.scroll_from, s.scroll_to, s.tick]).collect();
    assert_eq!(got_scroll, want_scroll);
    let want_afm = [[3, 5, 1], [4, 5, 0], [5, 3, 0], [6, 5, 0], [7, 3, 0], [8, 5, 0], [0, 1, 0], [1, 1, 1]];
    let got_afm: Vec<[i64; 3]> =
        CUT_SCRIPT.iter().filter(|s| s.kind == "afm").map(|s| [s.res, s.tick, s.skip as i64]).collect();
    assert_eq!(got_afm, want_afm);
}

fn cutscene_game(cut_idx: i64, cut_frame: i64, cut_tick: i64) -> Game {
    Game { title_phase: String::from("cutscene"), cut_idx, cut_frame, cut_tick, ..Default::default() }
}

/// Go：`title_menu_state_test.go:73` `TestTitleCutAdvanceRestoresNativeScrollBoundary`
#[test]
fn test_title_cut_advance_restores_native_scroll_boundary() {
    let mut g = cutscene_game(0, 7, 3);
    g.cut_advance();
    assert!(g.cut_idx == 1 && g.cut_frame == 0 && g.cut_tick == 0, "publisher boundary");
    g.cut_advance();
    assert!(g.cut_idx == 2 && g.scroll_y == 535.0, "first scroll boundary y={}", g.scroll_y);
    g.cut_idx = 3; // guardian static → native resumes at the same esi=450
    g.scroll_y = 450.0;
    g.cut_advance();
    assert!(g.cut_idx == 4 && g.scroll_y == 450.0, "guardian resume");
}

/// Go：`title_menu_state_test.go:91` `TestTitleCutAdvanceStopsANI1CompanionVoice`
#[test]
fn test_title_cut_advance_stops_ani1_companion_voice() {
    let index = CUT_SCRIPT.iter().position(|s| s.kind == "afm" && s.res == 1).expect("ANI #1 cut step missing") as i64;
    let mut g = cutscene_game(index, 0, 0);
    g.title_ani1_voice = Some(AudioVoice { playing: true, ..Default::default() });
    g.cut_advance();
    assert!(g.title_ani1_voice.is_none() && g.cut_idx == index + 1);
}

/// Go：`title_menu_state_test.go:140` `TestTitleLogoPaletteTransitionSchedule`
#[test]
fn test_title_logo_palette_transition_schedule() {
    let start = title_logo_transition_index();
    assert!(start >= 0 && start + 3 < CUT_SCRIPT.len() as i64, "logo transition start={start}");
    let want = [("redfade", 20, 0, false), ("redhold", 6, 0, false), ("afm", 1, 1, true), ("titlefade", 20, 0, false)];
    for (i, (kind, tick, res, skip)) in want.into_iter().enumerate() {
        let got = &CUT_SCRIPT[start as usize + i];
        assert!(got.kind == kind && got.tick == tick && got.res == res && got.skip == skip, "step {i}={got:?}");
    }
    for (tick, expected) in [(-1, 0), (0, 0), (1, 2), (9, 19), (10, 21), (18, 38), (19, 40), (20, 40)] {
        assert_eq!(title_palette_phase(tick), expected, "palette phase tick {tick}");
    }
}

/// Go `decodeNativeTitlePublisher`（`r8`）的測試用重寫：直接讀玩家自備 `FDOTHER.DAT` 的 #74／#76。
fn decode_archive_publisher(archive: &std::path::Path) -> (Vec<u8>, fdother::Palette) {
    let frame = fdother::parse_single_frame(&fd2_formats::archive::read_resource(archive, 74).unwrap()).unwrap();
    assert_eq!((frame.width, frame.height), (320, 200));
    let mut pixels = vec![0u8; 320 * 200];
    frame.blit(&mut pixels, 320, -1).unwrap();
    let palette = fdother::parse_vga_palette(&fd2_formats::archive::read_resource(archive, 76).unwrap()).unwrap();
    (pixels, palette)
}

/// Go：`title_menu_state_test.go:216` `TestSeparatedTitlePublisherMatchesFixedArchive`
#[test]
fn test_separated_title_publisher_matches_fixed_archive() {
    let (Some(archive), Some(_)) =
        (test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT"), test_support::input("FD2_ASSET_PACK", "surfaces"))
    else {
        return;
    };
    let pack = std::env::var("FD2_ASSET_PACK").unwrap();
    let (want_pixels, want_palette) = decode_archive_publisher(&archive);
    let got = load_separated_title_publisher(&pack).unwrap().unwrap();
    assert_eq!(got.pix, want_pixels, "separated publisher indexed pixels differ");
    assert_eq!(got.palette, color_palette_from_nrgba(&want_palette), "publisher palette differs");
    // Go `TestNativeTitlePublisherUsesFixedFDOTHERResources` 的雜湊。
    assert_eq!(crate::game::sha256_hex(&got.pix), "9ffe75b509e191db498528a584e63f4b048075257c0ab4e2ffe6a0140182f7bf");
}

/// Go：`title_menu_state_test.go:242` `TestSeparatedTitlePublisherFailsClosedWithoutPack`
#[test]
fn test_separated_title_publisher_fails_closed_without_pack() {
    let dir = test_support::temp_dir("title-publisher");
    assert!(load_separated_title_publisher(&dir.to_string_lossy()).is_err(), "missing publisher pack was accepted");
}

/// Go：`title_menu_state_test.go:248` `TestSeparatedTitleFDOTHERMatchesFixedArchive`
#[test]
fn test_separated_title_fdother_matches_fixed_archive() {
    let (Some(archive), Some(_)) =
        (test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT"), test_support::input("FD2_ASSET_PACK", "surfaces"))
    else {
        return;
    };
    let pack = std::env::var("FD2_ASSET_PACK").unwrap();
    let surfaces = join_path(&[&pack, "surfaces"]);
    for resource in [69usize, 70, 71, 72, 73, 75, 100] {
        let want =
            fdother::parse_single_frame(&fd2_formats::archive::read_resource(&archive, resource).unwrap()).unwrap();
        let (want_indexed, want_mask) = want.indexed_layers().unwrap();
        let got = fdother::load_separated_single_frame(&surfaces, "FDOTHER.DAT", resource as i64).unwrap();
        assert!(
            got.width == want.width
                && got.height == want.height
                && got.indexed == want_indexed
                && got.mask == want_mask,
            "separated title root FDOTHER #{resource} differs"
        );
    }
    for resource in [8usize, 99, 101, 102] {
        let want = fd2_formats::archive::read_resource(&archive, resource).unwrap();
        let (got, _) =
            fdother::load_separated_fdother_palette(join_path(&[&pack, "palette"]), resource as i64).unwrap();
        assert_eq!(got, want, "separated title palette FDOTHER #{resource} differs");
    }
    let outer = fd2_formats::archive::read_resource(&archive, 7).unwrap();
    assert_eq!(fd2_formats::archive::archive_data_resource_count(&outer).unwrap(), 8);
    assert!(fd2_formats::archive::archive_entry(&outer, 7).unwrap().is_empty());
    for nested in 0..7usize {
        let raw = fd2_formats::archive::archive_entry(&outer, nested).unwrap();
        let want = fdother::parse_single_frame(raw).unwrap();
        let (want_indexed, want_mask) = want.indexed_layers().unwrap();
        let got = fdother::load_separated_nested_single_frame(&surfaces, 7, nested as i64).unwrap();
        assert!(
            got.width == want.width
                && got.height == want.height
                && got.indexed == want_indexed
                && got.mask == want_mask,
            "separated title nested FDOTHER #7/{nested} differs"
        );
    }
    let mut assets = TitleAssets::default();
    load_separated_title_fdother(&pack, Some(&mut assets)).unwrap();
    assert!(
        assets.scroll.is_some()
            && assets.scroll_pulse.is_some()
            && assets.title.is_some()
            && assets.cut_static.iter().all(Option::is_some),
        "separated title bundle omitted a required full-screen surface"
    );
    for (item, pair) in assets.items.iter().enumerate() {
        assert!(pair[0].is_some() && pair[1].is_some(), "separated title bundle omitted menu item {item}");
    }
}

/// Go：`title_menu_state_test.go:333` `TestSeparatedTitleFDOTHERFailsClosedWithoutPack`
#[test]
fn test_separated_title_fdother_fails_closed_without_pack() {
    let dir = test_support::temp_dir("title-fdother");
    let mut assets = TitleAssets::default();
    assert!(
        load_separated_title_fdother(&dir.to_string_lossy(), Some(&mut assets)).is_err(),
        "missing separated title bundle was accepted"
    );
    assert!(load_separated_title_fdother(&dir.to_string_lossy(), None).is_err());
}

/// Go：`title_menu_state_test.go:358` `TestTitleCutSkipIsPerStepAndDoesNotGrantEscapeOverride`
#[test]
fn test_title_cut_skip_is_per_step_and_does_not_grant_escape_override() {
    let mut g = cutscene_game(2, 7, 3);
    let skippable = CutStep { skip: true, ..Default::default() };
    assert!(g.try_skip_title_cut_step(&skippable, true), "skippable AFM step rejected a pressed key");
    assert!(g.cut_idx == 3 && g.title_phase == "cutscene" && g.cut_frame == 0 && g.cut_tick == 0);
    assert!(!g.try_skip_title_cut_step(&CutStep::default(), true), "non-skippable AFM step accepted a pressed key");
    assert!(g.cut_idx == 3 && g.title_phase == "cutscene", "rejected skip changed the opening state");
    assert!(!g.try_skip_title_cut_step(&skippable, false), "skippable AFM step advanced without input");
}

/// Go：`title_menu_state_test.go:377` `TestTitleScrollKeyJumpsToNativeLogoBoundary`
#[test]
fn test_title_scroll_key_jumps_to_native_logo_boundary() {
    let mut g = cutscene_game(4, 4, 17);
    g.scroll_y = 441.0;
    let scroll = CutStep { kind: String::from("scroll"), ..Default::default() };
    assert!(g.try_skip_title_cut_step(&scroll, true), "native scroll key did not enter the logo boundary");
    assert!(g.cut_idx == title_logo_transition_index() && g.cut_cur.is_empty() && g.cut_frame == 0 && g.cut_tick == 0);
}

/// Go：`title_menu_state_test.go:414` `TestTitleMenuTraceWrapsAndConfirmsAfterFlash`
#[test]
fn test_title_menu_trace_wraps_and_confirms_after_flash() {
    let mut s = TitleMenuState::default();
    assert!(s.step(TITLE_MENU_UP) == TITLE_MENU_NO_ACTION && s.selection == 2);
    assert!(s.step(TITLE_MENU_DOWN) == TITLE_MENU_NO_ACTION && s.selection == 0);
    s.step(TITLE_MENU_DOWN);
    assert_eq!(s.selection, 1);
    s.step(TITLE_MENU_CONFIRM);
    assert_eq!(s.flash_ticks, 24);
    for i in 0..23 {
        assert_eq!(s.step(TITLE_MENU_TICK), TITLE_MENU_NO_ACTION, "action fired early at tick {i}");
    }
    assert_eq!(s.step(TITLE_MENU_TICK), TITLE_MENU_LOAD_SLOTS);
}

/// Go：`title_menu_state_test.go:440` `TestTitleMenuThirdSelectionIsNativeContinue`
#[test]
fn test_title_menu_third_selection_is_native_continue() {
    let mut s = TitleMenuState { selection: 2, ..Default::default() };
    s.step(TITLE_MENU_CONFIRM);
    for i in 0..23 {
        assert_eq!(s.step(TITLE_MENU_TICK), TITLE_MENU_NO_ACTION, "action fired early at tick {i}");
    }
    assert_eq!(s.step(TITLE_MENU_TICK), TITLE_MENU_CONTINUE);
}

/// `titleUpdate` 的主選單分支只讀 `tick_input`：↓、Enter 後 24 個 tick 才發出 LOAD。
#[test]
fn title_update_menu_reads_tick_input() {
    let mut g = Game { title_phase: String::from("menu"), ..Default::default() };
    g.tick_input.pressed.insert(Key::ArrowDown);
    assert!(g.title_update());
    assert_eq!(g.title_sel, 1);
    g.tick_input = TickInput { pressed: [Key::Enter].into(), ..Default::default() };
    assert!(g.title_update());
    // Confirm 設 24，同一幀的 Tick 再減 1。
    assert_eq!(g.title_flash, 23);
    g.tick_input = TickInput::default();
    for _ in 0..22 {
        g.title_update();
    }
    assert_eq!(g.title_phase, "menu");
    g.title_update();
    assert!(g.title_phase == "loadslots" && g.title_slot_sel == 0);
    // 讀檔槽沒有按鍵時照 Go 留在原地。
    assert!(g.title_update());
    // logozoom：任意鍵進選單。
    let mut g = Game { title_phase: String::from("logozoom"), ..Default::default() };
    g.tick_input.pressed.insert(Key::Z);
    assert!(g.title_update());
    assert!(g.title_phase == "menu" && g.title_sel == 0);
    assert!(!Game::default().title_update(), "unknown phase must leave title flow");
}
