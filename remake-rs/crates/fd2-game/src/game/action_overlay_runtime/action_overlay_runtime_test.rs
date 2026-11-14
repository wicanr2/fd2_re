//! C 組：指令環與系統游標選單的測試。Go 來源：`native_action_overlay_test.go`（指令環生命週期、
//! availability、系統游標選單）。Go 的 `markActionOverlayDrawn` 由 `Draw` 呼叫；這裡照 Go 測試直接呼叫，
//! 等同呈現回報（PORTING.md「C 組：呈現回報」）。

use super::*;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};

/// Go：`native_continue_handoff_test.go:755` `nativeSystemOverlayTestCells`。
pub(crate) fn native_system_overlay_test_cells() -> Vec<Option<Arc<Picture>>> {
    let mut cells = vec![None; NATIVE_ACTION_OVERLAY_CELL_COUNT as usize];
    for index in [12, 15, 18, 21] {
        cells[index] = Some(Arc::new(Picture { width: 1, height: 1, pix: vec![0; 4] }));
    }
    cells
}

fn empty_cursor_game(cells: Vec<Option<Arc<Picture>>>) -> Game {
    let mut st = battle::State::default();
    (st.w, st.h) = (20, 20);
    Game {
        m: Some(MapData { w: 20, h: 20, tile_w: 24, tile_h: 24, ..Default::default() }),
        st: Some(st),
        cur_x: 9,
        cur_y: 7,
        native_action_cells: cells,
        ..Default::default()
    }
}

/// Go：`native_action_overlay_test.go:57` `TestNativeSystemOverlayUses16F55CellState`
#[test]
fn test_native_system_overlay_uses16_f55_cell_state() {
    let mut g = Game { native_system_cursor_overlay: true, ..Default::default() };
    let state = g.native_action_overlay_state();
    let want = [21, 15, 18, 12];
    for (direction, want) in want.into_iter().enumerate() {
        let got = state.cell_index(direction as i64);
        assert_eq!(got.ok(), Some(want), "direction {direction}");
    }
}

/// Go：`native_action_overlay_test.go:69` `TestSharedBattleEmptyCursorOpensNativeSystemOverlay`
#[test]
fn test_shared_battle_empty_cursor_opens_native_system_overlay() {
    let mut g = empty_cursor_game(native_system_overlay_test_cells());
    g.confirm();
    assert!(
        g.ring
            && g.native_system_cursor_overlay
            && g.ring_sel == 0
            && g.action_overlay_phase == ACTION_OVERLAY_OPENING
            && g.sel.is_none(),
        "shared 0x117E7 empty-cursor overlay: ring={} system={} sel={} phase={:?}",
        g.ring,
        g.native_system_cursor_overlay,
        g.ring_sel,
        g.action_overlay_phase
    );
}

/// Go：`native_action_overlay_test.go:84` `TestSharedBattleEmptyCursorRejectsIncompleteNativeOverlay`
#[test]
fn test_shared_battle_empty_cursor_rejects_incomplete_native_overlay() {
    let mut cells = native_system_overlay_test_cells();
    cells[18] = None;
    let mut g = empty_cursor_game(cells);
    g.confirm();
    assert!(
        !g.ring && !g.native_system_cursor_overlay && g.sel.is_none(),
        "incomplete FDOTHER #2 opened invisible system overlay"
    );
}

/// Go：`native_action_overlay_test.go:218` `TestActionOverlayLifecyclePresentsAllOpeningAndClosingFrames`
#[test]
fn test_action_overlay_lifecycle_presents_all_opening_and_closing_frames() {
    let mut g = Game::default();
    g.begin_action_overlay_open(2);
    assert!(g.ring && g.ring_sel == 2 && g.action_overlay_blocks_input(), "opening phase {:?}", g.action_overlay_phase);
    for want in 0..4 {
        let (frame, closing) = g.action_overlay_render_state();
        assert!(frame == want && !closing, "opening present {want} = frame{frame} closing={closing}");
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    assert!(
        g.ring && g.action_overlay_phase == ACTION_OVERLAY_OPEN && !g.action_overlay_blocks_input(),
        "settled open phase {:?}",
        g.action_overlay_phase
    );

    let completed = Arc::new(AtomicBool::new(false));
    let flag = Arc::clone(&completed);
    g.begin_action_overlay_close(Some(Box::new(move |_: &mut Game| flag.store(true, Ordering::SeqCst))));
    for want in 0..4 {
        let (frame, closing) = g.action_overlay_render_state();
        let done = completed.load(Ordering::SeqCst);
        assert!(
            frame == want && closing && !done,
            "closing present {want} = frame{frame} closing={closing} completed={done}"
        );
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    assert!(
        !g.ring && g.action_overlay_phase.is_empty() && completed.load(Ordering::SeqCst),
        "completed close phase {:?}",
        g.action_overlay_phase
    );
}

/// Go：`native_action_overlay_test.go:251` `TestActionOverlayCloseDefersChildMenuUntilFourthPresent`
#[test]
fn test_action_overlay_close_defers_child_menu_until_fourth_present() {
    let mut g = Game::default();
    g.begin_action_overlay_open(1);
    while g.action_overlay_blocks_input() {
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    g.begin_action_overlay_close(Some(Box::new(|g: &mut Game| g.native_command_open = true)));
    for present in 0..4 {
        assert!(!g.native_command_open, "child menu opened beneath close frame {present}");
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    assert!(g.native_command_open && !g.ring, "close did not hand off to child menu");
}

/// Go：`native_action_overlay_test.go:330` `TestActionOverlayLifecycleCannotSkipAnUndrawnFrame`
#[test]
fn test_action_overlay_lifecycle_cannot_skip_an_undrawn_frame() {
    let mut g = Game::default();
    g.begin_action_overlay_open(1);
    g.step_action_overlay_lifecycle();
    g.step_action_overlay_lifecycle();
    assert_eq!(g.action_overlay_render_state(), (0, false), "undrawn opening frame advanced");
    g.mark_action_overlay_drawn();
    g.step_action_overlay_lifecycle();
    assert_eq!(g.action_overlay_render_state().0, 1, "drawn opening frame did not advance");
}

/// Go 測試把選取單位放在 `g.sel`、不在 `st.Units` 裡；Rust 的 `sel` 是索引，先把它放在最後
/// （R2 所有權第 6 條）。availability 只掃非我方單位，多出的我方單位不影響結果。
fn availability_game(selected: battle::Unit, others: Vec<battle::Unit>) -> Game {
    let mut st = battle::State::default();
    st.units = others;
    st.units.push(selected);
    let sel = st.units.len() - 1;
    Game { st: Some(st), sel: Some(sel), ..Default::default() }
}

fn sel_unit(g: &mut Game) -> &mut battle::Unit {
    let sel = g.sel.unwrap();
    &mut g.st.as_mut().unwrap().units[sel]
}

/// Go：`native_action_overlay_test.go:345` `TestActionOverlayAvailabilityUsesCurrentRemakeGates`
#[test]
fn test_action_overlay_availability_uses_current_remake_gates() {
    let mut selected = battle::Unit::default();
    (selected.spells, selected.inventory) = (vec![1], vec![2]);
    let mut g = availability_game(selected, Vec::new());
    assert_eq!(g.action_overlay_availability(), [1, 0, 0, 0]);
    sel_unit(&mut g).sealed = true;
    sel_unit(&mut g).inventory = Vec::new();
    assert_eq!(g.action_overlay_availability(), [1, 1, 1, 0]);
}

fn enemy_at(x: i64, y: i64) -> battle::Unit {
    let mut enemy = battle::Unit::default();
    (enemy.on_field, enemy.hp, enemy.camp, enemy.x, enemy.y) = (true, 10, battle::ENEMY, x, y);
    enemy
}

/// Go：`native_action_overlay_test.go:357` `TestActionOverlayAvailabilityRequiresRecoveredNativeInventoryGates`
#[test]
fn test_action_overlay_availability_requires_recovered_native_inventory_gates() {
    let mut selected = battle::Unit::default();
    (selected.on_field, selected.hp, selected.atk_min, selected.atk_max) = (true, 10, 1, 1);
    (selected.inventory, selected.equipped) = (vec![3], vec![false]);
    let mut g = availability_game(selected, vec![enemy_at(1, 0)]);
    assert_eq!(g.action_overlay_availability(), [1, 1, 0, 0], "without equipped weapon/raw command");
    sel_unit(&mut g).equipped[0] = true;
    sel_unit(&mut g).native_command_mask[0] = 0x01;
    assert_eq!(g.action_overlay_availability(), [0, 0, 0, 0], "with equipped weapon/raw command");
}

/// Go：`native_action_overlay_test.go:371` `TestActionOverlayNativeCommandGateUsesRawOffset27NotLegacySeal`
#[test]
fn test_action_overlay_native_command_gate_uses_raw_offset27_not_legacy_seal() {
    let mut selected = battle::Unit::default();
    (selected.on_field, selected.hp, selected.atk_min, selected.atk_max) = (true, 10, 1, 1);
    (selected.inventory, selected.equipped, selected.native_command_mask) = (vec![3], vec![true], [1, 0, 0, 0, 0]);
    let mut g = availability_game(selected, vec![enemy_at(1, 0)]);
    sel_unit(&mut g).sealed = true; // normalized legacy state is not the native +0x27 gate.
    assert_eq!(g.action_overlay_availability(), [0, 0, 0, 0], "legacy seal must not disable raw command");
    sel_unit(&mut g).native_transient[5] = 1; // exact raw unit+0x27.
    assert_eq!(g.action_overlay_availability(), [0, 1, 0, 0], "raw +0x27 must disable command");
}

/// 0x18a5f..0x18a72：走過路之後只有 raw +7 為 0x12／0x13／0x22 的單位還能選指令；
/// 原地開環沒有這一條。
///
/// Go：`native_action_overlay_test.go:391` `TestActionOverlayCommandDisabledAfterMoveExceptThreeFigs`
#[test]
fn test_action_overlay_command_disabled_after_move_except_three_figs() {
    let mut selected = battle::Unit::default();
    (selected.on_field, selected.hp, selected.atk_min, selected.atk_max, selected.x, selected.y) =
        (true, 10, 1, 1, 2, 2);
    (selected.inventory, selected.equipped, selected.native_command_mask, selected.battle_fig) =
        (vec![3], vec![true], [1, 0, 0, 0, 0], 9);
    let mut g = availability_game(selected, vec![enemy_at(3, 2)]);
    (g.moved, g.sel_orig_x, g.sel_orig_y) = (true, 2, 2);
    assert_eq!(g.action_overlay_availability(), [0, 0, 0, 0], "原地開環");
    (g.sel_orig_x, g.sel_orig_y) = (2, 5);
    assert_eq!(g.action_overlay_availability(), [0, 1, 0, 0], "走過路");
    for fig in [0x12, 0x13, 0x22] {
        sel_unit(&mut g).battle_fig = fig;
        assert_eq!(g.action_overlay_availability(), [0, 0, 0, 0], "fig {fig:#x} 走過路");
    }
}

/// Go：`native_action_overlay_test.go:415` `TestNativeActionSelectableRejectsDisabledWordAndInvalidDirection`
#[test]
fn test_native_action_selectable_rejects_disabled_word_and_invalid_direction() {
    let availability = [0, 1, 0, 0];
    for direction in [-1, 1, 4] {
        assert!(!native_action_selectable(availability, direction), "direction {direction} must not be selectable");
    }
    assert!(native_action_selectable(availability, 0) && native_action_selectable(availability, 3));
}

/// Rust 新增：`resetActionOverlayLifecycle` 清掉 END 對話與指令環的全部狀態；只有 END 對話存在時才順帶清
/// `nativeClassUIJob`（`action_overlay_runtime.go:158`）。
#[test]
#[ignore = "等 D 合併（cancel_native_command32_presentation 等四個指令演出取消）"]
fn reset_action_overlay_lifecycle_clears_end_turn_owner_and_ring() {
    let mut g = Game::default();
    g.begin_action_overlay_open(3);
    (g.native_system_cursor_overlay, g.native_system_end_turn_delay, g.native_system_exit_requested) = (true, 5, true);
    g.native_class_ui_job = Some(NativeClassUIJob::default());
    g.reset_action_overlay_lifecycle();
    assert!(!g.ring && g.action_overlay_phase.is_empty() && g.native_class_ui_job.is_some());
    assert!(!g.native_system_cursor_overlay && g.native_system_end_turn_delay == 0 && !g.native_system_exit_requested);
    g.native_system_end_turn_ui = Some(Box::default());
    g.reset_action_overlay_lifecycle();
    assert!(g.native_class_ui_job.is_none() && g.native_system_end_turn_ui.is_none());
}

/// Rust 新增：`currentNativeSystemOptions` 沒有設定時回原版預設值，有設定時回設定值。
#[test]
fn current_native_system_options_defaults_to_original() {
    let mut g = Game::default();
    assert_eq!(g.current_native_system_options(), fdother::default_native_system_options());
    let mut options = fdother::default_native_system_options();
    options.raw_53af9 = 1;
    g.native_system_options = Some(options);
    assert_eq!(g.current_native_system_options(), options);
}

/// Rust 新增：方向鍵只在 availability 為零時寫入 `ringSel`（0x17835..0x17897），讀的是本幀 `TickInput`。
#[test]
fn overlay_direction_input_respects_availability_gate() {
    let mut selected = battle::Unit::default();
    (selected.spells, selected.inventory) = (vec![1], vec![2]);
    let mut g = availability_game(selected, Vec::new()); // availability [1,0,0,0]
    g.ring_sel = 3;
    g.tick_input.pressed.insert(Key::ArrowUp);
    g.apply_native_overlay_direction_input();
    assert_eq!(g.ring_sel, 3, "不可用的攻擊方向不得改變選擇");
    g.tick_input.pressed = BTreeSet::from([Key::ArrowLeft]);
    g.apply_native_overlay_direction_input();
    assert_eq!(g.ring_sel, 1);
    g.begin_battle_action_overlay();
    assert_eq!(g.ring_sel, 1, "sub_173E7 起始方向是第一個可用方向");
}

/// Rust 新增：END 延遲倒數到零時若狀態不完整就什麼都不做，不建立收合工作。
#[test]
fn native_system_end_turn_delay_without_state_is_noop() {
    let mut g = Game { native_system_end_turn_delay: 2, ..Default::default() };
    g.step_native_system_end_turn();
    assert_eq!(g.native_system_end_turn_delay, 1);
    g.step_native_system_end_turn();
    assert_eq!(g.native_system_end_turn_delay, 0);
    assert!(g.native_class_ui_job.is_none());
}
