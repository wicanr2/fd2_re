//! 移植自 `remake/internal/battle/native_turn_event_test.go`。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

fn write_native_turn_scenario(tag: &str, raw: &str) -> String {
    let path = temp_dir(tag).join("scenario.json");
    std::fs::write(&path, raw).unwrap();
    path.to_string_lossy().into_owned()
}

fn pending_set(st: &State) -> BTreeMap<i64, bool> {
    st.pending_groups.clone()
}

/// Go：`TestLoadScenarioValidatesNativeTurnStagingAndPendingGroups`
#[test]
fn load_scenario_validates_native_turn_staging_and_pending_groups() {
    let path = write_native_turn_scenario(
        "turn-event-staging",
        r#"{
  "runtime_append_groups": true,
  "initial_groups": [0],
  "native_turn_events": [{
    "event_id": 63, "raw_camp": 0, "handler": "0x358c7",
    "staging": {
      "helper": "0x35822", "pan_helper": "0x135dd", "spawn_helper": "0x10b4e",
      "delay_before_flash_ms": 300,
      "palette_helper": "0x11df2", "palette_start": 0, "palette_end": 255,
      "flash_delta": 255, "flash_hold_ms": 200, "restore_delta": 0,
      "redraw_helper": "0x11cac", "raw_placement_gate": 0,
      "calls": [
        {"group": 1, "x": 3, "y": 27, "source": "0x358d7"},
        {"group": 2, "x": 15, "y": 27, "source": "0x358e5"}
      ]
    }
  }]
}"#,
    );
    let sc = load_scenario(&path).unwrap();
    let mut st = State::default();
    sc.materialize_pending_groups(&mut st);
    assert_eq!(pending_set(&st), BTreeMap::from([(1, true), (2, true)]), "pending groups");

    // Go 以 map 逐一 t.Run；子測試互不相依，Rust 依名稱排序跑完。
    let invalid_fixtures = [
        (
            "duplicate tuple",
            r#"{"runtime_append_groups":true,"native_turn_events":[{"event_id":63,"raw_camp":0,"handler":"0x358c7","staging":{"helper":"0x35822","pan_helper":"0x135dd","spawn_helper":"0x10b4e","palette_helper":"0x11df2","palette_end":255,"flash_delta":255,"redraw_helper":"0x11cac","calls":[{"group":1,"x":3,"y":27,"source":"0x358d7"}]}},{"event_id":63,"raw_camp":0,"handler":"0x358c7","staging":{"helper":"0x35822","pan_helper":"0x135dd","spawn_helper":"0x10b4e","palette_helper":"0x11df2","palette_end":255,"flash_delta":255,"redraw_helper":"0x11cac","calls":[{"group":2,"x":15,"y":27,"source":"0x358e5"}]}}]}"#,
        ),
        (
            "initial group",
            r#"{"runtime_append_groups":true,"initial_groups":[1],"native_turn_events":[{"event_id":63,"raw_camp":0,"handler":"0x358c7","staging":{"helper":"0x35822","pan_helper":"0x135dd","spawn_helper":"0x10b4e","palette_helper":"0x11df2","palette_end":255,"flash_delta":255,"redraw_helper":"0x11cac","calls":[{"group":1,"x":3,"y":27,"source":"0x358d7"}]}}]}"#,
        ),
    ];
    for (name, invalid) in invalid_fixtures {
        let path = write_native_turn_scenario("turn-event-invalid", invalid);
        assert!(load_scenario(&path).is_err(), "{name}: malformed native turn event accepted");
    }
}

/// Go：`TestNativeTurnEventsAtPreservesRawRowOrderAndFailsClosed`
#[test]
fn native_turn_events_at_preserves_raw_row_order_and_fails_closed() {
    let rule63 = NativeTurnEvent { event_id: 63, raw_camp: 0, handler: "0x358c7".into(), ..Default::default() };
    let rule64 = NativeTurnEvent { event_id: 64, raw_camp: 0, handler: "0x35900".into(), ..Default::default() };
    let sc = Scenario { native_turn_events: vec![rule64.clone(), rule63.clone()], ..Default::default() };
    let mut st = State { native_round_counter: 9, has_native_turn_event_control_state: true, ..Default::default() };
    for row in &mut st.native_turn_event_controls {
        row.turn = 0xff;
    }
    st.native_turn_event_controls[0] = NativeTurnEventControl { turn: 9, event_id: 63, raw_camp: 0 };
    st.native_turn_event_controls[2] = NativeTurnEventControl { turn: 9, event_id: 64, raw_camp: 0 };
    let got = sc.native_turn_events_at(Some(&st), 0).unwrap();
    assert_eq!(got, [rule63, rule64], "native turn order");
    st.native_turn_event_controls[1] = NativeTurnEventControl { turn: 9, event_id: 65, raw_camp: 0 };
    assert!(sc.native_turn_events_at(Some(&st), 0).is_err(), "unknown live event row was silently skipped");
}

/// Go：`TestChapter27KeepsDeferredGroupsOutOfInitialRoster`
#[test]
fn chapter27_keeps_deferred_groups_out_of_initial_roster() {
    let Some(path) = remake_asset("scenarios/ch27.json") else { return };
    let sc = load_scenario(&path).unwrap();
    assert!(
        sc.runtime_append_groups && sc.initial_groups == [0] && sc.native_turn_events.len() == 1,
        "chapter27 staging boundary={sc:?}"
    );
    let mut st = State::default();
    sc.materialize_pending_groups(&mut st);
    assert_eq!(pending_set(&st), BTreeMap::from([(1, true), (2, true)]), "chapter27 pending groups");
}

/// Go：`TestChapter29KeepsDynamicEvent74GroupsPending`
#[test]
fn chapter29_keeps_dynamic_event74_groups_pending() {
    let Some(path) = remake_asset("scenarios/ch29.json") else { return };
    let sc = load_scenario(&path).unwrap();
    assert!(
        sc.runtime_append_groups
            && sc.initial_groups == [8]
            && sc.native_field_event_rules.len() == 1
            && sc.native_turn_events.len() == 3
            && sc.native_turn_events[0].dynamic_group.is_some(),
        "chapter29 event boundary={sc:?}"
    );
    let mut st = State::default();
    sc.materialize_pending_groups(&mut st);
    assert_eq!(
        pending_set(&st),
        BTreeMap::from([(1, true), (4, true), (5, true), (6, true), (7, true)]),
        "chapter29 pending groups"
    );
}

fn chapter29_setup() -> Option<(State, Scenario)> {
    let map = remake_asset("maps/map28/map28_units.json")?;
    let scenario = remake_asset("scenarios/ch29.json")?;
    let mut st = load(&map).unwrap();
    let mut sc = load_scenario(&scenario).unwrap();
    sc.setup_checked(Some(&mut st)).unwrap();
    Some((st, sc))
}

/// Go：`TestEvent76RepeatCommitsRawBitStateAndNextRowAtomically`
#[test]
fn event76_repeat_commits_raw_bit_state_and_next_row_atomically() {
    let Some((mut st, sc)) = chapter29_setup() else { return };
    let event = sc.native_turn_events[1].clone();
    st.native_round_counter = 9;
    st.native_event_state[17] = 1;
    st.native_turn_event_controls[1] = NativeTurnEventControl { turn: 9, event_id: 76, raw_camp: 2 };
    assert!(st.units.len() >= 2 && st.units[1].has_native_record_byte5, "chapter29 slot1 lacks raw +5 provenance");
    let before = st.units[1].native_record_byte5;
    let plan = plan_native_turn_event76(Some(&st), &event);
    assert!(plan.as_ref().is_ok_and(|p| !p.r#final), "event76 repeat plan={plan:?}");
    commit_native_turn_event76_repeat(Some(&mut st), &plan.unwrap()).unwrap();
    assert!(
        st.units[1].native_record_byte5 == before | 0x80
            && st.native_event_state[17] == 2
            && st.native_turn_event_controls[1] == NativeTurnEventControl { turn: 10, event_id: 76, raw_camp: 2 },
        "event76 repeat byte5={:#x} state17={} row={:?}",
        st.units[1].native_record_byte5,
        st.native_event_state[17],
        st.native_turn_event_controls[1]
    );
}

/// Go：`TestEvent76RepeatMissingRawProvenanceFailsWithoutMutation`
#[test]
fn event76_repeat_missing_raw_provenance_fails_without_mutation() {
    let Some((mut st, sc)) = chapter29_setup() else { return };
    let event = sc.native_turn_events[1].clone();
    st.native_round_counter = 9;
    st.native_event_state[17] = 1;
    st.native_turn_event_controls[1] = NativeTurnEventControl { turn: 9, event_id: 76, raw_camp: 2 };
    st.units[1].has_native_record_byte5 = false;
    let (before_state, before_rows) = (st.native_event_state, st.native_turn_event_controls);
    assert!(plan_native_turn_event76(Some(&st), &event).is_err(), "event76 accepted missing slot1 raw +5 provenance");
    assert!(
        st.native_event_state == before_state && st.native_turn_event_controls == before_rows,
        "event76 rejected plan partially mutated state"
    );
}
