//! 移植自 `remake/internal/battle/growth_native_test.go`。
//!
//! Go 測試裡不在 `State.Units` 的單位，Rust 依模組文件第 6 點先 push 再傳索引（`AwardExp`、
//! `AwardExpNative`、`GainExp` 只動傳入的單位）。

use super::*;
use crate::test_support::{docs_file, temp_dir};
use std::sync::Arc;
use std::sync::atomic::{AtomicUsize, Ordering};

/// Go：`TestNativeEnemyGrowthMatchesChapter15Oracle`——原版第十五章 seq1049～1061：敵人 record40
/// 承接共享累計19。固定seed4，每個成長欄位再承接原版0x1E54A的受控輸入；不要求演出骰序一致。
#[test]
fn native_enemy_growth_matches_chapter15_oracle() {
    let Some(slots) = docs_file("data/exe_tables/native_growth_slots.json") else { return };
    let Some(evidence) = docs_file("data/ida/fd2_ch15_ai_growth_20261002.json") else { return };
    let rows = load_native_growth_slots(&slots).unwrap();
    let evidence: serde_json::Value = serde_json::from_slice(&std::fs::read(evidence).unwrap()).unwrap();
    let seeds: Vec<u16> = evidence["oracle"]["growth_trace"]
        .as_array()
        .unwrap()
        .iter()
        .map(|row| row["rng_word"].as_i64().unwrap() as u16)
        .collect();
    assert_eq!(seeds.len(), 10, "原版成長收據應為10");
    let cursor = Arc::new(AtomicUsize::new(0));
    let observer_cursor = Arc::clone(&cursor);
    let observer: Arc<dyn Fn(&Unit, u16) -> u16 + Send + Sync> = Arc::new(move |_, _| {
        let n = observer_cursor.fetch_add(1, Ordering::SeqCst);
        assert!(n < seeds.len(), "重製成長多擲");
        seeds[n]
    });
    let mut st = State {
        native_growth_rows: Some(rows),
        native_growth_roll_observer: Some(Callback::new(observer)),
        ..Default::default()
    };
    st.units.push(Unit {
        camp: ENEMY,
        lv: 19,
        exp: 255.0,
        hp: 161,
        max_hp: 304,
        max_mp: 0,
        ap: 232,
        dp: 102,
        dx: 19,
        hit: 139,
        ev: 24,
        equipment_base_set: true,
        base_ap: 152,
        base_dp: 38,
        base_hit: 139,
        base_ev: 24,
        battle_fig: 102,
        has_battle_fig: true,
        native_record_byte5: 0,
        has_native_record_byte5: true,
        native_record_byte6: 0,
        has_native_record_byte6: true,
        native_record_word42: 304,
        has_native_record_word42: true,
        native_record_word46: 0,
        has_native_record_word46: true,
        ..Default::default()
    });
    let u = 0;
    let (got, ups, _) = st.award_exp_native(Some(u), 19, 4);
    let unit = &st.units[u];
    assert!(
        !(got != 19
            || ups.len() != 2
            || cursor.load(Ordering::SeqCst) != 10
            || unit.lv != 21
            || unit.exp != 74.0
            || unit.hp != 161
            || unit.max_hp != 629
            || unit.max_mp != 376
            || unit.ap != 456
            || unit.dp != 159
            || unit.dx != 191
            || unit.hit != 311
            || unit.ev != 196
            || unit.base_ap != 376
            || unit.base_dp != 95
            || unit.native_record_word42 != 629
            || unit.native_record_word46 != 376),
        "敵方原生成長不符同槽收據：got={got} ups={ups:?} rolls={} unit={unit:?}",
        cursor.load(Ordering::SeqCst)
    );
    st.units.push(Unit { camp: ENEMY, lv: 19, exp: 255.0, hp: 161, max_hp: 304, ..Default::default() });
    let legacy = 1;
    let (accepted, _, _) = st.award_exp_native(Some(legacy), 19, 4);
    assert!(accepted == 0 && st.units[legacy].lv == 19, "缺原始來源的正規化敵人被套用原生例外");
    let (accepted, _) = st.award_exp(Some(u), 19.0, &mut GoMathRand::new(4));
    assert!(accepted == 0.0 && st.units[u].lv == 21, "一般AwardExp陣營範圍被改動");
}

/// Go：`TestNativeGrowthSlotsRejectMissingAndNonByteRows`
#[test]
fn native_growth_slots_reject_missing_and_non_byte_rows() {
    let Some(path) = docs_file("data/exe_tables/native_growth_slots.json") else { return };
    let raw = std::fs::read(&path).unwrap();
    for name in ["缺槽", "超出byte"] {
        let mut copy_rows: Vec<serde_json::Map<String, serde_json::Value>> = serde_json::from_slice(&raw).unwrap();
        if name == "缺槽" {
            copy_rows.truncate(255);
        } else {
            copy_rows[102].insert("ap".into(), serde_json::json!([256, 5]));
        }
        let data = serde_json::to_vec(&copy_rows).unwrap();
        let file = temp_dir("growth-slots").join("slots.json");
        std::fs::write(&file, data).unwrap();
        assert!(load_native_growth_slots(&file.to_string_lossy()).is_err(), "{name}：錯誤成長槽未被拒絕");
    }
    assert!(load_native_growth_rows(&path).is_err(), "作者表讀取器接受了負跨距相鄰資料");
}

/// Go：`TestNativeGrowthRowFollowsRecordPlus7`——原生戰場的單位不帶名字（身分走 NativeIdentity），升級時
/// 成長列由記錄 +7 選（0x1E2F8 → 0x4E4D1 = 0x620A1 + 11 × selector）。用名字查會一律查不到，等級
/// 照加、數值不長——第一關跑完身分 4 升到 lv2 而 MaxHP 仍是 48 就是這個樣子。
#[test]
fn native_growth_row_follows_record_plus7() {
    let Some(path) = docs_file("data/exe_tables/growth.json") else { return };
    let rows = load_native_growth_rows(&path).unwrap();
    assert_eq!(rows.len(), 68, "成長表列數");
    // idx 4：ap [6,9) dp [4,5) dx [2,2] hp [8,11) mp [0,0]，即手寫表的亞雷斯 騎士。
    // dp 跨距 1 仍會擲（Native 且非 Fixed），dx／mp 跨距 0 才是 Fixed。
    let want = GrowthRow {
        ap: StatRange { min: 6, max: 8, native: true, fixed: false },
        dp: StatRange { min: 4, max: 4, native: true, fixed: false },
        dx: StatRange { min: 2, max: 2, native: true, fixed: true },
        hp: StatRange { min: 8, max: 10, native: true, fixed: false },
        mp: StatRange { min: 0, max: 0, native: true, fixed: true },
        balance_rates: None,
    };
    assert_eq!(rows[&4], want, "row4");
    let legacy_row = GROWTH_TABLE["亞雷斯"]["騎士"];
    for (a, b) in [
        (rows[&4].ap, legacy_row.ap),
        (rows[&4].dp, legacy_row.dp),
        (rows[&4].dx, legacy_row.dx),
        (rows[&4].hp, legacy_row.hp),
        (rows[&4].mp, legacy_row.mp),
    ] {
        assert!(
            a.min == b.min && a.max == b.max,
            "EXE row4 與手寫表的亞雷斯 騎士不一致：{:?} vs {legacy_row:?}",
            rows[&4]
        );
    }

    let mut st = State { native_growth_rows: Some(rows), ..Default::default() };
    st.units.push(Unit {
        camp: OWN,
        lv: 1,
        max_hp: 48,
        hp: 48,
        battle_fig: 4,
        has_battle_fig: true,
        ..Default::default()
    });
    let events = st.gain_exp(Some(0), 100.0, &mut GoMathRand::new(1));
    assert!(events.len() == 1 && st.units[0].lv == 2, "升級 events={} lv={}", events.len(), st.units[0].lv);
    let gain = st.units[0].max_hp - 48;
    assert!((8..=10).contains(&gain), "MaxHP 成長 {gain}，應在 row4 的 8..10 之內");
    // 0x1E529 只寫 +0x42，目前 HP +0x40 不變。
    assert_eq!(st.units[0].hp, 48, "升級後目前 HP，原版不補，應維持 48");

    // 反對照：沒有原版 +7、名字也查不到，升級只加等級。
    st.units.push(Unit { camp: OWN, lv: 1, max_hp: 48, hp: 48, ..Default::default() });
    let events = st.gain_exp(Some(1), 100.0, &mut GoMathRand::new(1));
    let legacy = &st.units[1];
    assert!(
        events.is_empty() && legacy.lv == 2 && legacy.max_hp == 48,
        "無 +7 的單位：events={} lv={} maxhp={}",
        events.len(),
        legacy.lv,
        legacy.max_hp
    );
}

/// Go：`TestNativeLevelCapFollows1E292` 釘住 0x1E292 的上限與經驗歸零：
///
///     0x1E2E0 cmp edx,0x1E / 0x1E2E5 cmp edx,0x1F → 機兵比 0x63（99），其餘比 0x28（40）
///     0x1E2F2 je 0x1317D                          → 相等就整段返回，連經驗都不加
///     0x1E3EE sub [esp+4],0x64                     → 每升一級扣 100
///     0x1E403 cmp eax,0x63 / 0x1E4F6 cmp eax,0x1E  → 機兵到 99、任何人到 30，剩下的經驗歸零
///
/// 修改指南 references/text/modify1.md 第 11～13 條改的正是這幾個常數，其中 30 級那條
/// 被它稱為原版錯誤；忠實模式照原版。
#[test]
fn native_level_cap_follows1_e292() {
    let Some(path) = docs_file("data/exe_tables/growth.json") else { return };
    let rows = load_native_growth_rows(&path).unwrap();
    let mut st = State { native_growth_rows: Some(rows), ..Default::default() };
    let mut rng = GoMathRand::new(1);
    let unit = |fig: i64, lv: i64, exp: f64| Unit {
        camp: OWN,
        lv,
        exp,
        hp: 50,
        max_hp: 50,
        battle_fig: fig,
        has_battle_fig: true,
        ..Default::default()
    };

    struct Case {
        name: &'static str,
        fig: i64,
        lv: i64,
        exp: f64,
        award: f64,
        want_accepted: f64,
        want_lv: i64,
        want_exp: f64,
    }
    let case = |name, fig, lv, exp, award, want_accepted, want_lv, want_exp| Case {
        name,
        fig,
        lv,
        exp,
        award,
        want_accepted,
        want_lv,
        want_exp,
    };
    let cases = [
        case("一般單位在 40 級整段返回", 4, 40, 10.0, 50.0, 0.0, 40, 10.0),
        case("一般單位升到 40 級不歸零", 4, 39, 90.0, 50.0, 50.0, 40, 40.0),
        case("升到 30 級剩下的經驗歸零（原版錯誤）", 4, 29, 90.0, 50.0, 50.0, 30, 0.0),
        case("機兵過了 40 級照常收經驗", 0x1e, 40, 10.0, 50.0, 50.0, 40, 60.0),
        case("機兵升到 30 級同樣歸零", 0x1f, 29, 90.0, 50.0, 50.0, 30, 0.0),
        case("機兵升到 99 級歸零", 0x1e, 98, 90.0, 50.0, 50.0, 99, 0.0),
        case("機兵在 99 級整段返回", 0x1f, 99, 0.0, 50.0, 0.0, 99, 0.0),
    ];
    for c in cases {
        st.units.push(unit(c.fig, c.lv, c.exp));
        let u = st.units.len() - 1;
        let (accepted, _) = st.award_exp(Some(u), c.award, &mut rng);
        let got = &st.units[u];
        assert!(
            accepted == c.want_accepted && got.lv == c.want_lv && got.exp == c.want_exp,
            "{}：收下 {accepted} lv {} exp {}，應為 {}／{}／{}",
            c.name,
            got.lv,
            got.exp,
            c.want_accepted,
            c.want_lv,
            c.want_exp
        );
    }

    // 0x1E2D6 test [esi+5],1：死亡旗標在的單位不收經驗。
    let mut dead = unit(4, 5, 0.0);
    dead.hp = 0;
    (dead.native_record_byte5, dead.has_native_record_byte5) = (1, true);
    st.units.push(dead);
    let d = st.units.len() - 1;
    let (accepted, events) = st.award_exp(Some(d), 50.0, &mut rng);
    assert!(
        accepted == 0.0 && events.is_empty() && st.units[d].exp == 0.0,
        "陣亡單位收下 {accepted}、exp {}",
        st.units[d].exp
    );

    // 沒有原版 +7 的單位（重製端自訂資料）不套上限。
    st.units.push(Unit { camp: OWN, lv: 40, exp: 10.0, hp: 50, max_hp: 50, ..Default::default() });
    let l = st.units.len() - 1;
    let (accepted, _) = st.award_exp(Some(l), 50.0, &mut rng);
    assert!(accepted == 50.0 && st.units[l].exp == 60.0, "無 +7 的單位收下 {accepted}、exp {}", st.units[l].exp);
}

/// Go：`TestCappedAttackReportsNoExperience`：到上限時攻擊結果的 ExpGained 必須是 0，
/// 否則畫面會照樣附上「得到經驗值」，而原版在上限時整段不顯示。
#[test]
fn capped_attack_reports_no_experience() {
    let Some(path) = docs_file("data/exe_tables/growth.json") else { return };
    let rows = load_native_growth_rows(&path).unwrap();
    let mut st = State { native_growth_rows: Some(rows), ..Default::default() };
    st.units.push(Unit {
        camp: OWN,
        lv: 40,
        hp: 50,
        max_hp: 50,
        battle_fig: 4,
        has_battle_fig: true,
        ..Default::default()
    });
    let (accepted, events) = st.award_exp(Some(0), 30.0, &mut GoMathRand::new(1));
    assert!(accepted == 0.0 && events.is_empty(), "上限單位收下 {accepted}、升級 {} 次", events.len());
}
