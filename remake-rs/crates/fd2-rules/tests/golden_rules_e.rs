//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-e` 的黃金差分：fd2-rules E 組（戰鬥、成長與法術）。
//!
//! 由 `tools/rust_golden_rules_e.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_E`；真實輸入（受版控的
//! `remake/assets`）掛在 `FD2_GOLDEN_REPO` 底下，先核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：戰場由黃金檔的「底稿＋逐單位修改」組成，兩邊都把合併後的 JSON 物件解碼成 `Unit` 再綁定
//! 原版物品列。每個案例依 `op` 在 Rust 端重做同一串呼叫，整個結果物件與 Go 逐欄比對（鍵集合也要相同）：
//! `*Unit` 是 `State::units` 的索引（nil 為 -1），結構以 Go 欄位名輸出且只含 Go 匯出欄位，單位以 Go 的
//! JSON 比對。帶 `math/rand` 的案例另比呼叫後的下一個 `Int63`，確認亂數消耗量相同。E 組自己的錯誤
//! 文字逐字相同；Go 標準函式庫產生的錯誤（JSON 解碼、讀檔）只比接受或拒收。比對時的表示法正規化
//! 列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。
//!
//! 黃金不涵蓋（會走到 D 組的 `todo!()`，或以時間播種）：`NextAIPlan`、`NextAllyAIPlan`、`AITurn`、
//! `Attack`、`Cast`，以及只經由它們抵達的 crate 私有函式（`estDamage`、`aiTargets`、`aiSpell*`、
//! `nextAISpellPlan`、`nextNativeScannedAIPlan`、`nativeAIScanEligible`、`nativeAIPreselected`、
//! `nextAIPlanForUnit`、`aiActUnit`、`aiApproachPath`）。

use fd2_rules::battle::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use std::sync::Arc;

/// 刻意不照做 Go 的行為（PORTING.md 決定 2、`rust-port-no-go-stdlib`）。黃金案例都不落在這些輸入上，
/// 或只比接受與拒收。
const KNOWN_DIFFERENCES: &[&str] = &[
    // combat.go 202、235、621、921 行在 reach（Go map）裡以嚴格 < 挑最近格，距離平手時取先迭代到的；
    // Go 的 map 順序隨機，Rust 依 Cell 升冪。這四處只經由 NextAIPlan／AITurn 抵達，不在黃金裡。
    "combat.go reach 平手時的挑選順序",
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文（成長表另收 Go 欄位名 AP…MP 與小寫 ap…mp）。
    "JSON 欄位名大小寫",
    // 成長表的 [2]int 欄位：Go 對長度不是 2 的陣列截斷或補零，Rust 拒收；JSON 解碼錯誤文字只比接受或拒收。
    "成長表 JSON 解碼",
    // Go 可以把不在 State.Units 裡的 *Unit 或 nil 傳進來；Rust 以索引表示單位，黃金只傳 State.Units 裡的單位。
    "不在 State.Units 的單位指標",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列、空字串相同",
    // Go 把整數值的 float64 寫成 1，serde_json 寫成 1.0；只比數值。
    "數字只比數值",
];

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

fn digest(b: &[u8]) -> String {
    hex(&Sha256::digest(b))
}

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn b(v: &Value, key: &str) -> bool {
    v[key].as_bool().unwrap_or(false)
}

fn s<'a>(v: &'a Value, key: &str) -> &'a str {
    v[key].as_str().unwrap_or_else(|| panic!("欄位 {key} 不是字串：{}", v[key]))
}

fn ints(v: &Value) -> Vec<i64> {
    v.as_array().map(|a| a.iter().map(|x| x.as_i64().unwrap()).collect()).unwrap_or_default()
}

fn unit_from(v: &Value) -> Unit {
    serde_json::from_value(v.clone()).unwrap_or_else(|e| panic!("Unit JSON 讀不進來：{e}\n{v}"))
}

fn unit_val(u: &Unit) -> Value {
    serde_json::to_value(u).unwrap()
}

fn is_empty_like(v: &Value) -> bool {
    match v {
        Value::Null => true,
        Value::Array(a) => a.is_empty(),
        Value::String(s) => s.is_empty(),
        _ => false,
    }
}

/// 依 [`NORMALIZATIONS`] 比對兩個 JSON 值，回傳第一個不同的位置。
fn diff(path: &str, go: &Value, rust: &Value) -> Option<String> {
    match (go, rust) {
        (Value::Number(a), Value::Number(b)) => {
            let same = match (a.as_i64(), b.as_i64()) {
                (Some(x), Some(y)) => x == y,
                _ => match (a.as_u64(), b.as_u64()) {
                    (Some(x), Some(y)) => x == y,
                    _ => a.as_f64() == b.as_f64(),
                },
            };
            (!same).then(|| format!("{path}: go={a} rust={b}"))
        }
        (Value::Array(a), Value::Array(b)) => {
            if a.len() != b.len() {
                return Some(format!("{path}: 長度 go={} rust={}", a.len(), b.len()));
            }
            a.iter().zip(b).enumerate().find_map(|(k, (x, y))| diff(&format!("{path}[{k}]"), x, y))
        }
        (Value::Object(a), Value::Object(b)) => {
            for (k, x) in a {
                match b.get(k) {
                    Some(y) => {
                        if let Some(d) = diff(&format!("{path}.{k}"), x, y) {
                            return Some(d);
                        }
                    }
                    None => return Some(format!("{path}.{k}: Rust 沒有這個欄位")),
                }
            }
            b.keys().find(|k| !a.contains_key(*k)).map(|k| format!("{path}.{k}: Go 沒有這個欄位"))
        }
        _ if is_empty_like(go) && is_empty_like(rust) => None,
        _ => (go != rust).then(|| format!("{path}: go={go} rust={rust}")),
    }
}

/// Rust 結果先寫成 JSON 文字再讀回。serde_json 沒開 `float_roundtrip`，讀小數時可能差 1 ulp；Go 與
/// Rust 都把 f64 寫成最短可還原的十進位，讓兩邊的值經過同一個讀取器，才能逐位元比較小數。
fn check(ctx: &str, go: &Value, rust: &Value) {
    if let Some(d) = diff("", go, rust) {
        panic!("{ctx}：{d}");
    }
}

fn err_of<T>(r: &Result<T, BattleError>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

fn idx(u: Option<UnitIdx>) -> i64 {
    u.map_or(-1, |x| x as i64)
}

// ---- Go 結構的 JSON（沒有 json 標籤，鍵是 Go 欄位名）----

fn level_up_val(e: &LevelUpEvent) -> Value {
    json!({"NewLv": e.new_lv, "ApGain": e.ap_gain, "DpGain": e.dp_gain, "DxGain": e.dx_gain, "HpGain": e.hp_gain,
        "MpGain": e.mp_gain, "LearnedCommandIDs": e.learned_command_ids})
}

fn level_ups_val(v: &[LevelUpEvent]) -> Value {
    Value::Array(v.iter().map(level_up_val).collect())
}

fn attack_val(r: &AttackResult) -> Value {
    json!({"Amount": r.amount, "Missed": r.missed, "Crit": r.crit, "ExpGained": r.exp_gained,
        "LevelUps": level_ups_val(&r.level_ups), "Counter": r.counter.as_deref().map(attack_val)})
}

fn roll_result_val(r: &NativePhysicalRollResult) -> Value {
    json!({"Missed": r.missed, "Crit": r.crit, "Status": r.status, "Extra": r.extra, "Damage": r.damage,
        "StatusValue": r.status_value, "RNGState": r.rng_state})
}

fn strike_val(s: &NativePhysicalStrike) -> Value {
    json!({"Roll": roll_result_val(&s.roll), "Damage": s.damage, "DefenderHP": s.defender_hp})
}

fn strikes_val(v: &[NativePhysicalStrike]) -> Value {
    Value::Array(v.iter().map(strike_val).collect())
}

fn physical_val(r: &NativePhysicalAttackResult) -> Value {
    json!({"Attack": attack_val(&r.attack), "AttackStrikes": strikes_val(&r.attack_strikes),
        "Counter": r.counter.as_ref().map(attack_val), "CounterStrikes": strikes_val(&r.counter_strikes),
        "CounterExpGained": r.counter_exp_gained, "CounterLevelUps": level_ups_val(&r.counter_level_ups),
        "RNGState": r.rng_state})
}

fn cast_val(r: &CastResult) -> Value {
    json!({"Target": idx(r.target), "Amount": r.amount, "Missed": r.missed, "ExpGained": r.exp_gained,
        "LevelUps": level_ups_val(&r.level_ups)})
}

fn spell_val(sp: &Spell) -> Value {
    json!({"ID": sp.id, "Dmg": sp.dmg, "Hit": sp.hit, "Dist": sp.dist, "Range": sp.range, "MP": sp.mp,
        "Target": sp.target, "Name": sp.name})
}

fn spell_from(v: &Value) -> Spell {
    Spell {
        id: i(v, "ID"),
        dmg: i(v, "Dmg"),
        hit: i(v, "Hit"),
        dist: i(v, "Dist"),
        range: i(v, "Range"),
        mp: i(v, "MP"),
        target: i(v, "Target"),
        name: s(v, "Name").to_string(),
    }
}

fn stat_range_val(r: &StatRange) -> Value {
    json!({"Min": r.min, "Max": r.max, "Native": r.native, "Fixed": r.fixed})
}

fn growth_row_val(r: &GrowthRow) -> Value {
    json!({"AP": stat_range_val(&r.ap), "DP": stat_range_val(&r.dp), "DX": stat_range_val(&r.dx),
        "HP": stat_range_val(&r.hp), "MP": stat_range_val(&r.mp)})
}

fn growth_rows_val(rows: &BTreeMap<i64, GrowthRow>) -> Value {
    Value::Object(rows.iter().map(|(k, r)| (k.to_string(), growth_row_val(r))).collect())
}

fn rates(v: &[i64]) -> GrowthRates {
    GrowthRates { hp: v[0], mp: v[1], ap: v[2], dp: v[3], dx: v[4] }
}

fn balance_from(v: &Value) -> Option<BalanceOptions> {
    if v.is_null() {
        return None;
    }
    let mut o = BalanceOptions {
        growth: rates(&ints(&v["growth"])),
        enemy_percent: i(v, "enemy_percent"),
        ..Default::default()
    };
    for row in v["characters"].as_array().into_iter().flatten() {
        let r = ints(row);
        o.characters.insert(r[0], rates(&r[1..]));
    }
    for row in v["skill_hit"].as_array().into_iter().flatten() {
        let r = ints(row);
        o.skill_hit.entry(r[0]).or_default().insert(r[1], r[2]);
    }
    Some(o)
}

fn snapshot(units: &[Unit]) -> Vec<Value> {
    units.iter().map(unit_val).collect()
}

fn changed(before: &[Value], units: &[Unit]) -> Value {
    let mut out = serde_json::Map::new();
    for (k, u) in units.iter().enumerate() {
        let v = unit_val(u);
        if v != before[k] {
            out.insert(k.to_string(), v);
        }
    }
    Value::Object(out)
}

/// `State::native_growth_roll_observer` 的閉包型別。
type Observer = dyn Fn(&Unit, u16) -> u16 + Send + Sync;

struct Fixture {
    w: i64,
    h: i64,
    events: Vec<u8>,
    codes: Vec<u8>,
    cost: Option<Vec<i64>>,
    units: Vec<Value>,
}

struct Ctx {
    repo: PathBuf,
    fixtures: Vec<Fixture>,
    item_rows: Vec<u8>,
    spells: Vec<Spell>,
    growth_rows: BTreeMap<i64, GrowthRow>,
    growth_slots: BTreeMap<i64, GrowthRow>,
    learn: BTreeMap<i64, Vec<CommandLearnEntry>>,
    selectors: BTreeMap<i64, i64>,
    tmp: PathBuf,
}

impl Ctx {
    fn real(&self, rel: &str) -> String {
        self.repo.join("remake").join(rel).to_string_lossy().into_owned()
    }

    fn units(&self, sp: &Value) -> Vec<Unit> {
        let fx = &self.fixtures[i(sp, "fixture") as usize];
        let mut maps = fx.units.clone();
        for ed in sp["edits"].as_array().into_iter().flatten() {
            let unit = i(ed, "unit") as usize;
            for (k, v) in ed["set"].as_object().unwrap() {
                maps[unit][k] = v.clone();
            }
        }
        maps.iter().map(unit_from).collect()
    }

    /// Go `buildState`：鍵的意思見 Go 端同名函式。
    fn state(&self, sp: &Value) -> State {
        let fx = &self.fixtures[i(sp, "fixture") as usize];
        let mut st = State::default();
        st.w = fx.w;
        st.h = fx.h;
        st.units = self.units(sp);
        st.native_composition_event_bytes = fx.events.clone();
        st.native_terrain_move_codes = fx.codes.clone();
        st.cost = fx.cost.clone();
        if !b(sp, "no_items") {
            st.bind_native_future_item_rows(&self.item_rows).unwrap();
        }
        match sp["growth"].as_str() {
            Some("rows") => st.native_growth_rows = Some(self.growth_rows.clone()),
            Some("slots") => st.native_growth_rows = Some(self.growth_slots.clone()),
            _ => {}
        }
        if b(sp, "learn") {
            st.command_learn = Some(self.learn.clone());
            st.command_learn_selectors = Some(self.selectors.clone());
        }
        if b(sp, "cancel") {
            let f: Arc<dyn Fn(&Unit) -> bool + Send + Sync> = Arc::new(|u: &Unit| u.lv % 3 == 0);
            st.native_death_exp_cancel = Some(Callback::new(f));
        }
        if b(sp, "observer") {
            let f: Arc<Observer> = Arc::new(|u: &Unit, s: u16| s.wrapping_add(u.lv as u16));
            st.native_growth_roll_observer = Some(Callback::new(f));
        }
        st.balance = balance_from(&sp["balance"]);
        if let Some(acc) = sp["acc"].as_i64() {
            st.native_experience_accumulator = acc;
        }
        if b(sp, "spellbook") {
            st.spell_book = self.spells.clone();
        }
        for kv in sp["ai_spell"].as_array().into_iter().flatten() {
            st.ai_command_spell.insert(kv[0].as_i64().unwrap(), kv[1].as_i64().unwrap());
        }
        st
    }
}

fn rng_next(r: &mut GoMathRand) -> i64 {
    r.int63()
}

fn unit_arg(sp: &Value, key: &str) -> Option<UnitIdx> {
    Some(i(sp, key) as usize)
}

// ---- 各 op ----

fn attack_rng(ctx: &Ctx, sp: &Value) -> Value {
    let mut st = ctx.state(sp);
    let before = snapshot(&st.units);
    let mut r = GoMathRand::new(i(sp, "seed"));
    let res = st.attack_with_rng(unit_arg(sp, "a"), i(sp, "d") as usize, &mut r);
    json!({"result": attack_val(&res), "changed": changed(&before, &st.units), "rng_next": rng_next(&mut r)})
}

fn attack_native_exp(ctx: &Ctx, sp: &Value) -> Value {
    let mut st = ctx.state(sp);
    let before = snapshot(&st.units);
    let mut r = GoMathRand::new(i(sp, "seed"));
    let res = st.attack_with_native_experience(unit_arg(sp, "a"), unit_arg(sp, "d"), Some(&mut r));
    json!({"result": attack_val(res.as_ref().unwrap_or(&AttackResult::default())), "err": err_of(&res),
        "changed": changed(&before, &st.units), "rng_next": rng_next(&mut r)})
}

fn attack_native_physical(ctx: &Ctx, sp: &Value) -> Value {
    let mut st = ctx.state(sp);
    let before = snapshot(&st.units);
    let mut r = GoMathRand::new(i(sp, "seed"));
    let (a, d, state) = (unit_arg(sp, "a"), unit_arg(sp, "d"), i(sp, "state") as u16);
    let res = if b(sp, "with_exp") {
        st.attack_native_physical_with_experience(a, d, state, Some(&mut r))
    } else {
        st.attack_native_physical(a, d, state, &mut r)
    };
    let eligible = st.native_physical_counterattack_eligible(a, d);
    json!({"result": physical_val(res.as_ref().unwrap_or(&NativePhysicalAttackResult::default())),
        "err": err_of(&res), "changed": changed(&before, &st.units), "acc": st.native_experience_accumulator,
        "eligible_after": eligible, "rng_next": rng_next(&mut r)})
}

fn counter_eligible(ctx: &Ctx, sp: &Value) -> Value {
    let st = ctx.state(sp);
    json!({"eligible": st.native_physical_counterattack_eligible(unit_arg(sp, "a"), unit_arg(sp, "d"))})
}

fn player_attack_field(ctx: &Ctx, sp: &Value) -> Value {
    let st = ctx.state(sp);
    let res = st.native_player_attack_target_field(unit_arg(sp, "u"));
    json!({"field": hex(res.as_deref().unwrap_or_default()), "err": err_of(&res)})
}

fn terrain_cells(ctx: &Ctx, sp: &Value) -> Value {
    let mut sp = sp.clone();
    sp["no_items"] = json!(true);
    let st = ctx.state(&sp);
    let mut cells = Vec::new();
    for y in -1..=st.h {
        for x in -1..=st.w {
            let (ap, dp) = st.terrain_apdp_pct(x, y);
            cells.push(json!([ap, dp]));
        }
    }
    json!({"cells": cells})
}

fn cast_area(ctx: &Ctx, sp: &Value) -> Value {
    let mut st = ctx.state(sp);
    let before = snapshot(&st.units);
    let spell = spell_from(&sp["spell"]);
    let caster = i(sp, "caster") as usize;
    let in_range = st.in_cast_range(caster, &spell, i(sp, "tx"), i(sp, "ty"));
    let mut r = GoMathRand::new(i(sp, "seed"));
    let res = st.cast_area(Some(caster), i(sp, "cx"), i(sp, "cy"), &spell, &mut r);
    json!({"results": res.iter().map(cast_val).collect::<Vec<_>>(), "nil": res.is_empty(), "in_range": in_range,
        "changed": changed(&before, &st.units), "rng_next": rng_next(&mut r)})
}

fn ai_spells(ctx: &Ctx, sp: &Value) -> Value {
    let st = ctx.state(sp);
    let u = unit_arg(sp, "u");
    let available = st.ai_available_spells(u);
    let candidates = st.ai_spell_candidates(u, &spell_from(&sp["spell"]));
    json!({"available": available.iter().map(spell_val).collect::<Vec<_>>(),
        "candidates": candidates.iter().map(|&x| x as i64).collect::<Vec<_>>()})
}

fn result_op(ctx: &Ctx, sp: &Value) -> Value {
    json!({"result": ctx.state(sp).result(s(sp, "protect"))})
}

fn growth(ctx: &Ctx, op: &str, sp: &Value) -> Value {
    let mut st = ctx.state(sp);
    let before = snapshot(&st.units);
    let u = unit_arg(sp, "u");
    let (row, ok) = st.native_growth_row_for(u);
    let amount = i(sp, "amount");
    let mut r = GoMathRand::new(i(sp, "seed"));
    match op {
        "award_exp" => {
            let famount = sp["famount"].as_f64().unwrap();
            let (got, ups) = st.award_exp(u, famount, &mut r);
            json!({"got": got, "ups": level_ups_val(&ups), "row": growth_row_val(&row), "row_ok": ok,
                "changed": changed(&before, &st.units), "rng_next": rng_next(&mut r)})
        }
        "state_gain_exp" => {
            let ups = st.gain_exp(u, amount as f64, &mut r);
            json!({"ups": level_ups_val(&ups), "row": growth_row_val(&row), "row_ok": ok,
                "changed": changed(&before, &st.units), "rng_next": rng_next(&mut r)})
        }
        "award_exp_native" => {
            let (got, ups, next) = st.award_exp_native(u, amount, i(sp, "state") as u16);
            json!({"got": got, "ups": level_ups_val(&ups), "next": next, "row": growth_row_val(&row), "row_ok": ok,
                "changed": changed(&before, &st.units)})
        }
        _ => {
            let ups = gain_exp(Some(&mut st.units[u.unwrap()]), amount as f64, &mut r);
            json!({"ups": level_ups_val(&ups), "changed": changed(&before, &st.units), "rng_next": rng_next(&mut r)})
        }
    }
}

fn roll_physical(input: &Value) -> Value {
    let v = &input["in"];
    let w = &v["Weapon"];
    let roll = NativePhysicalRoll {
        attacker_ap: i(v, "AttackerAP"),
        defender_dp: i(v, "DefenderDP"),
        attacker_hit: i(v, "AttackerHit"),
        defender_ev: i(v, "DefenderEV"),
        attacker_crit_pct: i(v, "AttackerCritPct"),
        attacker_terrain_ap_pct: i(v, "AttackerTerrainAPPct"),
        defender_terrain_dp_pct: i(v, "DefenderTerrainDPPct"),
        weapon: NativePhysicalWeapon { effect: i(w, "Effect"), param: i(w, "Param"), reach: i(w, "Reach") },
        rng_state: i(v, "RNGState") as u16,
    };
    let res = roll_native_physical_damage(roll);
    json!({"result": roll_result_val(res.as_ref().unwrap_or(&NativePhysicalRollResult::default())), "err": err_of(&res)})
}

fn counter_raw(input: &Value) -> Value {
    let d = &input["d"];
    let def = NativeCounterattackDefender {
        x: i(d, "X"),
        y: i(d, "Y"),
        byte38: i(d, "Byte38"),
        has_weapon: b(d, "HasWeapon"),
        weapon_reach: i(d, "WeaponReach"),
        alive_hp: i(d, "AliveHP"),
    };
    json!({"eligible": native_counterattack_eligible(i(input, "ax"), i(input, "ay"), def)})
}

fn terrain_code(input: &Value) -> Value {
    let (ap, dp, ok) = native_terrain_apdp_pct(i(input, "code") as u8);
    json!({"ap": ap, "dp": dp, "ok": ok})
}

fn exp_formula(input: &Value) -> Value {
    let a = ints(&input["args"]);
    json!({"attack": attack_exp(a[0], a[1], a[2], a[3], a[4]), "teleport": teleport_exp(a[0], a[1]),
        "action": action_exp(a[0], a[1])})
}

fn battle_result(input: &Value) -> Value {
    let res = native_battle_result_code205_b4(&unhex(s(input, "records")), i(input, "count"));
    json!({"code": res.as_ref().copied().unwrap_or(0), "err": err_of(&res)})
}

fn load_growth_at(path: &str) -> Value {
    let rows = load_native_growth_rows(path);
    let slots = load_native_growth_slots(path);
    json!({"rows": growth_rows_val(rows.as_ref().unwrap_or(&BTreeMap::new())), "err": err_of(&rows),
        "slots": growth_rows_val(slots.as_ref().unwrap_or(&BTreeMap::new())), "slots_err": err_of(&slots)})
}

/// 損壞 JSON 的錯誤文字屬標準函式庫（[`KNOWN_DIFFERENCES`]），只比接受或拒收；成長表自己的錯誤逐字比。
fn own_growth_error(v: &Value) -> bool {
    v.as_str().is_some_and(|e| e.starts_with("native growth"))
}

fn normalize_growth_errors(go: &mut Value, rust: &mut Value) {
    for key in ["err", "slots_err"] {
        if !go[key].is_null() && !own_growth_error(&go[key]) && !rust[key].is_null() {
            go[key] = json!("JSON 解碼錯誤");
            rust[key] = json!("JSON 解碼錯誤");
        }
    }
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_E").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let (Some(path), Some(repo)) = (path, repo) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_RULES_E 或 FD2_GOLDEN_REPO");
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_E／FD2_GOLDEN_REPO");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let fixtures = doc["fixtures"]
        .as_array()
        .unwrap()
        .iter()
        .map(|f| Fixture {
            w: i(f, "w"),
            h: i(f, "h"),
            events: unhex(s(f, "events")),
            codes: unhex(s(f, "codes")),
            cost: (!f["cost"].is_null()).then(|| ints(&f["cost"])),
            units: f["units"].as_array().unwrap().clone(),
        })
        .collect();
    let real = |rel: &str| repo.join("remake").join(rel).to_string_lossy().into_owned();
    let ctx = Ctx {
        fixtures,
        item_rows: load_native_item_effect_row_prefix(&real("assets/data/native_item_effect_rows.json")).unwrap(),
        spells: load_spells(&real("assets/spells.json")).unwrap(),
        growth_rows: load_native_growth_rows(&real("assets/data/class_change_growth.json")).unwrap(),
        growth_slots: load_native_growth_slots(&real("assets/data/native_growth_slots.json")).unwrap(),
        learn: load_command_learn(&real("assets/data/command_learn.json")).unwrap(),
        selectors: load_command_learn_selectors(&real("assets/data/class_change_growth.json")).unwrap(),
        tmp: std::env::temp_dir().join(format!("fd2-golden-rules-e-{}", std::process::id())),
        repo,
    };
    let _ = std::fs::remove_dir_all(&ctx.tmp);
    std::fs::create_dir_all(&ctx.tmp).unwrap();
    Some((doc, ctx))
}

#[test]
fn golden_rules_e() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-e/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    for file in doc["real"]["files"].as_array().unwrap() {
        let raw = std::fs::read(ctx.real(s(file, "path"))).unwrap_or_else(|e| panic!("讀不到 {}：{e}", file["path"]));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    // 黃金檔內嵌的物品列與法術表，必須與 Rust 端自己載入真實檔的結果相同。
    assert_eq!(hex(&ctx.item_rows), s(&doc["real"], "item_rows"), "物品效果列");
    check("法術表", &doc["real"]["spells"], &json!(ctx.spells.iter().map(spell_val).collect::<Vec<_>>()));

    let mut counts = BTreeMap::<String, usize>::new();
    for case in doc["cases"].as_array().unwrap() {
        let op = s(case, "op");
        let name = format!("case {} {op}", case["index"]);
        let (input, result) = (&case["input"], &case["result"]);
        let mut go = result.clone();
        let mut rust = match op {
            "attack_rng" => attack_rng(&ctx, input),
            "attack_native_exp" => attack_native_exp(&ctx, input),
            "attack_native_physical" => attack_native_physical(&ctx, input),
            "counter_eligible" => counter_eligible(&ctx, input),
            "player_attack_field" => player_attack_field(&ctx, input),
            "terrain_cells" => terrain_cells(&ctx, input),
            "cast_area" => cast_area(&ctx, input),
            "ai_spells" => ai_spells(&ctx, input),
            "result" => result_op(&ctx, input),
            "award_exp" | "state_gain_exp" | "award_exp_native" | "gain_exp" => growth(&ctx, op, input),
            "roll_physical" => roll_physical(input),
            "counter_raw" => counter_raw(input),
            "terrain_code" => terrain_code(input),
            "exp_formula" => exp_formula(input),
            "battle_result" => battle_result(input),
            "load_growth" => load_growth_at(&ctx.real(s(input, "path"))),
            "load_growth_text" => {
                let path = ctx.tmp.join(s(input, "name"));
                std::fs::write(&path, s(input, "text")).unwrap();
                load_growth_at(&path.to_string_lossy())
            }
            "load_spells" => {
                let res = load_spells(&ctx.real(s(input, "path")));
                json!({"spells": res.as_deref().unwrap_or_default().iter().map(spell_val).collect::<Vec<_>>(),
                    "err": err_of(&res)})
            }
            other => panic!("未知的 op {other}"),
        };
        if op.starts_with("load_growth") {
            normalize_growth_errors(&mut go, &mut rust);
        }
        check(&name, &go, &rust);
        *counts.entry(op.to_string()).or_default() += 1;
    }
    let total: usize = counts.values().sum();
    eprintln!("黃金案例 {total}：{counts:?}");
    assert!(total > 0, "沒有任何黃金案例");
    let _ = std::fs::remove_dir_all(&ctx.tmp);
}
