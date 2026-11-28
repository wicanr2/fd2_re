//! Go 來源：`remake/cmd/fd2/native_field_event61_test.go`（期望值不改）。
//!
//! Go 以 `t.Setenv` 設的環境變數由父測試以子行程傳入（`f_child_env`）。Go 另設 `FD2_ORIGINAL_FDOTHER` 供其他
//! 地圖家族讀封存檔；Rust 的地圖載入只讀分離素材包，不設。Go 的 `assetPackRootWithLocales` 把受版控的語系疊到
//! 素材包；Rust 的語系一律從 `asset_path("assets/locales")` 讀，不另組。
//!
//! `TestNativeEvent61ProductionOwnersRejectMissingSeparatedBank` 在 Go 載入後才把 `FD2_ASSET_PACK` 換成空目錄；
//! Rust 工作區禁用 `unsafe`，不能在行程中改環境變數，改由父行程組一個少了 `animations/fdother_045_event61`
//! 的素材包（其餘項目以符號連結指回原包），子行程從頭到尾用它。

use super::*;
use crate::test_support;

const EVENT61_BANK: &str = "animations/fdother_045_event61";

/// 子行程的環境；素材包不在時回 `false`，父行程略過。`pack` 指定子行程的 `FD2_ASSET_PACK`。
fn event61_child(name: &str, pack: Option<std::path::PathBuf>) -> bool {
    // 子行程的素材包可能刻意少了事件 61 圖格表，這裡只檢查素材包本身在不在。
    if test_support::input("FD2_ASSET_PACK", "manifest.json").is_none() {
        return false;
    }
    f_child_env(module_path!(), name, |dir| {
        let mut envs = vec![
            ("FD2_MUTE", "1".into()),
            ("FD2_TITLE", "0".into()),
            ("FD2_CAMPAIGN", "assets/scenarios/campaign_full.json".into()),
            ("FD2_CAMP_NODE", "battle_ch26".into()),
            ("FD2_NATIVE_SAVE", dir.join("FD2.SAV").into_os_string()),
            ("XDG_DATA_HOME", dir.as_os_str().to_os_string()),
        ];
        if let Some(pack) = pack.as_ref() {
            envs.push(("FD2_ASSET_PACK", pack.clone().into_os_string()));
        }
        envs
    })
    .is_some()
}

/// Go：`native_field_event61_test.go:12` `nativeEvent61PlayerGame`
fn native_event61_player_game(items: &[i64]) -> (Game, UnitIdx) {
    let mut g = load_game().expect("loadGame");
    assert!(g.load_err.is_empty(), "{}", g.load_err);
    assert!(g.st.as_ref().is_some_and(|st| !st.units.is_empty()), "chapter26 battle state unavailable");
    // Go 的原生地圖時鐘讀 time.Now；Rust 讀虛擬時鐘，沒有跑過 Update 時是零值，給一個非零時刻。
    g.virtual_now = VIRTUAL_CLOCK_ORIGIN;
    {
        let st = g.st.as_ref().unwrap();
        let (view, hud) = (&st.native_map_view_state, &st.native_map_hud_state);
        assert!(
            st.has_native_map_view_state
                && st.has_native_map_hud_state
                && view.camera_x == 9
                && view.camera_y == 39
                && view.cursor_x == 15
                && view.cursor_y == 46
                && view.visible_cursor_x == 6
                && view.visible_cursor_y == 7
                && hud.display_gate_a == 1
                && hud.display_gate_b == 1
                && hud.anchor_x == 1,
            "chapter26 pre-handler runtime view={view:?} hud={hud:?}"
        );
    }
    {
        let trigger = &mut g.st.as_mut().unwrap().units[0];
        trigger.acted = false;
        trigger.native_record_word42 = trigger.max_hp as u16;
        trigger.has_native_record_word42 = true;
        *trigger.inventory = items.to_vec();
        *trigger.equipped = vec![false; items.len()];
        *trigger.inventory_slots = vec![0xff; 8];
        *trigger.native_inventory_flags = vec![0x80; 8];
        for (i, &item) in items.iter().enumerate() {
            trigger.inventory_slots[i] = item;
            trigger.native_inventory_flags[i] = 0;
        }
    }
    assert!(g.position_screenshot_cursor(1, 46), "chapter26 native cursor could not reach event61");
    let trigger = &mut g.st.as_mut().unwrap().units[0];
    (trigger.x, trigger.y) = (1, 46);
    (g.sel, g.cur_x, g.cur_y) = (Some(0), 1, 46);
    (g.sel_orig_x, g.sel_orig_y) = (1, 46);
    g.moved = true;
    (g, 0)
}

fn trigger_inventory(g: &Game, trigger: UnitIdx) -> Vec<i64> {
    g.st.as_ref().unwrap().units[trigger].inventory.to_vec()
}

fn acted(g: &Game, trigger: UnitIdx) -> bool {
    g.st.as_ref().unwrap().units[trigger].acted
}

fn state12(g: &Game) -> u8 {
    g.st.as_ref().unwrap().native_event_state[12]
}

/// 59 幀的演出：每一幀標成畫過，第一次先給 tick 100，之後每次前進 2 tick。
fn present_event61(g: &mut Game, check_frames: bool) {
    for frame in 0..59 {
        let job = g.native_field_event61.as_mut().unwrap_or_else(|| panic!("frame {frame}: job disappeared"));
        if check_frames {
            assert_eq!(job.frame, frame, "frame {frame}");
        }
        job.drawn = true;
        if !job.has_tick {
            g.step_native_field_event61_tick(100);
        }
        g.step_native_field_event61_tick(102 + frame * 2);
    }
}

/// Go：`native_field_event61_test.go:84` `TestNativeEvent61MissingItemRunsOnlyEditableFDTXT2`
#[test]
fn test_native_event61_missing_item_runs_only_editable_fdtxt2() {
    if !event61_child("test_native_event61_missing_item_runs_only_editable_fdtxt2", None) {
        return;
    }
    let (mut g, trigger) = native_event61_player_game(&[0x20]);
    g.finish_selected_wait();
    assert!(
        g.battle_event.is_some() && g.dialog.len() == 1 && g.dialog[0].text == "那是什麼奇怪的東西?頭部還開著?",
        "missing item dialogue={:?} err={:?}",
        g.dialog.iter().map(|d| &d.text).collect::<Vec<_>>(),
        g.load_err
    );
    assert!(
        g.native_field_event61.is_none() && state12(&g) == 0 && trigger_inventory(&g, trigger) == [0x20],
        "missing item path mutated or started presentation"
    );
    g.dialog.clear();
    g.advance_battle_event();
    assert!(
        g.battle_event.is_none() && g.sel.is_none() && acted(&g, trigger),
        "missing item path did not finish the successful wait action"
    );
}

/// Go：`native_field_event61_test.go:102` `TestNativeEvent61AttackWaitsForPresentationCompletion`
#[test]
fn test_native_event61_attack_waits_for_presentation_completion() {
    if !event61_child("test_native_event61_attack_waits_for_presentation_completion", None) {
        return;
    }
    let (mut g, trigger) = native_event61_player_game(&[0x20]);
    {
        // map26 的原生戰鬥 selector 113 的 FIGANI PNG 尚未匯出；改用已配對的玩家資源，隔離 selector1 延後。
        let unit = &mut g.st.as_mut().unwrap().units[trigger];
        unit.battle_fig = 4;
        // 物理 target 也走 0x115B6；fixture 必須先裝備武器並建立正式射程 field。
        unit.equipped[0] = true;
        unit.native_inventory_flags[0] = 0x40;
    }
    let mut target = battle::Unit::default();
    target.name = String::from("測試敵兵");
    (target.battle_fig, target.camp, target.x, target.y) = (96, battle::ENEMY, 2, 46);
    (target.hp, target.max_hp, target.on_field, target.has_battle_fig, target.lv) = (20, 28, true, true, 2);
    target.has_native_record_byte5 = true;
    (target.native_record_byte6, target.has_native_record_byte6) = (0, true);
    // map0/map0_units.json 的 selector96 原始欄位，與下列建構列相同來源。
    (target.map_selector_key, target.has_map_selector_key) = (96, true);
    (target.native_record_word42, target.has_native_record_word42) = (28, true);
    (target.native_record_race, target.has_native_record_race) = (1, true);
    (target.native_record_class, target.has_native_record_class) = (7, true);
    (target.native_record_byte8, target.has_native_record_byte8) = (96, true);
    // 原生 EXP 預檢需要實際 selector96 的 high_class 建構列。
    target.native_constructor = Some(battle::NativeConstructorTable {
        branch: String::from("high_class"),
        index: 28,
        record: vec![1, 7, 14, 0, 0, 7, 1, 1, 4, 21],
        ..Default::default()
    });
    let (tx, ty) = (target.x, target.y);
    // 手動建構的敵兵也走既有 selector slot 分配，才能驗攻擊返回的完整地圖。
    g.st.as_mut().unwrap().append_native_map_selector_batch(vec![target]).expect("AppendNativeMapSelectorBatch");
    g.begin_player_attack_targeting();
    (g.cur_x, g.cur_y) = (tx, ty);
    g.confirm();
    assert!(
        g.atk.as_ref().is_some_and(|atk| atk.after.is_some()),
        "successful attack did not retain the deferred selector1 owner: {}",
        g.load_err
    );
    assert!(
        g.atk.as_ref().unwrap().native_scene.is_some(),
        "normal player attack did not publish its native scene snapshot"
    );
    assert!(g.battle_event.is_none(), "attack committed selector1 before the full-screen presentation");
    g.finish_attack_presentation();
    assert!(
        g.atk.is_none()
            && g.battle_event.is_some()
            && g.dialog.len() == 1
            && g.dialog[0].text == "那是什麼奇怪的東西?頭部還開著?",
        "post-presentation dialog={:?} acted={} err={:?} job={} state12={}",
        g.dialog.iter().map(|d| &d.text).collect::<Vec<_>>(),
        acted(&g, trigger),
        g.load_err,
        g.native_field_event61.is_some(),
        state12(&g)
    );
    g.dialog.clear();
    g.advance_battle_event();
    assert!(g.battle_event.is_none() && acted(&g, trigger), "attack action did not finish after selector1 dialogue");
}

/// Go：`native_field_event61_test.go:159` `TestNativeEvent61ImmediateItemRunsAfterSuccessfulMutation`
#[test]
fn test_native_event61_immediate_item_runs_after_successful_mutation() {
    if !event61_child("test_native_event61_immediate_item_runs_after_successful_mutation", None) {
        return;
    }
    let (mut g, trigger) = native_event61_player_game(&[198]);
    let st = g.st.as_mut().unwrap();
    st.native_composition_event_bytes = vec![0; (st.w * st.h) as usize];
    g.native_item_effect_rows =
        battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
            .expect("LoadNativeItemEffectRowPrefix");
    let applied = g.apply_native_immediate_item(0, 198);
    assert!(
        matches!(applied, Ok(true)),
        "immediate item applied={:?}",
        applied.as_ref().map_err(|err| err.message().to_string())
    );
    // 0x13512 在 handler 成功返回就設 +5 bit7，0x1198A 的格子事件對話在它之後。
    assert!(
        trigger_inventory(&g, trigger).is_empty()
            && acted(&g, trigger)
            && g.sel.is_none()
            && g.battle_event.is_some()
            && g.dialog.len() == 1,
        "post-item inventory={:?} acted={} dialog={}",
        trigger_inventory(&g, trigger),
        acted(&g, trigger),
        g.dialog.len()
    );
    g.dialog.clear();
    g.advance_battle_event();
    assert!(
        g.battle_event.is_none() && acted(&g, trigger),
        "immediate item action did not finish after selector1 dialogue"
    );
}

/// Go：`native_field_event61_test.go:188` `TestNativeEvent61MaterializedRuntimePresentsCommitsAndPersistsWold`
#[test]
fn test_native_event61_materialized_runtime_presents_commits_and_persists_wold() {
    if !event61_child("test_native_event61_materialized_runtime_presents_commits_and_persists_wold", None) {
        return;
    }
    let (mut g, trigger) = native_event61_player_game(&[0xD0, 0x20]);
    {
        let st = g.st.as_ref().unwrap();
        let (view, hud) = (&st.native_map_view_state, &st.native_map_hud_state);
        assert!(
            st.has_native_map_view_state
                && st.has_native_map_hud_state
                && view.camera_x == 0
                && view.camera_y == 39
                && view.cursor_x == 1
                && view.cursor_y == 46
                && view.visible_cursor_x == 1
                && view.visible_cursor_y == 7
                && hud.anchor_x == 0xf2,
            "event61 campaign runtime view={view:?} hud={hud:?}"
        );
    }
    g.finish_selected_wait();
    assert!(
        g.battle_event.is_some()
            && g.dialog.len() == 1
            && g.dialog[0].text == "這機兵的頭部怎麼開著?這個金屬盒子..好像滿適合的,應該是這樣放進去.....咦!",
        "success dialogue3={:?} err={:?}",
        g.dialog.iter().map(|d| &d.text).collect::<Vec<_>>(),
        g.load_err
    );
    g.dialog.clear();
    g.advance_battle_event();
    let job = g.native_field_event61.as_ref();
    assert!(
        job.is_some_and(|job| job.frames.len() == 59 && job.frame == 0),
        "event61 job frames={:?} loadErr={:?}",
        job.map(|job| job.frames.len()),
        g.load_err
    );
    present_event61(&mut g, true);
    assert!(
        g.native_field_event61.is_none()
            && state12(&g) == 1
            && trigger_inventory(&g, trigger) == [0x20]
            && g.party_members.get(&31).copied().unwrap_or(false)
            && g.party_join_order.last() == Some(&31),
        "commit state={} inventory={:?} order={:?} err={:?}",
        state12(&g),
        trigger_inventory(&g, trigger),
        g.party_join_order,
        g.load_err
    );
    assert!(g.party_roster.contains_key(&31), "JOIN31 did not persist the materialized Wold record");
    assert!(g.battle_event.is_some() && g.dialog.len() == 1, "FDTXT4 did not begin: dialog={}", g.dialog.len());
    while g.battle_event.is_some() {
        g.dialog.clear();
        g.advance_battle_event();
    }
    assert!(g.sel.is_none() && acted(&g, trigger), "final FDTXT4 did not return to the completed wait action");
}

/// Go：`native_field_event61_test.go:246` `TestNativeSystemGroupMarchPausesForEvent61PresentationAndResumes`
#[test]
fn test_native_system_group_march_pauses_for_event61_presentation_and_resumes() {
    if !event61_child("test_native_system_group_march_pauses_for_event61_presentation_and_resumes", None) {
        return;
    }
    let (mut g, trigger) = native_event61_player_game(&[0xD0, 0x20]);
    {
        let unit = &mut g.st.as_mut().unwrap().units[trigger];
        (unit.x, unit.y) = (2, 46);
        (unit.native_map_presentation.x, unit.native_map_presentation.y) = (2, 46);
    }
    let step = battle::NativeSystemGroupMarchStep {
        unit_index: 0,
        path: vec![battle::Cell { x: 2, y: 46 }, battle::Cell { x: 1, y: 46 }],
        events: vec![battle::NativeSystemGroupMarchEvent {
            path_index: 1,
            event_id: 61,
            text_index: 3,
            presentation: true,
        }],
    };
    let plan = battle::NativeSystemGroupMarchPlan { destination: battle::Cell { x: 1, y: 46 }, steps: vec![step] };
    assert!(
        g.preflight_native_system_group_march_events(&plan),
        "event61 presentation assets failed group-march preflight"
    );
    g.native_system_group_march = Some(plan);
    g.start_next_native_system_group_march_step();
    for _ in 0..7 {
        g.step_battle_walk();
    }
    assert!(
        g.walk.as_ref().is_some_and(|w| w.native_group_march_paused)
            && g.battle_event.is_some()
            && g.native_field_event61.is_none()
            && state12(&g) == 0,
        "event61 did not pause at text3: walk={} event={} job={} state={} err={:?}",
        g.walk.is_some(),
        g.battle_event.is_some(),
        g.native_field_event61.is_some(),
        state12(&g),
        g.load_err
    );
    g.dialog.clear();
    g.advance_battle_event();
    assert!(
        g.native_field_event61.as_ref().is_some_and(|job| job.frames.len() == 59),
        "event61 presentation did not start: err={:?}",
        g.load_err
    );
    present_event61(&mut g, false);
    while g.battle_event.is_some() {
        g.dialog.clear();
        g.advance_battle_event();
    }
    assert!(
        g.walk.as_ref().is_some_and(|w| !w.native_group_march_paused)
            && state12(&g) == 1
            && g.party_members.get(&31).copied().unwrap_or(false)
            && trigger_inventory(&g, trigger) == [0x20],
        "event61 did not resume after commit: walk={} state={} inventory={:?} err={:?}",
        g.walk.is_some(),
        state12(&g),
        trigger_inventory(&g, trigger),
        g.load_err
    );
    g.step_battle_walk();
    let unit = &g.st.as_ref().unwrap().units[trigger];
    assert!(
        g.walk.is_none() && g.native_system_group_march.is_none() && unit.acted && unit.native_record_byte5 & 0x80 != 0,
        "group march did not finish after event61: walk={} plan={} acted={} byte5={:#x}",
        g.walk.is_some(),
        g.native_system_group_march.is_some(),
        unit.acted,
        unit.native_record_byte5
    );
}

/// 父行程組的素材包：原包每一項以符號連結指回，`animations` 底下少了事件 61 的圖格表。
fn pack_without_event61_bank() -> Option<std::path::PathBuf> {
    let source = std::path::PathBuf::from(std::env::var_os("FD2_ASSET_PACK")?);
    let root = test_support::temp_dir("event61-missing-bank-pack");
    let link = |from: std::path::PathBuf, to: std::path::PathBuf| {
        std::os::unix::fs::symlink(from, to).expect("建立素材包符號連結");
    };
    for entry in std::fs::read_dir(&source).ok()?.flatten() {
        let name = entry.file_name();
        if name == "animations" {
            let animations = root.join("animations");
            std::fs::create_dir_all(&animations).ok()?;
            for anim in std::fs::read_dir(entry.path()).ok()?.flatten() {
                if anim.file_name() != "fdother_045_event61" {
                    link(anim.path(), animations.join(anim.file_name()));
                }
            }
        } else {
            link(entry.path(), root.join(&name));
        }
    }
    Some(root)
}

/// Go：`native_field_event61_test.go:300` `TestNativeEvent61ProductionOwnersRejectMissingSeparatedBank`
#[test]
fn test_native_event61_production_owners_reject_missing_separated_bank() {
    let name = "test_native_event61_production_owners_reject_missing_separated_bank";
    let pack = if std::env::var_os("FD2_GAME_F_TEST_DIR").is_some() { None } else { pack_without_event61_bank() };
    if std::env::var_os("FD2_GAME_F_TEST_DIR").is_none() && pack.is_none() {
        return;
    }
    if !event61_child(name, pack) {
        return;
    }
    assert!(
        !std::path::Path::new(&separated_asset_path(EVENT61_BANK)).join("bank.json").exists(),
        "子行程的素材包仍有事件 61 圖格表"
    );
    let (mut g, trigger) = native_event61_player_game(&[0xD0, 0x20]);
    let step = battle::NativeSystemGroupMarchStep {
        unit_index: 0,
        events: vec![battle::NativeSystemGroupMarchEvent {
            path_index: 0,
            event_id: 61,
            text_index: 3,
            presentation: true,
        }],
        ..Default::default()
    };
    let plan = battle::NativeSystemGroupMarchPlan { steps: vec![step], ..Default::default() };
    assert!(
        !g.preflight_native_system_group_march_events(&plan),
        "group-march preflight accepted a missing event61 separated bank"
    );
    let st = g.st.as_ref().unwrap();
    let (x, y) = (st.units[trigger].x, st.units[trigger].y);
    let plan = battle::plan_native_field_event61(Some(st), Some(trigger), x, y).expect("PlanNativeFieldEvent61");
    assert!(
        g.begin_native_field_event61_presentation(plan, None).is_err(),
        "event61 presentation accepted a missing separated bank"
    );
    assert!(
        g.native_field_event61.is_none() && state12(&g) == 0 && trigger_inventory(&g, trigger) == [0xD0, 0x20],
        "missing event61 bank partially published the presentation or mutation"
    );
}
