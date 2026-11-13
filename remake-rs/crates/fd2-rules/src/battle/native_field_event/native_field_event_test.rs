//! 移植自 `remake/internal/battle/native_field_event_test.go`。
//!
//! Go 測試常把不在 `st.Units` 的觸發者 `&Unit{...}` 傳進去；Rust 依 `battle` 模組說明第 6 點，
//! 先 push 到 `units` 尾端再傳索引。被測函式只在規則列出的記錄範圍內掃描單位，多出的一筆
//! 不在範圍內。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

fn load_map(rel: &str) -> Option<State> {
    let path = remake_asset(rel)?;
    Some(load(&path).unwrap())
}

/// 把 Go 測試裡的外部觸發者放進 `units`，回傳索引。
fn push_trigger(st: &mut State, trigger: Unit) -> UnitIdx {
    st.units.push(trigger);
    st.units.len() - 1
}

/// Go：`TestNativeFieldEventIDAtMatchesSelector`
#[test]
fn native_field_event_id_at_matches_selector() {
    let mut st = State {
        w: 2,
        h: 2,
        native_field_event_slots: vec![-1, 3, -1, -1],
        native_field_events: vec![NativeFieldEvent::default(); 16],
        ..Default::default()
    };
    st.native_field_events[3] = NativeFieldEvent { event_id: 82, selector: 1 };
    let got = native_field_event_id_at(Some(&st), 1, 0, 1);
    assert_eq!(got, (82, true), "event, want (82,true)");
    assert!(!native_field_event_id_at(Some(&st), 1, 0, 0).1, "selector mismatch unexpectedly accepted");
}

/// Go：`TestNativeFieldEventIDAtFailsClosed`
#[test]
fn native_field_event_id_at_fails_closed() {
    let mut st = State { w: 1, h: 1, native_field_event_slots: vec![0], ..Default::default() };
    assert!(!native_field_event_id_at(Some(&st), 0, 0, 0).1, "missing table unexpectedly accepted");
    st.native_field_events = vec![NativeFieldEvent::default(); 16];
    st.native_field_events[0] = NativeFieldEvent { event_id: 0xff, ..Default::default() };
    assert!(!native_field_event_id_at(Some(&st), 0, 0, 0).1, "0xff event unexpectedly accepted");
}

/// Go：`TestAllEditableMapsCarryNativeRendererInputs`
///
/// Go `filepath.Glob("maps/map*/map*_units.json")`；Rust 列出 `maps/map*` 目錄後找同名檔。
#[test]
fn all_editable_maps_carry_native_renderer_inputs() {
    let Some(maps) = remake_asset("maps") else { return };
    let mut paths = Vec::new();
    for entry in std::fs::read_dir(&maps).unwrap() {
        let dir = entry.unwrap().path();
        let name = dir.file_name().unwrap().to_string_lossy().into_owned();
        if !dir.is_dir() || !name.starts_with("map") {
            continue;
        }
        for file in std::fs::read_dir(&dir).unwrap() {
            let file = file.unwrap().path();
            let fname = file.file_name().unwrap().to_string_lossy().into_owned();
            if fname.starts_with("map") && fname.ends_with("_units.json") {
                paths.push(file);
            }
        }
    }
    paths.sort();
    assert_eq!(paths.len(), 33, "map unit assets");
    for path in &paths {
        let path = path.to_string_lossy();
        let st = load(&path).unwrap_or_else(|err| panic!("{path}: {err}"));
        assert!(
            st.native_tile_blit_modes.len() as i64 == st.w * st.h
                && !st.native_terrain_control.is_empty()
                && st.native_terrain_control.len().is_multiple_of(4)
                && st.has_native_turn_event_control_state,
            "{path}: renderer modes={} cells={} controls={} turn-controls={}",
            st.native_tile_blit_modes.len(),
            st.w * st.h,
            st.native_terrain_control.len(),
            st.has_native_turn_event_control_state
        );
        assert_eq!(st.native_round_counter, 1, "{path}: native round seed, want 1");
    }
}

/// Go：`TestNativeTurnEventCatalogRejectsTampering`
#[test]
fn native_turn_event_catalog_rejects_tampering() {
    let Some(path) = remake_asset("maps/native_turn_event_controls.json") else { return };
    let raw = std::fs::read_to_string(path).unwrap();
    for (name, old, new) in [
        ("round writer", r#""writer": "0x2066e""#, r#""writer": "0x2066f""#),
        ("control row", r#""turn": 255"#, r#""turn": 254"#),
        ("map identity", r#""map": 0"#, r#""map": 1"#),
    ] {
        let corrupt = raw.replacen(old, new, 1);
        assert_ne!(corrupt, raw, "{name}: {old:?} was not found in the catalog fixture");
        let map_dir = temp_dir("field-catalog").join("maps").join("map26");
        std::fs::create_dir_all(&map_dir).unwrap();
        std::fs::write(map_dir.parent().unwrap().join("native_turn_event_controls.json"), corrupt).unwrap();
        let (controls, seed, ok) = load_native_turn_event_controls(&map_dir.join("map.json").to_string_lossy(), 29, 20);
        assert!(
            !ok && seed == 0 && controls == [NativeTurnEventControl::default(); 16],
            "{name}: tampered catalog accepted: ok={ok} seed={seed} controls={controls:?}"
        );
    }
}

/// Go：`TestMap25LoadsEditableNativeFieldEventRules`
#[test]
fn map25_loads_editable_native_field_event_rules() {
    let Some(st) = load_map("maps/map25/map25_units.json") else { return };
    assert_eq!(st.native_field_event_rules.len(), 3, "field event rules");
    let got = &st.native_field_event_rules[0];
    assert!(
        got.event_id == 59
            && got.selector == 0
            && got.trigger_gate == "record_byte6_nonzero"
            && got.set_mode_ranges == [NativeFieldModeRange { start: 39, end: 44, mode: 0 }],
        "event59 rule={got:?}"
    );
    let got61 = &st.native_field_event_rules[2];
    assert!(
        got61.event_id == 61
            && got61.selector == 1
            && got61.once_state == Some(12)
            && got61.required_item == Some(0xD0)
            && got61.spawn_group == Some(1)
            && got61.join_character == Some(31)
            && got61.text_indices == Some(NativeFieldTextIndices { missing_item: 2, success: 3, r#final: 4 })
            && got61.presentation
                == Some(NativeFieldPresentation {
                    archive: "FDOTHER.DAT".into(),
                    resource: 45,
                    frames: 59,
                    helper: "0x2935b".into(),
                    destination_offset: 48356,
                    stride: 320,
                    transparent: -1,
                    delay_helper: "0x17aa9".into(),
                    delay_ticks: 2,
                }),
        "event61 rule={got61:?}"
    );
}

/// Go `map26Event62Cell`
fn map26_event62_cell(st: &State) -> (i64, i64) {
    for (cell, &slot) in st.native_field_event_slots.iter().enumerate() {
        if slot < 0 || slot >= st.native_field_events.len() as i64 {
            continue;
        }
        let event = st.native_field_events[slot as usize];
        if event.event_id == 62 && event.selector == 0 {
            return (cell as i64 % st.w, cell as i64 / st.w);
        }
    }
    panic!("map26 has no event62 selector0 cell");
}

/// Go：`TestMap26LoadsDormantTurnRowsAndActivatesEvent63`
#[test]
fn map26_loads_dormant_turn_rows_and_activates_event63() {
    let Some(mut st) = load_map("maps/map26/map26_units.json") else { return };
    assert!(st.has_native_turn_event_control_state, "complete native turn controls were not loaded");
    assert_eq!(
        st.native_turn_event_controls[0],
        NativeTurnEventControl { turn: 0xff, event_id: 63, raw_camp: 0 },
        "map26 dormant row0"
    );
    assert_eq!(
        st.native_turn_event_controls[1],
        NativeTurnEventControl { turn: 0xff, event_id: 65, raw_camp: 0 },
        "map26 dormant row1"
    );
    assert!(
        st.native_field_event_rules.len() == 1 && st.native_field_event_rules[0].turn_activation.is_some(),
        "map26 event rules={:?}",
        st.native_field_event_rules
    );
    let (x, y) = map26_event62_cell(&st);
    st.native_round_counter = 8;
    let got = apply_native_field_turn_activation_event(Some(&mut st), x, y, 0);
    assert!(matches!(got, Ok(62)), "event62 activation={got:?}");
    assert!(
        st.native_turn_event_controls[0] == NativeTurnEventControl { turn: 9, event_id: 63, raw_camp: 0 }
            && st.native_event_state[17] == 1,
        "activated row={:?} state17={}",
        st.native_turn_event_controls[0],
        st.native_event_state[17]
    );
    let before = st.native_turn_event_controls;
    let before_raw = st.native_field_control_raw.clone();
    for once in [1u8, 2, 255] {
        st.native_event_state[17] = once;
        let id = apply_native_field_turn_activation_event(Some(&mut st), x, y, 0);
        assert!(
            matches!(id, Ok(62))
                && st.native_turn_event_controls == before
                && st.native_field_control_raw == before_raw
                && st.native_event_state[17] == once,
            "repeated event62 must return without mutation: once={once} id={id:?} rows={:?}",
            st.native_turn_event_controls
        );
    }
}

/// Go：`TestEvent62RawDisagreementFailsAtomically`
#[test]
fn event62_raw_disagreement_fails_atomically() {
    let Some(mut st) = load_map("maps/map26/map26_units.json") else { return };
    let (x, y) = map26_event62_cell(&st);
    st.native_round_counter = 3;
    st.has_native_field_control_state = true;
    st.native_field_control_raw = vec![0; 6];
    st.native_field_control_raw[3] = 0; // typed row says 0xff
    st.native_field_control_raw[4] = 63;
    let before_rows = st.native_turn_event_controls;
    let before_raw = st.native_field_control_raw.clone();
    assert!(
        apply_native_field_turn_activation_event(Some(&mut st), x, y, 0).is_err(),
        "disagreeing raw row unexpectedly activated"
    );
    assert!(
        st.native_turn_event_controls == before_rows
            && st.native_field_control_raw == before_raw
            && st.native_event_state[17] == 0,
        "failed event62 activation partially mutated state"
    );
}

/// Go：`TestChapter28FieldActivationsPreserveGuardsAndRawRows`
#[test]
fn chapter28_field_activations_preserve_guards_and_raw_rows() {
    for (name, x, y, slot, event, raw6, state18, want_activate) in [
        ("first line accepts enemy", 30, 23, 0usize, 65u8, 0u8, 0u8, true),
        ("third wave waits for second", 20, 20, 2, 69, 1, 0, false),
        ("third wave ignores enemy", 20, 20, 2, 69, 0, 1, false),
        ("third wave accepts own", 20, 20, 2, 69, 1, 1, true),
    ] {
        let Some(mut st) = load_map("maps/map27/map27_units.json") else { return };
        st.native_round_counter = 7;
        st.native_event_state[18] = state18;
        let trigger = push_trigger(
            &mut st,
            Unit { x, y, native_record_byte6: raw6, has_native_record_byte6: true, ..Default::default() },
        );
        let before = st.native_turn_event_controls;
        let id = apply_native_chapter28_field_turn_activation(Some(&mut st), Some(trigger), x, y, 0);
        assert!(matches!(id, Ok(got) if got == event), "{name}: id={id:?}");
        let mut want = before;
        if want_activate {
            want[slot].turn = 7;
        }
        assert_eq!(st.native_turn_event_controls, want, "{name}: rows");
        let state_index = if event == 69 { 17 } else { 16 };
        assert_eq!(st.native_event_state[state_index] != 0, want_activate, "{name}: state guard changed incorrectly");
        st.native_round_counter = 8;
        let again = apply_native_chapter28_field_turn_activation(Some(&mut st), Some(trigger), x, y, 0);
        assert!(
            again.is_ok() && st.native_turn_event_controls == want,
            "{name}: repeat rescheduled a completed event {again:?}"
        );
    }
}

/// Go：`TestChapter28FieldActivationRejectsWrongRawRowAtomically`
#[test]
fn chapter28_field_activation_rejects_wrong_raw_row_atomically() {
    for x in [30i64, 20] {
        let Some(mut st) = load_map("maps/map27/map27_units.json") else { return };
        let (mut y, mut slot) = (23, 0);
        if x == 20 {
            (y, slot) = (20, 2);
            st.native_event_state[18] = 1;
        }
        st.native_round_counter = 7;
        st.has_native_field_control_state = true;
        st.native_field_control_raw = vec![0; 51];
        let (before_rows, before_state) = (st.native_turn_event_controls, st.native_event_state);
        let before_raw = st.native_field_control_raw.clone();
        let trigger = push_trigger(
            &mut st,
            Unit { x, y, native_record_byte6: 1, has_native_record_byte6: true, ..Default::default() },
        );
        assert!(
            apply_native_chapter28_field_turn_activation(Some(&mut st), Some(trigger), x, y, 0).is_err(),
            "row{slot} mismatch accepted"
        );
        assert!(
            st.native_turn_event_controls == before_rows
                && st.native_event_state == before_state
                && st.native_field_control_raw == before_raw,
            "failed activation partially published"
        );
    }
}

fn chapter29_setup() -> Option<State> {
    let mut st = load_map("maps/map28/map28_units.json")?;
    let path = remake_asset("scenarios/ch29.json")?;
    let mut sc = load_scenario(&path).unwrap();
    sc.setup_checked(Some(&mut st)).unwrap();
    Some(st)
}

/// Go：`TestChapter29Event75PlansAndCommitsExactTurnChain`
#[test]
fn chapter29_event75_plans_and_commits_exact_turn_chain() {
    let Some(mut st) = chapter29_setup() else { return };
    let trigger = push_trigger(
        &mut st,
        Unit {
            x: 15,
            y: 21,
            has_native_record_byte6: true,
            native_record_byte6: 1,
            has_native_record_byte8: true,
            native_record_byte8: 9,
            ..Default::default()
        },
    );
    st.native_round_counter = 8;
    let before_rows = st.native_turn_event_controls;
    let before_raw = st.native_field_control_raw.clone();
    let plan = plan_native_field_event75(Some(&st), Some(trigger), 15, 21).unwrap();
    assert!(
        plan.activate
            && !plan.noop
            && plan.text_index == 1
            && st.native_turn_event_controls == before_rows
            && st.native_field_control_raw == before_raw,
        "event75 plan={plan:?} mutated state"
    );
    commit_native_field_event75(Some(&mut st), &plan).unwrap();
    assert!(
        st.native_event_state[17] == 1
            && st.native_event_state[16] == 4
            && st.native_turn_event_controls[1] == NativeTurnEventControl { turn: 9, event_id: 76, raw_camp: 2 }
            && st.native_turn_event_controls[0] == NativeTurnEventControl { turn: 8, event_id: 74, raw_camp: 0 },
        "event75 committed state16={} state17={} rows={:?}",
        st.native_event_state[16],
        st.native_event_state[17],
        &st.native_turn_event_controls[..2]
    );
}

/// Go：`TestChapter29Event75MismatchAndMissingProvenanceDoNotActivate`
#[test]
fn chapter29_event75_mismatch_and_missing_provenance_do_not_activate() {
    let Some(mut st) = chapter29_setup() else { return };
    let trigger = push_trigger(
        &mut st,
        Unit {
            x: 15,
            y: 21,
            has_native_record_byte6: true,
            native_record_byte6: 1,
            has_native_record_byte8: true,
            native_record_byte8: 8,
            ..Default::default()
        },
    );
    let plan = plan_native_field_event75(Some(&st), Some(trigger), 15, 21);
    assert!(plan.as_ref().is_ok_and(|p| !p.activate && !p.noop && p.text_index == 0), "event75 mismatch plan={plan:?}");
    st.units[trigger].has_native_record_byte8 = false;
    assert!(
        plan_native_field_event75(Some(&st), Some(trigger), 15, 21).is_err(),
        "event75 accepted missing raw +8 provenance"
    );
}

fn chapter26_setup() -> Option<State> {
    let mut st = load_map("maps/map25/map25_units.json")?;
    let path = remake_asset("scenarios/ch26.json")?;
    let mut sc = load_scenario(&path).unwrap();
    assert!(
        sc.runtime_append_groups && sc.initial_groups == [0],
        "ch26 constructor policy: runtime={} initial={:?}",
        sc.runtime_append_groups,
        sc.initial_groups
    );
    sc.setup(Some(&mut st));
    Some(st)
}

fn wold_counts(st: &State, require_on_field: bool) -> (usize, usize) {
    let active = st.units.iter().filter(|u| u.group == 1 && u.fig == 31 && (!require_on_field || u.on_field)).count();
    let pending = st.roster.iter().filter(|u| u.group == 1 && u.fig == 31).count();
    (active, pending)
}

/// Go：`TestChapter26KeepsWoldPendingUntilEvent61`
#[test]
fn chapter26_keeps_wold_pending_until_event61() {
    let Some(st) = chapter26_setup() else { return };
    let (active, pending) = wold_counts(&st, false);
    assert!(active == 0 && pending == 1, "event61 前渥德 active={active} pending={pending}");
}

/// Go `setupChapter26Event61`：觸發者是 `st.units[0]`。
fn setup_chapter26_event61(items: &[i64]) -> Option<State> {
    let mut st = chapter26_setup()?;
    let trigger = &mut st.units[0];
    (trigger.x, trigger.y) = (1, 46);
    trigger.inventory = items.to_vec();
    trigger.equipped = vec![false; items.len()];
    trigger.inventory_slots = vec![0xff; 8];
    trigger.native_inventory_flags = vec![0x80; 8];
    for (i, &item) in items.iter().enumerate() {
        trigger.inventory_slots[i] = item;
        trigger.native_inventory_flags[i] = 0;
    }
    Some(st)
}

/// Go：`TestEvent61MissingItemPlansOnlyOriginalText`
#[test]
fn event61_missing_item_plans_only_original_text() {
    let Some(mut st) = setup_chapter26_event61(&[0x20]) else { return };
    let plan = plan_native_field_event61(Some(&st), Some(0), 1, 46).unwrap();
    assert!(plan.missing_item && plan.text_index == 2 && plan.final_text == 4, "missing-item plan={plan:?}");
    assert!(
        st.native_event_state[12] == 0 && !st.roster.is_empty() && st.units[0].inventory == [0x20],
        "missing-item planning mutated battle state"
    );
    assert!(commit_native_field_event61(Some(&mut st), &plan, 59).is_err(), "missing-item plan unexpectedly committed");
}

/// Go：`TestEvent61CommitsAfterAllFramesAndReturnsJoin`
#[test]
fn event61_commits_after_all_frames_and_returns_join() {
    let Some(mut st) = setup_chapter26_event61(&[0xD0, 0x20]) else { return };
    let plan = plan_native_field_event61(Some(&st), Some(0), 1, 46).unwrap();
    assert!(
        !plan.missing_item && plan.text_index == 3 && plan.final_text == 4 && plan.presentation.frames == 59,
        "success plan={plan:?}"
    );
    assert!(
        commit_native_field_event61(Some(&mut st), &plan, 58).is_err(),
        "incomplete presentation unexpectedly committed"
    );
    assert!(
        st.native_event_state[12] == 0 && st.units[0].inventory[0] == 0xD0,
        "failed commit partially mutated state"
    );
    let joined = commit_native_field_event61(Some(&mut st), &plan, 59).unwrap();
    let trigger = &st.units[0];
    assert!(
        joined == 31
            && st.native_event_state[12] == 1
            && trigger.inventory == [0x20]
            && trigger.inventory_slots[0] == 0x20
            && trigger.inventory_slots[1] == 0xff,
        "event61 commit join={joined} state={} inventory={:?} slots={:?}",
        st.native_event_state[12],
        trigger.inventory,
        trigger.inventory_slots
    );
    let (active, pending) = wold_counts(&st, true);
    assert!(active == 1 && pending == 0, "event61 後渥德 active={active} pending={pending}");
}

/// Go：`TestEvent61RevalidatesInventoryBeforeCommit`
#[test]
fn event61_revalidates_inventory_before_commit() {
    let Some(mut st) = setup_chapter26_event61(&[0xD0]) else { return };
    let plan = plan_native_field_event61(Some(&st), Some(0), 1, 46).unwrap();
    (st.units[0].inventory[0], st.units[0].inventory_slots[0]) = (0x20, 0x20);
    assert!(commit_native_field_event61(Some(&mut st), &plan, 59).is_err(), "changed inventory unexpectedly committed");
    assert!(st.native_event_state[12] == 0 && !st.roster.is_empty(), "revalidation failure partially mutated state");
}

/// Go：`TestMap25Event59AppliesModeRangeAtomically`
#[test]
fn map25_event59_applies_mode_range_atomically() {
    let Some(mut st) = load_map("maps/map25/map25_units.json") else { return };
    for u in &mut st.units[39..=44] {
        u.native_record_byte34 |= 0xA0;
        u.has_native_record_byte34 = true;
    }
    let trigger =
        push_trigger(&mut st, Unit { native_record_byte6: 1, has_native_record_byte6: true, ..Default::default() });
    let got = apply_native_field_mode_event(Some(&mut st), Some(trigger), 10, 36, 0);
    assert_eq!(got, (59, true), "event59");
    for index in 39..=44 {
        assert_eq!(st.units[index].native_record_byte34, 0xA0, "unit{index} byte34, want 0xa0");
    }
}

/// Go：`TestNativeFieldModeEventFailsClosedBeforePartialWrite`
#[test]
fn native_field_mode_event_fails_closed_before_partial_write() {
    let Some(mut st) = load_map("maps/map25/map25_units.json") else { return };
    let before = st.units[23].native_record_byte34;
    st.units[56].has_native_record_byte34 = false;
    let trigger =
        push_trigger(&mut st, Unit { native_record_byte6: 1, has_native_record_byte6: true, ..Default::default() });
    assert!(
        !apply_native_field_mode_event(Some(&mut st), Some(trigger), 10, 22, 0).1,
        "incomplete event60 target provenance unexpectedly accepted"
    );
    assert_eq!(st.units[23].native_record_byte34, before, "event60 partially mutated before validation completed");
}
