//! 戰場快照的擷取、阻擋條件、節點驗證與套用。Go `battle_snapshot_test.go` 的整合測試要沿正常劇情快進到
//! 第一關（`fastForwardShotCampaign`，Rust 尚未移植），這裡改用合成的戰鬥節點驗同一組函式。

use super::*;

fn snapshot_test_game() -> Game {
    let node = campaign::Node {
        r#type: "battle".into(),
        units: "units/ch01.json".into(),
        scenario: "scenarios/ch01.json".into(),
        ..Default::default()
    };
    let c = campaign::Campaign {
        start: "battle_ch01".into(),
        nodes: BTreeMap::from([("battle_ch01".to_string(), node)]),
        ..Default::default()
    };
    let mut st = battle::State::default();
    (st.w, st.h, st.turn) = (3, 2, 2);
    let mut unit = battle::Unit::default();
    (unit.camp, unit.on_field, unit.hp, unit.max_hp, unit.x, unit.y) = (battle::OWN, true, 30, 40, 1, 1);
    st.units = vec![unit];
    Game {
        camp: Some(campaign::new_runner(&c)),
        st: Some(st),
        sc: Some(battle::Scenario::default()),
        m: Some(MapData { w: 3, h: 2, tile_w: 24, tile_h: 24, tiles: vec![1, 2, 3, 4, 5, 6], ..Default::default() }),
        native_rng_state: 0x1234,
        cur_x: 2,
        cur_y: 1,
        ..Default::default()
    }
}

#[test]
fn snapshot_round_trip_restores_state_and_clears_transients() {
    let mut g = snapshot_test_game();
    assert_eq!(g.battle_snapshot_blocker(), "");
    let snap = g.capture_battle_snapshot(false).expect("capture");
    assert_eq!((snap.node.as_str(), snap.cursor_x, snap.native_rng_state), ("battle_ch01", 2, 0x1234));
    assert_eq!(snap.map_sha256, g.battle_snapshot_map_digest());
    // 讀檔前的剛進場狀態：數值、游標與亂數都不同，還留有開場暫態。
    let st = g.st.as_mut().unwrap();
    (st.units[0].hp, st.turn) = (5, 1);
    (g.cur_x, g.cur_y, g.native_rng_state) = (0, 0, 9);
    (g.banner, g.banner_t) = ("第一章".into(), 30);
    g.apply_battle_snapshot(&snap).expect("apply");
    let st = g.st.as_ref().unwrap();
    assert_eq!((st.units[0].hp, st.turn), (30, 2));
    assert_eq!((g.cur_x, g.cur_y, g.prev_cur_x, g.native_rng_state), (2, 1, 2, 0x1234));
    assert!(g.banner.is_empty() && g.banner_t == 0, "opening banner survived the restore");
    assert_eq!(g.battle_snapshot_blocker(), "");
}

#[test]
fn snapshot_blocker_rejects_busy_battle_and_allows_system_menu() {
    let mut g = snapshot_test_game();
    g.ai_busy = true;
    assert_eq!(g.battle_snapshot_blocker(), "enemy phase");
    assert!(g.capture_battle_snapshot(false).is_err());
    g.ai_busy = false;
    (g.ring, g.native_system_cursor_overlay) = (true, true);
    assert_eq!(g.battle_snapshot_blocker(), "menu open");
    assert_eq!(g.battle_snapshot_blocker_for(true), "", "system menu save was rejected");
    g.sel = Some(0);
    assert_eq!(g.battle_snapshot_blocker_for(true), "unit action");
}

#[test]
fn snapshot_rejects_mismatched_version_node_and_map_without_mutation() {
    let mut g = snapshot_test_game();
    let snap = g.capture_battle_snapshot(false).expect("capture");
    assert!(g.validate_battle_snapshot_node(&snap, "battle_ch01").is_ok());
    let mut other = snap.clone();
    other.version = 2;
    assert!(g.validate_battle_snapshot_node(&other, "battle_ch01").is_err());
    let mut other = snap.clone();
    other.units = "units/ch02.json".into();
    assert!(g.validate_battle_snapshot_node(&other, "battle_ch01").is_err());
    assert!(g.validate_battle_snapshot_node(&snap, "story_ch02").is_err());
    g.st.as_mut().unwrap().units[0].hp = 7;
    g.m.as_mut().unwrap().tiles[0] = 99;
    assert!(g.apply_battle_snapshot(&snap).is_err(), "changed map accepted");
    assert_eq!(g.st.as_ref().unwrap().units[0].hp, 7, "rejected snapshot mutated the battle");
    g.m.as_mut().unwrap().tiles[0] = 1;
    let mut out_of_range = snap.clone();
    out_of_range.native_map_clock_tick = Some(0x8000);
    assert!(g.apply_battle_snapshot(&out_of_range).is_err(), "clock outside int16 accepted");
    assert_eq!(g.st.as_ref().unwrap().units[0].hp, 7, "rejected snapshot mutated the battle");
}
