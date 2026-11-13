//! 移植自 `remake/internal/battle/native_ch30_chain_test.go`。

use super::*;
use crate::test_support::remake_asset;

/// Go `native_result_test.go` 的 `nativeResultRecords`（同組另一個測試檔的輔助函式，在這裡照抄一份）。
fn native_result_records(count: usize) -> State {
    let mut st = State { native_round_counter: 1, ..Default::default() };
    for _ in 0..count {
        st.units.push(Unit {
            hp: 999,
            has_native_record_byte5: true,
            has_native_record_byte6: true,
            native_record_byte6: 2,
            ..Default::default()
        });
    }
    st
}

/// Go：`TestChapter30ResultRequiresBossFlagAndDefeatOverridesWin`
#[test]
fn chapter30_result_requires_boss_flag_and_defeat_overrides_win() {
    let Some(path) = remake_asset("scenarios/ch30.json") else { return };
    let sc = load_scenario(&path).unwrap();
    for (boss, leader, yuni, want) in [(0u8, 0u8, 0u8, 0i64), (0x80, 0, 0, 0), (1, 0, 0, 2), (1, 1, 0, 1), (1, 0, 1, 1)]
    {
        let mut st = native_result_records(33);
        st.has_native_turn_event_control_state = true;
        st.units[20].native_record_byte5 = boss;
        st.units[0].native_record_byte5 = leader;
        st.units[1].native_record_byte5 = yuni;
        // HP與其他敵人是否在場都不能取代record20的raw+5。
        st.units[20].hp = 0;
        let mut code = native_initial_result_code(Some(&st), Some(&sc)).unwrap();
        for rule in &sc.native_result_rules {
            if rule.r#match(Some(&st)).unwrap() {
                code = rule.code;
            }
        }
        assert_eq!(code, want, "flags={:?} result", (boss, leader, yuni));
        st.units[20].has_native_record_byte5 = false;
        assert!(sc.native_result_rules[0].r#match(Some(&st)).is_err(), "缺boss raw旗標仍猜測勝利");
    }
    assert!(
        sc.native_result_rules[2].actions.len() == 1
            && sc.native_result_rules[2].actions[0].native_text_index == Some(7),
        "悠妮敗北文字庫索引錯誤"
    );
}

/// Go：`TestChapter30DormantEvent82RejectsInvalidProgress`
#[test]
fn chapter30_dormant_event82_rejects_invalid_progress() {
    let Some(path) = remake_asset("scenarios/ch30.json") else { return };
    let sc = load_scenario(&path).unwrap();
    for phase in [0u8, 1, 2, 3, 4, 5] {
        let mut st = State { has_native_turn_event_control_state: true, native_round_counter: 2, ..Default::default() };
        st.native_turn_event_controls[0] = NativeTurnEventControl { turn: 2, event_id: 82, raw_camp: 0 };
        st.native_event_state[16] = phase;
        let events = sc.native_turn_events_at(Some(&st), 0);
        if phase == 0 || phase == 5 {
            assert!(events.is_err(), "progress{phase}沒有拒收");
            continue;
        }
        let events = events.unwrap_or_else(|err| panic!("progress{phase}: {err}"));
        assert_eq!(events.len(), 1, "progress{phase}");
        let mut placements = Vec::new();
        let mut groups = Vec::new();
        for action in &events[0].actions {
            // Go 的 `(*NativeActionWhen)(nil).Match` 恆為 true。
            let active = action.native_when.as_ref().map_or(Ok(true), |w| w.r#match(Some(&st))).unwrap();
            if !active {
                continue;
            }
            if let Some(p) = action.native_staging_present {
                placements.push(p);
            }
            groups.extend(action.groups.iter().copied());
        }
        let p = phase as i64;
        assert!(
            !placements.is_empty() && placements[0] == NativeStagingPlacement { slot: 24 - p, x: 22, y: 18 },
            "progress{phase}未保留原始魔神slot與目的座標: {placements:?}"
        );
        if phase < 4 {
            assert!(
                groups == [p]
                    && placements.len() == 3
                    && placements[1] == NativeStagingPlacement { slot: 25 + 2 * p, x: 21, y: 18 }
                    && placements[2] == NativeStagingPlacement { slot: 26 + 2 * p, x: 23, y: 18 },
                "progress{phase}增援順序不符: {groups:?} {placements:?}"
            );
        } else {
            assert!(groups.is_empty() && placements.len() == 1, "最後phase不應追加增援");
        }
    }
}

/// Go：`TestChapter30Field80AlwaysGateStillRequiresSource`
///
/// Go 的觸發者 `&Unit{}` 不在 `st.Units`；Rust 先把它放進 `units[21]` 再傳索引。模式範圍只到
/// 記錄 20，多出的一筆不在掃描範圍內。
#[test]
fn chapter30_field80_always_gate_still_requires_source() {
    for missing in [false, true] {
        let mut st = State {
            w: 1,
            h: 1,
            native_field_event_slots: vec![0],
            native_field_events: vec![NativeFieldEvent::default(); 16],
            ..Default::default()
        };
        st.native_field_events[0] = NativeFieldEvent { event_id: 80, selector: 1 };
        st.native_field_event_rules = vec![NativeFieldEventRule {
            event_id: 80,
            handler: "0x35f5a".into(),
            selector: 1,
            trigger_gate: "always".into(),
            set_mode_ranges: vec![NativeFieldModeRange { start: 20, end: 20, mode: 11 }],
            ..Default::default()
        }];
        st.units = vec![Unit::default(); 21];
        st.units[20] = Unit { native_record_byte34: 0xa8, has_native_record_byte34: !missing, ..Default::default() };
        st.units.push(Unit::default());
        // 35F5A不讀取actor raw+6，不能套其他格子事件的陣營gate。
        let (_, applied) = apply_native_field_mode_event(Some(&mut st), Some(21), 0, 0, 1);
        if missing {
            assert!(!applied && st.units[20].native_record_byte34 == 0xa8, "缺來源仍發布模式");
        } else {
            assert!(applied && st.units[20].native_record_byte34 == 0xab, "未保留mode高位或誤套actor gate");
        }
    }
}
