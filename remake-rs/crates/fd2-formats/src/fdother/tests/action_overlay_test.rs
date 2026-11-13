//! 移植自 `action_overlay_test.go` 與 `action_overlay_selection_test.go`。

use crate::fdother::*;

// Go: TestActionOverlayCellIndex
#[test]
fn action_overlay_cell_index() {
    let state = ActionOverlayState { availability: [0, 1, 0, 1], direction_state: [0x12, 0x14, 0x16, 0x18] };
    let want = [54, 62, 66, 74];
    for (direction, &w) in want.iter().enumerate() {
        assert_eq!(state.cell_index(direction as i64).unwrap(), w, "direction {direction}");
    }
    assert!(state.cell_index(4).is_err(), "invalid direction was accepted");
}

// Go: TestBattleActionOverlayStateMatches18D8C
#[test]
fn battle_action_overlay_state_matches_18d8c() {
    let state = battle_action_overlay_state([0, 1, 0, 1]);
    assert_eq!(state.direction_state, [0, 1, 2, 3], "direction states");
    for (direction, &w) in [0, 5, 6, 11].iter().enumerate() {
        assert_eq!(state.cell_index(direction as i64).unwrap(), w, "direction {direction}");
    }
}

// Go: TestNativeContinueActionOverlayStateMatches16F55InitialTables
#[test]
fn native_continue_action_overlay_state_matches_16f55_initial_tables() {
    let state = native_continue_action_overlay_state();
    assert!(state.availability == [0; 4] && state.direction_state == [7, 5, 6, 4], "current-runtime state={state:?}");
    for (direction, &w) in [21, 15, 18, 12].iter().enumerate() {
        assert_eq!(state.cell_index(direction as i64).unwrap(), w, "direction {direction}");
    }
}

// Go: TestNativeNestedSystemActionOverlayStateMatches19DF7Tables
#[test]
fn native_nested_system_action_overlay_state_matches_19df7_tables() {
    let state = native_nested_system_action_overlay_state(true, true);
    assert!(
        state.direction_state == [12, 13, 14, 15] && state.availability == [0, 1, 1, 0],
        "nested system state={state:?}"
    );
    for (direction, &w) in [36, 41, 44, 45].iter().enumerate() {
        assert_eq!(state.cell_index(direction as i64).unwrap(), w, "direction {direction}");
    }
}

// Go: TestNativeSystemOptionsMatch1728CCellsAndBooleanToggles
#[test]
fn native_system_options_match_1728c_cells_and_boolean_toggles() {
    let mut options = default_native_system_options();
    let mut state = options.action_overlay_state().unwrap();
    let want = [54, 60, 66, 72];
    for (direction, &w) in want.iter().enumerate() {
        let d = direction as i64;
        assert_eq!(state.cell_index(d).unwrap(), w, "default direction {direction}");
        options = options.toggle(d).unwrap();
        state = options.action_overlay_state().unwrap();
        assert_eq!(state.cell_index(d).unwrap(), w + 3, "toggled direction {direction}");
    }
    let mut invalid = default_native_system_options();
    invalid.raw_51e62 = 2;
    assert!(invalid.action_overlay_state().is_err(), "out-of-domain native system option was accepted");
    assert!(options.toggle(4).is_err(), "out-of-range native system selector was accepted");
}

// Go: TestActionOverlayOriginMatchesNativeAddressExpression
#[test]
fn action_overlay_origin_matches_native_address_expression() {
    let got = action_overlay_origin(6, 5).unwrap();
    assert_eq!(got, 0x8088 + 6 * 0x18 + 5 * 0x18 * 0x1c8);
    assert!(action_overlay_origin(-1, 0).is_err(), "negative origin was accepted");
    // 13×8 是這條位址式自己的界線（0x174AE／0x174B0 的 push 0xD／push 8）。可見游標的狀態可以合法地
    // 出界——走行捲動在鏡頭到邊界時會把它減到 -1——所以擋的地方在這裡，不在狀態物化時。
    for cursor in [(13, 0), (0, 8)] {
        assert!(action_overlay_origin(cursor.0, cursor.1).is_err(), "視窗外的可見游標 {cursor:?} 被接受了");
    }
}

// Go: TestActionOverlaySnapshotOriginMatchesNativePredecessorCell
#[test]
fn action_overlay_snapshot_origin_matches_native_predecessor_cell() {
    let got = action_overlay_snapshot_origin(6, 5).unwrap();
    assert_eq!(got, 0x8088 + 5 * 0x18 + 4 * 0x18 * 0x1c8);
    for cursor in [(0, 1), (1, 0), (-1, 1), (13, 1), (1, 8)] {
        assert!(action_overlay_snapshot_origin(cursor.0, cursor.1).is_err(), "invalid snapshot cursor {cursor:?}");
    }
}

// Go: TestActionOverlaySnapshotCapturesAndRestoresExplicit72By72Region
#[test]
fn action_overlay_snapshot_captures_and_restores_explicit_72_by_72_region() {
    const STRIDE: usize = 100;
    const HEIGHT: usize = 100;
    let mut src = vec![0u8; STRIDE * HEIGHT];
    for y in 0..HEIGHT {
        for x in 0..STRIDE {
            src[y * STRIDE + x] = ((x + 3 * y) & 0xff) as u8;
        }
    }
    let original = src.clone();
    let snapshot = capture_action_overlay_snapshot(&src, STRIDE as i64, 11, 13).unwrap();
    assert_eq!(snapshot.len() as i64, NATIVE_ACTION_OVERLAY_SNAPSHOT_BYTES);
    for y in 13..13 + NATIVE_ACTION_OVERLAY_SNAPSHOT_HEIGHT as usize {
        for x in 11..11 + NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH as usize {
            src[y * STRIDE + x] = 0xee;
        }
    }
    restore_action_overlay_snapshot(&mut src, &snapshot, STRIDE as i64, 11, 13).unwrap();
    assert_eq!(src, original, "snapshot restore did not recover the original rectangle");
    assert!(capture_action_overlay_snapshot(&src, STRIDE as i64, 29, 29).is_err(), "out-of-bounds 72x72 snapshot");
    assert!(
        restore_action_overlay_snapshot(&mut src, &snapshot[..snapshot.len() - 1], STRIDE as i64, 11, 13).is_err(),
        "malformed snapshot was accepted"
    );
}

// Go: TestActionOverlayFrameOffsets
#[test]
fn action_overlay_frame_offsets_open_and_close() {
    let got = action_overlay_frame_offsets(1, false).unwrap();
    assert_eq!(got, [0x390 - 2 * 0x8e8, 0x390 - 12, 0x390 + 12, 0x390 + 2 * 0x8e8], "opening offsets");
    let got = action_overlay_frame_offsets(0, true).unwrap();
    assert_eq!(got, [-0x23a0 + 0x8e8, 0x378 + 6, 0x3a8 - 6, 0x2ac0 - 0x8e8], "closing offsets");
    assert!(action_overlay_frame_offsets(4, false).is_err(), "invalid frame was accepted");
}

// Go: TestBlitActionOverlayFrameUsesNativeCellsAndTransparency
#[test]
fn blit_action_overlay_frame_uses_native_cells_and_transparency() {
    let mut cells: Vec<RawCell> =
        (0..75).map(|i| RawCell { width: 1, height: 1, pixels: vec![(i + 1) as u8] }).collect();
    // 方向 1 刻意放一個透明像素，必須保留目的端。
    cells[62].pixels[0] = 0;
    let state = ActionOverlayState { availability: [0, 1, 0, 1], direction_state: [0x12, 0x14, 0x16, 0x18] };
    const STRIDE: i64 = 500;
    let mut dst = vec![0xeeu8; (STRIDE * 30) as usize];
    const ORIGIN: i64 = STRIDE * 10 + 100;
    blit_action_overlay_frame(&cells, state, &mut dst, STRIDE, ORIGIN, 1, false).unwrap();
    let offsets = action_overlay_frame_offsets(1, false).unwrap();
    let at = |k: usize| dst[(ORIGIN + offsets[k]) as usize];
    assert_eq!(at(0), 55, "up pixel"); // cell 54
    assert_eq!(at(1), 0xee, "left transparent pixel"); // transparent cell 62
    assert_eq!(at(2), 67, "right pixel"); // cell 66
    assert_eq!(at(3), 75, "down pixel"); // cell 74
}

// Go: TestActionOverlaySelectionBlinkFollowsProvenTickWindow
// 直接指令依據：sub_17898 的 0x178BF..0x178F8。差值 0..3 不動；大於 3 或為負（BIOS 低字回繞）才把
// 相位在 0／1 之間翻動並更新時間戳。
#[test]
fn action_overlay_selection_blink_follows_proven_tick_window() {
    let mut blink = ActionOverlaySelectionBlink::default();
    for tick in [0, 1, 2, 3] {
        blink.advance(tick);
        assert!(blink.phase == 0 && blink.tick == 0, "差值 {tick} 之內不得翻相位，得到 {blink:?}");
    }
    blink.advance(4);
    assert!(blink.phase == 1 && blink.tick == 4, "差值 4 應翻到相位 1 並更新時間戳，得到 {blink:?}");
    blink.advance(7);
    assert_eq!(blink.phase, 1, "差值 3 不得再翻");
    blink.advance(8);
    assert!(blink.phase == 0 && blink.tick == 8, "第二次翻動應回到相位 0，得到 {blink:?}");
    // 0x178D3 的 test/jge：負差值同樣進入翻動分支。
    blink.advance(-1);
    assert!(blink.phase == 1 && blink.tick == -1, "負差值應翻動，得到 {blink:?}");
}

// Go: TestBattleActionOverlaySelectedCellAddsBlinkPhase
// 直接指令依據：sub_179D5 的 0x17A60..0x17A8B。未選中的方向維持 3*first+2*second；選中方向再加上
// [0x53c13]。
#[test]
fn battle_action_overlay_selected_cell_adds_blink_phase() {
    let state = battle_action_overlay_state([1, 1, 0, 0]);
    for (direction, &want) in [2, 5, 6, 9].iter().enumerate() {
        assert_eq!(state.cell_index(direction as i64).unwrap(), want, "方向 {direction} 基礎格");
    }
    for phase in [0, 1] {
        assert_eq!(state.selected_cell_index(2, phase).unwrap(), 6 + phase, "選中方向 2 相位 {phase}");
    }
    assert!(state.selected_cell_index(2, 2).is_err(), "相位只證實 0..1，其他值必須拒絕");
}

// Go: TestActionOverlayInitialDirectionPicksFirstAvailable
// 直接指令依據：sub_173E7 的 0x173F5..0x1741B。
#[test]
fn action_overlay_initial_direction_picks_first_available() {
    let cases = [([0, 0, 0, 0], 0), ([1, 0, 0, 0], 1), ([1, 1, 0, 0], 2), ([1, 1, 1, 0], 3), ([1, 1, 1, 1], 4)];
    for (availability, want) in cases {
        assert_eq!(action_overlay_initial_direction(availability), want, "availability {availability:?}");
    }
}

// Go: TestActionOverlayRejectsUnavailableDirection
// 直接指令依據：sub_177FC 的 0x17835..0x17897，四個掃描碼各自檢查自己那一格。
#[test]
fn action_overlay_rejects_unavailable_direction() {
    let availability = [1, 0, 1, 0];
    for (direction, &want) in [false, true, false, true].iter().enumerate() {
        assert_eq!(action_overlay_accepts_direction(availability, direction as i64), want, "方向 {direction}");
    }
    assert!(
        !action_overlay_accepts_direction(availability, -1) && !action_overlay_accepts_direction(availability, 4),
        "範圍外的方向必須拒絕"
    );
}
