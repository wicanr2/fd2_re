//! 移植自 `remake/internal/battle/native_end_turn_recovery_test.go`。

use super::*;

/// Go `native_ai_record_test.go` 的 `completeNativeAIScoringUnit`（A 組測試檔的輔助函式，在這裡照抄一份）。
fn complete_native_ai_scoring_unit() -> Unit {
    Unit {
        x: 99,
        y: 98,
        native_map_presentation: NativeMapPresentationState { x: 7, y: 8, pose: 3, motion: 4 },
        has_native_map_presentation: true,
        battle_fig: 9,
        has_battle_fig: true,
        native_identity: 10,
        has_native_identity: true,
        native_record_byte8: 0x60,
        has_native_record_byte8: true,
        native_record_race: 11,
        has_native_record_race: true,
        native_record_class: 12,
        has_native_record_class: true,
        native_record_byte5: 0x20,
        has_native_record_byte5: true,
        native_record_byte6: 1,
        has_native_record_byte6: true,
        native_record_byte34: 0x81,
        has_native_record_byte34: true,
        native_record_byte35: 13,
        has_native_record_byte35: true,
        native_record_byte36: 14,
        has_native_record_byte36: true,
        native_record_word42: 234,
        has_native_record_word42: true,
        native_record_word46: 18,
        has_native_record_word46: true,
        inventory_slots: vec![1, 2, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        native_command_mask: [1, 2, 3, 4, 5],
        native_transient: [6, 7, 8, 9, 10, 0],
        lv: 15,
        mv: 6,
        hp: 123,
        max_hp: 234,
        mp: 17,
        max_mp: 18,
        ap: 19,
        dp: 20,
        hit: 21,
        ev: 22,
        ..Default::default()
    }
}

/// Go：`TestNativeEndTurnRecoveryPlanAndWholeBatchRejection`
#[test]
fn native_end_turn_recovery_plan_and_whole_batch_rejection() {
    let a = Unit { on_field: true, camp: OWN, hp: 10, max_hp: 50, ..Default::default() };
    let b = Unit { on_field: true, camp: OWN, hp: 20, max_hp: 100, ..Default::default() };
    let mut s = State { units: vec![a, b], ..Default::default() };
    let plan = s.plan_native_end_turn_recovery();
    assert!(plan.len() == 2 && s.units[0].hp == 10 && s.units[1].hp == 20, "計畫已寫入或排序錯誤");
    s.units[1].max_hp = 101; // same quotient; invalidates the captured input nonetheless
    assert!(
        s.commit_native_end_turn_recovery(&plan).is_err() && s.units[0].hp == 10 && s.units[1].hp == 20,
        "後筆過期仍部分提交前筆"
    );
}

/// Go：`TestNativeEndTurnRecoveryFollows1A30BGates`
#[test]
fn native_end_turn_recovery_follows_1a30b_gates() {
    let mk = |hp, max_hp, byte5| {
        let mut u = complete_native_ai_scoring_unit();
        (u.on_field, u.camp) = (true, OWN);
        (u.native_record_byte6, u.native_record_byte5) = (2, byte5);
        u.native_transient = [0; 6];
        (u.hp, u.max_hp) = (hp, max_hp);
        u
    };
    let fresh = mk(87, 147, 0); // 147/5 = 29 → 116（第四章 r8 收據 seq 671 的 id 4）
    let near_full = mk(140, 147, 0); // 140+29 > 147 → 取 147
    let acted = mk(50, 100, 0x80); // 已行動不回復
    let dead = mk(0, 100, 1); // 不在場／陣亡不回復
    let full = mk(100, 100, 0); // 滿血不回復
    let mut transient = mk(50, 100, 0); // +0x26 非零不回復
    transient.native_transient[4] = 3;
    let mut enemy = mk(50, 100, 0);
    enemy.native_record_byte6 = 0;
    let legacy = Unit { on_field: true, camp: OWN, hp: 10, max_hp: 50, ..Default::default() }; // 沒有 raw 旗標走 Camp／Acted
    let mut s =
        State { units: vec![fresh, near_full, acted, dead, full, transient, enemy, legacy], ..Default::default() };
    let got = s.apply_native_end_turn_recovery();
    assert!(
        got.len() == 3 && got[0].unit == Some(0) && got[1].unit == Some(1) && got[2].unit == Some(7),
        "回復名單={got:?}"
    );
    let hp: Vec<i64> = s.units.iter().map(|u| u.hp).collect();
    assert_eq!(hp, [116, 147, 50, 0, 100, 50, 50, 20], "HP：fresh nearFull acted dead full transient enemy legacy");
    // 0x1A477 → 0x13512：回復過的記錄 +5 bit7 設起來；沒回復的不動。
    assert!(
        s.units[0].native_record_byte5 == 0x80
            && s.units[1].native_record_byte5 == 0x80
            && s.units[4].native_record_byte5 == 0
            && s.units[5].native_record_byte5 == 0,
        "+5：fresh={:#x} nearFull={:#x} full={:#x} transient={:#x}",
        s.units[0].native_record_byte5,
        s.units[1].native_record_byte5,
        s.units[4].native_record_byte5,
        s.units[5].native_record_byte5
    );
}

/// Go：`TestMarkNativeDeadRecordsWritesByte5AfterCommandDamage`
#[test]
fn mark_native_dead_records_writes_byte5_after_command_damage() {
    let mut dead = complete_native_ai_scoring_unit();
    (dead.hp, dead.native_record_byte5) = (0, 0x80); // 指令傷害只寫 HP，+5 還留著已行動位
    let mut alive = complete_native_ai_scoring_unit();
    (alive.hp, alive.native_record_byte5) = (5, 0x80);
    let legacy = Unit { hp: 0, ..Default::default() };
    let mut s = State { units: vec![dead, alive, legacy], ..Default::default() };
    let got = s.mark_native_dead_records();
    assert!(
        got == [0] && s.units[0].native_record_byte5 == 1 && s.units[1].native_record_byte5 == 0x80,
        "marked={got:?} dead.+5={:#x} alive.+5={:#x}",
        s.units[0].native_record_byte5,
        s.units[1].native_record_byte5
    );
    let again = s.mark_native_dead_records();
    assert!(again.is_empty(), "第二次掃描不該再標：{again:?}");
    // 標記後 0x1A30B 的回復不會再碰它。
    let dead = &mut s.units[0];
    (dead.on_field, dead.camp, dead.native_record_byte6, dead.max_hp) = (true, OWN, 2, 100);
    dead.native_transient = [0; 6];
    let rec = s.apply_native_end_turn_recovery();
    assert!(rec.is_empty() && s.units[0].hp == 0, "陣亡單位被回復：{rec:?} hp={}", s.units[0].hp);
}
