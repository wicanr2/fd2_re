//! 移植自 `remake/internal/battle/snapshot_test.go`。

use super::*;

/// Go 欄位名與 Rust 欄位名的比較鍵：去掉底線後轉小寫（`NativeMapHUDState` ↔ `native_map_hud_state`）。
fn field_key(name: &str) -> String {
    name.replace('_', "").to_lowercase()
}

/// `model.rs` 裡 `pub struct State` 的欄位名（Rust 沒有反射，從原始碼讀）。
fn state_fields() -> Vec<String> {
    let src = include_str!("../model.rs");
    let start = src.find("\npub struct State {\n").expect("model.rs 有 State 定義");
    let body = &src[start..];
    let end = body.find("\n}\n").expect("State 定義有結尾");
    body[..end]
        .lines()
        .filter_map(|line| {
            let line = line.trim_start();
            let rest = line.strip_prefix("pub(crate) ").or_else(|| line.strip_prefix("pub "))?;
            let (name, _) = rest.split_once(':')?;
            name.chars().all(|c| c.is_ascii_alphanumeric() || c == '_').then(|| name.to_string())
        })
        .collect()
}

/// Go：`TestStateSnapshotClassifiesEveryField`
///
/// 每個 State 欄位都要明確分類；新增欄位卻沒想過讀檔怎麼辦，在這裡失敗。Go 以反射列舉欄位；
/// Rust 讀 `model.rs` 的 `State` 定義，欄位名以 `field_key` 對應 Go 名稱。
#[test]
fn state_snapshot_classifies_every_field() {
    let fields = state_fields();
    assert!(fields.len() > 40, "State 欄位解析失敗：{fields:?}");
    let classified: BTreeSet<String> = STATE_SNAPSHOT_FIELD_CLASS.keys().map(|k| field_key(k)).collect();
    let seen: BTreeSet<String> = fields.iter().map(|f| field_key(f)).collect();
    let mut errors = Vec::new();
    for name in &fields {
        if !classified.contains(&field_key(name)) {
            errors.push(format!("State.{name} 沒有快照分類（saved／bound／transient）"));
        }
    }
    for name in STATE_SNAPSHOT_FIELD_CLASS.keys() {
        if !seen.contains(&field_key(name)) {
            errors.push(format!("快照分類列了不存在的 State.{name}"));
        }
    }
    assert!(errors.is_empty(), "{errors:#?}");
}

/// Go：`TestStateSnapshotRoundTripPreservesSavedFields`
///
/// saved 類欄位逐一在 StateSnapshot 有對應；以 JSON 往返後再還原，與原值深度相等。Go 的 Units 含一個
/// nil 元素（`[]*Unit{u1, nil, u2}`）；Rust 的 `Vec<Unit>` 沒有 nil，改用兩個單位。
#[test]
fn state_snapshot_round_trip_preserves_saved_fields() {
    let mut cache = fdicon::NativeSelectorCache::default();
    for key in [7, 3, 9] {
        cache.slot_for(key).unwrap();
    }
    let pending_unit = Unit { name: "r1".into(), hp: 9, native_constructor_pending: true, ..Default::default() };
    let mut src = State {
        w: 2,
        h: 2,
        units: vec![
            Unit { name: "u1".into(), hp: 5, acted: true, ..Default::default() },
            Unit { name: "u2".into(), x: 1, y: 1, ..Default::default() },
        ],
        roster: vec![pending_unit],
        native_map_selector_cache: Some(cache),
        native_map_selector_error: Some(BattleError::new("selector rejected")),
        has_native_map_cycle_state: true,
        native_map_cycle_state: fdicon::NativeMapSpriteCycleState { idle: 2, moving: 1, last_timer_tick: 40 },
        native_map_view_state: NativeMapViewState { camera_x: 1, cursor_x: 1, ..Default::default() },
        has_native_map_view_state: true,
        native_map_range_mode: 1,
        has_native_map_range_mode_state: true,
        pending_groups: BTreeMap::from([(3, true)]),
        own_deploy: vec![Cell { x: 1, y: 0 }],
        turn: 4,
        native_round_counter: 3,
        flags: BTreeMap::from([("x".to_string(), true)]),
        cost: Some(vec![1, 2, 3, 4]),
        native_composition_event_bytes: vec![0, 1, 2, 3],
        native_map_event_grid: vec![2, 0, 2, 0, 1, 2, 3, 4],
        has_native_map_event_grid: true,
        native_field_event_slots: vec![-1, 0, -1, 1],
        native_tile_blit_modes: vec![0, 0xff, 0, 1],
        spell_book: vec![Spell { id: 1, ..Default::default() }],
        ai_command_spell: BTreeMap::from([(4, 1)]),
        treasures: Some(BTreeMap::from([
            (Cell { x: 1, y: 1 }, Treasure { slot: 2, kind: "item".into(), value: 5, ..Default::default() }),
            (Cell { x: 0, y: 1 }, Treasure { slot: 1, ..Default::default() }),
        ])),
        opened_treasure: Some(BTreeMap::from([(1, true)])),
        ..Default::default()
    };
    src.native_event_state[3] = 9;
    src.native_turn_event_controls[2] = NativeTurnEventControl { turn: 3, event_id: 5, ..Default::default() };
    src.has_native_turn_event_control_state = true;

    let snap = src.snapshot().unwrap();
    let raw = serde_json::to_vec(&snap).unwrap();
    let decoded: StateSnapshot = serde_json::from_slice(&raw).unwrap();
    let mut dst = State { flags: BTreeMap::from([("stale".to_string(), true)]), ..Default::default() };
    dst.restore_snapshot(&decoded).unwrap();

    macro_rules! same {
        ($($field:ident),* $(,)?) => {
            $(assert!(src.$field == dst.$field, "State.{} 往返後不同", stringify!($field));)*
        };
    }
    same!(
        w,
        h,
        units,
        native_map_cycle_state,
        has_native_map_cycle_state,
        native_terrain_phase_state,
        has_native_terrain_phase_state,
        native_terrain_flip_state,
        native_unit_pixel_shift_state,
        has_native_map_binary_timing_state,
        native_map_hud_state,
        has_native_map_hud_state,
        native_map_view_state,
        has_native_map_view_state,
        native_map_range_mode,
        has_native_map_range_mode_state,
        roster,
        pending_groups,
        own_deploy,
        turn,
        native_round_counter,
        flags,
        native_event_state,
        cost,
        native_composition_event_bytes,
        native_map_event_grid,
        has_native_map_event_grid,
        native_field_control_raw,
        native_turn_event_controls,
        has_native_turn_event_control_state,
        native_chest_controls,
        has_native_chest_control_state,
        native_field_unit_controls,
        has_native_field_control_state,
        native_runtime_records,
        has_native_runtime_unit_projection,
        has_native_pending_group_binding,
        native_field_event_slots,
        native_field_events,
        native_tile_blit_modes,
        native_terrain_control,
        native_terrain_move_codes,
        spell_book,
        ai_command_spell,
        treasures,
        opened_treasure,
    );
    assert!(
        dst.native_map_selector_error.as_ref().map(|e| e.to_string()) == Some("selector rejected".to_string()),
        "NativeMapSelectorError = {:?}",
        dst.native_map_selector_error
    );
    let dst_cache = dst.native_map_selector_cache.as_mut().unwrap();
    assert_eq!(dst_cache.keys(), src.native_map_selector_cache.as_ref().unwrap().keys(), "selector keys");
    for (key, want) in [(7, 0), (3, 1), (9, 2)] {
        assert_eq!(dst_cache.slot_for(key).unwrap(), want, "selector slot {key}");
    }
    assert!(
        dst.roster[0].native_constructor_pending && !dst.units[0].native_constructor_pending,
        "nativeConstructorPending 沒有保存"
    );
}

/// Go：`TestStateSnapshotRejectsTransientAndBadInput`
///
/// Go 另測「Units 與 Roster 共用同一個 `*Unit`」會被拒絕；Rust 的兩個陣列各自擁有單位，這種狀態
/// 無法建構，該分支不移植。
#[test]
fn state_snapshot_rejects_transient_and_bad_input() {
    let st = State { w: 1, h: 1, native_experience_accumulator: 3, ..Default::default() };
    assert!(st.snapshot().is_err(), "行動經驗未結算時不應受理快照");
    let mut st = State { w: 1, h: 1, ..Default::default() };
    st.native_ai_scan.active = true;
    assert!(st.snapshot().is_err(), "敵方 AI 掃描中不應受理快照");
    let mut good = State { w: 1, h: 1, turn: 2, ..Default::default() };
    let base = StateSnapshot { version: STATE_SNAPSHOT_VERSION, w: 1, h: 1, ..Default::default() };
    for (name, snap) in [
        ("version", StateSnapshot { version: 99, ..base.clone() }),
        ("size", StateSnapshot { version: STATE_SNAPSHOT_VERSION, ..Default::default() }),
        ("cost", StateSnapshot { cost: Some(vec![1, 2]), ..base.clone() }),
        (
            "dup",
            StateSnapshot { has_native_map_selector_cache: true, native_map_selector_keys: vec![1, 1], ..base.clone() },
        ),
    ] {
        assert!(good.restore_snapshot(&snap).is_err(), "{name}: 應拒絕");
        assert!(good.turn == 2 && good.w == 1, "{name}: 拒絕後 State 被改動");
    }
}

/// Go：`TestScenarioFiredEventsRoundTrip`
#[test]
fn scenario_fired_events_round_trip() {
    let events = || {
        vec![
            Event { id: "a".into(), once: true, ..Default::default() },
            Event { id: String::new(), once: true, ..Default::default() },
            Event { id: "c".into(), ..Default::default() },
        ]
    };
    let mut sc = Scenario { events: events(), ..Default::default() };
    (sc.events[0].fired, sc.events[1].fired) = (true, true);
    let refs = sc.fired_events();
    let mut other = Scenario { events: events(), ..Default::default() };
    other.events[2].fired = true;
    other.restore_fired_events(&refs).unwrap();
    assert!(
        other.events[0].fired && other.events[1].fired && !other.events[2].fired,
        "fired = {} {} {}",
        other.events[0].fired,
        other.events[1].fired,
        other.events[2].fired
    );
    assert!(other.restore_fired_events(&[FiredEventRef { index: 0, id: "b".into() }]).is_err(), "ID 不符應拒絕");
    assert!(other.events[0].fired, "拒絕後劇本被改動");
}
