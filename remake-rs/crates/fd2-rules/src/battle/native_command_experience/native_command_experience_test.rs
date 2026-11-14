//! 移植自 `remake/internal/battle/native_command_experience_test.go`。

use super::*;

/// 帶原版高階職業表的守方：列表[9] = row9。
fn high_class_target(fig: i64, lv: i64, hp: i64, max_hp: i64, row9: u8) -> Unit {
    let mut record = vec![0u8; 10];
    record[9] = row9;
    Unit {
        camp: ENEMY,
        lv,
        hp,
        max_hp,
        battle_fig: fig,
        has_battle_fig: true,
        native_constructor: Some(NativeConstructorTable {
            branch: "high_class".to_string(),
            index: fig - 0x44,
            record,
            ..Default::default()
        }),
        ..Default::default()
    }
}

fn ally(lv: i64, class: u8) -> Unit {
    Unit {
        camp: OWN,
        lv,
        hp: 40,
        max_hp: 80,
        battle_fig: 4,
        has_battle_fig: true,
        native_record_class: class,
        has_native_record_class: true,
        ..Default::default()
    }
}

#[test]
fn test_native_damage_experience_follows_1c81f() {
    let target = high_class_target(0x44, 5, 70, 100, 12);
    // 存活：12 × 5 × 擲出 30 / 最大 HP 100 = 18。
    assert_eq!(native_damage_experience(&target, 30, 70).unwrap(), 18, "存活經驗");
    // 擊倒：整份 12 × 5，不乘傷害比例。
    assert_eq!(native_damage_experience(&target, 300, 0).unwrap(), 60, "擊倒經驗");
    // 擲出值超過剩餘 HP 時仍用擲出值（0x1C8CF imul edx,ebp），不截斷。
    assert_eq!(native_damage_experience(&target, 150, 1).unwrap(), 90, "擲出值經驗");
    // 0x1C8AA jl：圖號 0x43 不累加，也不需要表。
    let low = Unit { lv: 5, hp: 10, max_hp: 100, battle_fig: 0x43, has_battle_fig: true, ..Default::default() };
    assert_eq!(native_damage_experience(&low, 30, 10).unwrap(), 0, "圖號 0x43 經驗");
    // 沒有原版圖號的正規化單位回 0。
    let normalized = Unit { lv: 5, max_hp: 100, ..Default::default() };
    assert_eq!(native_damage_experience(&normalized, 30, 10).unwrap(), 0, "正規化單位經驗");
    // 高階圖號缺表：失敗即關閉。
    let missing = Unit { lv: 5, hp: 10, max_hp: 100, battle_fig: 0x44, has_battle_fig: true, ..Default::default() };
    assert!(native_damage_experience(&missing, 30, 10).is_err(), "缺高階職業表卻沒有拒收");
    let mut wrong_index = high_class_target(0x45, 5, 10, 100, 12);
    wrong_index.native_constructor.as_mut().unwrap().index = 0;
    assert!(native_damage_experience(&wrong_index, 30, 10).is_err(), "高階職業表索引和圖號不符卻沒有拒收");
}

#[test]
fn test_native_restore_experience_follows_1c916() {
    let target = |fig: i64, class: u8| Unit { hp: 60, battle_fig: fig, ..ally(5, class) };
    for (name, fig, class, want) in [
        ("職業 8 不加 30", 4, 8, 40 * 5 * 20 / 80),
        ("職業 9 加 30", 4, 9, 40 * 35 * 20 / 80),
        ("職業 24 加 30", 4, 24, 40 * 35 * 20 / 80),
        ("職業 25 不加 30", 4, 25, 40 * 5 * 20 / 80),
        ("圖號 0x4A 仍累加", 0x4a, 8, 40 * 5 * 20 / 80),
        ("圖號 0x4B 不累加", 0x4b, 8, 0),
    ] {
        assert_eq!(native_restore_experience(Some(&target(fig, class)), 20), want, "{name}");
    }
    let normalized = Unit { lv: 5, max_hp: 80, ..Default::default() };
    assert_eq!(native_restore_experience(Some(&normalized), 20), 0, "正規化單位回復經驗");
    let units = vec![target(4, 9), target(4, 8)];
    let heal = native_command_heal_experience(
        &units,
        &[
            NativeCommandHealResult { target: Some(0), restore: NativeCommandRestore { rolled: 30, actual: 20 } },
            NativeCommandHealResult { target: Some(1), restore: NativeCommandRestore { rolled: 30, actual: 0 } },
        ],
    );
    assert_eq!(heal, 40 * 35 * 20 / 80, "回復指令經驗");
}

#[test]
fn test_native_command_damage_experience_skips_misses() {
    let units = vec![high_class_target(0x44, 5, 70, 100, 12), high_class_target(0x44, 5, 100, 100, 12)];
    let got = native_command_damage_experience(
        &units,
        &[
            NativeCommandDamageResult {
                target: Some(0),
                native_command_damage: NativeCommandDamage { hit: true, damage: 30 },
                hp_before: 100,
                hp_after: 70,
            },
            NativeCommandDamageResult {
                target: Some(1),
                native_command_damage: NativeCommandDamage { hit: false, damage: 0 },
                hp_before: 100,
                hp_after: 100,
            },
        ],
    )
    .unwrap();
    assert_eq!(got, 18, "指令傷害經驗");
    let units = vec![high_class_target(0x44, 5, 0, 100, 12)];
    let strike = native_command_derived_strike_experience(
        &units,
        &[NativeCommand24Damage {
            target: Some(0),
            rolled: 140,
            damage: 100,
            hp_before: 100,
            hp_after: 0,
            ..Default::default()
        }],
    )
    .unwrap();
    assert_eq!(strike, 60, "衍生打擊擊倒經驗");
}

#[test]
fn test_native_command2022_experience_follows_tail_writers() {
    // 25 → 0x22C04：只有 +5 bit7 被清掉的目標，8 × L。
    let units = vec![ally(5, 9), ally(5, 9)];
    let plan = NativeAICommand2022Plan {
        command_id: 25,
        results: vec![
            NativeAICommand2022Result { target: Some(0), command25: true, cleared_byte5: true, ..Default::default() },
            NativeAICommand2022Result { target: Some(1), command25: true, ..Default::default() },
        ],
        ..Default::default()
    };
    assert_eq!(native_command2022_experience(&units, &plan).unwrap(), 8 * 35, "指令 25 經驗");
    // 20／21 → 0x22AF6：4 × L 加上 0x1C916 的回復經驗。
    let units = vec![ally(5, 8), ally(5, 8)];
    let plan = NativeAICommand2022Plan {
        command_id: 20,
        results: vec![
            NativeAICommand2022Result {
                target: Some(0),
                restore: Some(NativeRawFlagRestoreResult {
                    applied: true,
                    restore: NativeRawRestoreResult { score: 7, ..Default::default() },
                    ..Default::default()
                }),
                ..Default::default()
            },
            NativeAICommand2022Result {
                target: Some(1),
                restore: Some(NativeRawFlagRestoreResult::default()),
                ..Default::default()
            },
        ],
        ..Default::default()
    };
    assert_eq!(native_command2022_experience(&units, &plan).unwrap(), 7 + 4 * 5, "指令 20 經驗");
    // 22／26／27 → 0x22D1B：0x1C81F 傷害經驗加 8 × +0x21（不加 30）。
    let mut enemy = high_class_target(0x44, 6, 91, 100, 10);
    enemy.native_record_class = 9;
    enemy.has_native_record_class = true;
    let units = vec![enemy];
    let plan = NativeAICommand2022Plan {
        command_id: 26,
        targets: vec![0],
        results: vec![NativeAICommand2022Result {
            target: Some(0),
            apply: Some(NativeRawApplicationResult {
                target_index: 0,
                applied: true,
                damage: NativeRawDamageResult { rolled: 9, ..Default::default() },
                ..Default::default()
            }),
            ..Default::default()
        }],
        after: vec![NativeAICommand2022TargetState { target: Some(0), hp: 91, ..Default::default() }],
        ..Default::default()
    };
    assert_eq!(native_command2022_experience(&units, &plan).unwrap(), 10 * 6 * 9 / 100 + 8 * 6, "指令 26 經驗");
}

#[test]
fn test_native_compound_command_experience() {
    let units = vec![high_class_target(0x44, 5, 70, 100, 12)];
    let got = native_compound_command32_experience(
        &units,
        &NativeCompoundCommand32Result {
            targets: vec![NativeCompoundCommand32Target {
                target: Some(0),
                damage: NativeCommandDamage { hit: true, damage: 30 },
                hp_before: 100,
                hp_after: 70,
            }],
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(got, 18, "指令 32 經驗");
    let c33 = NativeCompoundCommand33Result {
        restore: NativeRawRestoreBatch {
            results: vec![
                NativeRawRestoreResult { score: 3, ..Default::default() },
                NativeRawRestoreResult { score: 4, ..Default::default() },
            ],
            ..Default::default()
        },
        ..Default::default()
    };
    assert_eq!(native_compound_command33_experience(&c33), 7, "指令 33 經驗");
    let c34 = NativeCompoundCommand34Result {
        stages: [10, 12, 0]
            .into_iter()
            .map(|accumulator| NativeCommandModifierResult { accumulator, ..Default::default() })
            .collect(),
        ..Default::default()
    };
    assert_eq!(native_compound_command34_experience(&c34), 22, "指令 34 經驗");
    let c35 = NativeCompoundCommand35Result {
        stages: vec![NativeCompoundCommand35Stage {
            accumulator: 40,
            results: vec![NativeRawApplicationResult {
                target_index: 0,
                applied: true,
                damage: NativeRawDamageResult { rolled: 10, ..Default::default() },
                ..Default::default()
            }],
            ..Default::default()
        }],
        stage_states: vec![vec![NativeCommand35TargetState { target: Some(0), hp: 60, ..Default::default() }]],
        ..Default::default()
    };
    assert_eq!(native_compound_command35_experience(&units, &c35).unwrap(), 40 + 12 * 5 * 10 / 100, "指令 35 經驗");
}

#[test]
fn test_award_native_player_action_experience_follows_117e7() {
    let player = |lv: i64| Unit {
        camp: OWN,
        lv,
        hp: 50,
        max_hp: 50,
        battle_fig: 4,
        has_battle_fig: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 2,
        has_native_record_byte6: true,
        ..Default::default()
    };
    // 0x11959：超過 99 壓到 99；收下後 0x1E51A 清零。
    let mut st = State { units: vec![player(5)], ..Default::default() };
    let (got, _, _) = st.award_native_player_action_experience(Some(0), 150, 1, &[]);
    assert!(
        got == 99 && st.units[0].exp == 99.0 && st.native_experience_accumulator == 0,
        "壓 99：收下 {got} exp {} 累計 {}",
        st.units[0].exp,
        st.native_experience_accumulator
    );
    // 0x1E2F2：滿級整段返回，不清累計。
    let mut st = State { units: vec![player(40)], ..Default::default() };
    let (got, _, _) = st.award_native_player_action_experience(Some(0), 30, 1, &[]);
    assert!(
        got == 0 && st.units[0].exp == 0.0 && st.native_experience_accumulator == 30,
        "滿級：收下 {got} exp {} 累計 {}",
        st.units[0].exp,
        st.native_experience_accumulator
    );
    // 0x34B48：擊倒帶無條件 exp_cancel 的單位，整次行動經驗清零。Go 以指標比對受害者；這裡只有它是敵方。
    let victim = Unit { camp: ENEMY, hp: 0, ..Default::default() };
    let cancel: std::sync::Arc<dyn Fn(&Unit) -> bool + Send + Sync> = std::sync::Arc::new(|u: &Unit| u.camp == ENEMY);
    let mut st = State {
        units: vec![player(5), victim],
        native_death_exp_cancel: Some(Callback::new(cancel)),
        ..Default::default()
    };
    let (got, _, _) = st.award_native_player_action_experience(Some(0), 40, 1, &[1]);
    assert!(
        got == 0 && st.units[0].exp == 0.0 && st.native_experience_accumulator == 0,
        "exp_cancel：收下 {got} exp {} 累計 {}",
        st.units[0].exp,
        st.native_experience_accumulator
    );
    // 存活的同一單位不觸發取消。
    st.units[1].hp = 5;
    let (got, _, _) = st.award_native_player_action_experience(Some(0), 40, 1, &[1]);
    assert_eq!(got, 40, "目標存活時收下");
}
