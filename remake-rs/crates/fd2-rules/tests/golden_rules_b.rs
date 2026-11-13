//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-b` 的黃金差分：fd2-rules B 組（指令、目標與死亡程式）。
//!
//! 由 `tools/rust_golden_rules_b.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_B`；真實輸入（受版控的
//! `remake/assets`）掛在 `FD2_GOLDEN_REPO` 底下，先核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：戰場由黃金檔的「底稿＋逐單位修改」組成，兩邊都把合併後的 JSON 物件解碼成 `Unit`。每個
//! 案例依 `op` 在 Rust 端重做同一串呼叫，整個結果物件與 Go 逐欄比對（鍵集合也要相同）：`*Unit` 是
//! `State::units` 的索引（nil 為 -1），結構以 Go 欄位名輸出且只含 Go 匯出欄位，單位以 Go 欄位名的
//! JSON 比對。B 組與 A 組自己的錯誤文字逐字相同；Go 標準函式庫產生的錯誤（JSON 解碼）只比接受或
//! 拒收。比對時的表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。

use fd2_formats::native_rng::native_rng_step;
use fd2_rules::battle::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為（PORTING.md 決定 2、`rust-port-no-go-stdlib`）。黃金案例都不落在這些輸入上。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 可以把不在 State.Units 裡的 *Unit 或 nil 接收者傳進來；Rust 以索引表示單位，黃金只傳 State.Units
    // 裡的單位或 nil（Rust 的 None）。
    "不在 State.Units 的單位指標與 nil 接收者",
    // Go 的 NativeActionWhen.Match 遇到 any_active 起點為負時索引越界 panic；黃金只放非負起點。
    "any_active 負起點的 panic",
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文。
    "JSON 欄位名大小寫",
    // LoadNativeCommandResistances 的 JSON 解碼錯誤屬標準函式庫，只比接受或拒收。
    "抗性表 JSON 解碼錯誤文字",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列、空字串相同",
    // Go 把整數值的 float64 寫成 0，serde_json 寫成 0.0；只比數值。
    "數字只比數值",
    // 交易 plan 只比 Go 匯出欄位（Rust 的 pub 欄位）；未匯出的進度旗標經由後續步驟的錯誤與單位狀態比對。
    "plan 只比匯出欄位",
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
    v[key].as_bool().unwrap_or_else(|| panic!("欄位 {key} 不是布林：{}", v[key]))
}

fn s<'a>(v: &'a Value, key: &str) -> &'a str {
    v[key].as_str().unwrap_or_else(|| panic!("欄位 {key} 不是字串：{}", v[key]))
}

fn ints(v: &Value) -> Vec<i64> {
    v.as_array().map(|a| a.iter().map(|x| x.as_i64().unwrap()).collect()).unwrap_or_default()
}

fn cell(v: &Value) -> Cell {
    let a = ints(v);
    Cell { x: a[0], y: a[1] }
}

fn cell_val(c: Cell) -> Value {
    json!([c.x, c.y])
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

fn idxs(v: &[UnitIdx]) -> Value {
    json!(v.iter().map(|&x| x as i64).collect::<Vec<_>>())
}

fn opt_idx(v: i64) -> Option<UnitIdx> {
    (v >= 0).then_some(v as usize)
}

// ---- Go 結構的 JSON（沒有 json 標籤，鍵是 Go 欄位名）----

fn damage_val(d: &NativeCommandDamage) -> Value {
    json!({"Hit": d.hit, "Damage": d.damage})
}

fn damage_result_val(r: &NativeCommandDamageResult) -> Value {
    json!({"Target": idx(r.target), "NativeCommandDamage": damage_val(&r.native_command_damage),
        "HPBefore": r.hp_before, "HPAfter": r.hp_after})
}

fn damage_plan_val(p: &NativeCommandDamagePlan) -> Value {
    json!({"Actor": idx(p.actor), "CommandID": p.command_id, "MPBefore": p.mp_before, "MPAfter": p.mp_after,
        "ActorActedBefore": p.actor_acted_before, "RNGBefore": p.rng_before, "RNGAfter": p.rng_after,
        "Results": p.results.iter().map(damage_result_val).collect::<Vec<_>>(), "DamageStages": p.damage_stages})
}

fn restore_val(r: &NativeCommandRestore) -> Value {
    json!({"Rolled": r.rolled, "Actual": r.actual})
}

fn heal_val(r: &NativeCommandHealResult) -> Value {
    json!({"Target": idx(r.target), "Restore": restore_val(&r.restore)})
}

fn word_step_val(r: &NativeRawWordStepResult) -> Value {
    json!({"UnitIndex": r.unit_index, "Processed": r.processed, "Marker": r.marker, "Delta": r.delta})
}

fn pair_step_val(r: &NativeRawPairStepResult) -> Value {
    json!({"UnitIndex": r.unit_index, "Processed": r.processed, "Marker": r.marker})
}

fn modifier_val(r: &NativeCommandModifierResult) -> Value {
    json!({"CommandID": r.command_id, "WordSteps": r.word_steps.iter().map(word_step_val).collect::<Vec<_>>(),
        "PairSteps": r.pair_steps.iter().map(pair_step_val).collect::<Vec<_>>(), "RNGState": r.rng_state,
        "Accumulator": r.accumulator})
}

fn state34_val(x: &NativeCommand34TargetState) -> Value {
    json!({"Target": idx(x.target), "NativeTransient": x.native_transient, "AP": x.ap, "DP": x.dp, "HIT": x.hit, "EV": x.ev})
}

fn modifier_plan_val(p: &NativeAICommandModifierPlan) -> Value {
    json!({"Actor": idx(p.actor), "CommandID": p.command_id, "Targets": idxs(&p.targets), "Result": modifier_val(&p.result),
        "After": p.after.iter().map(state34_val).collect::<Vec<_>>()})
}

fn clear_restore_val(r: &NativeCommandClearRestoreResult) -> Value {
    json!({"Target": idx(r.target), "Offset": r.offset, "Cleared": r.cleared, "Restore": restore_val(&r.restore)})
}

fn raw_restore_val(r: &NativeRawRestoreResult) -> Value {
    json!({"UnitIndex": r.unit_index, "Rolled": r.rolled, "Actual": r.actual, "Score": r.score, "RNGState": r.rng_state})
}

fn raw_damage_val(r: &NativeRawDamageResult) -> Value {
    json!({"UnitIndex": r.unit_index, "Rolled": r.rolled, "Actual": r.actual, "RNGState": r.rng_state})
}

fn raw_application_val(r: &NativeRawApplicationResult) -> Value {
    json!({"TargetIndex": r.target_index, "Applied": r.applied, "Damage": raw_damage_val(&r.damage), "Marker": r.marker})
}

fn result2022_val(r: &NativeAICommand2022Result) -> Value {
    let restore = r
        .restore
        .as_ref()
        .map(|x| json!({"TargetIndex": x.target_index, "Applied": x.applied, "Restore": raw_restore_val(&x.restore)}));
    json!({"Target": idx(r.target), "Offset": r.offset, "Command25": r.command25, "ClearedByte5": r.cleared_byte5,
        "Restore": restore, "Apply": r.apply.as_ref().map(raw_application_val)})
}

fn state2022_val(x: &NativeAICommand2022TargetState) -> Value {
    json!({"Target": idx(x.target), "HP": x.hp, "NativeRecordByte5": x.native_record_byte5, "NativeTransient": x.native_transient})
}

fn plan2022_val(p: &NativeAICommand2022Plan) -> Value {
    json!({"Actor": idx(p.actor), "CommandID": p.command_id, "Targets": idxs(&p.targets),
        "Results": p.results.iter().map(result2022_val).collect::<Vec<_>>(), "RNGState": p.rng_state,
        "Before": p.before.iter().map(state2022_val).collect::<Vec<_>>(),
        "After": p.after.iter().map(state2022_val).collect::<Vec<_>>()})
}

fn c25_val(r: &NativeCommand25Result) -> Value {
    json!({"Target": idx(r.target), "Cleared": r.cleared})
}

fn app_val(r: &NativeCommandApplicationResult) -> Value {
    json!({"Target": idx(r.target), "Offset": r.offset, "Applied": r.applied, "Damage": r.damage, "Duration": r.duration})
}

fn c32_result_val(r: &NativeCompoundCommand32Result) -> Value {
    let targets: Vec<Value> = r
        .targets
        .iter()
        .map(|t| {
            json!({"Target": idx(t.target), "Damage": damage_val(&t.damage), "HPBefore": t.hp_before, "HPAfter": t.hp_after})
        })
        .collect();
    json!({"Targets": targets, "RNGState": r.rng_state})
}

fn c32_plan_val(p: &NativeCompoundCommand32Plan) -> Value {
    json!({"Actor": idx(p.actor), "Result": c32_result_val(&p.result), "ActorActedBefore": p.actor_acted_before,
        "RNGBefore": p.rng_before})
}

fn c33_result_val(r: &NativeCompoundCommand33Result) -> Value {
    let targets: Vec<Value> = r
        .targets
        .iter()
        .map(|t| {
            json!({"Target": idx(t.target), "HPBefore": t.hp_before, "HPAfter": t.hp_after,
                "TransientBefore": t.transient_before, "TransientAfter": t.transient_after})
        })
        .collect();
    json!({"Restore": {"Results": r.restore.results.iter().map(raw_restore_val).collect::<Vec<_>>(),
        "RNGState": r.restore.rng_state, "Score": r.restore.score}, "Targets": targets})
}

fn c33_plan_val(p: &NativeCompoundCommand33Plan) -> Value {
    json!({"Actor": idx(p.actor), "Result": c33_result_val(&p.result), "ActorActedBefore": p.actor_acted_before,
        "RNGBefore": p.rng_before})
}

fn c34_result_val(r: &NativeCompoundCommand34Result) -> Value {
    json!({"Stages": r.stages.iter().map(modifier_val).collect::<Vec<_>>(),
        "StageStates": r.stage_states.iter().map(|row| row.iter().map(state34_val).collect::<Vec<_>>()).collect::<Vec<_>>(),
        "RNGState": r.rng_state})
}

fn c35_result_val(r: &NativeCompoundCommand35Result) -> Value {
    let stages: Vec<Value> = r
        .stages
        .iter()
        .map(|x| {
            json!({"CommandID": x.command_id, "MarkerOffset": x.marker_offset,
                "Results": x.results.iter().map(raw_application_val).collect::<Vec<_>>(),
                "Accumulator": x.accumulator, "RNGState": x.rng_state})
        })
        .collect();
    let states: Vec<Vec<Value>> = r
        .stage_states
        .iter()
        .map(|row| {
            row.iter()
                .map(|x| json!({"Target": idx(x.target), "HP": x.hp, "NativeTransient": x.native_transient}))
                .collect()
        })
        .collect();
    json!({"Stages": stages, "StageStates": states, "RNGState": r.rng_state})
}

fn command_val(r: &NativeCommandRecord) -> Value {
    json!({"ID": r.id, "Damage": r.damage, "Hit": r.hit, "SelectionMode": r.selection_mode,
        "EffectMode": r.effect_mode, "MPCost": r.mp_cost, "TargetCode": r.target_code})
}

fn command_from(v: &Value) -> NativeCommandRecord {
    NativeCommandRecord {
        id: i(v, "ID"),
        damage: i(v, "Damage"),
        hit: i(v, "Hit"),
        selection_mode: i(v, "SelectionMode"),
        effect_mode: i(v, "EffectMode"),
        mp_cost: i(v, "MPCost"),
        target_code: i(v, "TargetCode"),
    }
}

fn stats_val(x: &BalanceStats) -> Value {
    json!({"HP": x.hp, "MP": x.mp, "AP": x.ap, "DP": x.dp, "DX": x.dx})
}

fn view_arr(v: NativeMapViewState) -> Value {
    json!([v.camera_x, v.camera_y, v.cursor_x, v.cursor_y, v.visible_cursor_x, v.visible_cursor_y])
}

fn view_from(v: &Value) -> NativeMapViewState {
    let a = ints(v);
    NativeMapViewState {
        camera_x: a[0],
        camera_y: a[1],
        cursor_x: a[2],
        cursor_y: a[3],
        visible_cursor_x: a[4],
        visible_cursor_y: a[5],
    }
}

fn cells_of(m: &BTreeMap<Cell, bool>) -> Value {
    let mut out: Vec<Cell> = m.iter().filter(|(_, ok)| **ok).map(|(c, _)| *c).collect();
    out.sort_by_key(|c| (c.y, c.x));
    json!(out.into_iter().map(cell_val).collect::<Vec<_>>())
}

// ---- 黃金檔與戰場組裝 ----

struct Fixture {
    w: i64,
    h: i64,
    events: Vec<u8>,
    cost: Option<Vec<i64>>,
    units: Vec<Value>,
}

struct Ctx {
    repo: PathBuf,
    fixtures: Vec<Fixture>,
    book: Vec<NativeCommandRecord>,
    resist: BTreeMap<i64, i64>,
    tmp: PathBuf,
}

impl Ctx {
    fn real(&self, rel: &str) -> String {
        self.repo.join("remake").join(rel).to_string_lossy().into_owned()
    }

    fn temp_file(&self, name: &str, text: &str) -> String {
        let path = self.tmp.join(name);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, text).unwrap();
        path.to_string_lossy().into_owned()
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

    fn book(&self, sp: &Value) -> Vec<NativeCommandRecord> {
        let mut book = self.book.clone();
        if let Some(n) = sp["book_len"].as_i64() {
            while (book.len() as i64) < n {
                book.push(NativeCommandRecord { id: book.len() as i64, ..Default::default() });
            }
            book.truncate(n as usize);
        }
        for ed in sp["book_edits"].as_array().into_iter().flatten() {
            let (id, field, value) =
                (ed[0].as_i64().unwrap() as usize, ed[1].as_str().unwrap(), ed[2].as_i64().unwrap());
            let Some(r) = book.get_mut(id) else { continue };
            match field {
                "ID" => r.id = value,
                "Damage" => r.damage = value,
                "Hit" => r.hit = value,
                "SelectionMode" => r.selection_mode = value,
                "EffectMode" => r.effect_mode = value,
                "MPCost" => r.mp_cost = value,
                "TargetCode" => r.target_code = value,
                other => panic!("未知的指令欄位 {other}"),
            }
        }
        book
    }

    fn state(&self, sp: &Value) -> State {
        let fx = &self.fixtures[i(sp, "fixture") as usize];
        let mut st = State::default();
        st.w = fx.w;
        st.h = fx.h;
        st.units = self.units(sp);
        st.native_composition_event_bytes = fx.events.clone();
        st.cost = fx.cost.clone();
        st.native_command_book = self.book(sp);
        st.native_command_resistances = self.resist.clone();
        for k in ints(&sp["resist_drop"]) {
            st.native_command_resistances.remove(&k);
        }
        for kv in sp["resist_set"].as_array().into_iter().flatten() {
            st.native_command_resistances.insert(kv[0].as_i64().unwrap(), kv[1].as_i64().unwrap());
        }
        st.balance = balance_from(&sp["balance"]);
        st
    }
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

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_B").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let (Some(path), Some(repo)) = (path, repo) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_RULES_B 或 FD2_GOLDEN_REPO");
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_B／FD2_GOLDEN_REPO");
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
            cost: (!f["cost"].is_null()).then(|| ints(&f["cost"])),
            units: f["units"].as_array().unwrap().clone(),
        })
        .collect();
    let book = doc["real"]["book"].as_array().unwrap().iter().map(command_from).collect();
    let resist = doc["real"]["resistances"]
        .as_array()
        .unwrap()
        .iter()
        .map(|kv| (kv[0].as_i64().unwrap(), kv[1].as_i64().unwrap()))
        .collect();
    let tmp = std::env::temp_dir().join(format!("fd2-golden-rules-b-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&tmp);
    Some((doc, Ctx { repo, fixtures, book, resist, tmp }))
}

#[test]
fn golden_rules_b() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-b/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    for file in doc["real"]["files"].as_array().unwrap() {
        let raw = std::fs::read(ctx.real(s(file, "path"))).unwrap_or_else(|e| panic!("讀不到 {}：{e}", file["path"]));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    // 黃金檔內嵌的指令表、抗性表與移動成本列，必須與 Rust 端自己載入真實檔的結果相同。
    let book = load_native_command_records(&ctx.real("assets/spells.json")).unwrap();
    check("指令表", &doc["real"]["book"], &json!(book.iter().map(command_val).collect::<Vec<_>>()));
    let resist = load_native_command_resistances(&ctx.real("assets/data/native_command_resistances.json")).unwrap();
    check("抗性表", &doc["real"]["resistances"], &json!(resist.iter().map(|(k, v)| [*k, *v]).collect::<Vec<_>>()));
    let rows = load_native_movement_cost_rows(&ctx.real("assets/data/native_movement_cost_rows.json")).unwrap();
    check("移動成本列", &doc["real"]["cost_rows"], &json!(rows.iter().map(|r| hex(r)).collect::<Vec<_>>()));
    // 物品效果列（C 組 native_item_table 載入）只在 Go 端挑選列；案例輸入直接帶列的位元組，這裡只核對檔案雜湊。

    let mut counts = BTreeMap::<String, usize>::new();
    for case in doc["cases"].as_array().unwrap() {
        let op = s(case, "op");
        let name = format!("case {} {op}", case["index"]);
        let (input, result) = (&case["input"], &case["result"]);
        let rust = match op {
            "composition_flags" => composition_flags(input),
            "target_cells" => target_cells(input),
            "target_matches" => target_matches(),
            "compound_plan" => compound_plan(),
            "overlay_selector" => overlay_selector(),
            "resolve_damage" => resolve_damage(input),
            "apply_damage" => apply_damage(&ctx, input),
            "restore" => restore(&ctx, input),
            "modifier_raw" => modifier_raw(input),
            "load_resistances" => load_resistances(&ctx, input, result, case["index"].as_i64().unwrap()),
            "select_destination" => select_destination(input),
            "nearest_opposite" => nearest_opposite(input),
            "blocked_coordinate" | "path_directions" => paths(op, input),
            "scored_groups" => scored_groups(input),
            "map_view" => map_view(input),
            "balance_validate" => balance_validate(input),
            "hit_roll" => hit_roll(),
            "balance_identity" => balance_identity_case(&ctx, input),
            "balance_stats" => balance_stats(&ctx, input),
            "enemy_strength" => enemy_strength(&ctx, input),
            "move" => movement(&ctx, input),
            "death_program" => death_program(&ctx, input),
            "when_match" => when_match(&ctx, input),
            "death_ops" => death_ops(&ctx, input),
            "runtime_flags"
            | "future_placement"
            | "cursor_confirm"
            | "command_targets"
            | "attack_candidates"
            | "effect_targets"
            | "effect_targets_cursor"
            | "self_centered"
            | "command30"
            | "item_effect_targets"
            | "base_flags" => targets(&ctx, op, input),
            "command" => command(&ctx, input),
            "nil_plans" => nil_plans(),
            _ => panic!("{name}：未知的 op"),
        };
        check(&name, result, &rust);
        let key = if op == "command" { format!("command/{}", s(input, "kind")) } else { op.to_string() };
        *counts.entry(key).or_default() += 1;
    }
    let _ = std::fs::remove_dir_all(&ctx.tmp);
    let total: usize = counts.values().sum();
    eprintln!("rules-b 黃金差分：{total} 案例全部相符 {counts:?}");
}

// ---- 純函式 ----

fn composition_flags(input: &Value) -> Value {
    let r = native_composition_base_flags(i(input, "w"), i(input, "h"), &unhex(s(input, "events")));
    json!({"flags": hex(r.as_deref().unwrap_or_default()), "err": err_of(&r)})
}

fn target_cells(input: &Value) -> Value {
    let (w, h, origin, dist, inner) =
        (i(input, "w"), i(input, "h"), cell(&input["origin"]), i(input, "dist"), i(input, "inner"));
    let flags = unhex(s(input, "flags"));
    let cells = native_command_target_cells(w, h, origin, dist, &flags);
    let field = native_command_target_field_bytes(w, h, origin, dist, inner, &flags);
    json!({"cells": cells.as_ref().map(cells_of).unwrap_or(Value::Null), "err": err_of(&cells),
        "field": hex(field.as_deref().unwrap_or_default()), "field_err": err_of(&field)})
}

fn target_matches() -> Value {
    let rows: Vec<Vec<bool>> =
        (-1..=6).map(|code| (0..3).map(|camp| native_command_target_matches(code, Camp(camp))).collect()).collect();
    json!({"matches": rows})
}

fn compound_plan() -> Value {
    let plans: Vec<Value> = (-1..=40)
        .map(|id| {
            let (steps, ok) = native_compound_command_plan(id);
            let steps: Vec<Value> = steps
                .iter()
                .map(|x| json!({"Callee": x.callee, "CommandID": x.command_id, "MarkerOffset": x.marker_offset, "Amount": x.amount}))
                .collect();
            json!({"steps": steps, "ok": ok})
        })
        .collect();
    json!({"plans": plans})
}

fn overlay_selector() -> Value {
    json!({"values": (0..=255u8).map(native_map_overlay_selector_from_record_byte).collect::<Vec<_>>()})
}

fn resolve_damage(input: &Value) -> Value {
    let rng = i(input, "rng") as u16;
    let r = resolve_native_command_damage(i(input, "damage"), i(input, "hit"), i(input, "resist"), rng);
    let (d, next) = r.clone().unwrap_or((NativeCommandDamage::default(), rng));
    json!({"result": damage_val(&d), "rng": next, "err": err_of(&r)})
}

fn apply_damage(ctx: &Ctx, input: &Value) -> Value {
    let mut units = ctx.units(&input["state"]);
    let before = snapshot(&units);
    let rng = i(input, "rng") as u16;
    let target = opt_idx(i(input, "target"));
    let r = apply_native_command_damage(
        target.map(|t| &mut units[t]),
        i(input, "damage"),
        i(input, "hit"),
        i(input, "resist"),
        rng,
    );
    let (d, next) = r.clone().unwrap_or((NativeCommandDamage::default(), rng));
    json!({"result": damage_val(&d), "rng": next, "err": err_of(&r), "changed": changed(&before, &units)})
}

fn restore(ctx: &Ctx, input: &Value) -> Value {
    let mut units = ctx.units(&input["state"]);
    let before = snapshot(&units);
    let target = opt_idx(i(input, "target"));
    let amount = i(input, "amount");
    let mut res = if b(input, "native") {
        let rng = i(input, "rng") as u16;
        let r = apply_native_command_restore_native(target.map(|t| &mut units[t]), amount, rng);
        let (x, next) = r.clone().unwrap_or((NativeCommandRestore::default(), rng));
        json!({"result": restore_val(&x), "rng": next, "err": err_of(&r)})
    } else {
        let mut source = GoMathRand::new(i(input, "seed"));
        let rng: Option<&mut dyn GoRand> = if b(input, "nil_rng") { None } else { Some(&mut source) };
        let r = apply_native_command_restore(target.map(|t| &mut units[t]), amount, rng);
        json!({"result": restore_val(&r.clone().unwrap_or_default()), "err": err_of(&r)})
    };
    res["changed"] = changed(&before, &units);
    res
}

fn modifier_raw(input: &Value) -> Value {
    let mut records = unhex(s(input, "records"));
    let indices: Vec<u8> = ints(&input["indices"]).into_iter().map(|x| x as u8).collect();
    let r = apply_native_command_modifier(&mut records, &indices, i(input, "command"), i(input, "rng") as u16);
    json!({"result": modifier_val(&r.clone().unwrap_or_default()), "err": err_of(&r), "records": hex(&records)})
}

fn load_resistances(ctx: &Ctx, input: &Value, go: &Value, index: i64) -> Value {
    let path = match input["path"].as_str() {
        Some(rel) => ctx.real(rel),
        None => ctx.temp_file(&format!("resist-{index}.json"), s(input, "text")),
    };
    let r = load_native_command_resistances(&path);
    let table: Vec<[i64; 2]> = r.as_ref().map(|t| t.iter().map(|(k, v)| [*k, *v]).collect()).unwrap_or_default();
    // JSON 解碼錯誤屬標準函式庫：Rust 只需同樣拒收，錯誤文字沿用 Go 的值。
    let err = if b(go, "json_err") {
        assert!(r.is_err(), "case {index}：Go 拒收的 JSON Rust 接受了");
        go["err"].clone()
    } else {
        err_of(&r)
    };
    json!({"table": table, "err": err, "json_err": b(go, "json_err")})
}

fn select_destination(input: &Value) -> Value {
    let cands: Vec<Cell> = input["candidates"].as_array().unwrap().iter().map(cell).collect();
    let (best, ok) = select_native_movement_destination(&cands, cell(&input["intended"]));
    json!({"best": cell_val(best), "found": ok})
}

fn nearest_opposite(input: &Value) -> Value {
    let r = select_native_nearest_opposite_coordinate(
        &unhex(s(input, "records")),
        i(input, "count"),
        i(input, "actor"),
        i(input, "selector"),
    );
    let (c, ok) = r.clone().unwrap_or_default();
    json!({"best": cell_val(c), "found": ok, "err": err_of(&r)})
}

fn paths(op: &str, input: &Value) -> Value {
    let (w, h, start, budget) = (i(input, "w"), i(input, "h"), cell(&input["start"]), i(input, "budget"));
    let flags = unhex(s(input, "flags"));
    let terrain = unhex(s(input, "terrain"));
    let row = unhex(s(input, "cost_row"));
    if op == "blocked_coordinate" {
        let r = native_path_blocked_coordinate(w, h, start, budget, &flags, &terrain, &row);
        let (c, ok) = r.clone().unwrap_or_default();
        return json!({"cell": cell_val(c), "found": ok, "err": err_of(&r)});
    }
    let r = native_path_directions(w, h, start, cell(&input["dest"]), budget, i(input, "mode"), &flags, &terrain, &row);
    let (p, ok) = r.clone().unwrap_or_default();
    json!({"path": p, "found": ok, "err": err_of(&r)})
}

fn scored_groups(input: &Value) -> Value {
    let r = native_ai_scored_command_candidate_groups(
        i(input, "w"),
        i(input, "h"),
        &unhex(s(input, "records")),
        i(input, "count"),
        i(input, "actor"),
        i(input, "selector"),
        command_from(&input["command"]),
        &unhex(s(input, "flags")),
        &unhex(s(input, "terrain")),
        &unhex(s(input, "cost_row")),
    );
    let groups: Vec<Value> = r
        .as_ref()
        .map(|g| {
            g.iter()
                .map(|x| json!({"Destination": {"X": x.destination.x, "Y": x.destination.y}, "TargetIndices": x.target_indices}))
                .collect()
        })
        .unwrap_or_default();
    json!({"groups": groups, "err": err_of(&r)})
}

// ---- 地圖視圖 ----

fn map_view(input: &Value) -> Value {
    let mut st = State::default();
    st.w = i(input, "w");
    st.h = i(input, "h");
    let mut outs = Vec::new();
    for op in input["ops"].as_array().unwrap() {
        let out = match s(op, "op") {
            "materialize" => json!({"err": err_of(&st.materialize_native_map_view_state(view_from(&op["view"])))}),
            "move" => {
                let (moved, ok) = st.move_native_map_cursor(i(op, "dx"), i(op, "dy"));
                json!({"moved": moved, "ok": ok})
            }
            "focus" => json!({"ok": st.focus_native_map_cursor(i(op, "x"), i(op, "y"))}),
            "focus_steps" => {
                // 重繪判定只讀 NativeMapRangeMode，聚焦過程不改它；用複本避免與 &mut st 重疊借用。
                let probe = st.clone();
                let mut steps = Vec::new();
                let mut on_step = |before: NativeMapViewState, after: NativeMapViewState| {
                    steps.push(json!({"before": view_arr(before), "after": view_arr(after),
                        "redraws": probe.native_map_cursor_step_redraws(before, after)}));
                };
                let ok = st.focus_native_map_cursor_steps(
                    i(op, "x"),
                    i(op, "y"),
                    Some(&mut on_step as &mut dyn FnMut(_, _)),
                );
                json!({"ok": ok, "steps": steps})
            }
            "walk" => {
                json!({"ok": st.advance_native_map_walk_step_view(i(op, "ux"), i(op, "uy"), i(op, "dx"), i(op, "dy"))})
            }
            "walk_state" => {
                let v = view_from(&op["view"]);
                let (w, h) = (i(op, "w"), i(op, "h"));
                let (nv, ok) = advance_native_map_walk_step_view_state(
                    v,
                    w,
                    h,
                    i(op, "ux"),
                    i(op, "uy"),
                    i(op, "dx"),
                    i(op, "dy"),
                );
                json!({"view": view_arr(nv), "ok": ok, "in_field": v.cursor_in_field(w, h), "in_viewport": v.visible_cursor_in_viewport()})
            }
            "range_mode" => json!({"ok": st.materialize_native_map_range_mode(i(op, "mode"))}),
            "redraws" => {
                json!({"redraws": st.native_map_cursor_step_redraws(view_from(&op["before"]), view_from(&op["after"]))})
            }
            other => panic!("未知的視圖操作 {other}"),
        };
        outs.push(out);
    }
    json!({"outs": outs, "view": view_arr(st.native_map_view_state), "has_view": st.has_native_map_view_state,
        "range_mode": st.native_map_range_mode, "has_range_mode": st.has_native_map_range_mode_state})
}

// ---- 平衡設定 ----

fn balance_validate(input: &Value) -> Value {
    let o = balance_from(&input["balance"]).unwrap();
    json!({"err": err_of(&o.validate()), "original": o.is_original()})
}

fn hit_roll() -> Value {
    let g = original_growth_rates();
    json!({"rows": (-2..=40).map(native_command_has_hit_roll).collect::<Vec<_>>(), "original": new_balance_options().is_original(),
        "rates": {"HP": g.hp, "MP": g.mp, "AP": g.ap, "DP": g.dp, "DX": g.dx}})
}

fn balance_identity_case(ctx: &Ctx, input: &Value) -> Value {
    let st = ctx.state(&input["state"]);
    let actor = i(input, "actor") as usize;
    let (id, ok) = balance_identity(Some(&st.units[actor]));
    let hits: Vec<i64> =
        [0, 5, 12, 13, 22, 26, 32, 33].iter().map(|&cmd| st.command_hit(Some(actor), cmd, 77)).collect();
    json!({"id": id, "ok": ok, "hits": hits, "nil_hit": st.command_hit(None, 0, 77)})
}

fn balance_stats(ctx: &Ctx, input: &Value) -> Value {
    let mut units = ctx.units(&input["state"]);
    let u = &mut units[i(input, "unit") as usize];
    let v = &input["stats"];
    let stats = BalanceStats { hp: i(v, "HP"), mp: i(v, "MP"), ap: i(v, "AP"), dp: i(v, "DP"), dx: i(v, "DX") };
    let before = stats_val(&u.balance_stats());
    let r = u.set_balance_stats(stats);
    json!({"before": before, "err": err_of(&r), "unit_after": unit_val(u)})
}

fn enemy_strength(ctx: &Ctx, input: &Value) -> Value {
    let mut st = ctx.state(&input["state"]);
    let before = snapshot(&st.units);
    for p in input["percents"].as_array().unwrap() {
        st.balance = p.as_i64().map(|pct| BalanceOptions { enemy_percent: pct, ..Default::default() });
        st.apply_enemy_strength();
    }
    json!({"changed": changed(&before, &st.units)})
}

// ---- 移動 ----

fn movement(ctx: &Ctx, input: &Value) -> Value {
    let st = ctx.state(&input["state"]);
    let u = i(input, "unit") as usize;
    let reach = st.reachable(u);
    let (mut costs, mut ranges, mut paths) = (Vec::new(), Vec::new(), Vec::new());
    for p in input["probes"].as_array().unwrap() {
        let c = cell(p);
        costs.push(st.move_cost(c.x, c.y));
        ranges.push(st.in_attack_range(u, c.x, c.y));
        let path = st.path(u, c.x, c.y);
        paths.push(if path.is_empty() {
            Value::Null
        } else {
            json!(path.into_iter().map(cell_val).collect::<Vec<_>>())
        });
    }
    json!({"reachable": cells_of(&reach), "costs": costs, "in_range": ranges, "paths": paths})
}

// ---- 死亡程式 ----

fn when_from(v: &Value) -> NativeActionWhen {
    let pair = |key: &str| (!v[key].is_null()).then(|| <[i64; 2]>::try_from(ints(&v[key])).unwrap());
    NativeActionWhen {
        state_eq: pair("state_eq"),
        state_ne: pair("state_ne"),
        round_lt: v["round_lt"].as_i64(),
        any_active: pair("any_active"),
    }
}

fn death_op_from(m: &Value) -> NativeDeathOp {
    NativeDeathOp {
        op: s(m, "op").to_string(),
        unit: i(m, "unit"),
        first: i(m, "first"),
        last: i(m, "last"),
        mode: i(m, "mode"),
        value: i(m, "value"),
        mask: i(m, "mask"),
        slot: i(m, "slot"),
        delta: i(m, "delta"),
        index: i(m, "index"),
        x: i(m, "x"),
        y: i(m, "y"),
        group: i(m, "group"),
        kind: i(m, "kind"),
        writes: m["writes"].as_array().into_iter().flatten().map(|w| <[i64; 3]>::try_from(ints(w)).unwrap()).collect(),
    }
}

fn apply_death_extra(st: &mut State, extra: &Value) {
    for (k, v) in ints(&extra["event_state"]).into_iter().enumerate() {
        st.native_event_state[k] = v as u8;
    }
    st.native_round_counter = i(extra, "round");
    st.has_native_turn_event_control_state = b(extra, "has_turn");
    for (k, c) in extra["turn_controls"].as_array().unwrap().iter().enumerate() {
        let c = ints(c);
        st.native_turn_event_controls[k] =
            NativeTurnEventControl { turn: c[0] as u8, event_id: c[1] as u8, raw_camp: c[2] as u8 };
    }
    st.has_native_field_control_state = b(extra, "has_field");
    st.native_field_control_raw = unhex(s(extra, "field_raw"));
    st.has_native_runtime_unit_projection = b(extra, "has_proj");
    for r in extra["runtime_records"].as_array().unwrap() {
        let mut rec = NativeRuntimeRecordState::default();
        rec.raw.copy_from_slice(&unhex(r.as_str().unwrap()));
        st.native_runtime_records.push(rec);
    }
}

fn death_program(ctx: &Ctx, input: &Value) -> Value {
    let units = ctx.units(&input["state"]);
    let u = &units[i(input, "unit") as usize];
    let (kind, value, ok) = native_death_effect_of(Some(u));
    let mut sc = Scenario::default();
    for p in input["programs"].as_array().unwrap() {
        let list: Vec<Action> = p["actions"]
            .as_array()
            .unwrap()
            .iter()
            .map(|a| Action {
                r#type: "native_death_op".to_string(),
                native_death_op: a["op"].as_str().map(|op| NativeDeathOp { op: op.to_string(), ..Default::default() }),
                native_when: b(a, "when").then(NativeActionWhen::default),
                ..Default::default()
            })
            .collect();
        sc.native_death_programs.insert(s(p, "key").to_string(), list);
    }
    let (prog, key, found) = sc.native_death_program(Some(u));
    json!({"kind": kind, "value": value, "ok": ok, "actions": prog.len(), "key": key, "found": found,
        "cancels": sc.native_death_cancels_exp(Some(u)), "inactive": native_record_inactive(u)})
}

fn when_match(ctx: &Ctx, input: &Value) -> Value {
    let mut st = ctx.state(&input["state"]);
    apply_death_extra(&mut st, &input["extra"]);
    let outs: Vec<Value> = input["whens"]
        .as_array()
        .unwrap()
        .iter()
        .map(|w| {
            let r = when_from(w).r#match(Some(&st));
            json!({"ok": r.clone().unwrap_or(false), "err": err_of(&r)})
        })
        .collect();
    let nil = NativeActionWhen::default().r#match(None);
    json!({"outs": outs, "nil_state": {"ok": nil.clone().unwrap_or(false), "err": err_of(&nil)}})
}

fn death_ops(ctx: &Ctx, input: &Value) -> Value {
    let mut st = ctx.state(&input["state"]);
    apply_death_extra(&mut st, &input["extra"]);
    let before = snapshot(&st.units);
    let errs: Vec<Value> = input["ops"]
        .as_array()
        .unwrap()
        .iter()
        .map(|op| err_of(&st.apply_native_death_op(&death_op_from(op))))
        .collect();
    let clear_from = i(input, "clear_from");
    let cleared = if clear_from >= 0 { idxs(&st.clear_native_hp_from(clear_from)) } else { Value::Null };
    json!({"event_state": st.native_event_state, "turn_controls":
        st.native_turn_event_controls.iter().map(|c| [c.turn, c.event_id, c.raw_camp]).collect::<Vec<_>>(),
        "field_raw": hex(&st.native_field_control_raw),
        "runtime_records": st.native_runtime_records.iter().map(|r| hex(&r.raw)).collect::<Vec<_>>(),
        "range_mode": st.native_map_range_mode, "has_range_mode": st.has_native_map_range_mode_state,
        "changed": changed(&before, &st.units), "errs": errs, "cleared": cleared})
}

// ---- 目標選擇（戰場底稿）----

fn targets(ctx: &Ctx, op: &str, input: &Value) -> Value {
    let st = ctx.state(&input["state"]);
    let (w, h, u) = (st.w, st.h, &st.units);
    let flags = unhex(input["flags"].as_str().unwrap_or_default());
    let list = |r: &Result<Vec<UnitIdx>, BattleError>| -> Value {
        json!({"targets": r.as_ref().map(|t| idxs(t)).unwrap_or(Value::Null), "err": err_of(r)})
    };
    match op {
        "runtime_flags" => {
            let r =
                native_command_runtime_flags(w, h, &st.native_composition_event_bytes, u, i(input, "selector") as u8);
            let base = st.native_command_base_flags();
            json!({"flags": hex(r.as_deref().unwrap_or_default()), "err": err_of(&r),
                "base": hex(base.as_deref().unwrap_or_default()), "base_err": err_of(&base)})
        }
        "future_placement" => {
            let pos = ints(&input["pos"]);
            let position = NativePositionRecord { x_word: pos[0] as u16, y_word: pos[1] as u16, ..Default::default() };
            let r = native_future_group_placement(
                w,
                h,
                &st.native_composition_event_bytes,
                u,
                position,
                i(input, "gate") as u8,
            );
            json!({"cell": cell_val(r.clone().unwrap_or_default()), "err": err_of(&r)})
        }
        "cursor_confirm" => {
            let r = native_cursor_confirmation_allowed(
                cell(&input["cursor"]),
                i(input, "field") as u8,
                i(input, "overlay"),
                i(input, "code"),
                u,
            );
            json!({"ok": r.clone().unwrap_or(false), "err": err_of(&r)})
        }
        "command_targets" => {
            list(&native_command_targets(w, h, cell(&input["origin"]), i(input, "dist"), i(input, "code"), &flags, u))
        }
        "attack_candidates" => list(&native_attack_candidates(
            w,
            h,
            cell(&input["origin"]),
            i(input, "mode"),
            i(input, "inner"),
            i(input, "code"),
            &flags,
            u,
        )),
        "effect_targets" => list(&native_command_effect_targets(
            w,
            h,
            opt_idx(i(input, "actor")),
            opt_idx(i(input, "confirmed")),
            i(input, "selection"),
            i(input, "effect"),
            i(input, "code"),
            &flags,
            u,
        )),
        "effect_targets_cursor" => list(&native_command_effect_targets_at_cursor(
            w,
            h,
            opt_idx(i(input, "actor")),
            cell(&input["cursor"]),
            i(input, "selection"),
            i(input, "effect"),
            i(input, "code"),
            &flags,
            u,
        )),
        "self_centered" => list(&native_command_self_centered_targets(
            w,
            h,
            opt_idx(i(input, "actor")),
            cell(&input["cursor"]),
            i(input, "effect"),
            &flags,
            u,
        )),
        "command30" => list(&native_command30_targets(
            w,
            h,
            cell(&input["saved"]),
            cell(&input["confirmed_cursor"]),
            i(input, "steps"),
            u,
        )),
        "item_effect_targets" => {
            let plan = native_item_target_plan_from_row(&unhex(s(input, "row")));
            let p = plan.clone().unwrap_or_default();
            let mut out = json!({"plan": {"ItemType": p.item_type, "SelectionMode": p.selection_mode,
                "SelectionInnerMark": p.selection_inner_mark, "EffectMode": p.effect_mode, "TargetCode": p.target_code},
                "plan_err": err_of(&plan)});
            if plan.is_ok() {
                let r = native_item_effect_targets(
                    w,
                    h,
                    opt_idx(i(input, "actor")),
                    opt_idx(i(input, "confirmed")),
                    p,
                    &flags,
                    u,
                );
                out["targets"] = r.as_ref().map(|t| idxs(t)).unwrap_or(Value::Null);
                out["err"] = err_of(&r);
            }
            out
        }
        _ => {
            let r = st.native_command_base_flags();
            json!({"flags": hex(r.as_deref().unwrap_or_default()), "err": err_of(&r)})
        }
    }
}

// ---- 指令（戰場底稿）----

/// Go `makeWalk`：每個目標前多走 extra 步原版亂數、命中後再走一步；順序依 kind 改變。
fn make_walk(spec: &Value, start: u16) -> Option<Box<NativeCommandDamageRNGWalk<'static>>> {
    if spec.is_null() {
        return None;
    }
    let kind = s(spec, "kind").to_string();
    let extra = i(spec, "extra");
    Some(Box::new(
        move |count: i64,
              resolve: &mut dyn FnMut(i64, u16) -> Result<(u16, bool), BattleError>|
              -> Result<u16, BattleError> {
            let mut order: Vec<i64> = (0..count).collect();
            match kind.as_str() {
                "reverse" => order.reverse(),
                "skip_last" => {
                    order.pop();
                }
                "double" => {
                    if let Some(&first) = order.first() {
                        order.push(first);
                    }
                }
                _ => {}
            }
            let mut rng = start;
            for index in order {
                for _ in 0..extra {
                    rng = native_rng_step(rng);
                }
                let (next, hit) = resolve(index, rng)?;
                rng = next;
                if hit {
                    rng = native_rng_step(rng);
                }
            }
            Ok(rng)
        },
    ))
}

fn tamper(st: &mut State, op: &Value) {
    let u = &mut st.units[i(op, "unit") as usize];
    let v = i(op, "value");
    match s(op, "field") {
        "HP" => u.hp += v,
        "MP" => u.mp += v,
        "Acted" => u.acted = !u.acted,
        "AP" => u.ap += v,
        "T" => u.native_transient[((v & 0xff) % 6) as usize] ^= 1,
        "B5" => u.native_record_byte5 ^= v as u8,
        other => panic!("未知的竄改欄位 {other}"),
    }
}

/// 依 Go 的 script 逐步執行；竄改步驟沒有回傳值（null）。
fn run_script(
    st: &mut State,
    input: &Value,
    mut apply: impl FnMut(&mut [Unit], &Value) -> Result<(), BattleError>,
) -> Value {
    let errs: Vec<Value> = input["script"]
        .as_array()
        .unwrap()
        .iter()
        .map(|op| {
            if s(op, "op") == "tamper" {
                tamper(st, op);
                Value::Null
            } else {
                err_of(&apply(&mut st.units, op))
            }
        })
        .collect();
    json!(errs)
}

fn command(ctx: &Ctx, input: &Value) -> Value {
    let mut st = ctx.state(&input["state"]);
    let before = snapshot(&st.units);
    let kind = s(input, "kind");
    let (a, cf, cmd) = (opt_idx(i(input, "actor")), opt_idx(i(input, "confirmed")), i(input, "command"));
    let (cursor, origin) = (cell(&input["cursor"]), cell(&input["origin"]));
    let rng = i(input, "rng") as u16;
    let mut source = GoMathRand::new(i(input, "seed"));
    let nil_rng = b(input, "nil_rng");
    let mut walk = make_walk(&input["walk"], rng);
    let resist = st.native_command_resistances.clone();
    let mut res = json!({});
    macro_rules! set {
        ($r:expr, $f:expr) => {{
            let r = $r;
            res["result"] = r.as_ref().map($f).unwrap_or(Value::Null);
            res["err"] = err_of(&r);
            r
        }};
    }
    // Go 在錯誤時回傳零值結構（不是 nil 指標）的函式。
    macro_rules! set_zero {
        ($r:expr, $f:expr, $t:ty) => {{
            let r = $r;
            res["result"] = $f(r.as_ref().unwrap_or(&<$t>::default()));
            res["err"] = err_of(&r);
        }};
    }
    macro_rules! rng_of {
        () => {
            if nil_rng { None } else { Some(&mut source as &mut dyn GoRand) }
        };
    }
    match kind {
        "plan_damage"
        | "plan_damage_cursor"
        | "plan_damage6_cursor"
        | "plan_ai_damage_single"
        | "plan_ai_damage"
        | "bound0_plan" => {
            let w = walk.as_deref_mut();
            let r = match kind {
                "plan_damage" => st.plan_native_command_damage_walk(a, cf, cmd, &resist, rng, w),
                "plan_damage_cursor" => st.plan_native_command_damage_at_cursor(a, cursor, cmd, &resist, rng, w),
                "plan_damage6_cursor" => st.plan_native_command6_damage_at_cursor(a, cursor, &resist, rng, w),
                "plan_ai_damage_single" => st.plan_native_ai_command_damage_single_target(a, cf, cmd, &resist, rng),
                "plan_ai_damage" => st.plan_native_ai_command_damage_walk(a, origin, cmd, &resist, rng, w),
                _ => st.plan_bound_native_command0(a, cf, rng),
            };
            if let Ok(mut plan) = set!(r, damage_plan_val) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "mp" => apply_native_command_damage_mp(units, Some(&mut plan)),
                    "stage" => {
                        apply_native_command_damage_stage(units, Some(&mut plan), i(op, "index"), i(op, "stage"))
                    }
                    _ => complete_native_command_damage(units, Some(&mut plan)),
                });
                res["plan_after"] = damage_plan_val(&plan);
            }
        }
        "execute_damage" | "bound0_execute" => {
            let r = if kind == "execute_damage" {
                st.execute_native_command_damage(a, cf, cmd, &resist, rng)
            } else {
                st.execute_bound_native_command0(a, cf, rng)
            };
            res["result"] = r
                .as_ref()
                .map(|(x, _)| json!(x.iter().map(damage_result_val).collect::<Vec<_>>()))
                .unwrap_or(Value::Null);
            res["err"] = err_of(&r);
            res["rng"] = json!(r.as_ref().map_or(rng, |(_, n)| *n));
        }
        "heal_targets" => {
            let _ = set!(st.native_command_heal_targets(a, cf, cmd), |t| idxs(t));
        }
        "heal_exec" => {
            let _ = set!(st.execute_native_command_heal(a, cf, cmd, rng_of!()), |x| json!(
                x.iter().map(heal_val).collect::<Vec<_>>()
            ));
        }
        "heal_exec_native" | "ai_heal_exec_native" => {
            let r = if kind == "heal_exec_native" {
                st.execute_native_command_heal_native(a, cf, cmd, rng)
            } else {
                st.execute_native_ai_command_heal_native(a, origin, cmd, rng)
            };
            res["result"] =
                r.as_ref().map(|(x, _)| json!(x.iter().map(heal_val).collect::<Vec<_>>())).unwrap_or(Value::Null);
            res["err"] = err_of(&r);
            res["rng"] = json!(r.as_ref().map_or(rng, |(_, n)| *n));
        }
        "ai_heal_targets" => {
            let _ = set!(st.native_ai_command_heal_targets(a, origin, cmd), |t| idxs(t));
        }
        "ai_heal_exec" => {
            let _ = set!(st.execute_native_ai_command_heal(a, origin, cmd, rng_of!()), |x| json!(
                x.iter().map(heal_val).collect::<Vec<_>>()
            ));
        }
        "modifier_targets" => {
            let _ = set!(st.native_command_modifier_targets(a, cf, cmd), |t| idxs(t));
        }
        "modifier_exec" => {
            set_zero!(st.execute_native_command_modifier(a, cf, cmd, rng), modifier_val, NativeCommandModifierResult)
        }
        "ai_modifier_targets" => {
            let _ = set!(st.native_ai_command_modifier_targets(a, origin, cmd), |t| idxs(t));
        }
        "ai_modifier_exec" => {
            set_zero!(
                st.execute_native_ai_command_modifier(a, origin, cmd, rng),
                modifier_val,
                NativeCommandModifierResult
            )
        }
        "modifier_validate" => {
            let t: Vec<UnitIdx> = ints(&input["targets"]).into_iter().map(|x| x as usize).collect();
            res["err"] = err_of(&st.validate_native_command_modifier_targets(a, &t, cmd));
        }
        "ai_modifier_plan" => {
            if let Ok(mut plan) = set!(st.plan_native_ai_command_modifier(a, origin, cmd, rng), modifier_plan_val) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "publish" => publish_native_ai_command_modifier(units, Some(&mut plan)),
                    "complete" => complete_native_ai_command_modifier(units, Some(&mut plan)),
                    _ => abort_native_ai_command_modifier(units, Some(&mut plan)),
                });
            }
        }
        "clear_restore_targets" | "app_targets" => {
            let r = if kind == "clear_restore_targets" {
                st.native_command_clear_restore_targets(a, cf, cmd)
            } else {
                st.native_command_application_targets(a, cf, cmd)
            };
            res["result"] = r.as_ref().map(|(t, _)| idxs(t)).unwrap_or(Value::Null);
            res["err"] = err_of(&r);
            res["offset"] = json!(r.as_ref().map_or(0, |(_, off)| *off));
        }
        "clear_restore_exec" => {
            let _ = set!(st.execute_native_command_clear_restore(a, cf, cmd, rng_of!()), |x| json!(
                x.iter().map(clear_restore_val).collect::<Vec<_>>()
            ));
        }
        "ai2022_targets" => {
            let _ = set!(st.native_ai_command2022_targets(a, origin, cmd), |t| idxs(t));
        }
        "ai2022_plan" | "player2022_plan" => {
            let r = if kind == "ai2022_plan" {
                st.plan_native_ai_command2022(a, origin, cmd, rng)
            } else {
                st.plan_native_player_command2022(a, cf, cmd, rng)
            };
            if let Ok(mut plan) = set!(r, plan2022_val) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "publish" => publish_native_ai_command2022(units, Some(&mut plan)),
                    "complete" => complete_native_ai_command2022(units, Some(&mut plan)),
                    _ => abort_native_ai_command2022(units, Some(&mut plan)),
                });
            }
        }
        "cmd25_targets" => {
            let _ = set!(st.native_command25_targets(a, cf), |t| idxs(t));
        }
        "cmd25_exec" => {
            let _ = set!(st.execute_native_command25(a, cf), |x| json!(x.iter().map(c25_val).collect::<Vec<_>>()));
        }
        "app_exec" => {
            let _ = set!(st.execute_native_command_application(a, cf, cmd, rng_of!()), |x| json!(
                x.iter().map(app_val).collect::<Vec<_>>()
            ));
        }
        "c32_plan" => {
            if let Ok(mut plan) = set!(st.plan_native_compound_command32(a, cf, rng), c32_plan_val) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "apply" => apply_native_compound_command32_target(units, Some(&mut plan), i(op, "index")),
                    "complete" => complete_native_compound_command32(units, Some(&mut plan)),
                    _ => abort_native_compound_command32(units, Some(&mut plan)),
                });
            }
        }
        "c32_exec" => {
            set_zero!(st.execute_native_compound_command32(a, cf, rng), c32_result_val, NativeCompoundCommand32Result)
        }
        "c33_plan" => {
            if let Ok(mut plan) = set!(st.plan_native_compound_command33(a, cf, rng), c33_plan_val) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "publish" => publish_native_compound_command33(units, Some(&mut plan)),
                    "complete" => complete_native_compound_command33(units, Some(&mut plan)),
                    _ => abort_native_compound_command33(units, Some(&mut plan)),
                });
            }
        }
        "c33_exec" => {
            set_zero!(st.execute_native_compound_command33(a, cf, rng), c33_result_val, NativeCompoundCommand33Result)
        }
        "c34_plan" => {
            let r = st.plan_native_compound_command34(a, cf, rng);
            if let Ok(mut plan) = set!(r, |p| json!({"Actor": idx(p.actor), "Result": c34_result_val(&p.result)})) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "publish" => publish_native_compound_command34_stage(units, Some(&mut plan), i(op, "stage")),
                    "complete" => complete_native_compound_command34(units, Some(&mut plan)),
                    _ => abort_native_compound_command34(units, Some(&mut plan)),
                });
            }
        }
        "c34_exec" => {
            set_zero!(st.execute_native_compound_command34(a, cf, rng), c34_result_val, NativeCompoundCommand34Result)
        }
        "c35_plan" => {
            let r = st.plan_native_compound_command35(a, cf, rng);
            if let Ok(mut plan) = set!(r, |p| json!({"Actor": idx(p.actor), "Result": c35_result_val(&p.result)})) {
                res["script_errs"] = run_script(&mut st, input, |units, op| match s(op, "op") {
                    "publish" => publish_native_compound_command35_stage(units, Some(&mut plan), i(op, "stage")),
                    "complete" => complete_native_compound_command35(units, Some(&mut plan)),
                    _ => abort_native_compound_command35(units, Some(&mut plan)),
                });
            }
        }
        "c35_exec" => {
            set_zero!(st.execute_native_compound_command35(a, cf, rng), c35_result_val, NativeCompoundCommand35Result)
        }
        other => panic!("未知的指令種類 {other}"),
    }
    res["changed"] = changed(&before, &st.units);
    res
}

fn nil_plans() -> Value {
    let mut u: Vec<Unit> = Vec::new();
    let mut source = GoMathRand::new(1);
    let errs = json!({
        "damage_mp": err_of(&apply_native_command_damage_mp(&mut u, None)),
        "damage_stage": err_of(&apply_native_command_damage_stage(&mut u, None, 0, 1)),
        "damage_complete": err_of(&complete_native_command_damage(&mut u, None)),
        "mod_publish": err_of(&publish_native_ai_command_modifier(&mut u, None)),
        "mod_complete": err_of(&complete_native_ai_command_modifier(&mut u, None)),
        "mod_abort": err_of(&abort_native_ai_command_modifier(&mut u, None)),
        "t2022_publish": err_of(&publish_native_ai_command2022(&mut u, None)),
        "t2022_complete": err_of(&complete_native_ai_command2022(&mut u, None)),
        "t2022_abort": err_of(&abort_native_ai_command2022(&mut u, None)),
        "c32_apply": err_of(&apply_native_compound_command32_target(&mut u, None, 0)),
        "c32_complete": err_of(&complete_native_compound_command32(&mut u, None)),
        "c32_abort": err_of(&abort_native_compound_command32(&mut u, None)),
        "c33_publish": err_of(&publish_native_compound_command33(&mut u, None)),
        "c33_complete": err_of(&complete_native_compound_command33(&mut u, None)),
        "c33_abort": err_of(&abort_native_compound_command33(&mut u, None)),
        "c34_publish": err_of(&publish_native_compound_command34_stage(&mut u, None, 0)),
        "c34_complete": err_of(&complete_native_compound_command34(&mut u, None)),
        "c34_abort": err_of(&abort_native_compound_command34(&mut u, None)),
        "c35_publish": err_of(&publish_native_compound_command35_stage(&mut u, None, 0)),
        "c35_complete": err_of(&complete_native_compound_command35(&mut u, None)),
        "c35_abort": err_of(&abort_native_compound_command35(&mut u, None)),
        "restore_nil": err_of(&apply_native_command_restore(None, 1, Some(&mut source as &mut dyn GoRand))),
    });
    json!({"errs": errs})
}
