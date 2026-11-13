//! 移植自 `remake/internal/battle/native_death_program_test.go`。

use super::*;
use crate::test_support::{docs_file, remake_asset};

fn death_test_state(n: usize) -> State {
    let mut st = State { w: 24, h: 24, ..Default::default() };
    for _ in 0..n {
        st.units.push(Unit {
            camp: ENEMY,
            hp: 20,
            max_hp: 20,
            on_field: true,
            native_record_byte5: 0,
            has_native_record_byte5: true,
            native_record_byte6: 0,
            has_native_record_byte6: true,
            native_record_byte34: 0xa3,
            has_native_record_byte34: true,
            ..Default::default()
        });
    }
    st
}

fn op(name: &str) -> NativeDeathOp {
    NativeDeathOp { op: name.into(), ..Default::default() }
}

/// Go：`TestNativeDeathOpsFollowTranscription`
///
/// 各原語對照轉寫的語意：0x3419C 只動低四位、0x34A85 整個覆寫、0x34A0E AND、
/// 0x32975 整個 +5 寫 1、[0x53AD5] 設值與遞增、[0x51A83]=1。
#[test]
fn native_death_ops_follow_transcription() {
    let mut st = death_test_state(8);
    let mut must = |st: &mut State, o: NativeDeathOp| {
        if let Err(err) = st.apply_native_death_op(&o) {
            panic!("{}: {err}", o.op);
        }
    };
    must(&mut st, NativeDeathOp { first: 1, last: 2, mode: 7, ..op("ai_mode_range") });
    let b34 = |st: &State, i: usize| st.units[i].native_record_byte34;
    assert!(
        b34(&st, 1) == 0xa7 && b34(&st, 2) == 0xa7 && b34(&st, 0) == 0xa3 && b34(&st, 3) == 0xa3,
        "ai_mode_range 應只改 1..2 的低四位：{:#x} {:#x} {:#x} {:#x}",
        b34(&st, 0),
        b34(&st, 1),
        b34(&st, 2),
        b34(&st, 3)
    );
    must(&mut st, NativeDeathOp { first: 3, last: 20, value: 0, ..op("ai_byte_set_range") });
    assert!(b34(&st, 3) == 0 && b34(&st, 7) == 0, "ai_byte_set_range 應整個覆寫，超出現存單位的索引忽略");
    must(&mut st, NativeDeathOp { first: 0, last: 0, mask: 0x80, ..op("ai_byte_and_range") });
    assert!(b34(&st, 0) == 0x80, "ai_byte_and_range：{:#x}", b34(&st, 0));
    must(&mut st, NativeDeathOp { unit: 4, ..op("mark_inactive") });
    assert!(!st.units[4].on_field && st.units[4].native_record_byte5 == 1, "mark_inactive 應讓單位離場並把 +5 寫成 1");
    must(&mut st, NativeDeathOp { index: 0x10, value: 1, ..op("state_set") });
    must(&mut st, NativeDeathOp { index: 0x10, ..op("state_inc") });
    assert!(st.native_event_state[0x10] == 2, "狀態表 0x10 = {}，應為 2", st.native_event_state[0x10]);
    must(&mut st, op("range_one"));
    assert!(st.native_map_range_mode == 1 && st.has_native_map_range_mode_state, "range_one 應把地圖範圍選擇值設回 1");
    must(&mut st, op("range_zero"));
    assert!(st.native_map_range_mode == 0 && st.has_native_map_range_mode_state, "range_zero 應把地圖範圍選擇值關成 0");
    assert!(st.apply_native_death_op(&op("staging")).is_err(), "staging 由介面擁有者執行，狀態層不該默默接受");
    // 還沒建構的記錄：原版照寫 raw 陣列，之後建構器整筆覆寫，所以是無作用。
    if let Err(err) =
        st.apply_native_death_op(&NativeDeathOp { unit: 99, writes: vec![[6, 1, 1]], ..op("record_bytes") })
    {
        panic!("寫還沒建構的記錄應為無作用：{err}");
    }
    assert!(st.apply_native_death_op(&NativeDeathOp { unit: -1, ..op("record_bytes") }).is_err(), "負索引應該失敗");
}

/// Go：`TestNativeDeathRecordWritesReviveAsAlly`
///
/// 事件 30：萊汀倒下時被改寫成友軍身分 6、HP 1，死亡效果清掉，下一次也不會再觸發。
#[test]
fn native_death_record_writes_revive_as_ally() {
    let mut st = death_test_state(12);
    {
        let u = &mut st.units[11];
        (u.hp, u.native_record_byte5) = (0, 1);
        (u.battle_fig, u.has_battle_fig) = (118, true);
        (u.native_record_byte8, u.has_native_record_byte8) = (118, true);
        u.death_effect = Some(DeathEffect { r#type: 2, value: 30 });
        (u.native_record_death_effect, u.has_native_record_death_effect) = ([2, 30, 0], true);
    }
    st.apply_native_death_op(&NativeDeathOp {
        unit: 11,
        writes: vec![[5, 0, 1], [6, 1, 1], [7, 6, 1], [8, 6, 1], [0x31, 0xff, 1], [0x34, 0x80, 1], [0x40, 1, 2]],
        ..op("record_bytes")
    })
    .unwrap();
    let u = &st.units[11];
    assert!(
        !(u.hp != 1
            || u.native_record_byte5 != 0
            || !u.on_field
            || u.camp != ALLY
            || u.native_record_byte6 != 1
            || u.battle_fig != 6
            || u.native_record_byte8 != 6
            || u.native_record_byte34 != 0x80),
        "復活後 {u:?}"
    );
    let (_, _, ok) = native_death_effect_of(Some(u));
    assert!(!ok && u.death_effect.is_none(), "+0x31=0xFF 之後不應再有死亡效果");
    assert!(
        st.apply_native_death_op(&NativeDeathOp { unit: 11, writes: vec![[0x20, 1, 1]], ..op("record_bytes") })
            .is_err(),
        "沒有型別化欄位的 offset 應整筆拒絕"
    );
}

/// Go：`TestNativeDeathDestinationPairKeepsProjectionAndAtomicRejection`
///
/// event13的+0x35整byte寫入必須同時更新typed/raw；未知欄位仍整筆拒絕。
#[test]
fn native_death_destination_pair_keeps_projection_and_atomic_rejection() {
    let mut st = death_test_state(2);
    st.has_native_runtime_unit_projection = true;
    st.native_runtime_records = vec![NativeRuntimeRecordState::default(); 2];
    for i in 0..st.units.len() {
        let u = &mut st.units[i];
        (u.native_record_byte35, u.has_native_record_byte35) = (17, true);
        (u.native_record_byte36, u.has_native_record_byte36) = (29, true);
        st.native_runtime_records[i].raw[0x35] = 17;
        st.native_runtime_records[i].raw[0x36] = 29;
    }
    let mut o = NativeDeathOp { unit: 1, writes: vec![[0x35, 26, 1], [0x36, 15, 1]], ..op("record_bytes") };
    st.apply_native_death_op(&o).unwrap();
    let (u0, u1, r0, r1) = (&st.units[0], &st.units[1], &st.native_runtime_records[0], &st.native_runtime_records[1]);
    assert!(
        !(u1.native_record_byte35 != 26
            || !u1.has_native_record_byte35
            || r1.raw[0x35] != 26
            || u0.native_record_byte35 != 17
            || r0.raw[0x35] != 17
            || u1.native_record_byte36 != 15
            || !u1.has_native_record_byte36
            || r1.raw[0x36] != 15
            || u0.native_record_byte36 != 29
            || r0.raw[0x36] != 29),
        "+0x35／+0x36投影不一致或改到其他記錄"
    );
    o.writes = vec![[0x35, 9, 1], [0x20, 1, 1]];
    assert!(st.apply_native_death_op(&o).is_err(), "未知欄位不應被接受");
    assert!(
        !(st.units[1].native_record_byte35 != 26 || st.native_runtime_records[1].raw[0x35] != 26),
        "拒絕未知欄位前已部分改寫+0x35"
    );
    for invalid in [[0x36, 256, 1], [0x36, 15, 2]] {
        o.writes = vec![[0x35, 9, 1], invalid];
        let err = st.apply_native_death_op(&o);
        let (u1, r1) = (&st.units[1], &st.native_runtime_records[1]);
        assert!(
            err.is_err()
                && u1.native_record_byte35 == 26
                && u1.native_record_byte36 == 15
                && r1.raw[0x35] == 26
                && r1.raw[0x36] == 15,
            "目的地欄位越界或寬度錯誤沒有零修改拒收"
        );
    }
}

/// Go：`TestNativeDeathControlTurnNeedsRoundProvenance`
///
/// 回合事件控制列：[0x53A55]+3+3×slot = [0x53BEF] + delta，缺回合來源就拒絕。
#[test]
fn native_death_control_turn_needs_round_provenance() {
    let mut st = death_test_state(1);
    let o = NativeDeathOp { slot: 1, delta: 2, ..op("control_turn") };
    assert!(st.apply_native_death_op(&o).is_err(), "沒有 [0x53BEF] 來源時不該猜回合");
    (st.has_native_turn_event_control_state, st.native_round_counter) = (true, 5);
    st.native_turn_event_controls[1] = NativeTurnEventControl { turn: 0xff, event_id: 31, raw_camp: 0 };
    st.apply_native_death_op(&o).unwrap();
    let row = st.native_turn_event_controls[1];
    assert!(row.turn == 7 && row.event_id == 31, "控制列 {row:?}，turn 應為 7 且事件不變");
}

/// Go：`TestNativeActionWhenReadsStateAtExecution`
#[test]
fn native_action_when_reads_state_at_execution() {
    let mut st = death_test_state(4);
    let eq = NativeActionWhen { state_eq: Some([0x10, 1]), ..Default::default() };
    let got = eq.r#match(Some(&st));
    assert!(matches!(got, Ok(false)), "state_eq 還沒成立：{got:?}");
    st.native_event_state[0x10] = 1;
    assert!(matches!(eq.r#match(Some(&st)), Ok(true)), "state_eq 應成立");
    let ne = NativeActionWhen { state_ne: Some([0x13, 0]), ..Default::default() };
    assert!(!matches!(ne.r#match(Some(&st)), Ok(true)), "state_ne：值是 0 時不成立");
    let lt = NativeActionWhen { round_lt: Some(15), ..Default::default() };
    assert!(lt.r#match(Some(&st)).is_err(), "round_lt 沒有回合來源時應回錯");
    st.native_round_counter = 15;
    assert!(!matches!(lt.r#match(Some(&st)), Ok(true)), "第 15 回合不小於 15（0x3488A jge）");
    let active = NativeActionWhen { any_active: Some([1, 2]), ..Default::default() };
    (st.units[1].native_record_byte5, st.units[2].native_record_byte5) = (1, 1);
    assert!(!matches!(active.r#match(Some(&st)), Ok(true)), "1..2 都不在場時 any_active 不成立");
    st.units[2].native_record_byte5 = 0;
    assert!(matches!(active.r#match(Some(&st)), Ok(true)), "有一筆在場就成立");
}

/// Go：`TestClearNativeHPFromSyncsProjection`
///
/// 0x35BBA：從 first 起清 +0x40，帶 raw 投影時一起清。
#[test]
fn clear_native_hp_from_syncs_projection() {
    let mut st = death_test_state(4);
    st.has_native_runtime_unit_projection = true;
    st.native_runtime_records = vec![NativeRuntimeRecordState::default(); 4];
    for r in &mut st.native_runtime_records {
        r.raw[0x40] = 20;
    }
    let cleared = st.clear_native_hp_from(2);
    assert!(
        cleared.len() == 2 && st.units[1].hp == 20 && st.units[2].hp == 0 && st.units[3].hp == 0,
        "清除範圍錯誤：{} 筆",
        cleared.len()
    );
    assert!(
        !(st.native_runtime_records[1].raw[0x40] != 20 || st.native_runtime_records[3].raw[0x40] != 0),
        "raw 投影沒有跟著清"
    );
}

/// Go：`TestKillWithExpCancelAwardsNoExperience`
///
/// 事件 30 的 [0x53EC8]=0：擊倒帶無條件 exp_cancel 的單位，整次行動的經驗是 0。
#[test]
fn kill_with_exp_cancel_awards_no_experience() {
    let Some(path) = docs_file("data/exe_tables/growth.json") else { return };
    let rows = load_native_growth_rows(&path).unwrap();
    let sc = Scenario {
        native_death_programs: BTreeMap::from([(
            "2:30".to_string(),
            vec![Action {
                r#type: "native_death_op".into(),
                native_death_op: Some(op("exp_cancel")),
                ..Default::default()
            }],
        )]),
        ..Default::default()
    };
    // Go 傳方法值 `sc.NativeDeathCancelsExp`；Rust 包成擁有劇本複本的閉包。
    let cancel: Arc<dyn Fn(&Unit) -> bool + Send + Sync> =
        Arc::new(move |u: &Unit| sc.native_death_cancels_exp(Some(u)));
    let attacker = Unit {
        camp: OWN,
        lv: 5,
        hp: 40,
        max_hp: 40,
        ap: 99,
        hit: 200,
        on_field: true,
        battle_fig: 4,
        has_battle_fig: true,
        ..Default::default()
    };
    let boss = Unit {
        camp: ENEMY,
        lv: 5,
        hp: 1,
        max_hp: 30,
        dp: 0,
        ev: 0,
        on_field: true,
        x: 1,
        exp_per_level: 20,
        death_effect: Some(DeathEffect { r#type: 2, value: 30 }),
        ..Default::default()
    };
    let mut st = State {
        w: 8,
        h: 8,
        native_growth_rows: Some(rows),
        native_death_exp_cancel: Some(Callback::new(cancel)),
        units: vec![attacker, boss],
        ..Default::default()
    };
    let (attacker, boss) = (0, 1);
    let result = st.attack_with_rng(Some(attacker), boss, &mut GoMathRand::new(1));
    assert!(st.units[boss].hp == 0, "測試前提：攻擊應擊倒頭目，剩 HP {}", st.units[boss].hp);
    assert!(
        result.exp_gained == 0.0 && st.units[attacker].exp == 0.0,
        "擊倒萊汀不該有經驗：收下 {}、exp {}",
        result.exp_gained,
        st.units[attacker].exp
    );
    // 反對照：沒有取消的死亡效果照常給經驗。
    st.units[1] = Unit {
        camp: ENEMY,
        lv: 5,
        hp: 1,
        max_hp: 30,
        on_field: true,
        x: 1,
        exp_per_level: 20,
        death_effect: Some(DeathEffect { r#type: 3, value: 8 }),
        ..Default::default()
    };
    let r = st.attack_with_rng(Some(attacker), 1, &mut GoMathRand::new(1));
    assert!(r.exp_gained != 0.0, "反對照：一般擊倒應有經驗");
}

/// Go：`TestNativeDeathProgramsCoverEveryBattleMap`
///
/// 全部戰場的死亡效果都有程式可跑；死亡事件生成的群組不能在開局就在場。
/// Go 以 `filepath.Glob` 取劇本（字典序）；Rust 列目錄後排序，得到相同順序。Go 用 `t.Errorf` 收集
/// 全部錯誤，Rust 收進清單最後一次報出。
#[test]
#[ignore = "等 F 合併"]
fn native_death_programs_cover_every_battle_map() {
    let Some(scenarios) = remake_asset("scenarios") else { return };
    let Some(assets) = remake_asset("") else { return };
    let known = ["native_death_op", "dialogue", "pan", "native_acting", "reset_pose", "spawn_group", "delay"];
    let ops = [
        "ai_mode_range",
        "ai_byte_set_range",
        "ai_byte_and_range",
        "record_bytes",
        "control_turn",
        "state_set",
        "state_inc",
        "mark_inactive",
        "range_one",
        "range_zero",
        "exp_cancel",
        "staging",
        "clear_hp_from",
        "reward",
    ];
    let gaps: BTreeSet<String> = BTreeSet::new();
    let mut paths: Vec<std::path::PathBuf> = std::fs::read_dir(&scenarios)
        .unwrap()
        .map(|e| e.unwrap().path())
        .filter(|p| {
            let name = p.file_name().unwrap().to_string_lossy().into_owned();
            let b = name.as_bytes();
            name.len() == 9
                && name.starts_with("ch")
                && name.ends_with(".json")
                && b[2].is_ascii_digit()
                && b[3].is_ascii_digit()
        })
        .collect();
    paths.sort();
    assert!(paths.len() == 30, "劇本 {} 份，應為 30", paths.len());
    let mut errors: Vec<String> = Vec::new();
    for path in &paths {
        let sc = load_scenario(&path.to_string_lossy()).unwrap();
        let units_path = std::path::Path::new(&assets).join(format!("maps/map{0}/map{0}_units.json", sc.map));
        let raw = std::fs::read(&units_path).unwrap_or_else(|err| panic!("{}: {err}", units_path.display()));
        let units: serde_json::Value = serde_json::from_slice(&raw).unwrap();
        let source = format!("FDTXT_{:03}", sc.map + 1);
        let initial: BTreeSet<i64> = sc.initial_groups.iter().copied().collect();
        for u in units["units"].as_array().into_iter().flatten() {
            let e = &u["death_effect"];
            if e.is_null() {
                continue;
            }
            let (ty, value) = (e["type"].as_i64().unwrap_or(0), e["value"].as_i64().unwrap_or(0));
            if ty != 2 && ty != 3 {
                continue;
            }
            let key = format!("{ty}:{value}");
            let name = format!("{} {key}", path.file_name().unwrap().to_string_lossy());
            let Some(program) = sc.native_death_programs.get(&key) else {
                if !gaps.contains(&name) {
                    errors.push(format!("{name} 沒有死亡程式"));
                }
                continue;
            };
            if gaps.contains(&name) {
                errors.push(format!("{name} 已經有程式了，把它從已知缺口移走"));
            }
            for a in program {
                if !known.contains(&a.r#type.as_str()) {
                    errors.push(format!("{name} 有未知動作 {}", go_quote(&a.r#type)));
                }
                if let Some(o) = &a.native_death_op
                    && !ops.contains(&o.op.as_str())
                {
                    errors.push(format!("{name} 有未知原語 {}", go_quote(&o.op)));
                }
                if let Some(d) = &a.native_dialogue_ref
                    && d.source_dat != source
                {
                    errors.push(format!("{name} 的對白來自 {}，戰場文字庫應是 {source}", d.source_dat));
                }
                let mut spawned = a.groups.clone();
                if let Some(o) = &a.native_death_op
                    && o.op == "staging"
                {
                    spawned = vec![o.group];
                }
                for g in spawned {
                    if initial.contains(&g) {
                        errors.push(format!("{name} 會生成群組 {g}，它卻在開局就在場"));
                    }
                }
            }
        }
    }
    assert!(errors.is_empty(), "{}", errors.join("\n"));
}
