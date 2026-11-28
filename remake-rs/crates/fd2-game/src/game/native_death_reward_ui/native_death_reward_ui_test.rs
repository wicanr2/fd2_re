//! Go 來源：`remake/cmd/fd2/native_death_reward_ui_test.go`（期望值不改）。
//!
//! Go 以 `t.Setenv` 設的環境變數由父測試以子行程傳入（`f_child_env`）。Go 的 `assetPackRootWithLocales` 把
//! 受版控的 `assets/locales` 疊到素材包；Rust 的語系一律從 `asset_path("assets/locales")` 讀（`locale.rs`），
//! 素材包不必帶語系，不另組暫存目錄。

use super::super::parity_replay::{Driver, ch01_blockers};
use super::*;
use crate::test_support;

const JOURNEY_BATTLE_NODE: &str = "battle_ch01";

/// Go：`native_death_program_test.go:16` `deathBattleIdle`。
fn death_battle_idle(g: &Game) -> bool {
    g.st.is_some()
        && g.result.is_empty()
        && !g.ai_busy
        && g.battle_event.is_none()
        && g.native_turn_staging.is_none()
        && g.dialog.is_empty()
        && g.walk.is_none()
        && g.atk.is_none()
        && !g.ring
        && g.spawn_intro_transition.is_none()
        && g.indexed_transition.is_none()
        && g.native_unit_present.is_none()
        && g.act_job.is_none()
        && g.cam_pan.is_none()
        && g.focus_job.is_none()
}

/// 子行程的環境（Go `loadChapterOneDeathRewardBattle` 的 `t.Setenv`）；素材包不在時回 `None`，父行程略過。
fn death_reward_child(name: &str) -> bool {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return false;
    }
    f_child_env(module_path!(), name, |dir| {
        vec![
            ("FD2_MUTE", "1".into()),
            ("FD2_TITLE", "0".into()),
            ("FD2_CAMPAIGN", CANONICAL_CAMPAIGN_REFERENCE.into()),
            ("FD2_CAMP_NODE", JOURNEY_BATTLE_NODE.into()),
            ("FD2_NATIVE_SAVE", dir.join("FD2.SAV").into_os_string()),
            ("XDG_DATA_HOME", dir.as_os_str().to_os_string()),
        ]
    })
    .is_some()
}

/// Go：`native_death_reward_ui_test.go:15` `loadChapterOneDeathRewardBattle`。
fn load_chapter_one_death_reward_battle() -> Driver {
    let mut g = load_game().expect("loadGame");
    if g.load_err.starts_with("separated UI sounds:") {
        g.load_err.clear();
    }
    assert!(g.load_err.is_empty(), "{}", g.load_err);
    assert!(g.st.is_some() && g.sc.is_some(), "第一關戰場或劇本沒有建立");
    let mut d = Driver {
        g,
        clock: VirtualFrameClock::default(),
        frame_capture: None,
        map_layer_capture: None,
        dialogue_layer_capture: None,
        r5p_layer_capture: None,
        r5c_layer_capture: None,
        ttf_layer_capture: None,
        r5b_layer_capture: None,
        r5h_layer_capture: None,
        r5s_frame_capture: None,
        hp_policy: None,
    };
    let idle = d.pump(20000, |g| {
        if !g.dialog.is_empty() && g.battle_event.is_none() {
            g.dlg_advance();
        }
        death_battle_idle(g)
    });
    assert!(idle, "第一關的開場工作沒有收掉\n阻塞：{}", ch01_blockers(&d.g));
    // Go 的原生地圖時鐘讀 time.Now；Rust 讀虛擬時鐘，開場工作若在第一幀前就收掉，`virtual_now` 仍是零值，
    // 先推一格虛擬幀（不跑 Update）。
    d.g.advance_virtual_frame(&mut d.clock);
    d
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct InventorySnapshot {
    inventory: Vec<i64>,
    equipped: Vec<bool>,
    slots: Vec<i64>,
    flags: Vec<i64>,
}

/// Go：`native_death_reward_ui_test.go:51` `makeNativeDeathRewardKiller`。
fn make_native_death_reward_killer(g: &mut Game) -> UnitIdx {
    let st = g.st.as_mut().unwrap();
    for (index, unit) in st.units.iter_mut().enumerate() {
        if unit.camp != battle::OWN || !unit.alive() || !unit.on_field || !unit.has_battle_fig {
            continue;
        }
        *unit.inventory = vec![0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17];
        *unit.equipped = vec![true, false, true, false, false, true, false, true];
        *unit.inventory_slots = unit.inventory.to_vec();
        *unit.native_inventory_flags = vec![0x40, 0, 0x40, 0, 0, 0x40, 0, 0x40];
        if let Err(err) = battle::validate_native_inventory_projection(Some(unit)) {
            panic!("完整原生道具欄 fixture 無效：{err}");
        }
        return index;
    }
    panic!("第一關找不到具原生頭像的我方擊殺者");
}

fn snapshot(g: &Game, killer: UnitIdx) -> InventorySnapshot {
    let unit = &g.st.as_ref().unwrap().units[killer];
    InventorySnapshot {
        inventory: unit.inventory.to_vec(),
        equipped: unit.equipped.to_vec(),
        slots: unit.inventory_slots.to_vec(),
        flags: unit.native_inventory_flags.to_vec(),
    }
}

fn assert_inventory(g: &Game, killer: UnitIdx, want: &InventorySnapshot) {
    let got = snapshot(g, killer);
    assert_eq!(&got, want, "道具交易非原子");
}

/// 正式 Draw 發布每張索引畫面，再由正式 Update 推進生命週期。
///
/// Go：`native_death_reward_ui_test.go:93` `driveNativeDeathRewardUI`
fn drive_native_death_reward_ui(d: &mut Driver, screen: &mut Picture, done: impl Fn(&Game) -> bool) {
    for frame in 0..1024 {
        if done(&d.g) {
            return;
        }
        let had_job = d.g.native_class_ui_job.is_some();
        d.g.draw(screen);
        if had_job && d.g.native_class_ui_job.as_ref().is_some_and(|job| !job.drawn) {
            panic!("第 {frame} 幀未由正式 Draw 確認原生 UI 發布");
        }
        d.g.advance_virtual_frame(&mut d.clock);
        if let Err(err) = d.g.update() {
            panic!("第 {frame} 幀正式 Update：{}", err.message());
        }
        assert!(d.g.load_err.is_empty(), "第 {frame} 幀原生獎勵 UI 失敗：{}", d.g.load_err);
    }
    assert!(done(&d.g), "原生死亡獎勵 UI 超過 1024 幀仍未抵達預期邊界");
}

/// Go：`native_death_reward_ui_test.go:113` `beginFullNativeDeathRewardPrompt`
fn begin_full_native_death_reward_prompt(d: &mut Driver, killer: UnitIdx, rewards: &[i64]) -> Picture {
    for &reward in rewards {
        d.g.grant_native_death_reward(0, reward, Some(killer));
    }
    assert_eq!(d.g.pending_native_death_rewards.len(), rewards.len(), "完整道具欄未依序排入獎勵");
    d.g.finish_successful_unit_action(Some(killer), None);
    let acted = d.g.st.as_ref().unwrap().units[killer].acted;
    assert!(
        d.g.native_death_reward_ui.is_some()
            && d.g.native_system_end_turn_confirm
            && d.g.native_class_ui_job.is_some()
            && !acted,
        "未停在丟棄提示：confirm={} job={} acted={acted} err={:?}",
        d.g.native_system_end_turn_confirm,
        d.g.native_class_ui_job.is_some(),
        d.g.load_err
    );
    let mut screen = Picture::new_image(LOGICAL_W, LOGICAL_H);
    drive_native_death_reward_ui(d, &mut screen, |g| {
        g.native_class_ui_job.is_none() && g.native_system_end_turn_confirm
    });
    screen
}

/// Go：`native_death_reward_ui_test.go:133` `TestNativeDeathRewardFullInventoryNoAndEscapeRemainAtomic`。
/// Go 的兩個子測試（NO 與 Escape）走同一個取消分支，內容相同。
#[test]
fn test_native_death_reward_full_inventory_no_and_escape_remain_atomic() {
    if !death_reward_child("test_native_death_reward_full_inventory_no_and_escape_remain_atomic") {
        return;
    }
    for _route in ["NO", "Escape"] {
        let mut d = load_chapter_one_death_reward_battle();
        let killer = make_native_death_reward_killer(&mut d.g);
        let before = snapshot(&d.g, killer);
        let mut screen = begin_full_native_death_reward_prompt(&mut d, killer, &[0xd3]);
        // Escape 與選到 NO 都由 ringInput 匯入同一個原版取消分支。
        d.g.cancel_native_system_end_turn();
        assert_inventory(&d.g, killer, &before);
        drive_native_death_reward_ui(&mut d, &mut screen, |g| {
            g.native_death_reward_ui.is_none() && g.native_system_end_turn_ui.is_none()
        });
        assert_inventory(&d.g, killer, &before);
        let acted = d.g.st.as_ref().unwrap().units[killer].acted;
        assert!(acted && d.g.pending_native_death_rewards.is_empty(), "取消後未回到行動收尾：acted={acted}");
    }
}

/// Go：`native_death_reward_ui_test.go:155` `TestNativeDeathRewardDiscardCommitsOnlyAfterItemPanelClose`
#[test]
fn test_native_death_reward_discard_commits_only_after_item_panel_close() {
    if !death_reward_child("test_native_death_reward_discard_commits_only_after_item_panel_close") {
        return;
    }
    let mut d = load_chapter_one_death_reward_battle();
    let killer = make_native_death_reward_killer(&mut d.g);
    let before = snapshot(&d.g, killer);
    let mut screen = begin_full_native_death_reward_prompt(&mut d, killer, &[0xd3]);

    d.g.confirm_native_system_end_turn();
    drive_native_death_reward_ui(&mut d, &mut screen, |g| g.item_open && !g.item_closing && g.item_anim_step == 11);
    assert!(
        d.g.sel == Some(killer) && d.g.native_item_panel.is_some() && d.g.draw_native_item_panel(&mut screen),
        "YES 未進入擊殺者的實際八格原生道具面板"
    );
    assert_inventory(&d.g, killer, &before);

    for step in 0..3 {
        let down = NativeDeathRewardInput { down: true, ..Default::default() };
        assert!(d.g.handle_native_death_reward_item_input(down), "第 {step} 次 Down 未被獎勵選擇器擁有");
    }
    let confirm = NativeDeathRewardInput { confirm: true, ..Default::default() };
    assert!(
        d.g.item_sel == 3 && d.g.handle_native_death_reward_item_input(confirm),
        "選擇器未停在原始索引 3：itemSel={}",
        d.g.item_sel
    );
    assert_inventory(&d.g, killer, &before);
    drive_native_death_reward_ui(&mut d, &mut screen, |g| g.native_death_reward_ui.is_none());

    let want_items = vec![0x10, 0x11, 0x12, 0x14, 0x15, 0x16, 0x17, 0xd3];
    let want_equipped = vec![true, false, true, false, true, false, true, false];
    let unit = &d.g.st.as_ref().unwrap().units[killer];
    assert!(
        *unit.inventory == want_items
            && *unit.inventory_slots == want_items
            && *unit.equipped == want_equipped
            && unit.native_inventory_flags[7] == 0,
        "丟棄索引 3 後結果錯誤：inventory={:?} slots={:?} equipped={:?} flags={:?}",
        *unit.inventory,
        *unit.inventory_slots,
        *unit.equipped,
        *unit.native_inventory_flags
    );
    assert!(
        unit.acted && d.g.pending_native_death_rewards.is_empty(),
        "交易後未回到行動收尾：acted={} pending={}",
        unit.acted,
        d.g.pending_native_death_rewards.len()
    );
}

/// Go：`native_death_reward_ui_test.go:193` `TestNativeDeathRewardSelectorCancelAndSequentialQueue`
#[test]
fn test_native_death_reward_selector_cancel_and_sequential_queue() {
    if !death_reward_child("test_native_death_reward_selector_cancel_and_sequential_queue") {
        return;
    }
    let mut d = load_chapter_one_death_reward_battle();
    let killer = make_native_death_reward_killer(&mut d.g);
    let before = snapshot(&d.g, killer);
    let mut screen = begin_full_native_death_reward_prompt(&mut d, killer, &[0xd3, 0xd4]);

    // 第一筆 YES 後在道具選擇器按 Escape，應顯示 0x1B2 並零變更，接著由同一個 action continuation 開啟第二筆提示。
    d.g.confirm_native_system_end_turn();
    drive_native_death_reward_ui(&mut d, &mut screen, |g| g.item_open && !g.item_closing && g.item_anim_step == 11);
    let cancel = NativeDeathRewardInput { cancel: true, ..Default::default() };
    assert!(d.g.handle_native_death_reward_item_input(cancel), "道具選擇器 Escape 未被擁有");
    drive_native_death_reward_ui(&mut d, &mut screen, |g| {
        g.native_system_end_turn_confirm && g.native_class_ui_job.is_none() && g.pending_native_death_rewards.is_empty()
    });
    assert_inventory(&d.g, killer, &before);
    let acted = d.g.st.as_ref().unwrap().units[killer].acted;
    assert!(
        !acted && d.g.native_death_reward_ui.as_ref().is_some_and(|reward| reward.item == 0xd4),
        "第二筆獎勵未循序接手：acted={acted}"
    );

    // 第二筆選 NO，兩筆都放棄後才真正結束行動。
    d.g.cancel_native_system_end_turn();
    drive_native_death_reward_ui(&mut d, &mut screen, |g| g.native_death_reward_ui.is_none());
    assert_inventory(&d.g, killer, &before);
    assert!(d.g.st.as_ref().unwrap().units[killer].acted, "序列獎勵全部結束後仍未完成擊殺者行動");
}

/// Go：`native_death_reward_ui_test.go:225` `TestNativeDeathRewardMissingProvenanceFailsClosedWithoutMutation`
#[test]
fn test_native_death_reward_missing_provenance_fails_closed_without_mutation() {
    let mut st = battle::State::default();
    let mut killer = battle::Unit::default();
    killer.camp = battle::OWN;
    *killer.inventory = vec![1, 2, 3, 4, 5, 6, 7, 8];
    *killer.equipped = vec![false; 8];
    *killer.inventory_slots = vec![1, 2, 3, 4, 5, 6, 7, 8];
    *killer.native_inventory_flags = vec![0; 8];
    st.units.push(killer);
    let mut g = Game {
        st: Some(st),
        pending_native_death_rewards: vec![PendingNativeDeathReward { killer: Some(0), item: 0xd3 }],
        ..Default::default()
    };
    let before = snapshot(&g, 0);
    let continued = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let flag = continued.clone();
    let then: Box<dyn FnOnce(&mut Game) + Send> =
        Box::new(move |_| flag.store(true, std::sync::atomic::Ordering::SeqCst));
    assert!(g.run_pending_native_death_rewards(Some(then)), "缺來源的待處理獎勵未被失敗即關閉閘門擁有");
    let continued = continued.load(std::sync::atomic::Ordering::SeqCst);
    assert!(
        !g.load_err.is_empty() && !continued && g.native_death_reward_ui.is_none(),
        "缺來源未停住：err={:?} continued={continued}",
        g.load_err
    );
    assert_inventory(&g, 0, &before);
}
