//! R4 D 組：D 組在 `main.go` 名下函式的測試（`atk_animation_test.go`、`phase_banner_test.go` 的前半、
//! `native_death_program_test.go` 的死亡獎勵）。

use super::*;

/// `battle::Unit`／`battle::State`／`fdother::Frame` 有私有欄位，不能用結構更新語法；先 `default()` 再一次寫入欄位。
macro_rules! build {
    ($t:ty { $($f:ident : $v:expr),* $(,)? }) => {{
        let mut x = <$t>::default();
        ($(x.$f,)*) = ($($v,)*);
        x
    }};
}

/// Go `counterFixture`（`atk_animation_test.go:176`）：攻守雙方都有完整 FIGANI 資源的 `Game`。索引規則見
/// `figani_index`：fig×3 是待機、fig×3+1 是攻擊動作。Go 的畫格是 1×1 圖；Rust 只看幀數。
fn counter_fixture() -> Game {
    let frames = |n: usize| vec![Some(Arc::new(Picture { width: 1, height: 1, pix: vec![0; 4] })); n];
    Game {
        figani: BTreeMap::from([(12, frames(4)), (13, frames(3)), (288, frames(4)), (289, frames(2))]),
        figani_delays: BTreeMap::from([
            (12, vec![1, 1, 1, 1]),
            (13, vec![1, 2, 1]),
            (288, vec![1, 1, 1, 1]),
            (289, vec![2, 3]),
        ]),
        ..Default::default()
    }
}

/// Go 以 `t.Setenv("FD2_BATTLE_FPT", "2")`；Rust 在子行程設定（F 組的 `f_child_env`）。
fn with_battle_fpt_two(name: &str) -> bool {
    f_child_env(module_path!(), name, |_| vec![("FD2_BATTLE_FPT", "2".into())]).is_some()
}

/// Go：`atk_animation_test.go:12`
#[test]
fn test_new_atk_anim_requires_native_delay_pairing() {
    if !with_battle_fpt_two("test_new_atk_anim_requires_native_delay_pairing") {
        return;
    }
    let frames = || vec![Some(Arc::new(Picture { width: 1, height: 1, pix: vec![0; 4] })); 3];
    let mut g = Game {
        figani: BTreeMap::from([(13, frames()), (288, frames())]),
        figani_delays: BTreeMap::from([(13, vec![1, 2, 1]), (288, vec![1, 2, 1])]),
        ..Default::default()
    };
    let a = g
        .new_atk_anim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
        .expect("paired FIGANI delay schedule did not create an attack presentation");
    let timeline = a.figani_timeline.as_ref().expect("timeline");
    assert_eq!((a.body_ticks, a.total, a.frame_index), (8, 16, 0));
    assert_eq!(timeline.frame_start(2), Some(6));
    g.figani_delays = BTreeMap::new();
    assert!(
        g.new_atk_anim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true).is_none(),
        "unpaired FIGANI PNGs received a guessed attack timeline"
    );
}

/// Go：`atk_animation_test.go:38`
#[test]
fn test_new_atk_anim_leaves_native_impact_dac_unwired() {
    let g = counter_fixture();
    let a = g
        .new_atk_anim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
        .expect("paired FIGANI delay schedule did not create an attack presentation");
    assert!(a.native_impact_raw.is_none(), "attack animation inferred native DAC output without raw provenance");
}

/// 反擊測試的單位：Go 的單位不在 `g.st`，Rust 放進 `st.units`（0 攻方亞雷斯、1 守方盜賊）。
fn counter_units(g: &mut Game, actor_hp: i64) {
    let actor = build!(battle::Unit { battle_fig: 4, hp: actor_hp, max_hp: 48 });
    let target = build!(battle::Unit { battle_fig: 96, hp: 8, max_hp: 28 });
    g.st = Some(build!(battle::State { units: vec![actor, target] }));
}

/// Go：`atk_animation_test.go:199`
#[test]
fn test_counter_stage_swaps_resources_and_sides() {
    if !with_battle_fpt_two("test_counter_stage_swaps_resources_and_sides") {
        return;
    }
    let mut g = counter_fixture();
    let mut a = g
        .new_atk_anim(4, 96, "亞雷斯", "盜賊", 31, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
        .expect("主攻演出沒有建立");
    // 結算在演出之前，所以攻方 HP 已經是被反擊之後的 31；第一段要顯示 48。
    counter_units(&mut g, 31);
    let counter = battle::AttackResult { amount: 17, ..Default::default() };
    let result = battle::AttackResult { amount: 20, counter: Some(Box::new(counter)), ..Default::default() };
    g.attach_counter_presentation(Some(&mut a), Some(0), Some(1), &result);
    assert_eq!(a.atk_hp, 48, "第一段攻方 HP 應修回反擊前的 48");
    assert!(a.counter.is_some(), "反擊段沒有接上");
    let (first_total, first_fig) = (a.total, a.atk_fig);
    a.begin_counter_stage();
    assert_eq!((a.atk_fig, a.def_fig), (289, 12), "換裝後資源應為 289（盜賊攻擊）／12（亞雷斯待機）");
    assert_eq!(first_fig, 13);
    assert_eq!((a.atk_name.as_str(), a.def_name.as_str()), ("盜賊", "亞雷斯"));
    assert!(!a.atk_own, "陣營沒有換邊");
    assert_eq!((a.atk_hp, a.atk_max), (8, 28));
    assert_eq!((a.def_hp0, a.def_hp1, a.def_max), (48, 31, 48));
    assert_eq!((a.timer, a.frame_index), (a.total, 0), "時間軸沒有重新開始");
    // 兩段長度各自等於自己那組延遲和×fpt，加上尾段停格。
    assert_eq!(first_total, (1 + 2 + 1) * 2 + 4 * 2);
    assert_eq!(a.total, (2 + 3) * 2 + 4 * 2);
    assert!(a.counter.is_none(), "換裝之後 counter 應該清掉，否則會無限對打下去");
}

/// Go：`atk_animation_test.go:257`
#[test]
fn test_counter_stage_fails_closed_without_assets() {
    if !with_battle_fpt_two("test_counter_stage_fails_closed_without_assets") {
        return;
    }
    let mut g = counter_fixture();
    g.figani_delays.remove(&289); // 盜賊的攻擊動作沒有延遲表
    let mut a = g
        .new_atk_anim(4, 96, "亞雷斯", "盜賊", 31, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
        .expect("主攻演出不該因為反擊素材缺件就建不起來");
    counter_units(&mut g, 31);
    let counter = battle::AttackResult { amount: 17, ..Default::default() };
    let result = battle::AttackResult { amount: 20, counter: Some(Box::new(counter)), ..Default::default() };
    g.attach_counter_presentation(Some(&mut a), Some(0), Some(1), &result);
    assert!(a.counter.is_none(), "缺延遲表時仍接上了反擊段");
    assert_eq!(a.atk_hp, 48, "攻方 HP 還是要修回反擊前的 48");
    // 沒有反擊時完全不動。
    let mut b = g.new_atk_anim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true);
    counter_units(&mut g, 48);
    g.attach_counter_presentation(
        b.as_mut(),
        Some(0),
        Some(1),
        &battle::AttackResult { amount: 20, ..Default::default() },
    );
    let b = b.unwrap();
    assert!(b.counter.is_none() && b.atk_hp == 48, "沒有反擊時被動到了");
}

/// Go `TestPhaseBannerScheduleFollowsTheBiosTickContract`（`phase_banner_test.go:13`）的長度部分。逐幀走馬賽克
/// 步驟要 `phaseBannerStep`／`phaseBannerSlideOffset`（畫面用，`r8`），在 R5 連同繪圖一起移植。
#[test]
fn test_phase_banner_schedule_follows_the_bios_tick_contract_lengths() {
    for (banner, hold) in [
        ("ENEMY PHASE", PHASE_BANNER_ENEMY_HOLD_MILLIS as f64),
        (PHASE_BANNER_PLAYER_TEXT, PHASE_BANNER_PLAYER_HOLD_MILLIS as f64),
    ] {
        assert_eq!(phase_banner_hold_millis(banner), hold, "{banner} 中段停留");
        let steps = indexedmap::PHASE_BANNER_SLIDE_IN_STEPS
            + indexedmap::PHASE_BANNER_STEPS
            + indexedmap::PHASE_BANNER_SLIDE_OUT_STEPS;
        let want = steps as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS + hold;
        assert!((phase_banner_total_millis(banner) - want).abs() <= 1e-9, "{banner} 全長");
    }
}

/// D 組補充：橫幅幀數無條件進位（7＋33＋5 步各一個 BIOS tick，加中段停留，換成 60 Hz）。
#[test]
fn phase_banner_frames_round_up_and_show_banner_resets_snapshot() {
    // (45 × 54.925409453960306 + 20) × 60 / 1000 = 149.498... → 150；玩家回合 +150 毫秒 → 157.298... → 158。
    assert_eq!(phase_banner_frames("ENEMY PHASE"), 150);
    assert_eq!(phase_banner_frames(PHASE_BANNER_PLAYER_TEXT), 158);
    let mut g = Game { banner_frame: vec![1, 2, 3], ..Default::default() };
    g.show_banner(PHASE_BANNER_PLAYER_TEXT);
    assert_eq!((g.banner.as_str(), g.banner_t), (PHASE_BANNER_PLAYER_TEXT, 158));
    assert!(g.banner_frame.is_empty());
    let mut shot = Game { shot_path: String::from("x.png"), ..Default::default() };
    shot.show_banner("ENEMY PHASE");
    assert!(shot.banner.is_empty() && shot.banner_t == 0, "截圖模式不顯示橫幅");
}

/// Go：`native_death_program_test.go:189`。物品與金錢只給原版陣營 2 的擊殺者（0x1AC7B、0x1AB95
/// `cmp byte [esi+6], 2`）。Go 的單位不在 `g.st`；Rust 放進 `st.units`（0 友軍、1 我方）。
#[test]
fn test_native_death_reward_only_for_player_killer() {
    let empty = |camp, side| {
        build!(battle::Unit {
            camp: camp,
            native_record_byte6: side,
            has_native_record_byte6: true,
            inventory_slots: vec![0xff; 8],
            native_inventory_flags: vec![0x80; 8]
        })
    };
    let mut g = Game {
        st: Some(build!(battle::State { units: vec![empty(battle::ALLY, 1), empty(battle::OWN, 2)] })),
        ..Default::default()
    };
    g.grant_native_death_reward(1, 1000, Some(0));
    g.grant_native_death_reward(1, 1000, None);
    g.grant_native_death_reward(0, 0xd3, Some(0));
    g.grant_native_death_reward(0, 0xd3, None);
    assert_eq!(g.gold, 0, "友軍或狀態致死不該拿到金錢");
    assert!(g.st.as_ref().unwrap().units[0].inventory.is_empty() && g.pending_native_death_rewards.is_empty());
    g.grant_native_death_reward(1, 1000, Some(1));
    // 0x1ABFD：金額在訊息關框之後才加，grant 只排訊息。
    assert_eq!(g.gold, 0);
    assert_eq!(g.pending_native_reward_msgs.len(), 1);
    assert_eq!((g.pending_native_reward_msgs[0].kind, g.pending_native_reward_msgs[0].value), (1, 1000));
    g.grant_native_death_reward(0, 0xd3, Some(1));
    let own = &g.st.as_ref().unwrap().units[1];
    assert_eq!(own.inventory, vec![0xd3]);
    assert_eq!((own.inventory_slots[0], own.native_inventory_flags[0]), (0xd3, 0));
    assert!(g.pending_native_death_rewards.is_empty());
}

/// D 組補充：`awardDeathReward` 只處理一次死亡、活著的單位不處理；型態 0／1 直接給，型態 2／3 登記成死亡程式。
#[test]
fn award_death_reward_runs_once_and_queues_programs() {
    let mut g = Game::default();
    let dead =
        build!(battle::Unit { hp: 0, max_hp: 10, death_reward: Some(battle::DeathEffect { r#type: 1, value: 300 }) });
    let killer = build!(battle::Unit { camp: battle::OWN, hp: 10, max_hp: 10 });
    let alive = build!(battle::Unit { hp: 5, max_hp: 10 });
    let program =
        build!(battle::Unit { hp: 0, max_hp: 10, death_effect: Some(battle::DeathEffect { r#type: 2, value: 7 }) });
    g.st = Some(build!(battle::State { units: vec![dead, killer, alive, program] }));
    g.award_death_reward(Some(0), Some(1));
    g.award_death_reward(Some(0), Some(1));
    g.award_death_reward(Some(2), Some(1));
    g.award_death_reward(None, Some(1));
    assert_eq!(g.pending_native_reward_msgs.len(), 1, "同一次死亡只給一次");
    assert_eq!(g.death_rewarded.get(&0), Some(&true));
    assert!(!g.death_rewarded.contains_key(&2), "活著的單位不記");
    g.award_death_reward(Some(3), None);
    assert!(g.pending_death_programs.is_empty(), "沒有擊殺者不登記死亡程式");
    g.death_rewarded.clear();
    g.award_death_reward(Some(3), Some(1));
    assert_eq!(g.pending_death_programs, vec![PendingDeathProgram { dead: Some(3), killer: Some(1) }]);
    assert!(g.native_death_programs_pending());
}

/// D 組補充：`battleFPT` 沒有環境變數時是 3，`figaniIndex` 是 fig×3。
#[test]
fn battle_fpt_default_and_figani_index() {
    if std::env::var_os("FD2_BATTLE_FPT").is_none() {
        assert_eq!(battle_fpt(), 3);
    }
    assert_eq!(figani_index(96), 288);
}

/// D 組補充：`resetAllNativeMapPoses` 把全部有地圖呈現的記錄姿勢歸零（0x134E4），不動沒有呈現的。
#[test]
fn reset_all_native_map_poses_rests_every_presented_record() {
    let mut g = Game::default();
    let presented = || {
        build!(battle::Unit {
            has_native_map_presentation: true,
            native_map_presentation: battle::NativeMapPresentationState { pose: 3, ..Default::default() }
        })
    };
    g.st = Some(build!(battle::State { units: vec![presented(), presented(), battle::Unit::default()] }));
    g.reset_all_native_map_poses();
    let units = &g.st.as_ref().unwrap().units;
    assert!(units[..2].iter().all(|u| u.native_map_presentation.pose as i64 == NATIVE_MAP_REST_POSE));
}
