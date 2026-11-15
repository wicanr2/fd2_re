//! A 組在 `main.go` 名下函式的測試。Go 來源：`atk_animation_test.go`、`beatrunner_test.go`、
//! `persistent_native_command_test.go`、`native_map_presentation_test.go`、`native_continue_handoff_test.go`，
//! 以及 Rust 新增的單元測試（`story_walk`、`fade`、`transition_reveal`、`key_repeat` 等）。
//! 其他引用 A 組函式、但主題屬於別組或需要畫面的 Go 測試，見 `PORTING.md`「A 組測試」。

use super::*;
use crate::test_support;

/// Go：`atk_animation_test.go:127` `TestAttackPresentationTickFollowsTheBiosTick`
#[test]
fn test_attack_presentation_tick_follows_the_bios_tick() {
    let bios_tick_millis = 1000.0 / 18.2065097;
    for fpt in [1, 2, 3, 6] {
        let got = attack_presentation_tick_millis(fpt);
        let want = bios_tick_millis / fpt as f64;
        assert!((got - want).abs() <= 1e-9, "fpt={fpt} tick={got:.4} want {want:.4}");
        let unit = got * fpt as f64;
        assert!((unit - bios_tick_millis).abs() <= 1e-9, "fpt={fpt} unit={unit:.4}");
    }
    assert!(attack_presentation_tick_millis(3) > 1000.0 / 60.0);
    assert_eq!(attack_presentation_tick_millis(0), indexedmap::PHASE_BANNER_STEP_MILLIS);
}

/// Go：`atk_animation_test.go:149` `TestAttackPresentationTicksKeepsTheRemainder`
#[test]
fn test_attack_presentation_ticks_keeps_the_remainder() {
    let fpt = 3;
    let step = attack_presentation_tick_millis(fpt);
    let (mut accum, mut total) = (0.0, 0);
    let frames = 600;
    for _ in 0..frames {
        let (ticks, rest) = attack_presentation_ticks(accum + 1000.0 / 60.0, fpt);
        accum = rest;
        total += ticks;
    }
    let elapsed = frames as f64 * 1000.0 / 60.0;
    let want = (elapsed / step) as i64;
    assert!(total == want || total == want + 1, "{frames} 畫格推進了 {total} 個顯示 tick，應該是 {want} 個");
    let naive = frames * (1000.0 / 60.0 / step) as i64;
    assert_ne!(naive, total, "丟掉餘數的寫法應該少推進");
}

fn own_unit() -> battle::Unit {
    let mut u = battle::Unit::default();
    u.camp = battle::OWN;
    u
}

/// Go：`beatrunner_test.go:960` `TestSyncPartyUsesNativeIdentityWhenFigDiffers`
#[test]
fn test_sync_party_uses_native_identity_when_fig_differs() {
    let mut roster = battle::Unit::default();
    (roster.fig, roster.native_identity, roster.has_native_identity, roster.max_hp, roster.max_mp) =
        (4, 0x2a, true, 30, 7);
    let mut current = own_unit();
    (current.fig, current.native_identity, current.has_native_identity) = (99, 0x2a, true);
    (current.hp, current.max_hp, current.mp, current.max_mp, current.on_field) = (10, 30, 2, 7, true);
    let mut st = battle::State::default();
    st.units = vec![current];
    let mut g = Game {
        party_members: BTreeMap::from([(4, true)]),
        party_roster: BTreeMap::from([(4, roster)]),
        st: Some(st),
        ..Default::default()
    };
    g.sync_party_from_battle().unwrap();
    let got = &g.party_roster[&4];
    assert!(got.hp == 30 && got.mp == 7, "raw identity did not update persistent member: {got:?}");
    assert!(!g.party_roster.contains_key(&99), "Fig selector was incorrectly used as persistent key");
}

/// Go：`beatrunner_test.go:977` `TestSyncPartyClearsRawTransientStateWithoutChangingBattle`
#[test]
fn test_sync_party_clears_raw_transient_state_without_changing_battle() {
    for status in [0x80u8, 0x83] {
        let mut source = own_unit();
        (source.fig, source.hp, source.max_hp, source.mp, source.max_mp) = (4, 3, 30, 2, 7);
        (source.has_native_record_byte5, source.native_record_byte5, source.acted) = (true, status, true);
        source.native_transient = [1, 2, 3, 4, 5, 6];
        let mut st = battle::State::default();
        st.units = vec![source.clone()];
        let mut g = Game { party_members: BTreeMap::from([(4, true)]), st: Some(st), ..Default::default() };
        g.sync_party_from_battle().unwrap();
        let got = &g.party_roster[&4];
        let want_hp = if status & 1 != 0 { 3 } else { 30 };
        assert!(
            got.native_record_byte5 == status & 1
                && got.native_transient == [0; 6]
                && !got.acted
                && got.hp == want_hp
                && got.mp == 7,
            "sync status={status:#x}: {got:?}"
        );
        let after = &g.st.as_ref().unwrap().units[0];
        assert_eq!(after, &source, "sync changed battle source");
    }
}

/// Go：`beatrunner_test.go:1001` `TestSyncPartySkipsUnknownNativeIdentity`
#[test]
fn test_sync_party_skips_unknown_native_identity() {
    let mut roster = battle::Unit::default();
    (roster.fig, roster.native_identity, roster.has_native_identity, roster.hp, roster.max_hp) =
        (4, 0x2a, true, 21, 30);
    let mut current = own_unit();
    (current.fig, current.native_identity, current.has_native_identity, current.hp, current.max_hp) =
        (4, 0x2b, true, 1, 30);
    current.on_field = true;
    let mut st = battle::State::default();
    st.units = vec![current];
    let mut g = Game {
        party_members: BTreeMap::from([(4, true)]),
        party_roster: BTreeMap::from([(4, roster)]),
        st: Some(st),
        ..Default::default()
    };
    g.sync_party_from_battle().unwrap();
    assert_eq!(g.party_roster[&4].hp, 21, "unknown raw identity should fail closed");
}

/// Go：`persistent_native_command_test.go:59` `TestSyncPartyUsesRawByte5ForHPRefill`
#[test]
fn test_sync_party_uses_raw_byte5_for_hp_refill() {
    let mut current = own_unit();
    (current.fig, current.on_field, current.hp, current.max_hp, current.mp, current.max_mp) = (0, false, 0, 50, 2, 8);
    (current.has_native_record_byte5, current.native_record_byte5) = (true, 0);
    let mut roster = own_unit();
    (roster.max_hp, roster.max_mp) = (50, 8);
    let mut st = battle::State::default();
    st.units = vec![current];
    let mut g = Game {
        st: Some(st),
        party_members: BTreeMap::from([(0, true)]),
        party_roster: BTreeMap::from([(0, roster)]),
        ..Default::default()
    };
    g.sync_party_from_battle().unwrap();
    assert_eq!(g.party_roster[&0].hp, 50, "raw byte5=0 did not refill HP");
    let unit = &mut g.st.as_mut().unwrap().units[0];
    (unit.native_record_byte5, unit.hp) = (1, 0);
    g.sync_party_from_battle().unwrap();
    assert_eq!(g.party_roster[&0].hp, 0, "raw byte5=1 incorrectly refilled HP");
}

/// Go：`persistent_native_command_test.go:9` `TestApplyPersistentStatsPreservesDynamicNativeCommandMask`
#[test]
fn test_apply_persistent_stats_preserves_dynamic_native_command_mask() {
    let mut dst = battle::Unit::default();
    dst.native_command_mask = [1, 2, 3, 4, 5];
    (dst.map_selector_slot, dst.has_map_selector_slot) = (7, true);
    let mut src = battle::Unit::default();
    src.native_command_mask = [0x81, 0x01, 0, 0x80, 0x11];
    (src.native_identity, src.has_native_identity) = (9, true);
    (src.native_record_race, src.has_native_record_race) = (1, true);
    (src.native_record_class, src.has_native_record_class) = (5, true);
    src.native_transient = [1, 2, 3, 4, 5, 6];
    (src.native_record_byte5, src.has_native_record_byte5) = (1, true);
    (src.native_record_byte6, src.has_native_record_byte6) = (7, true);
    (src.native_record_byte34, src.has_native_record_byte34) = (0x81, true);
    (src.native_record_byte35, src.has_native_record_byte35) = (0x22, true);
    (src.native_record_byte36, src.has_native_record_byte36) = (0x33, true);
    (src.native_record_word42, src.has_native_record_word42) = (0x140, true);
    (src.native_record_word46, src.has_native_record_word46) = (0x24, true);
    src.native_inventory_flags = vec![0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80].into();
    apply_persistent_stats(Some(&mut dst), Some(&src));
    assert_eq!(dst.native_command_mask, src.native_command_mask);
    assert!(dst.native_record_byte5 == 1 && dst.has_native_record_byte5);
    assert!(dst.native_record_byte6 == 7 && dst.has_native_record_byte6);
    assert!(dst.native_record_byte34 == 0x81 && dst.native_record_byte35 == 0x22 && dst.native_record_byte36 == 0x33);
    assert!(dst.has_native_record_byte34 && dst.has_native_record_byte35 && dst.has_native_record_byte36);
    assert!(dst.native_record_word42 == 0x140 && dst.has_native_record_word42);
    assert!(dst.native_record_word46 == 0x24 && dst.has_native_record_word46);
    assert!(dst.map_selector_slot == 7 && dst.has_map_selector_slot, "battle-local selector slot was overwritten");
    assert!(dst.native_identity == 9 && dst.has_native_identity);
    assert!(dst.native_record_race == 1 && dst.has_native_record_race);
    assert!(dst.native_record_class == 5 && dst.has_native_record_class);
    assert_eq!(dst.native_transient, src.native_transient);
    assert!(dst.native_inventory_flags.len() == 8 && dst.native_inventory_flags[0] == 0x40);
    apply_persistent_stats(None, Some(&src));
}

fn party_member(fig: i64) -> battle::PartyMember {
    battle::PartyMember { fig, ..Default::default() }
}

/// Go：`beatrunner_test.go:1490` `TestReorderScenarioPartyUsesOriginalJoinSlots`
#[test]
fn test_reorder_scenario_party_uses_original_join_slots() {
    let mut sc = battle::Scenario::default();
    sc.party = vec![party_member(0), party_member(4), party_member(9), party_member(30)];
    sc.deploy_cells = vec![[7, 20], [10, 21], [8, 22], [11, 23]];
    reorder_scenario_party(Some(&mut sc), &[0, 9, 4, 30]).unwrap();
    for (slot, (id, x, y)) in [(0, 7, 20), (9, 10, 21), (4, 8, 22), (30, 11, 23)].into_iter().enumerate() {
        assert!(sc.party[slot].fig == id && sc.deploy_cells[slot] == [x, y], "slot {slot}");
    }
    // 名單沒有涵蓋全部隊員時拒收。
    let err = reorder_scenario_party(Some(&mut sc), &[0, 9]).unwrap_err();
    assert_eq!(err.message(), "JOIN order covers 2 of 4 scenario party members");
    assert!(reorder_scenario_party(None, &[1]).is_ok());
}

/// Go：`beatrunner_test.go:2262` `TestFilterScenarioPartyUsesJoinMembership`
#[test]
fn test_filter_scenario_party_uses_join_membership() {
    let mut sc = battle::Scenario::default();
    sc.party = vec![party_member(0), party_member(9), party_member(30), party_member(75)];
    sc.deploy_cells = vec![[1, 10], [2, 20], [3, 30], [4, 40]];
    filter_scenario_party(Some(&mut sc), &BTreeMap::from([(0, true), (9, true)]));
    assert!(sc.party.len() == 2 && sc.party[0].fig == 0 && sc.party[1].fig == 9);
    assert!(sc.deploy_cells.len() == 4 && sc.deploy_cells[0] == [1, 10] && sc.deploy_cells[1] == [2, 20]);
    let mut direct = battle::Scenario::default();
    direct.party = vec![party_member(0), party_member(9)];
    filter_scenario_party(Some(&mut direct), &BTreeMap::new());
    assert_eq!(direct.party.len(), 2, "direct scenario start must preserve authored party");
}

/// Go：`native_map_presentation_test.go:267` `TestGameMaterializesEditableNativeMapRuntime`
#[test]
fn test_game_materializes_editable_native_map_runtime() {
    let mut st = battle::State::default();
    (st.w, st.h) = (24, 24);
    let mut g = Game {
        m: Some(MapData { w: 24, h: 24, tile_w: 24, tile_h: 24, ..Default::default() }),
        st: Some(st),
        ..Default::default()
    };
    let n = campaign::Node {
        native_map_view: Some(campaign::NativeMapViewConfig {
            camera_x: 1,
            camera_y: 13,
            cursor_x: 8,
            cursor_y: 17,
            visible_cursor_x: 7,
            visible_cursor_y: 4,
            range_mode: Some(1),
        }),
        native_map_hud: Some(campaign::NativeMapHUDConfig { display_gate_a: 1, display_gate_b: 1, anchor_x: 1 }),
        ..Default::default()
    };
    assert!(g.materialize_native_map_runtime(Some(&n)), "{}", g.load_err);
    let st = g.st.as_ref().unwrap();
    assert!(st.has_native_map_view_state && st.has_native_map_hud_state && st.has_native_map_range_mode_state);
    assert_eq!(st.native_map_range_mode, 1);
    assert!(g.cur_x == 8 && g.cur_y == 17 && g.cam_x == 24.0 && g.cam_y == 13.0 * 24.0);
}

/// 沒有視圖設定的節點不動狀態；只有 HUD 沒有視圖時失敗即關閉（`materializeNativeMapRuntime`
/// 的前兩個閘門，不經過 `syncNativeMapView`）。
#[test]
fn materialize_native_map_runtime_gates_before_publishing() {
    let mut g = Game::default();
    assert!(g.materialize_native_map_runtime(None));
    assert!(g.materialize_native_map_runtime(Some(&campaign::Node::default())));
    let hud_only = campaign::Node {
        native_map_hud: Some(campaign::NativeMapHUDConfig { display_gate_a: 1, display_gate_b: 1, anchor_x: 1 }),
        ..Default::default()
    };
    assert!(!g.materialize_native_map_runtime(Some(&hud_only)));
    assert_eq!(g.load_err, "native map runtime: battle state is unavailable");
    g.st = Some(battle::State::default());
    assert!(!g.materialize_native_map_runtime(Some(&hud_only)));
    assert_eq!(g.load_err, "native map runtime: HUD lacks a sourced view");
}

/// Go：`native_continue_handoff_test.go:1117` `TestNativeNestedSystemExitPublishesEbitenTermination`
#[test]
fn test_native_nested_system_exit_publishes_ebiten_termination() {
    let mut g = Game { native_system_exit_requested: true, ..Default::default() };
    let err = g.update().unwrap_err();
    assert!(err.is(&ERR_TERMINATION), "Update error={err}");
}

/// `Update` 的 `defer` 在提早返回時照樣執行：F10 請求海報切換後，玩家面板請求旗標被清除。
#[test]
fn update_runs_deferred_cleanup_on_early_return() {
    let mut g = Game { player_theme_requested: true, ..Default::default() };
    g.tick_input.pressed.insert(Key::F10);
    g.update().unwrap();
    assert!(g.player_poster_cycle_requested && !g.player_theme_requested);
    assert_eq!(g.frame, 1);
}

/// Go `stepTransitionReveal`（`native_24b4d_test.go:25` 的步進部分）：只有呈現回報 `drawn` 後才換幀，
/// 共 30 次交替，最後一次呼叫延續。
#[test]
fn step_transition_reveal_alternates_frames_after_presentation() {
    let continued = Arc::new(std::sync::atomic::AtomicI64::new(0));
    let c = continued.clone();
    let mut g = Game {
        transition_reveal: Some(TransitionRevealJob {
            frames: [vec![1, 1], vec![2, 2]],
            remaining: 30,
            delay: 1,
            then: Some(Box::new(move |_: &mut Game| {
                c.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
            })),
            ..Default::default()
        }),
        ..Default::default()
    };
    g.step_transition_reveal();
    assert_eq!(g.transition_reveal.as_ref().unwrap().remaining, 30, "未呈現前不得前進");
    for step in 0..30 {
        let job = g.transition_reveal.as_mut().unwrap();
        (job.drawn, job.ticks) = (true, 0);
        g.step_transition_reveal();
        if let Some(job) = g.transition_reveal.as_ref() {
            let want = (step + 1) & 1;
            assert_eq!(job.index, want, "step{step}");
            assert_eq!(g.native_map_vga, job.frames[want as usize], "step{step}");
            assert!(!job.drawn);
        }
    }
    assert!(g.transition_reveal.is_none());
    assert_eq!(continued.load(std::sync::atomic::Ordering::Relaxed), 1);
}

#[test]
fn start_fade_transition_runs_action_between_out_and_in() {
    let mut g = Game::default();
    g.start_fade_transition(Some(Box::new(|g: &mut Game| g.msg = String::from("acted"))));
    for _ in 0..STORY_FADE_FRAMES - 1 {
        g.step_fade();
    }
    assert!(g.fade.as_ref().unwrap().out && g.msg.is_empty());
    g.step_fade();
    assert_eq!(g.msg, "acted");
    let fade = g.fade.as_ref().unwrap();
    assert!(!fade.out && fade.t == 0 && fade.total == STORY_FADE_FRAMES && fade.then.is_none());
    for _ in 0..STORY_FADE_FRAMES {
        g.step_fade();
    }
    assert!(g.fade.is_none());
}

#[test]
fn step_dlg_anim_collapses_before_switching_speaker() {
    let line = |speaker: i64| battle::DialogLine { speaker, ..Default::default() };
    let mut g = Game { dlg_shown: DLG_NONE, dialog: vec![line(3)], ..Default::default() };
    g.step_dlg_anim();
    assert_eq!((g.dlg_shown, g.dlg_phase, g.dlg_t), (3, 2, 0), "首句直接展開");
    for _ in 0..DLG_ANIM_FRAMES {
        g.step_dlg_anim();
    }
    assert_eq!(g.dlg_phase, 0);
    g.dialog.push(line(40));
    g.step_dlg_anim();
    assert_eq!((g.dlg_shown, g.dlg_phase), (3, 1), "換人先收合");
    for _ in 0..DLG_ANIM_FRAMES {
        g.step_dlg_anim();
    }
    assert_eq!((g.dlg_shown, g.dlg_phase), (40, 2));
    g.dialog.clear();
    g.step_dlg_anim();
    assert_eq!((g.dlg_shown, g.dlg_phase, g.dlg_t), (DLG_NONE, 0, 0));
}

#[test]
fn step_cam_pan_tile_step_moves_x_first_then_continues() {
    let mut g =
        Game { m: Some(MapData { w: 30, h: 30, tile_w: 24, tile_h: 24, ..Default::default() }), ..Default::default() };
    let done = Arc::new(std::sync::atomic::AtomicI64::new(0));
    let d = done.clone();
    g.cam_pan = Some(CamPanJob {
        to_x: 48.0,
        to_y: 24.0,
        tile_step: true,
        then: Some(Box::new(move |_: &mut Game| {
            d.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        })),
        ..Default::default()
    });
    let mut path = Vec::new();
    while g.cam_pan.is_some() {
        g.step_cam_pan();
        path.push((g.cam_x, g.cam_y));
    }
    assert_eq!(path, [(24.0, 0.0), (48.0, 0.0), (48.0, 24.0)]);
    assert_eq!(done.load(std::sync::atomic::Ordering::Relaxed), 1);
}

#[test]
fn key_repeat_fires_on_press_then_every_five_after_twelve() {
    let mut input = TickInput::default();
    let mut fired = Vec::new();
    for d in 0..=27 {
        input.press_duration.insert(Key::ArrowUp, d);
        if key_repeat(&input, Key::ArrowUp) {
            fired.push(d);
        }
    }
    assert_eq!(fired, [1, 12, 17, 22, 27]);
    assert!(!key_repeat(&input, Key::ArrowDown));
}

#[test]
fn small_main_helpers_follow_go() {
    assert!(equal_int_order(&[1, 2], &[1, 2]) && !equal_int_order(&[1, 2], &[2, 1]) && !equal_int_order(&[1], &[1, 2]));
    assert_eq!(fig_name(4), "亞雷斯");
    assert_eq!(fig_name(69), "守衛");
    assert_eq!(fig_name(7), "fig7");
    assert_eq!((sgn_int(-5), sgn_int(0), sgn_int(9)), (-1, 0, 1));
    let mut v = 5.0;
    clamp(&mut v, 0.0, -1.0);
    assert_eq!(v, 0.0, "hi < lo 時夾到 lo");
    assert_eq!(scan_numbered_frame("fig_012_f03.png", "fig_", "_f"), Some((12, 3)));
    assert_eq!(scan_numbered_frame("DATO_7_m-1.png", "DATO_", "_m"), Some((7, -1)));
    assert_eq!(scan_numbered_frame("fig_x_f1.png", "fig_", "_f"), None);
    assert_eq!(scan_numbered_frame("fig_1_f1.gif", "fig_", "_f"), None);
    assert_eq!(Game::default().layout(1, 1), (640, 400));
}

/// `loadStoryScriptWithIdentityAt` 對受版控的第一章劇本：每句都接上 canonical line_id（Go
/// `story_locale_test.go:939` `TestChapterOneStoryUsesAllOfficialContentCatalogs` 的第一段）。
#[test]
fn load_story_script_joins_canonical_line_ids() {
    if test_support::remake_asset("story/ch01.json").is_none() {
        return;
    }
    let lines = load_story_script_with_identity_at("assets/story/ch01.json", "", Some(&0)).unwrap();
    assert!(!lines.is_empty());
    assert_eq!(lines[0].line_id, "legacy/line/7ecb566a60db/scenes/0/lines/0");
    let all = load_story_script_with_identity_at("assets/story/ch01.json", "", None).unwrap();
    assert!(all.len() >= lines.len() && all.iter().all(|l| !l.line_id.is_empty()));
    let err = load_story_script_with_identity_at("assets/story/ch01.json", "", Some(&9999)).unwrap_err();
    assert_eq!(err.message(), "story \"assets/story/ch01.json\" scene index 9999 out of range");
    let err = load_story_script_with_identity_at("assets/story/ch01.json", "no-such-scene", None).unwrap_err();
    assert_eq!(err.message(), "story \"assets/story/ch01.json\" scene \"no-such-scene\" not found");
}

#[test]
fn load_figani_tables_from_remake_assets() {
    if test_support::remake_asset("figani/delays.json").is_none() {
        return;
    }
    let delays = load_figani_delays().unwrap();
    assert!(!delays.is_empty() && delays.values().all(|d| !d.is_empty() && d.iter().all(|&v| v >= 0)));
}

#[test]
fn load_native_command_labels_keeps_only_valid_entries() {
    // 檔案不存在時回空 map（Go 回 nil）；存在時每筆 id 在 0..40 且標籤非空。
    let labels = load_native_command_labels();
    assert!(labels.iter().all(|(id, label)| (0..40).contains(id) && !label.is_empty()));
}
