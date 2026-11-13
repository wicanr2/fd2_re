//! 移植自 `remake/internal/battle/event_test.go`。
//!
//! Go 以指標相等（`st.Units[1] == pending`）判斷「是不是同一個單位」；Rust 的單位由陣列擁有，
//! 改比對可辨識的欄位，或註明該檢查由型別保證。

use super::*;
use crate::test_support::{docs_file, remake_asset, temp_dir};

/// Go `bindNativeFutureItemRowsForTest`；缺少資產時回 `None`（呼叫端略過）。
fn bind_native_future_item_rows_for_test(st: &mut State) -> Option<()> {
    let path = remake_asset("data/native_item_effect_rows.json")?;
    let rows = load_native_item_effect_row_prefix(&path).unwrap();
    st.bind_native_future_item_rows(&rows).unwrap();
    Some(())
}

fn write_scenario(tag: &str, raw: &str) -> String {
    let path = temp_dir(tag).join("scenario.json");
    std::fs::write(&path, raw).unwrap();
    path.to_string_lossy().into_owned()
}

/// `remake/assets/<rel>` 的地圖與劇本：兩者都在才回傳。
fn load_map_and_scenario(map: &str, scenario: &str) -> Option<(State, Scenario)> {
    let map = remake_asset(map)?;
    let scenario = remake_asset(scenario)?;
    Some((load(&map).unwrap(), load_scenario(&scenario).unwrap()))
}

fn action_types(actions: &[Action]) -> Vec<&str> {
    actions.iter().map(|a| a.r#type.as_str()).collect()
}

/// Go：`TestNativeEventStateScenarioConditionFailsClosedWhenIncomplete`
#[test]
fn native_event_state_scenario_condition_fails_closed_when_incomplete() {
    let path = write_scenario(
        "event-partial-state",
        r#"{
		"events":[{
			"trigger":"on_turn_end",
			"when":{"native_event_state_index":16},
			"do":[]
		}]
	}"#,
    );
    assert!(load_scenario(&path).is_err(), "partial native event-state condition unexpectedly loaded");
    let w = When { native_event_state_index: Some(16), ..Default::default() };
    assert!(
        !w.r#match(Some(&State::default()), ""),
        "partial direct native event-state condition unexpectedly matched"
    );
    let action = Action {
        r#type: "set_native_event_state".into(),
        native_event_id: Some(90),
        event_state_index: Some(16),
        event_state_value: Some(1),
        native_source: "0x34996".into(),
        ..Default::default()
    };
    let mut st = State::default();
    assert!(
        Scenario::default().execute_action_checked(Some(&mut st), &action).is_err(),
        "out-of-range native event provenance unexpectedly executed"
    );
}

/// Go：`TestScenarioPartyUnitsPreserveRuntimeOrderAndDeployment`
#[test]
fn scenario_party_units_preserve_runtime_order_and_deployment() {
    let pm = |name: &str, fig, hp| PartyMember { name: name.into(), fig, hp, ..Default::default() };
    let sc = Scenario {
        party: vec![pm("索爾", 0, 42), pm("悠妮", 9, 28), pm("亞雷斯", 4, 48), pm("蓋亞", 30, 50)],
        deploy_cells: vec![[7, 20], [8, 22], [10, 21], [11, 23]],
        ..Default::default()
    };
    let units = sc.party_units(&[]);
    assert_eq!(units.len(), 4, "party units");
    for (slot, (fig, x, y)) in [(0, 7, 20), (9, 8, 22), (4, 10, 21), (30, 11, 23)].into_iter().enumerate() {
        let u = &units[slot];
        assert!(
            u.fig == fig && u.x == x && u.y == y && u.on_field && u.camp == OWN,
            "runtime slot {slot} = {u:?}, want fig={fig} at ({x},{y})"
        );
        assert!(
            u.battle_fig == fig && u.has_map_selector_key && u.map_selector_key == fig,
            "fresh JOIN selector source slot {slot} = battle={} key={} known={}",
            u.battle_fig,
            u.map_selector_key,
            u.has_map_selector_key
        );
    }
}

/// Go：`TestScenarioPartyUnitsUseFDFIELDFallbackCells`
#[test]
fn scenario_party_units_use_fdfield_fallback_cells() {
    let sc = Scenario {
        party: vec![PartyMember { fig: 0, ..Default::default() }, PartyMember { fig: 9, ..Default::default() }],
        ..Default::default()
    };
    let units = sc.party_units(&[Cell { x: 3, y: 4 }, Cell { x: 5, y: 6 }]);
    assert!(
        units[0].x == 3 && units[0].y == 4 && units[1].x == 5 && units[1].y == 6,
        "fallback deployment lost: {units:?}"
    );
}

/// Go：`TestScenarioPartyUnitsCaptureEffectiveStatsAsEquipmentBase`
#[test]
fn scenario_party_units_capture_effective_stats_as_equipment_base() {
    let sc = Scenario {
        party: vec![PartyMember {
            fig: 0,
            ap: 16,
            dp: 12,
            hit: 97,
            ev: 2,
            mv: 4,
            atk_min: 1,
            atk_max: 1,
            ..Default::default()
        }],
        ..Default::default()
    };
    let u = &sc.party_units(&[])[0];
    assert!(
        !u.equipment_base_set
            && u.base_ap == 16
            && u.base_dp == 12
            && u.base_hit == 97
            && u.base_ev == 2
            && u.base_mv == 4
            && u.equipped.is_empty(),
        "equipment base not captured: {u:?}"
    );
}

/// Go：`TestScenarioPartyUnitsMaterializeRawCommandMask`
#[test]
fn scenario_party_units_materialize_raw_command_mask() {
    let sc = Scenario {
        party: vec![PartyMember {
            name: "原始指令".into(),
            initial_command_mask: vec![0x81, 0x01, 0, 0x80],
            ..Default::default()
        }],
        ..Default::default()
    };
    let got = sc.party_units(&[])[0].native_command_ids();
    assert_eq!(got, [0, 7, 8, 31], "native command IDs");
}

/// Go：`TestScenarioPartyUnitsPreserveOptionalNativeIdentity`
#[test]
fn scenario_party_units_preserve_optional_native_identity() {
    let sc = Scenario {
        party: vec![PartyMember {
            fig: 9,
            native_identity: Some(0x2a),
            native_record_race: Some(1),
            native_record_class: Some(5),
            ..Default::default()
        }],
        ..Default::default()
    };
    let u = &sc.party_units(&[])[0];
    assert!(
        u.has_native_identity && u.native_identity == 0x2a,
        "native identity={} known={}, want 42/true",
        u.native_identity,
        u.has_native_identity
    );
    assert!(
        u.has_native_record_race
            && u.native_record_race == 1
            && u.has_native_record_class
            && u.native_record_class == 5,
        "native race/class={}/{} known={}/{}",
        u.native_record_race,
        u.native_record_class,
        u.has_native_record_race,
        u.has_native_record_class
    );
    let legacy = Scenario { party: vec![PartyMember { fig: 9, ..Default::default() }], ..Default::default() };
    assert!(!legacy.party_units(&[])[0].has_native_identity, "legacy Fig must not imply native +0x08 identity");
}

/// Go：`TestLoadScenarioRejectsNativeIdentityOutsideByte`
#[test]
fn load_scenario_rejects_native_identity_outside_byte() {
    let path = write_scenario("event-bad-identity", r#"{"party":[{"name":"bad","native_identity":256}]}"#);
    assert!(load_scenario(&path).is_err(), "out-of-range native identity accepted");
}

/// Go：`TestLoadScenarioRejectsIncompleteNativeSpawnCall`
#[test]
fn load_scenario_rejects_incomplete_native_spawn_call() {
    let path = write_scenario(
        "event-bad-native-spawn",
        r#"{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[2],"native_event_id":15,"native_spawns":[{"group":2,"via":"spawn_group","source":"0x3464b"}]}]}]}"#,
    );
    assert!(load_scenario(&path).is_err(), "缺少 raw_placement_gate 的原版增援呼叫被接受");
}

/// Go：`TestLoadScenarioRejectsNativeEventOutsideGlobalTable`
#[test]
fn load_scenario_rejects_native_event_outside_global_table() {
    let path = write_scenario(
        "event-bad-native-event",
        r#"{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[2],"native_event_id":90,"native_spawns":[{"group":2,"via":"spawn_group","source":"0x3464b","raw_placement_gate":1}]}]}]}"#,
    );
    assert!(load_scenario(&path).is_err(), "超出90項全域表的 native_event_id 被接受");
}

/// Go：`TestLoadScenarioRequiresFollowingActingForNativeIntroSpawn`
#[test]
fn load_scenario_requires_following_acting_for_native_intro_spawn() {
    let path = write_scenario(
        "event-bad-native-intro-1",
        r#"{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[4],"native_event_id":1,"native_spawns":[{"group":4,"via":"spawn_group_with_intro","source":"0x342ce","raw_placement_gate":0}]}]}]}"#,
    );
    assert!(load_scenario(&path).is_err(), "缺少呼叫端 following_acting 的原版 intro spawn 被接受");
    let path = write_scenario(
        "event-bad-native-intro-2",
        r#"{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[4],"native_event_id":1,"native_spawns":[{"group":4,"via":"spawn_group_with_intro","source":"0x342ce","raw_placement_gate":0,"following_acting":{"resource":3,"source":"0x342e7"}}]}]}]}"#,
    );
    assert!(load_scenario(&path).is_err(), "缺少 native_acting_resources 的原版 intro spawn 被接受");
    let path = write_scenario(
        "event-bad-native-intro-3",
        r#"{"native_acting_resources":"assets/cutscenes/acting/map32.json","events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[4],"native_event_id":1,"native_spawns":[{"group":4,"via":"spawn_group_with_intro","source":"0x342ce","raw_placement_gate":0,"following_acting":{"resource":3,"source":"0x342e7"}}]}]}]}"#,
    );
    if let Err(err) = load_scenario(&path) {
        panic!("完整 native intro provenance/resources 被拒絕：{err}");
    }
}

/// Go：`TestGeneratedTurnSpawnsCarryExactNativeCallMetadata`
///
/// Go `filepath.Glob("ch*.json")` 依字典序；Rust 列目錄後排序，得到相同順序。
#[test]
fn generated_turn_spawns_carry_exact_native_call_metadata() {
    let Some(dir) = remake_asset("scenarios") else { return };
    let mut paths: Vec<std::path::PathBuf> = std::fs::read_dir(&dir)
        .unwrap()
        .map(|e| e.unwrap().path())
        .filter(|p| {
            let name = p.file_name().unwrap().to_string_lossy().into_owned();
            name.starts_with("ch") && name.ends_with(".json")
        })
        .collect();
    paths.sort();
    let (mut calls, mut actions) = (0, 0);
    let mut gate_one = Vec::new();
    let mut intro = Vec::new();
    for path in &paths {
        let base = path.file_name().unwrap().to_string_lossy().into_owned();
        let sc = load_scenario(&path.to_string_lossy()).unwrap_or_else(|err| panic!("{base}: {err}"));
        for event in &sc.events {
            for action in &event.r#do {
                if action.r#type != "spawn_group" {
                    continue;
                }
                actions += 1;
                assert_eq!(
                    action.native_spawns.len(),
                    action.groups.len(),
                    "{base}/{}: groups={:?} native={:?}",
                    event.id,
                    action.groups,
                    action.native_spawns
                );
                assert!(action.native_event_id.is_some(), "{base}/{}: 缺少 native_event_id", event.id);
                for call in &action.native_spawns {
                    calls += 1;
                    if call.via == "spawn_group_with_intro" {
                        let acting = call.following_acting.as_ref().unwrap();
                        intro.push(format!("{}:{}:{}", call.group, acting.resource, acting.source));
                    }
                    if call.raw_placement_gate == Some(1) {
                        gate_one.push(format!("{base}:{}:{}", call.group, call.source));
                    }
                }
            }
        }
    }
    // 第十三章 event5 的入口前共用尾段加入第47個呼叫；原始來源
    // 與實際配置另由 TestChapter13Turn4AppendsOriginalRecord59 核對。
    assert!(actions == 47 && calls == 47, "產生的增援覆蓋 actions/calls={actions}/{calls}，預期 47/47");
    intro.sort();
    assert_eq!(intro, ["4:3:0x342e7", "5:4:0x3434f"], "intro 後續 acting");
    gate_one.sort();
    let mut want = vec![
        "ch01.json:6:0x34397",
        "ch02.json:3:0x3444c",
        "ch05.json:2:0x3464b",
        "ch07.json:2:0x34945",
        "ch12.json:2:0x34c95",
        "ch13.json:2:0x34d91",
    ];
    want.sort();
    assert_eq!(gate_one, want, "gate=1 呼叫");
}

/// Go：`TestChapter13Turn4AppendsOriginalRecord59`
#[test]
fn chapter13_turn4_appends_original_record59() {
    // 獨立 fixture 是 dosgolem 原版第四回合首次新增記錄的 raw checkpoint，
    // 不從 scenario／constructor 的輸出反推預期結果。
    let Some(evidence) = docs_file("data/ida/fd2_ch13_result_conditions_20261001.json") else { return };
    let evidence: serde_json::Value = serde_json::from_slice(&std::fs::read(evidence).unwrap()).unwrap();
    let mut want = Vec::new();
    for sample in evidence["runtime_samples"].as_array().unwrap() {
        if sample["control_seq"] == 2034 {
            for unit in sample["units"].as_array().unwrap() {
                if unit["index"] == 59 {
                    want = ["hp", "x", "y", "fig"].iter().map(|k| unit[k].as_i64().unwrap()).collect();
                }
            }
        }
    }
    assert_eq!(want.len(), 4, "原版第四回合記錄59的 fixture 缺漏");
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map12/map12_units.json", "scenarios/ch13.json") else {
        return;
    };
    bind_native_future_item_rows_for_test(&mut st).unwrap();
    sc.setup_checked(Some(&mut st)).unwrap();
    assert_eq!(st.units.len(), 59, "第四回合前名冊應有59筆");
    (st.turn, st.native_round_counter) = (3, 3);
    assert!(sc.trigger_actions(Some(&st), "on_turn_end", "").is_empty(), "第四回合援軍提前出現");
    (st.turn, st.native_round_counter) = (4, 4);
    let actions = sc.trigger_actions(Some(&st), "on_turn_end", "");
    assert!(
        actions.len() == 8
            && actions[0].r#type == "spawn_group"
            && actions[0].native_event_id == Some(5)
            && actions[0].native_spawns.len() == 1
            && actions[0].native_spawns[0].source == "0x34bee"
            && actions[0].native_spawns[0].raw_placement_gate == Some(0),
        "event5 缺少先生成group1的完整來源：{actions:?}"
    );
    sc.execute_action_checked(Some(&mut st), &actions[0]).unwrap();
    assert_eq!(st.units.len(), 60, "第四回合後名冊應有60筆");
    let u = &st.units[59];
    assert!(
        u.camp == ALLY && u.group == 1 && [u.hp, u.x, u.y, u.fig] == want[..],
        "新增記錄59與原版不符：{u:?}，原版{want:?}"
    );
    for (i, action) in actions[1..].iter().enumerate() {
        assert!(
            action.r#type == "dialogue"
                && action.native_text_index == Some(1)
                && action.native_dialogue_ref.as_ref().is_some_and(|r| r.utterance == i as i64),
            "援軍後第{i}句未按原始文字1順序接線：{action:?}"
        );
    }
    assert!(sc.trigger_actions(Some(&st), "on_turn_end", "").is_empty(), "event5 在同一回合重複觸發");
}

/// Go：`TestExecuteActionCheckedUsesNativeTurnSpawnPlacement`
#[test]
fn execute_action_checked_uses_native_turn_spawn_placement() {
    let active = Unit {
        x: 1,
        y: 1,
        map_selector_key: 2,
        has_map_selector_key: true,
        native_map_presentation: NativeMapPresentationState { x: 1, y: 1, ..Default::default() },
        has_native_map_presentation: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 2,
        has_native_record_byte6: true,
        ..Default::default()
    };
    let pending = Unit {
        group: 6,
        lv: 2,
        map_selector_key: 3,
        has_map_selector_key: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_position_record: NativePositionRecord { x_word: 1, y_word: 1, ..Default::default() },
        has_native_position_record: true,
        native_constructor: Some(NativeConstructorTable {
            branch: "high_class".into(),
            index: 0,
            record: vec![4, 5, 10, 0, 3, 6, 7, 8, 9, 0],
            ..Default::default()
        }),
        inventory: vec![0],
        equipped: vec![true],
        inventory_slots: vec![0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        ..Default::default()
    };
    let mut st =
        State { w: 3, h: 3, roster: vec![pending], native_composition_event_bytes: vec![0; 9], ..Default::default() };
    if bind_native_future_item_rows_for_test(&mut st).is_none() {
        return;
    }
    st.append_native_map_selector_batch(vec![active]).unwrap();
    let mut sc = Scenario { runtime_append_groups: true, ..Default::default() };
    let action = Action {
        r#type: "spawn_group".into(),
        groups: vec![6],
        native_event_id: Some(3),
        native_spawns: vec![NativeSpawnCall {
            group: 6,
            via: "spawn_group".into(),
            source: "0x34397".into(),
            raw_placement_gate: Some(1),
            ..Default::default()
        }],
        ..Default::default()
    };
    sc.execute_action_checked(Some(&mut st), &action).unwrap();
    // Go 另檢查來源名冊記錄沒被改寫（`pending.AP`、`HasNativeRecordRace`）；Rust 的來源列由 roster
    // 擁有、登場時移出，沒有外部指標能看到它，這一項由型別保證。
    assert!(
        st.units.len() == 2 && st.units[1].x == 1 && st.units[1].y == 1,
        "原版 gate=1 增援沒有落在直接座標：units={:?}",
        st.units
    );
}

/// Go：`TestExecuteActionCheckedFailsClosedWithoutRuntimeRoster`
#[test]
fn execute_action_checked_fails_closed_without_runtime_roster() {
    let mut sc = Scenario { runtime_append_groups: true, ..Default::default() };
    let action = Action {
        r#type: "spawn_group".into(),
        groups: vec![6],
        native_event_id: Some(3),
        native_spawns: vec![NativeSpawnCall {
            group: 6,
            via: "spawn_group".into(),
            source: "0x34397".into(),
            raw_placement_gate: Some(1),
            ..Default::default()
        }],
        ..Default::default()
    };
    assert!(
        sc.execute_action_checked(Some(&mut State::default()), &action).is_err(),
        "需要原版名冊的增援在名冊缺失時未採失敗即關閉"
    );
}

/// Go：`TestExecuteActionCheckedFailsClosedBeforeNativeIntroMutation`
#[test]
fn execute_action_checked_fails_closed_before_native_intro_mutation() {
    let mut st = State {
        units: vec![Unit { name: "active".into(), ..Default::default() }],
        roster: vec![Unit { name: "pending".into(), group: 4, ..Default::default() }],
        ..Default::default()
    };
    let mut sc = Scenario { runtime_append_groups: true, ..Default::default() };
    let action = Action {
        r#type: "spawn_group".into(),
        groups: vec![4],
        native_event_id: Some(1),
        native_spawns: vec![NativeSpawnCall {
            group: 4,
            via: "spawn_group_with_intro".into(),
            source: "0x342ce".into(),
            raw_placement_gate: Some(0),
            following_acting: Some(NativeFollowingActing { resource: 3, source: "0x342e7".into() }),
            ..Default::default()
        }],
        ..Default::default()
    };
    assert!(
        sc.execute_action_checked(Some(&mut st), &action).is_err(),
        "不具視覺／後續 acting adapter 的低階 action executor 把 0x32999 當成一般增援執行"
    );
    assert!(
        st.units.len() == 1 && st.units[0].name == "active" && st.roster.len() == 1,
        "失敗前已改變 runtime roster：{:?}",
        st.units
    );
}

/// Go：`TestChapter1SetupMaterializesYuniCommandZero`
#[test]
fn chapter1_setup_materializes_yuni_command_zero() {
    let Some((mut st, mut sc)) = load_map_and_scenario("map0_units.json", "scenarios/ch01.json") else { return };
    sc.setup(Some(&mut st));
    let yuni = st.units.iter().find(|u| u.name == "悠妮").expect("悠妮 was not materialized by ch01 spawn_party");
    assert_eq!(yuni.native_command_ids(), [0], "悠妮 native commands");
}

/// Go：`TestAdoptHandlerBattleStateSkipsRepeatedOpeningAndKeepsTurnGroups`
#[test]
fn adopt_handler_battle_state_skips_repeated_opening_and_keeps_turn_groups() {
    let mut sc = Scenario {
        runtime_append_groups: true,
        events: vec![
            Event {
                trigger: "on_battle_start".into(),
                once: true,
                r#do: vec![
                    Action { r#type: "spawn_party".into(), ..Default::default() },
                    Action { r#type: "dialogue".into(), text: "must not replay".into(), ..Default::default() },
                ],
                ..Default::default()
            },
            Event {
                trigger: "on_turn_end".into(),
                once: true,
                when: Some(When { turn: 3, ..Default::default() }),
                r#do: vec![Action { r#type: "spawn_group".into(), groups: vec![4], ..Default::default() }],
                ..Default::default()
            },
        ],
        ..Default::default()
    };
    let mut st = State {
        units: vec![Unit { on_field: true, ..Default::default() }],
        roster: vec![Unit { group: 4, ..Default::default() }],
        ..Default::default()
    };
    sc.adopt_handler_battle_state(Some(&mut st)).unwrap();
    let got = sc.fire(Some(&mut st), "on_battle_start", "");
    assert!(
        got.is_empty() && st.units.len() == 1,
        "adopted opening replayed: dialogue={got:?} units={}",
        st.units.len()
    );
    assert!(st.pending_groups.get(&4).copied().unwrap_or(false), "pending groups={:?}", st.pending_groups);
    st.turn = 3;
    sc.fire(Some(&mut st), "on_turn_end", "");
    assert!(st.units.len() == 2 && st.units[1].group == 4, "turn event did not retain adopted roster: {:?}", st.units);
}

/// Go：`TestAdoptHandlerBattleStateAppendsUnmaterializedInitialGroups`
#[test]
fn adopt_handler_battle_state_appends_unmaterialized_initial_groups() {
    let mut party =
        Unit { camp: OWN, map_selector_key: 0, has_map_selector_key: true, x: 1, y: 1, ..Default::default() };
    party.materialize_native_map_presentation().unwrap();
    let group0 = Unit {
        camp: ENEMY,
        group: 0,
        map_selector_key: 100,
        has_map_selector_key: true,
        x: 2,
        y: 2,
        ..Default::default()
    };
    let group1 = Unit {
        camp: ENEMY,
        group: 1,
        map_selector_key: 101,
        has_map_selector_key: true,
        x: 3,
        y: 3,
        ..Default::default()
    };
    let mut st = State { roster: vec![group0, group1], ..Default::default() };
    st.append_native_map_selector_batch(vec![party]).unwrap();
    let mut sc = Scenario { runtime_append_groups: true, initial_groups: vec![0, 1], ..Default::default() };
    sc.adopt_handler_battle_state(Some(&mut st)).unwrap();
    // Go 比的是指標；Rust 以 map_selector_key（100／101）辨識同一筆來源。
    assert!(
        st.units.len() == 3
            && st.units[1].map_selector_key == 100
            && st.units[2].map_selector_key == 101
            && st.roster.is_empty(),
        "adopted initial groups units={:?} roster={:?}",
        st.units,
        st.roster
    );
    for (index, unit) in st.units.iter().enumerate() {
        assert!(
            unit.has_map_selector_slot && unit.has_native_map_presentation,
            "adopted unit{index} lacks native selector/presentation: {unit:?}"
        );
    }
}

/// Go：`TestChapter8UsesNativePartyThenGroup0RuntimeOrder`
#[test]
fn chapter8_uses_native_party_then_group0_runtime_order() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map7/map7_units.json", "scenarios/ch08.json") else {
        return;
    };
    bind_native_future_item_rows_for_test(&mut st).unwrap();
    assert!(
        sc.runtime_append_groups && sc.initial_groups == [0],
        "chapter8 constructor policy runtime={} initial={:?}",
        sc.runtime_append_groups,
        sc.initial_groups
    );
    sc.setup(Some(&mut st));
    assert!(
        st.units.len() == 29 && !st.roster.is_empty(),
        "chapter8 opening units={} roster={}",
        st.units.len(),
        st.roster.len()
    );
    for slot in 0..10 {
        assert!(
            st.units[slot].camp == OWN && st.units[slot].has_native_identity,
            "chapter8 party slot{slot}={:?}",
            st.units[slot]
        );
    }
    for slot in 10..29 {
        assert_eq!(st.units[slot].group, 0, "chapter8 group0 slot{slot}={:?}", st.units[slot]);
    }
    let pending = |g: i64, st: &State| st.pending_groups.get(&g).copied().unwrap_or(false);
    for group in 2..=7 {
        assert!(
            pending(group, &st),
            "chapter8 reinforcement group{group} missing from pending={:?}",
            st.pending_groups
        );
    }
    for group in [1, 8, 9, 10] {
        assert!(
            !pending(group, &st),
            "chapter8 group{group} has no proven producer but is pending={:?}",
            st.pending_groups
        );
        for unit in &st.units {
            assert_ne!(unit.group, group, "chapter8 group{group} was materialized without a producer: {unit:?}");
        }
    }

    st.turn = 2;
    let actions = sc.trigger_actions(Some(&st), "on_turn_end", "");
    // event 27 0x349D9：pan (8,2) → delay(100) → 0x10B4E([0x53BEF]) → delay(100)。
    assert_eq!(action_types(&actions), ["pan", "delay", "spawn_group", "delay"], "chapter8 turn2 actions");
    sc.execute_action_checked(Some(&mut st), &actions[2]).unwrap();
    assert!(
        st.units.len() == 31 && st.units[29].group == 2 && st.units[30].group == 2,
        "chapter8 event27 group2 frontier={:?}",
        st.units
    );
}

/// Go：`TestLoadScenarioRejectsMalformedPartyCommandMask`
#[test]
fn load_scenario_rejects_malformed_party_command_mask() {
    let path = write_scenario("event-bad-mask", r#"{"party":[{"name":"bad","initial_command_mask":[1,2,3]}]}"#);
    assert!(load_scenario(&path).is_err(), "malformed party command mask was accepted");
}

/// Go：`TestChapter2RuntimeAppendOrderMatchesOriginalHandlerSlots`
#[test]
fn chapter2_runtime_append_order_matches_original_handler_slots() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map1/map1_units.json", "scenarios/ch02.json") else {
        return;
    };
    bind_native_future_item_rows_for_test(&mut st).unwrap();
    sc.setup(Some(&mut st));
    assert!(
        st.native_map_selector_error.is_none(),
        "native selector construction failed: {:?}",
        st.native_map_selector_error
    );
    for slot in 5..st.units.len() {
        let (key, ok) = st.native_map_sprite_key(Some(slot));
        assert!(ok, "runtime slot {slot} lacks an enabled native map key");
        let u = &st.units[slot];
        assert!(u.has_battle_fig, "runtime slot {slot} lacks FDFIELD b1 battle selector");
        let want = u.battle_fig; // scripted map key and unit+7 both source FDFIELD b1
        assert_eq!(key, want, "runtime slot {slot} native key");
    }
    assert_eq!(st.units.len(), 21, "setup runtime units, want party5 + group1/2=16");
    for (slot, portrait) in [0, 4, 9, 30, 1].into_iter().enumerate() {
        assert!(
            st.units[slot].portrait == portrait && st.units[slot].camp == OWN,
            "party slot {slot} = portrait{} camp{:?}",
            st.units[slot].portrait,
            st.units[slot].camp
        );
    }
    for (slot, portrait) in [134, 133, 134, 133, 134, 133].into_iter().enumerate() {
        let u = &st.units[slot + 5];
        assert!(
            u.portrait == portrait && u.group == 1 && u.camp == ALLY,
            "villager slot {} = portrait{} group{} camp{:?}",
            slot + 5,
            u.portrait,
            u.group,
            u.camp
        );
    }
    assert_eq!(st.pending_count(ENEMY), 6, "pending enemies, want scheduled group3 only");
    st.turn = 3;
    let actions = sc.trigger_actions(Some(&st), "on_turn_end", "");
    assert!(
        actions.len() == 20
            && actions[0].r#type == "pan"
            && actions[0].grid == Some([9, 1])
            && actions[3].r#type == "native_acting"
            && actions[3].native_acting.as_ref().is_some_and(|a| a.resource == 13),
        "turn3缺少來源PAN、ACT及完整20個event6動作"
    );
    for (index, action) in actions.iter().enumerate() {
        assert!(action.native_event_id == Some(6) && action.camp == "ally", "event6動作{index}缺selector1來源");
        if (5..14).contains(&index) {
            assert!(
                action.r#type == "dialogue" && action.native_text_index == Some(4),
                "event6對白{index}缺FDTXT002#4來源"
            );
        }
        // 純規則路徑只執行名冊與raw寫入；鏡頭／演出／對白另由正常Game章收據驗證。
        if action.r#type == "spawn_group" {
            sc.execute_action_checked(Some(&mut st), action).unwrap();
        } else if action.r#type == "native_death_op" {
            // GUI owner沿runNativeDeathOp呼叫同一規則；同步Scenario動作不消費此原語。
            let op = action.native_death_op.as_ref().expect("event6缺少具型別原生寫入");
            st.apply_native_death_op(op).unwrap();
        }
    }
    for slot in 5..=10 {
        let u = &st.units[slot];
        assert!(
            u.native_record_byte34 & 15 == 4 && u.native_record_byte35 == 26 && u.native_record_byte36 == 15,
            "村民{slot}欄位 mode={} x={} y={} has={}/{}/{}",
            u.native_record_byte34,
            u.native_record_byte35,
            u.native_record_byte36,
            u.has_native_record_byte34,
            u.has_native_record_byte35,
            u.has_native_record_byte36
        );
    }
    assert_eq!(st.units.len(), 27, "turn3 exact event6 runtime units");
    // turn_events 的第三個 byte 是 0x1A813 的 phase selector，不是陣營；登場單位
    // 的陣營跟著 FDFIELD record +6（這批是 0＝敵軍）。第四章 event 11 的原版收據
    // （parity-ch04.json，turn_events 同樣標 ally）登場的就是 camp 0。
    for unit in &st.units[21..] {
        assert!(
            unit.group == 3 && unit.camp == ENEMY && unit.native_record_byte6 == 0 && !unit.acted,
            "turn3 exact event6 unit={unit:?}"
        );
        assert!(
            unit.x == (unit.native_position_record.x_word as u8) as i64
                && unit.y == (unit.native_position_record.y_word as u8) as i64,
            "turn3 gate=1 未採原始 position row：unit={unit:?}"
        );
    }
    for slot in 0..st.units.len() {
        assert!(
            st.native_map_sprite_key(Some(slot)).1,
            "post-spawn runtime slot {slot} lacks an enabled native map key"
        );
    }
    let got = st.append_group(4);
    assert!(got == 1 && st.units.len() == 28, "post SPAWN4={got} runtime units={}, want 1/28", st.units.len());
    let hilia = &st.units[27];
    assert!(
        hilia.portrait == 8 && hilia.group == 4 && hilia.x == 22 && hilia.y == 4 && hilia.on_field,
        "post slot27 = {hilia:?}"
    );
    for u in &st.units {
        assert_ne!(u.group, 255, "group255 placeholder polluted canonical runtime slots");
    }
}

/// Go：`TestInitialGroupAbsentPartyConditionControlsOnlyOpeningVisibility`
#[test]
fn initial_group_absent_party_condition_controls_only_opening_visibility() {
    let make_state = || State {
        units: vec![
            Unit { group: 0, on_field: true, ..Default::default() },
            Unit { group: 1, on_field: true, ..Default::default() },
        ],
        ..Default::default()
    };
    let mut sc = Scenario {
        initial_groups: vec![0],
        initial_groups_absent: vec![InitialGroupAbsent { char_id: 18, group: 1 }],
        ..Default::default()
    };
    let mut present = make_state();
    sc.party = vec![PartyMember { fig: 18, ..Default::default() }];
    sc.setup(Some(&mut present));
    assert!(
        present.units[0].on_field && !present.units[1].on_field,
        "present char18 opening groups = {:?}",
        present.units
    );
    let mut absent = make_state();
    sc.party = Vec::new();
    sc.setup(Some(&mut absent));
    assert!(absent.units[0].on_field && absent.units[1].on_field, "absent char18 opening groups = {:?}", absent.units);
}

/// Go：`TestChapter1Turn3JoinsHanoBeforeSpawningHisGroup`
#[test]
fn chapter1_turn3_joins_hano_before_spawning_his_group() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map0/map0_units.json", "scenarios/ch01.json") else {
        return;
    };
    bind_native_future_item_rows_for_test(&mut st).unwrap();
    sc.setup(Some(&mut st));
    st.turn = 3;
    sc.fire(Some(&mut st), "on_turn_end", "");
    let joins = sc.take_party_joins();
    assert_eq!(joins, [1], "turn3 joins, want Hano char1");
    let hano = st.units.iter().rev().find(|u| u.fig == 1);
    let hawat = st.units.iter().rev().find(|u| u.fig == 3);
    assert!(hano.is_some_and(|u| u.on_field && u.camp == OWN), "Hano spawn = {hano:?}, want recruited OWN unit");
    // 哈瓦特是我方，不是 allied NPC：map0 建構資料明列 group7 的 raw `+6 = 2`
    // 且 camp 為 own，ch01 的 spawn_group 也寫 `"camp": "own"`。舊斷言來自手寫
    // 的 `camp: ally`，與原版建構資料矛盾。
    assert!(hawat.is_some_and(|u| u.on_field && u.camp == OWN), "Hawat spawn = {hawat:?}, want OWN unit");
    let got = sc.take_party_joins();
    assert!(got.is_empty(), "party joins were not consumed: {got:?}");
}

/// Go：`TestChapter3RuntimeAppendOrderMatchesPreHandlerSlots`
#[test]
fn chapter3_runtime_append_order_matches_pre_handler_slots() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map2/map2_units.json", "scenarios/ch03.json") else {
        return;
    };
    sc.setup(Some(&mut st));
    assert_eq!(st.units.len(), 15, "chapter3 initial slots, want six party + nine group1 records");
    for (slot, id) in [0, 4, 9, 30, 1, 8].into_iter().enumerate() {
        assert_eq!(st.units[slot].fig, id, "authored party slot{slot} fig before JOIN-order adapter");
    }
    assert!(st.units[6].fig == 2 && st.units[6].camp == ALLY, "chapter3 slot6 = {:?}, want Tino ally", st.units[6]);
    for slot in 7..=14 {
        assert_eq!(st.units[slot].camp, ENEMY, "chapter3 slot{slot} camp, want enemy");
    }
    for unit in &st.units {
        assert_ne!(unit.group, 255, "group255 source padding polluted runtime: {unit:?}");
    }
}

/// Go：`TestChapter3Turn3ReinforcementRequiresLivingTinoInRuntimeSlot6`
#[test]
fn chapter3_turn3_reinforcement_requires_living_tino_in_runtime_slot6() {
    let load_ch3 = || -> Option<(State, Scenario)> {
        let (mut st, mut sc) = load_map_and_scenario("maps/map2/map2_units.json", "scenarios/ch03.json")?;
        bind_native_future_item_rows_for_test(&mut st)?;
        sc.setup(Some(&mut st));
        st.turn = 3;
        Some((st, sc))
    };

    let Some((mut dead, mut dead_scenario)) = load_ch3() else { return };
    dead.units[6].hp = 0;
    let dead_dialogues = dead_scenario.fire(Some(&mut dead), "on_turn_end", "");
    assert_eq!(dead.units.len(), 15, "dead Tino spawned group2: runtime units, want 15");
    assert!(dead_dialogues.is_empty(), "dead Tino played living-only #4 dialogue: {dead_dialogues:?}");

    let Some((mut alive, mut alive_scenario)) = load_ch3() else { return };
    let alive_dialogues = alive_scenario.fire(Some(&mut alive), "on_turn_end", "");
    assert_eq!(alive.units.len(), 27, "living Tino runtime units, want 15+12 group2");
    assert!(
        alive_dialogues.len() == 7
            && alive_dialogues[0].speaker == 77
            && alive_dialogues[1].speaker == 2
            && alive_dialogues[6].speaker == 77,
        "turn3 FDTXT_003 #4 dialogues = {alive_dialogues:?}"
    );
    assert_eq!(
        alive_dialogues[1].text,
        "如果不是這些年輕人幫忙的話,我早就沒命了!不過既然我還活著,我還是要問你一個問題:到底是誰命令你來殺我?",
        "turn3 Tino line drifted"
    );
}

/// Go：`TestChapter3Turn3TriggerPreservesOriginalStagingOrder`
#[test]
fn chapter3_turn3_trigger_preserves_original_staging_order() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map2/map2_units.json", "scenarios/ch03.json") else {
        return;
    };
    sc.setup(Some(&mut st));
    st.turn = 3;
    let actions = sc.trigger_actions(Some(&st), "on_turn_end", "");
    let want = [
        "spawn_group",
        "pan",
        "delay",
        "pan",
        "delay",
        "dialogue",
        "dialogue",
        "dialogue",
        "dialogue",
        "dialogue",
        "dialogue",
        "dialogue",
    ];
    assert_eq!(action_types(&actions), want, "turn3 actions");
    assert!(
        actions[1].grid == Some([3, 0])
            && actions[2].ms == 800
            && actions[3].grid == Some([3, 17])
            && actions[4].ms == 200,
        "turn3 PAN/delay staging drifted: {:?}",
        &actions[1..5]
    );
    let again = sc.trigger_actions(Some(&st), "on_turn_end", "");
    assert!(again.is_empty(), "once event triggered twice: {again:?}");
}

/// Go：`TestChapter9Event31SpawnsGroupFromStateWhenControlRowIsLive`
///
/// 第九章：開場只有隊伍＋group 0（0x1088D 只呼叫 0x10B4E(0)）；事件 31 在 map 8 控制列是休眠列，
/// 由事件 30 的 control_turn 啟用後才在 selector 0 分派，登場的 group 讀 state16（事件 30 寫 2，
/// 所以第一次是 group 2），接著 state16 加一。
#[test]
fn chapter9_event31_spawns_group_from_state_when_control_row_is_live() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map8/map8_units.json", "scenarios/ch09.json") else {
        return;
    };
    bind_native_future_item_rows_for_test(&mut st).unwrap();
    assert!(
        sc.runtime_append_groups && sc.initial_groups == [0],
        "chapter9 constructor policy runtime={} initial={:?}",
        sc.runtime_append_groups,
        sc.initial_groups
    );
    sc.setup(Some(&mut st));
    assert!(
        st.has_native_turn_event_control_state
            && st.native_turn_event_controls[0] == NativeTurnEventControl { turn: 0xff, event_id: 31, raw_camp: 0 },
        "map8 control rows={:?} has={}",
        &st.native_turn_event_controls[..2],
        st.has_native_turn_event_control_state
    );
    let got = sc.native_turn_action_events_at(Some(&st), 0);
    assert!(got.is_empty(), "休眠列不該分派：{got:?}");
    let opening = st.units.len();
    st.native_event_state[0x10] = 2;
    st.native_turn_event_controls[0].turn = st.native_round_counter as u8;
    let events = sc.native_turn_action_events_at(Some(&st), 0);
    assert!(events.len() == 1 && events[0].event_id == 31, "啟用後應分派事件 31：{events:?}");
    let actions = &events[0].actions;
    assert!(
        actions[0].r#type == "spawn_group" && actions[1].r#type == "native_death_op",
        "事件 31 動作順序={actions:?}"
    );
    sc.execute_action_checked(Some(&mut st), &actions[0]).unwrap();
    assert!(
        st.units.len() != opening && st.units[opening].group == 2,
        "state16=2 應登場 group 2：units {opening}→{}",
        st.units.len()
    );
}

/// Go：`TestNativeResultCode1ChecksListedRecordsByte5Bit0`
#[test]
fn native_result_code1_checks_listed_records_byte5_bit0() {
    // 0x20707：0x3453E(0x32)／0x3453E(0x33) 任一非 0 就寫 [0x53ECC]=1。
    let sc = Scenario {
        native_result_handler: "0x20707".into(),
        native_result_code1_records: vec![2, 3, 9],
        ..Default::default()
    };
    let mut st = State {
        units: vec![
            Unit::default(),
            Unit::default(),
            Unit { has_native_record_byte5: true, ..Default::default() },
            Unit { has_native_record_byte5: true, native_record_byte5: 0x80, ..Default::default() },
        ],
        ..Default::default()
    };
    assert!(!sc.native_result_code1(Some(&st)), "bit0 都沒成立卻判成結果碼 1");
    st.units[3].native_record_byte5 = 0x81;
    assert!(sc.native_result_code1(Some(&st)), "記錄 3 的 +5 bit0 成立沒有判成結果碼 1");
    assert!(!Scenario::default().native_result_code1(Some(&st)), "沒有列記錄的章節不該覆寫結果");
}

/// Go：`TestChapter12ProtectsNativeRecord14WhilePartySurvives`
#[test]
fn chapter12_protects_native_record14_while_party_survives() {
    let Some((mut st, mut sc)) = load_map_and_scenario("maps/map11/map11_units.json", "scenarios/ch12.json") else {
        return;
    };
    sc.setup_checked(Some(&mut st)).unwrap();
    assert!(
        st.units.len() > 14 && st.units[14].fig == 17 && st.units[14].camp == ALLY,
        "chapter12 protected record has wrong construction order: {:?}",
        st.units
    );
    st.units[14].native_record_byte5 = 0x80;
    assert!(!sc.native_result_code1(Some(&st)), "acted bit7 caused defeat while the protected ally remains active");
    st.units[14].native_record_byte5 = 0x81;
    st.units[14].hp = 0;
    assert!(
        st.units[0].alive() && sc.native_result_code1(Some(&st)),
        "protected record bit0 did not cause defeat while the party survives"
    );
}
