//! C 組在 `main.go` 與其他 C 組檔名下函式的測試（Go 測試檔名不對應單一 C 組 Rust 檔的，集中在這裡）。
//! Go 來源：`native_ai_input_lock_test.go`、`native_player_movement_test.go`、`native_player_input_test.go`、
//! `attack_resolution_test.go`、`treasure_test.go`、`native_ai_item_presentation_test.go`、
//! `native_ai_modifier_presentation_test.go`、`native_command_locale_test.go`、`native_battle_name_test.go`、
//! `native_system_info_ui_test.go`、`chapter_parity_ring_open_test.go`。

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::{
    complete_native_map_frame_fixture, native_frame_test_sprite,
};
use crate::test_support;

fn own_unit(hp: i64, max_hp: i64, x: i64, y: i64) -> battle::Unit {
    let mut u = battle::Unit::default();
    (u.camp, u.on_field, u.hp, u.max_hp, u.x, u.y) = (battle::OWN, true, hp, max_hp, x, y);
    u
}

/// Go：`native_ai_input_lock_test.go:10` `TestNativeAIPhaseRejectsMapConfirmReentry`
#[test]
fn test_native_ai_phase_rejects_map_confirm_reentry() {
    let mut st = battle::State::default();
    (st.w, st.h) = (2, 2);
    st.units = vec![own_unit(20, 20, 0, 0)];
    let mut g = Game {
        st: Some(st),
        m: Some(MapData { w: 2, h: 2, ..Default::default() }),
        ai_busy: true,
        ..Default::default()
    };
    let before = g.st.as_ref().unwrap().units[0].clone();
    g.confirm();
    assert!(
        g.sel.is_none() && g.native_move_plan.is_none() && !g.ring && g.st.as_ref().unwrap().units[0] == before,
        "AI phase accepted a player selection or changed its actor"
    );
    g.ai_busy = false;
    g.confirm();
    assert_eq!(g.sel, Some(0), "player selection did not resume after AI completion");
}

/// Go：`native_player_movement_test.go:9` `TestNativePhysicalAttackOwnCellConfirmPreservesTargetState`
#[test]
fn test_native_physical_attack_own_cell_confirm_preserves_target_state() {
    let mut actor = own_unit(10, 20, 0, 0);
    actor.has_native_record_byte5 = true;
    let mut enemy = battle::Unit::default();
    (enemy.camp, enemy.on_field, enemy.x, enemy.y, enemy.hp, enemy.max_hp, enemy.has_native_record_byte5) =
        (battle::ENEMY, true, 1, 0, 12, 12, true);
    let mut st = battle::State::default();
    (st.w, st.h, st.units) = (2, 1, vec![actor, enemy]);
    (st.has_native_map_view_state, st.native_map_range_mode, st.native_tile_blit_modes) = (true, 1, vec![0xff, 0]);
    let mut g = Game {
        st: Some(st),
        m: Some(MapData { w: 2, h: 1, ..Default::default() }),
        sel: Some(0),
        moved: true,
        native_rng_state: 17791,
        ..Default::default()
    };
    let before_units = g.st.as_ref().unwrap().units.clone();
    let before_field = g.st.as_ref().unwrap().native_tile_blit_modes.clone();
    g.confirm();
    let st = g.st.as_ref().unwrap();
    assert!(
        g.sel == Some(0)
            && g.moved
            && !g.ring
            && g.atk.is_none()
            && g.native_rng_state == 17791
            && st.units == before_units
            && st.native_tile_blit_modes == before_field,
        "rejected own-cell confirmation changed target state: sel={:?}",
        g.sel
    );
    // 原生閘門在合法敵方格通過，自己格即使被誤標非FF仍不能當敵方。
    g.cur_x = 1;
    assert!(g.native_physical_attack_confirmation_allowed(), "active enemy inside marked field was rejected");
    g.cur_x = 0;
    g.st.as_mut().unwrap().native_tile_blit_modes[0] = 0;
    assert!(!g.native_physical_attack_confirmation_allowed(), "own unit was accepted as physical enemy target");
    g.st.as_mut().unwrap().native_tile_blit_modes = Vec::new();
    g.confirm();
    let actor = &g.st.as_ref().unwrap().units[0];
    assert!(!actor.acted && actor.hp == 10 && g.sel == Some(0), "missing native field submitted wait");
}

/// Go：`native_item_panel_ui_test.go:13` `nativeItemPanelTestUnit` 與 `native_player_input_test.go:9`
/// `nativeNeutralTestUnit`。
fn native_neutral_test_unit(x: u8, y: u8) -> battle::Unit {
    let mut u = battle::Unit::default();
    (u.battle_fig, u.native_identity, u.has_native_identity) = (0, 0, true);
    (u.native_record_byte6, u.has_native_record_byte6) = (1, true);
    (u.native_record_race, u.has_native_record_race) = (1, true);
    (u.native_record_class, u.has_native_record_class) = (1, true);
    (u.lv, u.mv, u.exp, u.dx) = (12, 5, 34.0, 56);
    (u.hp, u.max_hp, u.mp, u.max_mp) = (80, 100, 20, 40);
    (u.ap, u.dp, u.hit, u.ev) = (123, 98, 76, 54);
    u.inventory_slots = vec![0, 0xff, 79, 0xff, 0xff, 0xff, 0xff, 0xff].into();
    u.native_inventory_flags = vec![0x40, 0x80, 0, 0x80, 0x80, 0x80, 0x80, 0x80].into();
    (u.has_battle_fig, u.has_native_record_byte5) = (true, true);
    u.native_record_byte6 = 2;
    u.has_native_map_presentation = true;
    (u.native_map_presentation.x, u.native_map_presentation.y) = (x, y);
    (u.has_native_record_byte34, u.has_native_record_byte35, u.has_native_record_byte36) = (true, true, true);
    (u.has_native_record_word42, u.has_native_record_word46) = (true, true);
    (u.native_record_word42, u.native_record_word46) = (100, 40);
    u
}

/// Go：`native_player_input_test.go:72` `TestNativePlayerStatusGatesAndFailedResourcesDoNotSelect`。
/// Go 以 `t.Setenv("FD2_ASSET_PACK", 空目錄)` 的第三段在子行程執行。Go 失敗時回 `(true, err)`，Rust 的
/// `Result<bool>` 只能回 `Err`；呼叫端（`confirm`）同樣把它當成已消費。
#[test]
fn test_native_player_status_gates_and_failed_resources_do_not_select() {
    let name = "test_native_player_status_gates_and_failed_resources_do_not_select";
    let Some(dir) = f_child_env(module_path!(), name, |d| vec![("FD2_ASSET_PACK", d.into())]) else {
        return;
    };
    let mut st = battle::State::default();
    st.units = vec![native_neutral_test_unit(7, 14)];
    let mut g = Game { st: Some(st), ..Default::default() };
    let handled = g.inspect_native_player_unit(0, false);
    assert!(matches!(handled, Ok(false)), "normal unit was intercepted: {handled:?}");
    g.st.as_mut().unwrap().units[0].battle_fig = 0x79;
    let handled = g.inspect_native_player_unit(0, true);
    assert!(matches!(handled, Ok(true)) && g.native_player_status.is_none(), "special record did not reject");
    g.st.as_mut().unwrap().units[0].battle_fig = 0;
    g.st.as_mut().unwrap().units[0].native_record_byte5 = 0x80;
    assert_eq!(std::env::var_os("FD2_ASSET_PACK").as_deref(), Some(dir.as_os_str()));
    let handled = g.inspect_native_player_unit(0, false);
    assert!(
        handled.is_err() && g.native_player_status.is_none() && g.sel.is_none(),
        "missing status resources did not fail closed: {:?}",
        handled.map_err(|e| e.message().to_string())
    );
}

/// 重製端補充：用分離素材包的真實面板資源開啟玩家狀態。Go 沒有對應測試；本項只核對 0x17AED
/// 排程的幀數、穩定幀與頁別次序，不代表與原版逐像素一致。
#[test]
fn test_native_player_status_opens_from_separated_assets_and_closes() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    let (assets, field, mut state) = complete_native_map_frame_fixture();
    {
        let n = native_neutral_test_unit(0, 0);
        let u = &mut state.units[0];
        (u.native_identity, u.has_native_identity, u.battle_fig) = (n.native_identity, true, 0);
        (u.native_record_race, u.native_record_class) = (n.native_record_race, n.native_record_class);
        (u.lv, u.mv, u.exp, u.dx) = (n.lv, n.mv, n.exp, n.dx);
        (u.hp, u.max_hp, u.mp, u.max_mp) = (n.hp, n.max_hp, n.mp, n.max_mp);
        (u.ap, u.dp, u.hit, u.ev) = (n.ap, n.dp, n.hit, n.ev);
        (u.inventory_slots, u.native_inventory_flags) = (n.inventory_slots.clone(), n.native_inventory_flags.clone());
    }
    let unit = state.units[0].clone();
    let mut g = Game {
        m: Some(field),
        st: Some(state),
        native_map_assets: Some(assets),
        native_ui_palette: load_native_ui_palette(),
        // 地圖幀的輔助時序：測試以截圖模式的虛擬時鐘提供。
        shot_path: String::from("player-status-test"),
        virtual_now: VIRTUAL_CLOCK_ORIGIN,
        ..Default::default()
    };
    if let Err(err) = g.begin_native_player_status(&unit) {
        panic!("status did not open: {}", err.message());
    }
    let s = g.native_player_status.as_ref().unwrap();
    assert!(s.phase == "opening" && s.frames.len() == 12 && s.steady.len() == 64000 && s.source.len() == 64000);
    assert!(s.frames[11] == s.steady && s.frames[0] != s.steady, "opening schedule does not end on the steady frame");
    assert!(s.commands.is_empty() == unit.native_command_ids().is_empty(), "command page presence mismatch");
    for _ in 0..12 {
        g.native_player_status.as_mut().unwrap().drawn = true;
        g.step_native_player_status();
    }
    let s = g.native_player_status.as_ref().unwrap();
    assert!(s.phase == "status" && s.frames.is_empty(), "status page not steady: {}", s.phase);
    // 沒有指令頁時確認鍵直接關閉；有指令頁時先捲到指令頁。
    let pages = if s.commands.is_empty() { 1 } else { 2 };
    for page in 0..pages {
        g.advance_native_player_status();
        let n = g.native_player_status.as_ref().unwrap().frames.len();
        assert_eq!(n, if page + 1 < pages { 14 } else { 12 }, "page {page} transition frames");
        for _ in 0..n {
            g.native_player_status.as_mut().unwrap().drawn = true;
            g.step_native_player_status();
        }
    }
    assert!(g.native_player_status.is_none() && g.load_err.is_empty(), "status did not close: {:?}", g.load_err);
}

/// Go：`native_player_input_test.go:90` `TestNativePlayerStatusTwoPagesWaitForDrawAndRestore`
#[test]
fn test_native_player_status_two_pages_wait_for_draw_and_restore() {
    let s = NativePlayerStatusState {
        source: vec![0; 64000],
        status: vec![0; 64000],
        commands: vec![0; 64000],
        phase: String::from("status"),
        ..Default::default()
    };
    let mut g = Game { native_player_status: Some(s), ..Default::default() };
    g.advance_native_player_status();
    assert_eq!(g.native_player_status.as_ref().unwrap().frames.len(), 14, "command transition missing");
    g.step_native_player_status();
    assert_eq!(g.native_player_status.as_ref().unwrap().frame, 0, "undrawn frame advanced");
    for _ in 0..14 {
        g.native_player_status.as_mut().unwrap().drawn = true;
        g.step_native_player_status();
    }
    let s = g.native_player_status.as_ref().unwrap();
    assert!(s.phase == "commands" && s.frames.is_empty(), "command page not steady");
    g.advance_native_player_status();
    for _ in 0..12 {
        g.native_player_status.as_mut().unwrap().drawn = true;
        g.step_native_player_status();
    }
    assert!(g.native_player_status.is_none(), "status did not close");
}

/// Go：`native_player_input_test.go:49` `TestNativePlayerFocusUsesRawBattleViewAndXFirstSafeBand`
#[test]
fn test_native_player_focus_uses_raw_battle_view_and_x_first_safe_band() {
    let mut st = battle::State::default();
    (st.w, st.h) = (30, 30);
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        camera_y: 13,
        cursor_x: 7,
        cursor_y: 14,
        visible_cursor_x: 7,
        visible_cursor_y: 1,
        ..Default::default()
    })
    .expect("view");
    st.units = vec![native_neutral_test_unit(15, 20)];
    let mut g = Game {
        st: Some(st),
        m: Some(MapData { w: 30, h: 30, tile_w: 24, tile_h: 24, ..Default::default() }),
        has_story_native_map_view: true,
        story_native_map_view: battle::NativeMapViewState { cursor_x: 25, ..Default::default() },
        ..Default::default()
    };
    assert!(g.begin_native_player_focus(Some(0)), "{}", g.load_err);
    g.step_native_player_focus();
    let v = g.st.as_ref().unwrap().native_map_view_state;
    assert!(v.cursor_x == 8 && v.cursor_y == 14, "focus did not move X first");
    for _ in 0..50 {
        if g.native_player_focus.is_none() {
            break;
        }
        g.step_native_player_focus();
    }
    let v = g.st.as_ref().unwrap().native_map_view_state;
    assert!(
        g.native_player_focus.is_none()
            && v.cursor_x == 15
            && v.cursor_y == 20
            && v.camera_x == 4
            && v.camera_y == 14
            && g.story_native_map_view.cursor_x == 25,
        "focus state {v:?} err={}",
        g.load_err
    );
}

/// Go：`native_player_movement_test.go:44` `TestNativePlayerMovementCancelRestoresRawCursorBeforeSync`
/// （受測的 `restorePlayerMovementCursor` 隨 `ringInput` 補移植）。
#[test]
fn test_native_player_movement_cancel_restores_raw_cursor_before_sync() {
    let mut st = battle::State::default();
    (st.w, st.h) = (30, 30);
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        camera_x: 0,
        camera_y: 12,
        cursor_x: 7,
        cursor_y: 13,
        visible_cursor_x: 7,
        visible_cursor_y: 1,
    })
    .expect("view");
    let mut g = Game {
        st: Some(st),
        m: Some(MapData { w: 30, h: 30, tile_w: 24, tile_h: 24, ..Default::default() }),
        sel_orig_x: 7,
        sel_orig_y: 14,
        ..Default::default()
    };
    g.restore_player_movement_cursor();
    g.sync_native_map_view();
    let raw = g.st.as_ref().unwrap().native_map_view_state;
    assert!(g.cur_x == 7 && g.cur_y == 14 && raw.cursor_y == 14, "cancel lost its destination after sync: {raw:?}");
}

fn zh_hant_catalog() -> Option<localization::Catalog> {
    test_support::remake_asset("locales/zh-Hant/pack.json")?;
    Some(load_official_locale("zh-Hant").expect("zh-Hant").unwrap())
}

fn named(name: &str) -> battle::Unit {
    let mut u = battle::Unit::default();
    u.name = name.to_string();
    u
}

/// Go：`attack_resolution_test.go:51` `TestPlayerPhysicalAttackMessagePreservesSettlementResult`
#[test]
fn test_player_physical_attack_message_preserves_settlement_result() {
    let Some(catalog) = zh_hant_catalog() else { return };
    let (actor, target) = (named("亞雷斯"), named("盜賊"));
    let miss = battle::AttackResult { missed: true, ..Default::default() };
    let miss = player_physical_attack_message(Some(&catalog), Some(&actor), Some(&target), &miss).expect("miss");
    assert!(miss.contains("未命中"), "miss message={miss:?}");
    let hit = battle::AttackResult { amount: 12, crit: true, exp_gained: 8.0, ..Default::default() };
    let hit = player_physical_attack_message(Some(&catalog), Some(&actor), Some(&target), &hit).expect("hit");
    for want in ["造成 12 傷害", "暴擊", "經驗 +8"] {
        assert!(hit.contains(want), "hit message={hit:?} missing {want:?}");
    }
}

/// Go：`attack_resolution_test.go:76` `TestPlayerPhysicalAttackMessageUsesAllOfficialLocales`
#[test]
fn test_player_physical_attack_message_uses_all_official_locales() {
    if test_support::remake_asset("locales/en/pack.json").is_none() {
        return;
    }
    let (actor, target) = (named("Sol"), named("Bandit"));
    for (locale_id, want) in [("zh-Hant", "未命中"), ("zh-Hans", "未命中"), ("ja", "外れた"), ("en", "misses")]
    {
        let catalog = load_official_locale(locale_id).unwrap_or_else(|e| panic!("{locale_id}: {e}")).unwrap();
        let result = battle::AttackResult { missed: true, ..Default::default() };
        let message = player_physical_attack_message(Some(&catalog), Some(&actor), Some(&target), &result);
        assert!(message.as_ref().is_ok_and(|m| m.contains(want)), "{locale_id} message={message:?}, want {want:?}");
    }
}

/// Go：`attack_resolution_test.go:92` `TestPlayerPhysicalAttackMessageFailsClosedWithoutNames`
#[test]
fn test_player_physical_attack_message_fails_closed_without_names() {
    let Some(catalog) = zh_hant_catalog() else { return };
    let (thief, sol, empty) = (named("盜賊"), named("索爾"), battle::Unit::default());
    let result = battle::AttackResult { missed: true, ..Default::default() };
    for (name, actor, target) in [
        ("nil actor", None, Some(&thief)),
        ("nil target", Some(&sol), None),
        ("empty actor", Some(&empty), Some(&thief)),
        ("empty target", Some(&sol), Some(&empty)),
    ] {
        let got = player_physical_attack_message(Some(&catalog), actor, target, &result);
        assert!(got.is_err(), "{name}: message={got:?}");
    }
}

/// Go：`attack_resolution_test.go:120` `TestPhysicalAttackMessageStaysOutOfThePlayerPath`
#[test]
fn test_physical_attack_message_stays_out_of_the_player_path() {
    let mut g = Game::default();
    g.publish_physical_attack_message("索爾 攻擊 盜賊，造成 12 傷害");
    assert!(g.msg.is_empty(), "一般玩家路徑不得顯示戰鬥結果字串，實得 {:?}", g.msg);
    g.debug = true;
    g.publish_physical_attack_message("索爾 攻擊 盜賊，造成 12 傷害");
    assert!(!g.msg.is_empty(), "F3 診斷模式應保留字串");
}

fn treasure_state(cell: battle::Cell, treasure: battle::Treasure) -> battle::State {
    let mut st = battle::State::default();
    st.treasures = Some(BTreeMap::from([(cell, treasure)]));
    st.opened_treasure = Some(BTreeMap::new());
    st
}

fn opened(g: &Game, slot: i64) -> bool {
    g.st.as_ref().unwrap().opened_treasure.as_ref().is_some_and(|m| m.get(&slot).copied().unwrap_or(false))
}

#[test]
fn test_native_treasure_choice_publishes_reward_at_the_correct_boundary() {
    let Ok(Some(ui)) = load_native_preparation_ui_assets() else {
        test_support::skip("寶物確認介面素材");
        return;
    };
    let present_all = |g: &mut Game| {
        for _ in 0..32 {
            let Some(job) = g.native_class_ui_job.as_mut() else { return };
            job.drawn = true;
            g.step_native_class_ui_lifecycle(GameInstant::default());
        }
        panic!("寶物確認演出未結束");
    };
    for (kind, value, accepted) in [("item", 192, true), ("gold", 3500, true), ("item", 192, false)] {
        let reward = battle::Treasure { slot: 0, kind: kind.into(), value, ..Default::default() };
        let mut st = treasure_state(battle::Cell { x: 7, y: 13 }, reward.clone());
        st.units = vec![own_unit(40, 100, 7, 13), own_unit(40, 100, 8, 13)];
        let frame = vec![0; 320 * 200];
        let mut g = Game {
            st: Some(st),
            sel: Some(0),
            gold: 10,
            native_preparation_ui: Some(ui.clone()),
            native_system_end_turn_confirm: true,
            native_system_end_turn_ui: Some(Box::new(NativeSystemEndTurnUIState {
                source: frame.clone(),
                dialogue: frame.clone(),
                question: frame.clone(),
                accepted: vec![frame.clone()],
                canceled: vec![frame],
                treasure: Some(NativeTreasurePrompt { actor: Some(0), reward, x: 7, y: 13, ..Default::default() }),
                ..Default::default()
            })),
            ..Default::default()
        };
        g.tick_input.pressed.insert(Key::Enter);
        if !accepted {
            g.tick_input.pressed.insert(Key::ArrowRight);
        }
        assert!(g.ring_input());
        assert!(!opened(&g, 0));
        present_all(&mut g);
        assert_eq!(g.gold, 10);
        assert_eq!(opened(&g, 0), accepted && kind == "item");
        assert!(!g.st.as_ref().unwrap().units[0].acted);
        if accepted {
            assert!(g.ring_input());
        }
        for _ in 0..12 {
            g.step_native_system_end_turn();
        }
        present_all(&mut g);
        let unit = &g.st.as_ref().unwrap().units[0];
        assert!(g.load_err.is_empty() && g.sel.is_none() && unit.acted);
        assert_eq!(opened(&g, 0), accepted);
        assert_eq!(g.gold, if kind == "gold" { 3510 } else { 10 });
        assert_eq!(unit.inventory, if accepted && kind == "item" { vec![192] } else { vec![] });
    }
}

/// Go：`treasure_test.go:9` `TestFinishSelectedWaitRestsThenClaimsTreasure`
#[test]
fn test_finish_selected_wait_rests_then_claims_treasure() {
    let Some(catalog) = zh_hant_catalog() else { return };
    let mut st = treasure_state(
        battle::Cell { x: 18, y: 37 },
        battle::Treasure { slot: 0, kind: String::from("item"), value: 0xd2, ..Default::default() },
    );
    st.units = vec![own_unit(40, 100, 18, 37)];
    let mut g = Game {
        st: Some(st),
        sel: Some(0),
        sel_orig_x: 18,
        sel_orig_y: 37,
        moved: true,
        ring: true,
        locale_catalog: Some(catalog),
        ..Default::default()
    };
    g.finish_selected_wait();
    let u = &g.st.as_ref().unwrap().units[0];
    assert!(
        u.hp == 60 && u.inventory == vec![0xd2] && u.acted && opened(&g, 0),
        "rest/claim = hp{} inventory={:?} acted={}",
        u.hp,
        u.inventory,
        u.acted
    );
    assert!(g.msg == "取得物品 D2h" && g.sel.is_none() && !g.ring, "wait UI state msg={:?}", g.msg);
}

/// Go：`treasure_test.go:31` `TestFinishSelectedWaitAfterMoveDoesNotHealAndGoldUsesCampaignBank`
#[test]
fn test_finish_selected_wait_after_move_does_not_heal_and_gold_uses_campaign_bank() {
    let mut st = treasure_state(
        battle::Cell { x: 3, y: 4 },
        battle::Treasure { slot: 7, kind: String::from("gold"), value: 3000, hidden: true, ..Default::default() },
    );
    st.units = vec![own_unit(40, 100, 3, 4)];
    let mut g =
        Game { st: Some(st), sel: Some(0), sel_orig_x: 1, sel_orig_y: 1, moved: true, gold: 500, ..Default::default() };
    g.finish_selected_wait();
    assert!(
        g.st.as_ref().unwrap().units[0].hp == 40 && g.gold == 3500 && opened(&g, 7),
        "moved wait = hp{} gold{}",
        g.st.as_ref().unwrap().units[0].hp,
        g.gold
    );
}

/// Go：`treasure_test.go:46` `TestFinishSelectedWaitExecutesEditableNativeTreasureEvent`
#[test]
fn test_finish_selected_wait_executes_editable_native_treasure_event() {
    let Some(catalog) = zh_hant_catalog() else { return };
    let mut st = treasure_state(
        battle::Cell { x: 2, y: 3 },
        battle::Treasure { slot: 0, kind: String::from("event"), native_type: 2, value: 58, ..Default::default() },
    );
    st.native_treasure_event_rules = BTreeMap::from([(
        58,
        battle::NativeTreasureEventRule { event_id: 58, item_by_slot: vec![0x1d], open_slots: vec![0] },
    )]);
    st.units = vec![own_unit(10, 10, 2, 3)];
    let mut g = Game {
        st: Some(st),
        sel: Some(0),
        sel_orig_x: 2,
        sel_orig_y: 3,
        locale_catalog: Some(catalog),
        ..Default::default()
    };
    g.finish_selected_wait();
    let u = &g.st.as_ref().unwrap().units[0];
    assert!(
        u.inventory == vec![0x1d] && opened(&g, 0) && g.msg == "取得物品 1Dh",
        "event58 wait inventory={:?} msg={:?}",
        u.inventory,
        g.msg
    );
}

/// Go：`treasure_test.go:71` `TestFinishSelectedWaitHonorsNativeRecoveryGatesAndFloor`
#[test]
fn test_finish_selected_wait_honors_native_recovery_gates_and_floor() {
    for raw_index in [3, 4] {
        let mut u = own_unit(40, 100, 2, 3);
        u.native_transient[raw_index] = 1;
        let mut st = battle::State::default();
        st.units = vec![u];
        let mut g = Game { st: Some(st), sel: Some(0), sel_orig_x: 2, sel_orig_y: 3, ..Default::default() };
        g.finish_selected_wait();
        let hp = g.st.as_ref().unwrap().units[0].hp;
        assert_eq!(hp, 40, "raw +{:#x} gate healed", 0x22 + raw_index);
    }
    let mut st = battle::State::default();
    st.units = vec![own_unit(1, 4, 2, 3)];
    let mut g = Game { st: Some(st), sel: Some(0), sel_orig_x: 2, sel_orig_y: 3, ..Default::default() };
    g.finish_selected_wait();
    assert_eq!(g.st.as_ref().unwrap().units[0].hp, 1, "floor(maxHP/5) invented minimum heal");
}

/// Go：`native_ai_execute_test.go:153` `completeNativeAIExecutorUnit`。
fn complete_native_ai_executor_unit() -> battle::Unit {
    let mut u = battle::Unit::default();
    (u.on_field, u.hp, u.max_hp, u.mp, u.max_mp) = (true, 100, 100, 10, 10);
    (u.ap, u.dp, u.hit, u.ev, u.mv, u.lv) = (10, 20, 30, 40, 5, 1);
    u.has_native_map_presentation = true;
    (u.battle_fig, u.has_battle_fig) = (1, true);
    (u.native_record_byte8, u.has_native_record_byte8) = (1, true);
    (u.native_record_race, u.has_native_record_race) = (1, true);
    (u.native_record_class, u.has_native_record_class) = (1, true);
    (u.native_record_byte5, u.has_native_record_byte5) = (0, true);
    (u.native_record_byte6, u.has_native_record_byte6) = (0, true);
    (u.native_record_byte34, u.has_native_record_byte34) = (0x81, true);
    (u.native_record_byte35, u.has_native_record_byte35) = (0, true);
    (u.native_record_byte36, u.has_native_record_byte36) = (0, true);
    (u.native_record_word42, u.has_native_record_word42) = (100, true);
    (u.native_record_word46, u.has_native_record_word46) = (10, true);
    u.inventory_slots = vec![0xff; 8].into();
    u.native_inventory_flags = vec![0x80; 8].into();
    u
}

/// Go：`native_ai_item_presentation_test.go:513` `TestNativeAIItemPresentationRosterChangeCancelsWithoutPanic`
#[test]
fn test_native_ai_item_presentation_roster_change_cancels_without_panic() {
    let mut st = battle::State::default();
    st.units = vec![complete_native_ai_executor_unit()];
    let mut g = Game {
        st: Some(st),
        native_ai_item_presentation: Some(NativeAIItemPresentationJob {
            transaction: Some(NativeAIItemTransaction { before: vec![0; 160], rng_before: 7, ..Default::default() }),
            published: true,
            baseline_work: vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize],
            baseline_vga: vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
            ..Default::default()
        }),
        native_map_work: vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize],
        native_map_vga: vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
        ..Default::default()
    };
    g.cancel_native_ai_item_presentation();
    assert!(
        g.native_ai_item_presentation.is_none() && !g.load_err.is_empty(),
        "roster-change cancel err={:?}",
        g.load_err
    );
}

/// Go：`native_ai_modifier_presentation_test.go:98` `TestNativeAICommandModifierLateCancelRollsBack`
#[test]
fn test_native_ai_command_modifier_late_cancel_rolls_back() {
    let mut actor = complete_native_ai_executor_unit();
    (actor.camp, actor.mp, actor.ap, actor.native_record_byte6) = (battle::ENEMY, 10, 100, 0);
    let mut target = complete_native_ai_executor_unit();
    (target.camp, target.ap, target.native_record_byte6) = (battle::ENEMY, 200, 0);
    let mut book: Vec<battle::NativeCommandRecord> = (0..battle::NATIVE_COMMAND_RECORD_COUNT)
        .map(|id| battle::NativeCommandRecord { id, ..Default::default() })
        .collect();
    book[17] = battle::NativeCommandRecord { id: 17, effect_mode: 1, target_code: 1, ..Default::default() };
    book[18] = battle::NativeCommandRecord { id: 18, mp_cost: 4, ..Default::default() };
    (actor.native_map_presentation.x, target.native_map_presentation.x) = (0, 1);
    let mut st = battle::State::default();
    (st.w, st.h, st.units, st.native_composition_event_bytes, st.native_command_book) =
        (2, 1, vec![actor, target], vec![0; 2], book);
    let mut plan = st.plan_native_ai_command_modifier(Some(0), battle::Cell::default(), 17, 0).expect("plan");
    battle::publish_native_ai_command_modifier(&mut st.units, Some(&mut plan)).expect("publish");
    let base_work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let base_vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
    let rng = plan.result.rng_state;
    let mut g = Game {
        st: Some(st),
        native_rng_state: rng,
        native_map_work: base_work.clone(),
        native_map_vga: base_vga.clone(),
        ..Default::default()
    };
    g.native_ai_command_modifier = Some(NativeAICommandModifierPresentationJob {
        actor: Some(0),
        plan: Some(plan),
        rng_before: 0,
        baseline_work: base_work,
        baseline_vga: base_vga,
        ..Default::default()
    });
    g.cancel_native_ai_command_modifier_presentation();
    let units = &g.st.as_ref().unwrap().units;
    assert!(
        units[0].ap == 100
            && units[1].ap == 200
            && units[0].mp == 10
            && !units[0].acted
            && g.native_rng_state == 0
            && g.native_ai_command_modifier.is_none(),
        "cancel retained state actor={:?} target={:?}",
        (units[0].ap, units[0].mp, units[0].acted),
        units[1].ap
    );
}

/// Go：`native_command_locale_test.go:9` `TestLocalizedSpellBookUsesCommandCatalogAndKeepsEmptySlot`
#[test]
fn test_localized_spell_book_uses_command_catalog_and_keeps_empty_slot() {
    if test_support::remake_asset("locales/en/entities.json").is_none() {
        return;
    }
    let entities = load_official_locale_entities("en").expect("en").unwrap();
    let source: Vec<battle::Spell> =
        (0..36).map(|id| battle::Spell { id, name: String::from("舊名稱"), ..Default::default() }).collect();
    let got = localized_spell_book(Some(&entities), &source).expect("localized");
    assert!(
        got[21].name == "Cure Paralysis" && got[27].name == "Paralysis" && got[31].name.is_empty(),
        "localized spell names={:?}/{:?}/empty={:?}",
        got[21].name,
        got[27].name,
        got[31].name
    );
    assert_eq!(source[21].name, "舊名稱", "localized spell book mutated its source");
    assert!(localized_spell_book(None, &source).is_err());
    assert!(localized_spell_book(Some(&entities), &source[..35]).is_err());
}

/// Go：`native_battle_name_test.go:11` `TestBundledNativeBattleNameAssetsCoverImpactFixture` 的載入部分；
/// 後半的 `nativeBattleNameImage` 是 `r8` 的畫面轉換（R5）。Rust 改核對索引表收得到兩個字。
#[test]
fn test_bundled_native_battle_name_assets_cover_impact_fixture() {
    if test_support::input("FD2_ASSET_PACK", "fonts").is_none()
        || test_support::remake_asset("fonts/unicode_to_glyph.json").is_none()
    {
        return;
    }
    let (font, index) = load_native_battle_name_assets().expect("assets");
    let font = font.unwrap();
    for key in ["盜", "賊"] {
        let glyph = index.get(key).copied().unwrap_or(-1);
        assert!(glyph >= 0 && glyph < font.glyph_count() as i64, "glyph {key} = {glyph}");
    }
}

/// Go：`native_system_info_ui_test.go:16` `TestLoadNativeSystemInfoAssetsUsesSeparatedPackWithoutArchive`
#[test]
fn test_load_native_system_info_assets_uses_separated_pack_without_archive() {
    if test_support::input("FD2_ASSET_PACK", "ui/fdother_005_item_panel/resource.json").is_none() {
        return;
    }
    let assets = load_native_system_info_assets().expect("assets").unwrap();
    let want_geometry = [(102, 17), (170, 117), (170, 16), (63, 15)];
    for (index, panel) in assets.panels.iter().enumerate() {
        assert_eq!((panel.width, panel.height), want_geometry[index], "system info panel {index} geometry");
    }
}

/// Rust 新增：系統資訊畫面的開合十二幀與穩態調色盤循環（Go `TestPrepareNativeSystemInfoUIRunsTwelveOpenWaitTwelveClose`
/// 經 `prepareNativeSystemInfoUI`／`closeNativeSystemInfoUI`，兩者 `r8`；這裡直接建立同形狀的狀態）。
#[test]
fn step_native_system_info_ui_runs_opening_steady_and_closing() {
    let state = NativeSystemInfoUIState {
        opening: vec![Vec::new(); 12],
        closing: vec![Vec::new(); 12],
        phase: String::from("opening"),
        dac: vec![0; 768],
        ..Default::default()
    };
    let mut g = Game { native_system_info_ui: Some(state), ..Default::default() };
    g.step_native_system_info_ui();
    assert_eq!(g.native_system_info_ui.as_ref().unwrap().frame, 0, "undrawn frame advanced");
    for _ in 0..12 {
        g.native_system_info_ui.as_mut().unwrap().drawn = true;
        g.step_native_system_info_ui();
    }
    assert_eq!(g.native_system_info_ui.as_ref().unwrap().phase, "steady");
    let phase = g.native_fdother_palette_phase;
    g.native_system_info_ui.as_mut().unwrap().drawn = true;
    g.step_native_system_info_ui();
    assert_eq!(g.native_fdother_palette_phase, (phase + 1) & 15);
    assert_eq!(g.native_system_info_ui.as_ref().unwrap().palette.len(), 256);
    let s = g.native_system_info_ui.as_mut().unwrap();
    (s.phase, s.frame) = (String::from("closing"), 0);
    for _ in 0..12 {
        g.native_system_info_ui.as_mut().unwrap().drawn = true;
        g.step_native_system_info_ui();
    }
    assert!(g.native_system_info_ui.is_none(), "closing did not restore battlefield");
}

/// Go：`chapter_parity_ring_open_test.go:61` `TestNativeActionOverlayCursorRedrawUsesFirstActiveRawRecord`
#[test]
fn test_native_action_overlay_cursor_redraw_uses_first_active_raw_record() {
    for (name, inactive, missing, closing) in [
        ("first_active", false, false, false),
        ("skip_inactive", true, false, false),
        ("missing_raw", false, true, false),
        ("closing", false, false, true),
    ] {
        let (mut assets, _, mut state) = complete_native_map_frame_fixture();
        let view = battle::NativeMapViewState {
            cursor_x: 4,
            cursor_y: 4,
            visible_cursor_x: 4,
            visible_cursor_y: 4,
            ..Default::default()
        };
        state.materialize_native_map_view_state(view).expect("view");
        let first = &mut state.units[0];
        (first.native_map_presentation.x, first.native_map_presentation.y) = (4, 4);
        (first.x, first.y) = (9, 9); // Lookup must ignore normalized coordinates and g.sel.
        let mut second = first.clone();
        second.native_map_presentation.pose = 1;
        if inactive {
            first.native_record_byte5 |= 1;
        }
        if missing {
            first.has_native_map_presentation = false;
        }
        state.units.push(second);
        for (i, sprite) in assets.units.as_mut().unwrap().sprites.iter_mut().enumerate() {
            *sprite = native_frame_test_sprite((3 + i) as u8);
        }
        let cells = vec![
            fdother::RawCell { width: 80, height: 80, pixels: vec![240; 80 * 80] };
            NATIVE_ACTION_OVERLAY_CELL_COUNT as usize
        ];
        let mut g = Game {
            sel: Some(1),
            ring: true,
            native_map_assets: Some(assets),
            native_action_cells_raw: cells,
            action_overlay_phase: String::from(if closing { ACTION_OVERLAY_CLOSING } else { ACTION_OVERLAY_OPEN }),
            action_overlay_frame: 3,
            ..Default::default()
        };
        g.st = Some(state.clone());
        let overlay = g.native_map_action_overlay(&state);
        let mut buf = vec![0u8; 456 * 400];
        let result = overlay.render(&g, &state, &mut buf, 456, 0);
        if missing {
            assert!(result.is_err(), "{name}: missing raw provenance accepted");
            continue;
        }
        result.unwrap_or_else(|e| panic!("{name}: {e}"));
        let pos = fdicon::native_placement_offset(4, 4, 0, 0, 0, 0, 0, false).unwrap() as usize;
        if closing {
            assert_eq!(buf[pos + 8 * 456 + 8], 240, "{name}: closing acquired cursor redraw");
            continue;
        }
        let chosen = &state.units[if inactive { 1 } else { 0 }];
        let (entry, ok) = chosen.native_unit_layer_entry();
        assert!(ok, "fixture lacks raw entry");
        let cache = state.native_map_selector_cache.as_ref().unwrap();
        let sprite = g
            .native_map_assets
            .as_ref()
            .unwrap()
            .units
            .as_ref()
            .unwrap()
            .sprite_for_native_slot(cache, entry.slot, entry.pose, state.native_map_cycle_state.idle)
            .expect("sprite");
        assert_eq!(
            buf[pos + 8 * 456 + 8],
            sprite.pixels[8 * 24 + 8],
            "{name}: cursor redraw did not use first active raw record"
        );
    }
}

/// Go：`battle_protection_test.go:9` `TestBattleProtectionUsesSourceIdentityAcrossDisplayLocales`
#[test]
fn test_battle_protection_uses_source_identity_across_display_locales() {
    if test_support::remake_asset("locales/en/entities.json").is_none() {
        return;
    }
    for locale in ["zh-Hant", "en", "ja", "zh-Hans"] {
        let display = load_official_locale_entities(locale).unwrap_or_else(|e| panic!("{locale}: {e}"));
        let source = load_official_locale_entities("zh-Hant").expect("zh-Hant");
        for raw_byte in [false, true] {
            let mut u = battle::Unit::default();
            (u.hp, u.on_field, u.native_identity, u.has_native_identity) = (42, true, 0, true);
            (u.native_record_byte8, u.has_native_record_byte8) = (0, raw_byte);
            let mut st = battle::State::default();
            st.units = vec![u];
            let mut g = Game {
                st: Some(st),
                locale_entities: display.clone(),
                source_battle_entities: source.clone(),
                ..Default::default()
            };
            let alive = g.protected_battle_unit_alive("索爾");
            assert!(matches!(alive, Ok(true)), "{locale}/{raw_byte}: living source identity: {alive:?}");
            g.st.as_mut().unwrap().units[0].hp = 0;
            let alive = g.protected_battle_unit_alive("索爾");
            assert!(matches!(alive, Ok(false)), "{locale}/{raw_byte}: dead source identity: {alive:?}");
        }
    }
    let mut g = Game::default();
    assert!(matches!(g.protected_battle_unit_alive(""), Ok(true)), "空的保護名稱一律成立");
}

/// Rust 新增：`terrainAt` 越界回 -1；`hasNativeEquippedWeapon` 只接受已裝備、ID < 0x80 的物品。
#[test]
fn terrain_at_and_equipped_weapon_gates() {
    let g = Game { m: Some(MapData { w: 2, h: 1, tiles: vec![5, 6], ..Default::default() }), ..Default::default() };
    assert_eq!((g.terrain_at(1, 0), g.terrain_at(2, 0), g.terrain_at(-1, 0)), (6, -1, -1));
    assert_eq!(Game::default().terrain_at(0, 0), -1);
    let mut u = battle::Unit::default();
    (u.inventory, u.equipped) = (vec![0x80, 3].into(), vec![true, false].into());
    assert!(!has_native_equipped_weapon(Some(&u)));
    u.equipped[1] = true;
    assert!(has_native_equipped_weapon(Some(&u)));
    assert!(!has_native_equipped_weapon(None));
    assert!(native_command_action_blocked(None));
    assert!(native_command_allowed_after_move(0x22) && !native_command_allowed_after_move(9));
}

/// Go：`native_player_input_test.go:21` `TestNativePlayerCycleRawOrderSkipsEachGateAndWraps`
#[test]
fn test_native_player_cycle_raw_order_skips_each_gate_and_wraps() {
    let mut st = battle::State::default();
    (st.w, st.h) = (30, 30);
    st.units = (0..5).map(|i| native_neutral_test_unit(7 + i, 14 + i)).collect();
    st.units[1].native_record_byte5 = 1;
    st.units[2].native_record_byte5 = 4;
    st.units[3].native_record_byte5 = 0x80;
    st.units[4].native_record_byte6 = 0;
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        cursor_x: 1,
        cursor_y: 1,
        visible_cursor_x: 1,
        visible_cursor_y: 1,
        ..Default::default()
    })
    .expect("view");
    let mut g = Game { st: Some(st), native_next_player_index: 1, ..Default::default() };
    g.cycle_native_player_unit();
    assert!(
        g.load_err.is_empty()
            && g.native_player_focus == Some(battle::Cell { x: 7, y: 14 })
            && g.native_next_player_index == 1,
        "cycle failed: {} target={:?} next={}",
        g.load_err,
        g.native_player_focus,
        g.native_next_player_index
    );
    g.native_player_focus = None;
    g.st.as_mut().unwrap().units[0].native_record_byte5 = 0x80;
    g.cycle_native_player_unit();
    assert!(g.native_player_focus.is_none() && g.native_next_player_index == 1, "no candidate changed state");
    g.st.as_mut().unwrap().units[0].native_record_byte5 = 0;
    g.sel = Some(0);
    g.cycle_native_player_unit();
    assert!(g.native_player_focus.is_none(), "selection cancellation was bypassed");
}

/// Go：`native_action_overlay_test.go:271` `TestSystemOptionsOverlayUsesExactCellsAndReturnsAfterFourClosePresents`
#[test]
fn test_system_options_overlay_uses_exact_cells_and_returns_after_four_close_presents() {
    let mut g = Game {
        native_system_cursor_overlay: true,
        native_system_options_open: true,
        native_action_cells: vec![None; 76],
        ..Default::default()
    };
    for index in [54, 57, 60, 63, 66, 69, 72, 75] {
        g.native_action_cells[index] = Some(Arc::new(Picture::new_image(1, 1)));
    }
    assert!(g.native_system_options_ready(), "complete 0x1728c cell bank was rejected");
    g.begin_action_overlay_open(0);
    while g.action_overlay_blocks_input() {
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    assert!(g.toggle_native_system_option(0), "music option toggle failed");
    let state = g.native_action_overlay_state();
    assert_eq!(state.cell_index(0).ok(), Some(57), "toggled music cell");
    g.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
        g.native_system_options_open = false;
        g.native_system_cursor_overlay = false;
    })));
    for present in 0..4 {
        assert!(g.native_system_options_open, "system options closed before present {present}");
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    assert!(
        !g.ring && !g.native_system_options_open && !g.native_system_cursor_overlay,
        "system options did not return to field"
    );
}

/// Go：`native_system_info_ui_test.go:77` `nativeSystemInfoRuntimeTestGame`
fn native_system_info_runtime_test_game() -> Option<Game> {
    test_support::input("FD2_ASSET_PACK", "ui")?;
    let mut unit = battle::Unit::default();
    (unit.native_record_byte5, unit.has_native_record_byte5) = (0x80, true);
    (unit.native_record_byte6, unit.has_native_record_byte6) = (0, true);
    (unit.battle_fig, unit.has_battle_fig) = (1, true);
    (unit.native_record_race, unit.has_native_record_race) = (1, true);
    let mut st = battle::State::default();
    (st.units, st.native_round_counter) = (vec![unit], 3);
    Some(Game {
        st: Some(st),
        handler_chapter: 0,
        gold: 123,
        native_map_vga: vec![0; fdother::NATIVE_SYSTEM_INFO_BYTES],
        native_map_dac: vec![0; 256 * 3],
        native_system_info_assets: load_native_system_info_assets().expect("system info assets"),
        ..Default::default()
    })
}

/// Go：`native_system_info_ui_test.go:95` `TestPrepareNativeSystemInfoUIRunsTwelveOpenWaitTwelveClose`
#[test]
fn test_prepare_native_system_info_ui_runs_twelve_open_wait_twelve_close() {
    let Some(mut g) = native_system_info_runtime_test_game() else { return };
    let state = g.prepare_native_system_info_ui().expect("prepare");
    assert!(state.opening.len() == 12 && state.closing.len() == 12);
    assert_eq!(state.steady.len(), fdother::NATIVE_SYSTEM_INFO_BYTES);
    g.native_system_info_ui = Some(state);
    for _ in 0..12 {
        g.native_system_info_ui.as_mut().unwrap().drawn = true;
        g.step_native_system_info_ui();
    }
    assert_eq!(g.native_system_info_ui.as_ref().map(|s| s.phase.as_str()), Some("steady"));
    let phase = g.native_fdother_palette_phase;
    g.native_system_info_ui.as_mut().unwrap().drawn = true;
    g.step_native_system_info_ui();
    assert_eq!(g.native_fdother_palette_phase, (phase + 1) & 15);
    g.close_native_system_info_ui();
    for _ in 0..12 {
        g.native_system_info_ui.as_mut().unwrap().drawn = true;
        g.step_native_system_info_ui();
    }
    assert!(g.native_system_info_ui.is_none(), "closing did not restore battlefield");
}

/// Go：`native_system_info_ui_test.go:126` `TestPrepareNativeSystemInfoUIFailsBeforePublicationWithoutRawCount`
#[test]
fn test_prepare_native_system_info_ui_fails_before_publication_without_raw_count() {
    let Some(mut g) = native_system_info_runtime_test_game() else { return };
    g.st.as_mut().unwrap().units[0].has_native_record_race = false;
    assert!(g.prepare_native_system_info_ui().is_err(), "missing provenance published state");
}

/// Go：`native_system_info_ui_test.go:134` `TestNestedSystemOverlayUsesRuntimeSaveAndFD2SAVGates`
#[test]
fn test_nested_system_overlay_uses_runtime_save_and_fd2sav_gates() {
    let Some(_dir) =
        f_child_env(module_path!(), "test_nested_system_overlay_uses_runtime_save_and_fd2sav_gates", |dir| {
            vec![("FD2_NATIVE_SAVE", dir.join("missing-FD2.SAV").into_os_string())]
        })
    else {
        return;
    };
    let Some(mut g) = native_system_info_runtime_test_game() else { return };
    g.native_action_cells = vec![None; NATIVE_ACTION_OVERLAY_CELL_COUNT as usize];
    for index in [36, 41, 44, 45] {
        g.native_action_cells[index] = Some(Arc::new(Picture::new_image(1, 1)));
    }
    let state = g.native_nested_system_overlay_state().expect("nested system overlay was rejected");
    for (direction, want) in [36, 41, 44, 45].into_iter().enumerate() {
        assert_eq!(state.cell_index(direction as i64).ok(), Some(want), "direction {direction}");
    }
    // Rust 補充：巢狀選單開著時，指令環狀態（Go `nativeActionOverlayState`）就是這份；圖格缺件時四向都不可用。
    g.native_system_nested_open = true;
    assert_eq!(g.native_action_overlay_state(), state, "nested open must use the nested overlay state");
    g.native_action_cells[41] = None;
    assert_eq!(g.native_action_overlay_state().availability, [-1, -1, -1, -1], "missing cell must disable all");
}

/// Go：`native_action_overlay_test.go:592` `TestNativeCommandTargetFieldMaterializeAndReset`（Go 的施法者不在
/// `Units` 裡；Rust 的 `sel` 是索引，放進名單第 0 筆，位置相同）。
#[test]
fn test_native_command_target_field_materialize_and_reset() {
    let mut st = battle::State::default();
    (st.w, st.h) = (3, 1);
    st.native_composition_event_bytes = vec![0; 3];
    st.native_tile_blit_modes = vec![0xff; 3];
    st.units = vec![battle::Unit::default()];
    let mut g = Game { st: Some(st), sel: Some(0), ..Default::default() };
    let record = battle::NativeCommandRecord { selection_mode: 1, ..Default::default() };
    g.materialize_native_command_target_field(&record).expect("materialize");
    let st = g.st.as_ref().unwrap();
    assert_eq!(st.native_tile_blit_modes, vec![1, 0, 0xff]);
    assert!(st.has_native_map_range_mode_state && st.native_map_range_mode == 2, "target selector");
    assert!(g.reset_native_target_field());
    assert_eq!(g.st.as_ref().unwrap().native_tile_blit_modes, vec![0xff; 3]);
}

/// Go：`native_action_overlay_test.go:427` `TestNativeCommandTargetWhitelistKeepsUnresolvedIDsFailClosed`
#[test]
fn test_native_command_target_whitelist_keeps_unresolved_ids_fail_closed() {
    let g = Game::default();
    for id in (0..=35).filter(|&id| id != 9) {
        assert!(g.native_command_target_supported(id), "verified target/effect id {id} was rejected");
    }
    for id in [-1, 9, 36] {
        assert!(!g.native_command_target_supported(id), "unresolved target/effect id {id} was enabled");
    }
}

/// Go：`native_action_overlay_test.go:618` `TestPlayerNativeCommand0RunsIndexedPresentationThroughCursorConfirm`
/// （Go 另設 `FD2_ORIGINAL_*` 指到缺檔與原版檔；Rust 的指令 0 只讀分離素材包。）
#[test]
fn test_player_native_command0_runs_indexed_presentation_through_cursor_confirm() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    let Some(_dir) = f_child_env(
        module_path!(),
        "test_player_native_command0_runs_indexed_presentation_through_cursor_confirm",
        |_| vec![("FD2_MUTE", "1".into())],
    ) else {
        return;
    };
    let (Some(scene), Some(flash)) = (
        test_support::remake_asset("data/native_command_scene.json"),
        test_support::remake_asset("data/native_command_palette_flash.json"),
    ) else {
        return;
    };
    let (mut assets, field, mut state) = complete_native_map_frame_fixture();
    let mut book: Vec<battle::NativeCommandRecord> = (0..battle::NATIVE_COMMAND_RECORD_COUNT)
        .map(|id| battle::NativeCommandRecord { id, ..Default::default() })
        .collect();
    book[0] = battle::NativeCommandRecord {
        id: 0,
        damage: 50,
        hit: 100,
        selection_mode: 5,
        effect_mode: 0,
        mp_cost: 2,
        target_code: 0,
    };
    {
        let actor = &mut state.units[0];
        (actor.camp, actor.on_field, actor.hp, actor.max_hp, actor.mp) = (battle::OWN, true, 28, 28, 8);
        (actor.max_mp, actor.native_record_byte8, actor.has_native_record_byte8) = (8, 0, true);
        (actor.inventory_slots, actor.native_inventory_flags) = (vec![0; 8].into(), vec![0; 8].into());
    }
    let mut target = battle::Unit::default();
    (target.camp, target.on_field, target.x, target.y, target.hp, target.max_hp, target.class_id) =
        (battle::ENEMY, true, 1, 0, 100, 100, 5);
    (target.native_record_byte5, target.has_native_record_byte5) = (0, true);
    (target.native_record_byte6, target.has_native_record_byte6) = (1, true);
    (target.battle_fig, target.has_battle_fig) = (0, true);
    (target.native_record_byte8, target.has_native_record_byte8) = (1, true);
    (target.native_record_race, target.has_native_record_race) = (1, true);
    (target.native_record_class, target.has_native_record_class) = (1, true);
    (target.inventory_slots, target.native_inventory_flags) = (vec![0; 8].into(), vec![0; 8].into());
    let (tx, ty) = (target.x, target.y);
    state.units.push(target);
    state.native_command_book = book;
    state.native_composition_event_bytes = vec![0; (state.w * state.h) as usize];
    let w = state.w;
    state.native_tile_blit_modes[(ty * w + tx) as usize] = 0;
    state.materialize_native_map_range_mode(3);
    while assets.luts.len() <= 14 {
        assets.luts.push((0..=255).collect());
    }
    let mut g = Game {
        m: Some(field),
        st: Some(state),
        sel: Some(0),
        cur_x: tx,
        cur_y: ty,
        native_command0_targeting: true,
        native_command_target_id: 0,
        native_rng_state: 1,
        native_map_assets: Some(assets),
        native_ui_palette: load_native_ui_palette(),
        native_command_scene: Some(battle::load_native_command_scene_table(&scene).expect("scene")),
        native_command_palette_flash: Some(battle::load_native_command_palette_flash_table(&flash).expect("flash")),
        ..Default::default()
    };
    g.locale_id = String::from("zh-Hant");
    g.locale_catalog = load_official_locale("zh-Hant").expect("locale");
    g.locale_entities = load_official_locale_entities("zh-Hant").expect("entities");
    let snapshot = |g: &Game| {
        let st = g.st.as_ref().unwrap();
        (st.units[0].mp, st.units[0].acted, st.units[1].hp)
    };
    // 缺少原版職業抗性表時，正式確認保留選目標，MP、HP、行動與目標格都不動。
    g.confirm();
    assert_eq!(snapshot(&g), (8, false, 100), "missing resistance crossed transaction boundary");
    assert!(g.native_command0_targeting && g.sel == Some(0));
    assert_eq!(g.st.as_ref().unwrap().native_tile_blit_modes[(ty * w + tx) as usize], 0);
    g.st.as_mut().unwrap().native_command_resistances = BTreeMap::from([(1, 10)]);
    g.st.as_mut().unwrap().units[1].has_battle_fig = false;
    g.confirm();
    assert!(g.native_cmd0_presentation.is_none() && snapshot(&g) == (8, false, 100) && g.native_command0_targeting);
    g.st.as_mut().unwrap().units[1].has_battle_fig = true;
    g.confirm();
    assert!(g.native_cmd0_presentation.is_some(), "command0 did not start: {}", g.msg);
    assert_eq!(snapshot(&g), (8, false, 100), "command0 start crossed transaction boundary");
    g.step_native_command0_presentation();
    assert_eq!(snapshot(&g), (8, false, 100), "command0 advanced without Draw acknowledgement");
    let mut screen = Picture::new_image(640, 400);
    let (mut hp_changes, mut mp_changes) = (0, 0);
    for step in 0..512 {
        let (before_mp, _, before_hp) = snapshot(&g);
        let Some(job) = g.native_cmd0_presentation.as_mut() else { break };
        let (phase, frame) = (job.phase, job.frame as usize);
        let publish_mp = job.actor_specs.get(frame).is_some_and(|spec| spec.publish_mp);
        let impact = job.target_impact.get(frame).copied().unwrap_or(false);
        assert!(job.present(Some(&mut screen)), "command0 draw failed at step {step}");
        g.step_native_command0_presentation();
        let (mp, _, hp) = snapshot(&g);
        if mp != before_mp {
            mp_changes += 1;
            assert!(phase == NATIVE_COMMAND0_ACTOR && publish_mp, "MP changed outside actor marker");
        }
        if hp != before_hp {
            hp_changes += 1;
            assert!(phase == NATIVE_COMMAND0_TARGET && impact, "HP changed outside target marker");
        }
    }
    let (mp, acted, hp) = snapshot(&g);
    assert!(g.native_cmd0_presentation.is_none() && mp == 6 && acted && hp < 100, "result msg={}", g.msg);
    assert_eq!((mp_changes, hp_changes), (1, 7), "command0 publication counts");
    let st = g.st.as_ref().unwrap();
    assert!(!g.native_command0_targeting && g.sel.is_none() && st.native_map_range_mode == 1, "command0 cleanup");
    assert!(st.native_tile_blit_modes.iter().all(|&value| value == 0xff), "target field not reset");
    assert!(!g.msg.is_empty(), "command0 numeric result did not publish its result message");
}
