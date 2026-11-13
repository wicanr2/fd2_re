//! 移植自 `remake/internal/battle/native_result_test.go`。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

fn chapter_scenario(name: &str) -> Option<Scenario> {
    let path = remake_asset(&format!("scenarios/{name}"))?;
    Some(load_scenario(&path).unwrap())
}

/// Go `chapter13ResultScenario`
fn chapter13_result_scenario() -> Option<Scenario> {
    chapter_scenario("ch13.json")
}

/// Go `nativeResultRecords`
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

/// 依 Go 測試的寫法：先取初始碼，再讓每一條命中的規則覆寫。
fn resolve_code(st: &State, sc: &Scenario) -> i64 {
    let mut code = native_initial_result_code(Some(st), Some(sc)).unwrap();
    for rule in &sc.native_result_rules {
        if rule.r#match(Some(st)).unwrap() {
            code = rule.code;
        }
    }
    code
}

/// Go：`TestChapter29RequiresAllDragonDeathFlagsInsteadOfEnemyHP`
#[test]
fn chapter29_requires_all_dragon_death_flags_instead_of_enemy_hp() {
    let Some(sc) = chapter_scenario("ch29.json") else { return };
    for mask in 0..8 {
        let mut st = native_result_records(93);
        st.has_native_turn_event_control_state = true;
        for u in &mut st.units[20..] {
            u.hp = 0;
            u.native_record_byte6 = 0;
            u.native_record_byte5 = 1;
        }
        for bit in 0..3 {
            st.native_event_state[18 + bit] = ((mask >> bit) & 1) as u8;
        }
        let want = if mask == 7 { 2 } else { 0 };
        assert_eq!(resolve_code(&st, &sc), want, "flags={mask} result");
    }
}

/// Go：`TestChapter29DefeatOverridesDragonVictoryAndRequiresRawSource`
#[test]
fn chapter29_defeat_overrides_dragon_victory_and_requires_raw_source() {
    let Some(sc) = chapter_scenario("ch29.json") else { return };
    let mut st = native_result_records(93);
    st.has_native_turn_event_control_state = true;
    (st.native_event_state[18], st.native_event_state[19], st.native_event_state[20]) = (1, 1, 1);
    for index in [0, 1] {
        st.units[index].native_record_byte5 = 1;
        assert_eq!(resolve_code(&st, &sc), 1, "record{index} inactive result");
        st.units[index].native_record_byte5 = 0;
    }
    assert!(
        sc.native_result_rules[2].actions.len() == 1
            && sc.native_result_rules[2].actions[0].native_text_index == Some(9),
        "悠妮敗北未保留string9"
    );
    st.units[1].has_native_record_byte5 = false;
    assert!(sc.native_result_rules[2].r#match(Some(&st)).is_err(), "缺raw +5卻猜測結果");
    st.has_native_turn_event_control_state = false;
    assert!(sc.native_result_rules[0].r#match(Some(&st)).is_err(), "缺事件狀態來源卻猜測勝利");
}

/// Go：`TestChapter29RejectsInvalidEventStateResultRules`
#[test]
fn chapter29_rejects_invalid_event_state_result_rules() {
    let rule = |states: Vec<i64>, records: Vec<i64>| NativeResultRule {
        id: "bad".into(),
        source: "0x20b93".into(),
        code: 2,
        event_states_all_nonzero: states,
        records_all_inactive: records,
        ..Default::default()
    };
    let state_len = State::default().native_event_state.len() as i64;
    for r in
        [rule(vec![18, 18], vec![]), rule(vec![-1], vec![]), rule(vec![state_len], vec![]), rule(vec![18], vec![0])]
    {
        assert!(r.validate().is_err(), "不合法事件狀態規則被接受");
    }
}

/// Go：`TestChapter26OnlyRequiredInactiveRecordsOverrideResult`
#[test]
fn chapter26_only_required_inactive_records_override_result() {
    let Some(sc) = chapter_scenario("ch26.json") else { return };
    assert!(
        sc.native_result_handler == "0x20b3c" && sc.native_result_rules.len() == 2,
        "第26章原生結果來源或兩筆必存角色規則缺漏"
    );
    for (index, flags, want) in [(1usize, 0x80u8, 0), (2, 0x80, 0), (3, 1, 0), (1, 1, 1), (2, 0x81, 1)] {
        let mut st = native_result_records(57);
        st.units[index].native_record_byte5 = flags;
        let matched = sc.native_result_rules.iter().filter(|rule| rule.r#match(Some(&st)).unwrap()).count();
        assert_eq!(matched, want, "記錄{index} flags={flags:02x}命中列數");
    }
}

/// Go：`TestNativeDefaultResultUsesRawCampAndBit0Priority`
#[test]
fn native_default_result_uses_raw_camp_and_bit0_priority() {
    let mut st = native_result_records(2);
    st.units[1].native_record_byte6 = 0;
    (st.units[1].camp, st.units[1].hp) = (OWN, 0); // 正規化Camp／HP不可替代原始讀取。
    st.roster = vec![Unit { native_record_byte6: 0, hp: 999, ..Default::default() }];
    for (own, enemy, want) in [(0u8, 0u8, 0i64), (0, 0x80, 0), (0, 1, 2), (1, 0, 1), (1, 1, 1)] {
        (st.units[0].native_record_byte5, st.units[1].native_record_byte5) = (own, enemy);
        let code = native_default_result_code(Some(&st));
        assert!(matches!(code, Ok(c) if c == want), "raw旗標{own}/{enemy}結果={code:?}，預期{want}");
    }
    st.units[1].has_native_record_byte6 = false;
    assert!(native_default_result_code(Some(&st)).is_err(), "原始camp缺漏卻推測結果");
}

/// Go：`TestChapter13AllInitialAlliesUsesEveryRawRecord`
#[test]
fn chapter13_all_initial_allies_uses_every_raw_record() {
    let Some(sc) = chapter13_result_scenario() else { return };
    let rule = &sc.native_result_rules[0];
    let mut st = native_result_records(59);
    for u in &mut st.units[15..=26] {
        u.native_record_byte5 = 0x81;
    }
    assert!(matches!(rule.r#match(Some(&st)), Ok(true)), "全部bit0為1未命中");
    (st.units[20].hp, st.units[20].native_record_byte5) = (0, 0x80);
    assert!(matches!(rule.r#match(Some(&st)), Ok(false)), "HP0／bit7被誤判成bit0");
    st.units[26].has_native_record_byte5 = false;
    assert!(rule.r#match(Some(&st)).is_err(), "發現一名活動者後漏查後方原始欄位");
}

/// Go：`TestChapter13RoundGatePrecedesUncreatedRecord59`
#[test]
fn chapter13_round_gate_precedes_uncreated_record59() {
    let Some(sc) = chapter13_result_scenario() else { return };
    let rule = &sc.native_result_rules[1];
    let mut st = native_result_records(59);
    st.native_round_counter = 5;
    assert!(matches!(rule.r#match(Some(&st)), Ok(false)), "回合5提前讀取59");
    st.native_round_counter = 6;
    assert!(rule.r#match(Some(&st)).is_err(), "回合6缺記錄59卻猜成存活");
    st.units.push(Unit { has_native_record_byte5: true, native_record_byte5: 1, hp: 999, ..Default::default() });
    assert!(matches!(rule.r#match(Some(&st)), Ok(true)), "回合6的bit0未命中");
    st.native_round_counter = 0;
    assert!(rule.r#match(Some(&st)).is_err(), "原生回合缺漏卻借用正規化回合");
}

/// Go：`TestLoadScenarioRejectsUnknownOrIncompleteNativeResults`
///
/// Go 以 `json.Marshal(sc)` 寫回再載入；Rust 用 `Scenario` 的 serde 輸出（欄位名同 Go 標籤）。
#[test]
fn load_scenario_rejects_unknown_or_incomplete_native_results() {
    let mutations: [fn(&mut Scenario); 5] = [
        |sc| sc.native_result_rules[0].code = 2,
        |sc| sc.native_result_rules[0].records_all_inactive = Vec::new(),
        |sc| sc.native_result_rules[0].actions[0].native_dialogue_ref = None,
        |sc| sc.native_result_rules[1].id = sc.native_result_rules[0].id.clone(),
        |sc| sc.native_result_code1_records = vec![59],
    ];
    for mutate in mutations {
        let Some(mut sc) = chapter13_result_scenario() else { return };
        mutate(&mut sc);
        let raw = serde_json::to_vec(&sc).unwrap();
        let path = temp_dir("result-reject").join("scenario.json");
        std::fs::write(&path, raw).unwrap();
        assert!(load_scenario(&path.to_string_lossy()).is_err(), "不完整或未閉合的結果列被正式載入");
    }
}

/// Go：`TestChapter15ResultUsesOnlyRecord64RawInactiveBit`
#[test]
fn chapter15_result_uses_only_record64_raw_inactive_bit() {
    let Some(sc) = chapter_scenario("ch15.json") else { return };
    assert!(sc.native_result_handler == "0x20822" && sc.native_result_rules.len() == 1, "第十五章結果來源或規則缺漏");
    let rule = &sc.native_result_rules[0];
    assert!(rule.actions.is_empty() && rule.round_greater_than.is_none(), "無對白原版處理器被新增對白或回合閘門");
    let mut st = native_result_records(74);
    st.units[65].native_record_byte5 = 1;
    st.units[64].hp = 0;
    st.units[64].native_record_byte5 = 0x80;
    assert!(matches!(rule.r#match(Some(&st)), Ok(false)), "其他友軍倒下、HP0或bit7被當成record64 bit0");
    (st.units[64].hp, st.units[64].native_record_byte5) = (999, 0x81);
    assert!(matches!(rule.r#match(Some(&st)), Ok(true)), "原始bit0未使存活我方的戰場判敗");
    st.units[64].has_native_record_byte5 = false;
    assert!(rule.r#match(Some(&st)).is_err(), "record64缺原始欄位卻猜補結果");
}
