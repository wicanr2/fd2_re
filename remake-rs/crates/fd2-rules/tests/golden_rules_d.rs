//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-d` 的黃金差分：fd2-rules D 組（敵我 AI）。
//!
//! 由 `tools/rust_golden_rules_d.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_D`；真實輸入（受版控的
//! `remake/assets`）掛在 `FD2_GOLDEN_REPO` 底下，先核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：戰場由黃金檔的「底稿＋逐單位修改」組成，兩邊都把合併後的 JSON 物件解碼成 `Unit`；
//! `State` 由同一份規格（地圖大小、地形、事件格網、寶箱控制、事件狀態覆寫、移動成本列與物品效果列）
//! 建出。每個案例依 `op` 在 Rust 端重做同一串呼叫，整個結果物件與 Go 逐欄比對（鍵集合也要相同）：
//! `*Unit` 是 `State::units` 的索引（nil 為 -1），結構以 Go 欄位名輸出且只含 Go 匯出欄位，單位以
//! Go 欄位名的 JSON 比對。D 組與其他組的錯誤文字逐字相同。比對時的表示法正規化列在
//! [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。
//!
//! 經由 E 組（`NextAIPlan`、地形 AP/DP 表）的 runtime 規劃不在這份黃金裡，E 合併後才能涵蓋。

use fd2_rules::battle::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為（PORTING.md 決定 2、`rust-port-no-go-stdlib`、`battle.rs` 模組文件）。
/// 黃金案例都不落在這些輸入上。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 可以把不在 State.Units 裡的 *Unit 或 nil 接收者傳進來；Rust 以索引表示單位，黃金只傳 State.Units
    // 裡的單位或 nil（Rust 的 None）。
    "不在 State.Units 的單位指標與 nil 接收者",
    // ScoreNativePhysicalAttackCandidate／SelectNativePhysicalAttackCandidate 迭代 map 檢查欄位範圍，
    // 多欄同時越界時 Go 報哪一欄隨 map 順序而定，Rust 依鍵名升冪；黃金每案最多一個越界欄位。
    "native_ai_physical_score 多欄越界時的錯誤欄位",
    // BuildNativeAIPhaseDiagnosticPlan 迭代 map 找不合格輸入，多筆時 Go 報哪一筆不定，Rust 依索引升冪；
    // 黃金每案最多一筆不合格輸入。
    "native_ai_phase_diagnostic 多筆不合格輸入時的錯誤單位",
    // ApplyNativeAIIdleRecoveryWithPresentation 的「callback 改了記錄」分支：Rust 的 records 在呼叫期間
    // 被可變借用，safe code 的 callback 碰不到同一個切片，黃金的 callback 不改記錄。
    "閒置恢復 callback 改寫記錄",
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文。
    "JSON 欄位名大小寫",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列、空字串相同",
    // 只比數值，不比整數與浮點的表示法。
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

fn bytes_of(v: &Value) -> Vec<u8> {
    ints(v).into_iter().map(|x| x as u8).collect()
}

fn pair(v: &Value) -> Cell {
    let a = ints(v);
    Cell { x: a[0], y: a[1] }
}

fn cell_obj(c: Cell) -> Value {
    json!({"X": c.x, "Y": c.y})
}

fn cell_from(v: &Value) -> Cell {
    Cell { x: i(v, "X"), y: i(v, "Y") }
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
        Err(e) => json!(e.message()),
    }
}

fn idx(u: Option<UnitIdx>) -> i64 {
    u.map_or(-1, |x| x as i64)
}

// ---- 結構 → Go 欄位名物件 ----

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

fn input14ef0_from(v: &Value) -> NativeAI14EF0Input {
    NativeAI14EF0Input {
        score_c4_f: i(v, "ScoreC4F") as i32,
        score_c23: i(v, "ScoreC23") as i32,
        score_c33: i(v, "ScoreC33") as i32,
        command_id: i(v, "CommandID") as i32,
        command_word: i(v, "CommandWord") as u16,
        record34: i(v, "Record34") as u8,
        actor_word48: i(v, "ActorWord48") as u16,
        target_word4_a: i(v, "TargetWord4A") as u16,
        has_raw_score_c4_f: b(v, "HasRawScoreC4F"),
        has_raw_score_c23: b(v, "HasRawScoreC23"),
        has_raw_score_c33: b(v, "HasRawScoreC33"),
        has_raw_command_id: b(v, "HasRawCommandID"),
        has_raw_command_word: b(v, "HasRawCommandWord"),
        has_raw_record34: b(v, "HasRawRecord34"),
        has_raw_actor_word48: b(v, "HasRawActorWord48"),
        has_raw_target_word4_a: b(v, "HasRawTargetWord4A"),
    }
}

fn spell_candidate_val(c: &NativeAISpellCandidate) -> Value {
    json!({"CommandID": c.command_id, "CommandValue": c.command_value, "X": c.x, "Y": c.y, "Score": c.score,
        "TargetIndices": c.target_indices})
}

fn spell_candidate_from(v: &Value) -> NativeAISpellCandidate {
    NativeAISpellCandidate {
        command_id: i(v, "CommandID"),
        command_value: i(v, "CommandValue"),
        x: i(v, "X"),
        y: i(v, "Y"),
        score: i(v, "Score"),
        target_indices: bytes_of(&v["TargetIndices"]),
    }
}

fn r1598a_val(r: &NativeAI1598AScoreResult) -> Value {
    json!({"MaxScore": r.max_score, "PositiveWinner": spell_candidate_val(&r.positive_winner),
        "HasPositiveWinner": r.has_positive_winner})
}

fn r1567e_val(r: &NativeAI1567EScoreResult) -> Value {
    json!({"MaxScore": r.max_score, "X": r.x, "Y": r.y, "InventorySlot": r.inventory_slot, "ItemID": r.item_id,
        "TargetIndices": r.target_indices, "HasPositiveWinner": r.has_positive_winner})
}

fn group_score_val(g: &NativeAIScoredCommandGroupScore) -> Value {
    json!({"Destination": cell_obj(g.destination), "TargetIndices": g.target_indices, "Score": g.score})
}

fn decision_val(d: &NativeAIIdleRecoveryDecision) -> Value {
    json!({"Unit": d.unit, "Accepted": d.accepted, "CurrentHP": d.current_hp, "MaximumHP": d.maximum_hp,
        "NextHP": d.next_hp, "RawGate25": d.raw_gate25, "RawGate26": d.raw_gate26})
}

fn presentation_val(p: &NativeAIIdleRecoveryPresentation) -> Value {
    let write = |w: &NativeAIIdleRecoveryGlobalWrite| json!({"Address": w.address, "Value": w.value});
    let decode =
        |d: &NativeAIIdleRecoveryDecode| json!({"ResourceBase": d.resource_base, "Mode": d.mode, "Tail": d.tail});
    let copy = |c: &NativeAIIdleRecoveryCopy| {
        json!({"SourceOffset": c.source_offset, "DestinationOffset": c.destination_offset,
            "SourceStride": c.source_stride, "DestinationStride": c.destination_stride, "Rows": c.rows, "Length": c.length})
    };
    json!({"Unit": p.unit, "BeforeHP": p.before_hp, "AfterHP": p.after_hp,
        "BeforeGlobalWrite": write(&p.before_global_write),
        "CoordinateCall": {"Address": p.coordinate_call.address, "Unit": p.coordinate_call.unit},
        "SampleHandleExpr": p.sample_handle_expr, "SampleIndex": p.sample_index, "SampleLoopCount": p.sample_loop_count,
        "FirstDecode": decode(&p.first_decode), "SecondDecode": decode(&p.second_decode),
        "FirstCopy": copy(&p.first_copy), "SecondCopy": copy(&p.second_copy), "WaitTicks": p.wait_ticks,
        "AfterGlobalWrite": write(&p.after_global_write)})
}

fn tx_val(t: &NativeAIMode11Transaction) -> Value {
    json!({"FirstRoute": t.first_route.0, "SecondRoute": t.second_route.0})
}

fn stage_val(s: &NativeAIMode11Stage) -> Value {
    json!({"Ordinal": s.ordinal, "Route": s.route.0})
}

fn item_source_val(x: &NativeAIPhysicalItemSource) -> Value {
    json!({"Slot": x.slot, "ItemID": x.item_id, "Row": x.row, "RawGeometryByte0B": x.raw_geometry_byte0_b,
        "RawGeometryByte0C": x.raw_geometry_byte0_c})
}

fn score_input_val(x: &NativePhysicalAttackScoreInput) -> Value {
    json!({"ActorWord48": x.actor_word48, "ActorWord4A": x.actor_word4_a, "TargetWord48": x.target_word48,
        "TargetWord4A": x.target_word4_a, "TargetWord40": x.target_word40, "RawTargetByte8": x.raw_target_byte8,
        "RawHelper1DEBEResult": x.raw_helper1_debe_result})
}

fn score_input_from(v: &Value) -> NativePhysicalAttackScoreInput {
    NativePhysicalAttackScoreInput {
        actor_word48: i(v, "ActorWord48"),
        actor_word4_a: i(v, "ActorWord4A"),
        target_word48: i(v, "TargetWord48"),
        target_word4_a: i(v, "TargetWord4A"),
        target_word40: i(v, "TargetWord40"),
        raw_target_byte8: i(v, "RawTargetByte8") as u8,
        raw_helper1_debe_result: i(v, "RawHelper1DEBEResult"),
    }
}

fn score_val(x: &NativePhysicalAttackScore) -> Value {
    json!({"Priority": x.priority, "Score": x.score})
}

fn candidate_val(c: &NativePhysicalAttackCandidate) -> Value {
    json!({"DestinationX": c.destination_x, "DestinationY": c.destination_y, "TargetIndex": c.target_index,
        "Inputs": score_input_val(&c.inputs)})
}

fn candidate_from(v: &Value) -> NativePhysicalAttackCandidate {
    NativePhysicalAttackCandidate {
        destination_x: i(v, "DestinationX"),
        destination_y: i(v, "DestinationY"),
        target_index: i(v, "TargetIndex"),
        inputs: score_input_from(&v["Inputs"]),
    }
}

fn selection_val(x: &NativePhysicalAttackSelection) -> Value {
    json!({"Candidate": candidate_val(&x.candidate), "Ranking": score_val(&x.ranking)})
}

fn cue_val(c: &NativeAIMode5AudioCue) -> Value {
    json!({"HandleLinearAddress": c.handle_linear_address, "ResourceID": c.resource_id, "Index": c.index,
        "LoopCount": c.loop_count})
}

fn damage24_val(d: &NativeCommand24Damage) -> Value {
    json!({"Target": idx(d.target), "Amount": d.amount, "Rolled": d.rolled, "Damage": d.damage, "HPBefore": d.hp_before,
        "HPAfter": d.hp_after, "NativeRecordByte5Before": d.native_record_byte5_before,
        "NativeRecordByte5After": d.native_record_byte5_after})
}

fn strike_plan_val(p: &NativeCommandDerivedStrikePlan) -> Value {
    json!({"Actor": idx(p.actor), "CommandID": p.command_id, "MPBefore": p.mp_before, "MPAfter": p.mp_after,
        "Results": p.results.iter().map(damage24_val).collect::<Vec<_>>(), "TargetImpactCount": p.target_impact_count})
}

fn phase_plan_val(p: &fd2_formats::fdother::NativePhaseUnitScanPlan) -> Value {
    let cand = |c: &fd2_formats::fdother::NativePhaseDispatchCandidate| json!({"UnitIndex": c.unit_index, "SelectorArg": c.selector_arg});
    json!({
        "SelectorOne": p.selector_one.iter().map(cand).collect::<Vec<_>>(),
        "ZeroPreselection": p.zero_preselection.iter().map(|z| json!({"UnitIndex": z.unit_index,
            "SelectorArg": z.selector_arg, "RunUnitAction": z.run_unit_action})).collect::<Vec<_>>(),
        "ZeroSecondPass": p.zero_second_pass.iter().map(cand).collect::<Vec<_>>(),
    })
}

fn diagnostic_val(d: &NativeAIPhaseZeroDiagnostic) -> Value {
    json!({"UnitIndex": d.unit_index, "CommandMaskResult": r1598a_val(&d.command_mask_result),
        "ItemCommandResult": r1567e_val(&d.item_command_result),
        "PhaseGateScorePair": {"SpellCandidateScore53C23": d.phase_gate_score_pair.spell_candidate_score_53c23,
            "ItemCommandScore53C33": d.phase_gate_score_pair.item_command_score_53c33}})
}

// ---- 黃金檔與戰場 ----

struct Fixture {
    w: i64,
    h: i64,
    events: Vec<u8>,
    terrain: Vec<u8>,
    control: Vec<u8>,
    grid: Vec<u8>,
    has_grid: bool,
    chests: [NativeChestControl; 16],
    has_chest: bool,
    units: Vec<Value>,
    records: Option<Vec<u8>>,
}

struct Ctx {
    repo: PathBuf,
    fixtures: Vec<Fixture>,
    book: Vec<NativeCommandRecord>,
    cost_rows: Vec<Vec<u8>>,
    item_rows: Vec<u8>,
}

impl Ctx {
    fn real(&self, rel: &str) -> String {
        self.repo.join("remake").join(rel).to_string_lossy().into_owned()
    }

    /// Go `exporter.units`：底稿加逐單位修改。
    fn units(&self, k: usize, edits: &Value) -> Vec<Unit> {
        let mut maps: Vec<Value> = self.fixtures[k].units.clone();
        for ed in edits.as_array().map(|a| a.as_slice()).unwrap_or_default() {
            let target = maps[i(ed, "unit") as usize].as_object_mut().unwrap();
            for (key, value) in ed["set"].as_object().unwrap() {
                target.insert(key.clone(), value.clone());
            }
        }
        maps.iter().map(unit_from).collect()
    }

    /// Go `exporter.state`。
    fn state(&self, sp: &Value) -> State {
        let k = i(sp, "fixture") as usize;
        let fx = &self.fixtures[k];
        // State 有 crate 私有欄位，不能用結構更新語法；逐欄設定。
        let mut st = State::default();
        (st.w, st.h, st.units) = (fx.w, fx.h, self.units(k, &sp["edits"]));
        st.native_composition_event_bytes = fx.events.clone();
        st.native_terrain_move_codes = fx.terrain.clone();
        st.native_terrain_control = fx.control.clone();
        (st.native_chest_controls, st.has_native_chest_control_state) = (fx.chests, fx.has_chest);
        st.native_command_book = self.book.clone();
        if let Some(book) = sp.get("book").and_then(|v| v.as_array()) {
            st.native_command_book = book.iter().map(command_from).collect();
        }
        if !b(sp, "no_grid") {
            (st.native_map_event_grid, st.has_native_map_event_grid) = (fx.grid.clone(), fx.has_grid);
            if let Some(g) = sp.get("grid").and_then(|v| v.as_str()) {
                (st.native_map_event_grid, st.has_native_map_event_grid) = (unhex(g), true);
            }
        }
        for kv in sp["event_state"].as_array().unwrap() {
            st.native_event_state[kv[0].as_i64().unwrap() as usize] = kv[1].as_i64().unwrap() as u8;
        }
        if let Some(raw) = sp.get("field_raw").and_then(|v| v.as_str()) {
            (st.native_field_control_raw, st.has_native_field_control_state) = (unhex(raw), true);
        }
        st.has_native_map_view_state = b(sp, "map_view");
        if b(sp, "rows") {
            st.bind_native_movement_cost_rows(&self.cost_rows).unwrap();
            st.bind_native_future_item_rows(&self.item_rows).unwrap();
        }
        st
    }
}

fn snapshot(units: &[Unit]) -> Vec<Value> {
    units.iter().map(unit_val).collect()
}

fn changed(before: &[Value], units: &[Unit]) -> Value {
    let mut out = serde_json::Map::new();
    for (k, u) in units.iter().enumerate() {
        let now = unit_val(u);
        if diff("", &before[k], &now).is_some() {
            out.insert(k.to_string(), now);
        }
    }
    Value::Object(out)
}

/// Go 黃金工具的確定性閘門：mod=0 表示 nil，否則略過索引是 mod 倍數的目標。
fn skip_gate(m: i64) -> impl FnMut(i64) -> bool {
    move |index| index % m == 0
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_D").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let (Some(path), Some(repo)) = (path, repo) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_RULES_D 或 FD2_GOLDEN_REPO");
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_D／FD2_GOLDEN_REPO");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let fixtures = doc["fixtures"]
        .as_array()
        .unwrap()
        .iter()
        .map(|f| {
            let mut chests = [NativeChestControl::default(); 16];
            for (slot, c) in f["chests"].as_array().unwrap().iter().enumerate() {
                chests[slot] =
                    NativeChestControl { raw_type: c[0].as_i64().unwrap() as u8, value: c[1].as_i64().unwrap() as u16 };
            }
            Fixture {
                w: i(f, "w"),
                h: i(f, "h"),
                events: unhex(s(f, "events")),
                terrain: unhex(s(f, "terrain")),
                control: unhex(s(f, "control")),
                grid: unhex(s(f, "grid")),
                has_grid: b(f, "has_grid"),
                chests,
                has_chest: b(f, "has_chest"),
                units: f["units"].as_array().unwrap().clone(),
                records: f["records"].as_str().map(unhex),
            }
        })
        .collect();
    let book = doc["real"]["book"].as_array().unwrap().iter().map(command_from).collect();
    let cost_rows = doc["real"]["cost_rows"].as_array().unwrap().iter().map(|r| unhex(r.as_str().unwrap())).collect();
    let item_rows = unhex(s(&doc["real"], "item_rows"));
    Some((doc, Ctx { repo, fixtures, book, cost_rows, item_rows }))
}

#[test]
fn golden_rules_d() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-d/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    for file in doc["real"]["files"].as_array().unwrap() {
        let raw = std::fs::read(ctx.real(s(file, "path"))).unwrap_or_else(|e| panic!("讀不到 {}：{e}", file["path"]));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    // 黃金檔內嵌的指令表、移動成本列與物品效果列，必須與 Rust 端自己載入真實檔的結果相同。
    let book = load_native_command_records(&ctx.real("assets/spells.json")).unwrap();
    check("指令表", &doc["real"]["book"], &json!(book.iter().map(command_val).collect::<Vec<_>>()));
    let rows = load_native_movement_cost_rows(&ctx.real("assets/data/native_movement_cost_rows.json")).unwrap();
    check("移動成本列", &doc["real"]["cost_rows"], &json!(rows.iter().map(|r| hex(r)).collect::<Vec<_>>()));
    let items = load_native_item_effect_row_prefix(&ctx.real("assets/data/native_item_effect_rows.json")).unwrap();
    assert_eq!(hex(&items), s(&doc["real"], "item_rows"), "物品效果列");
    // 底稿的原始記錄（A 組 NativeAIScoringRecords）兩邊相同。
    for (k, fx) in ctx.fixtures.iter().enumerate() {
        let units = ctx.units(k, &Value::Null);
        let got = native_ai_scoring_records(&units).ok();
        assert_eq!(got.as_ref().map(|r| hex(r)), fx.records.as_ref().map(|r| hex(r)), "底稿 {k} 的原始記錄");
    }

    let mut counts = BTreeMap::<String, usize>::new();
    for case in doc["cases"].as_array().unwrap() {
        let op = s(case, "op");
        let name = format!("case {} {op}", case["index"]);
        let (input, result) = (&case["input"], &case["result"]);
        let rust = match op {
            "dispatch14ef0" => dispatch14ef0(input),
            "spell_candidate" => spell_candidate(input),
            "item_score" => item_score(&ctx, input),
            "group_score" => group_score(input),
            "spell_score" => spell_score(input),
            "idle" => idle(input),
            "mode11" => mode11(input),
            "mode_record" => mode_record(input),
            "mode_write" => mode_write(input),
            "physical_score" => physical_score(input),
            "physical_select" => physical_select(input),
            "score1598a" => score1598a(&ctx, input),
            "score1567e" => score1567e(&ctx, input),
            "item_source" => item_source(&ctx, input),
            "physical_build" => physical_build(&ctx, input),
            "phase" => phase(&ctx, input),
            "mode5" => mode5(&ctx, input),
            "player_move" => player_move(&ctx, input),
            "derived_damage" => derived_damage(input),
            "command24" => command24(&ctx, input),
            other => panic!("{name}：未知的 op {other}"),
        };
        check(&name, result, &rust);
        *counts.entry(op.to_string()).or_default() += 1;
    }
    eprintln!("rules-d 黃金案例：{counts:?}，共 {}", counts.values().sum::<usize>());
}

// ---- 純函式 ----

fn dispatch14ef0(input: &Value) -> Value {
    let r#in = input14ef0_from(&input["in"]);
    let tail = select_native_ai14_ef0_tail(r#in);
    let decision = plan_native_ai14_ef0_dispatch(r#in);
    let d = decision.as_ref().copied().unwrap_or_default();
    json!({"tail": tail.as_ref().map_or(0, |t| t.0), "err": err_of(&tail),
        "decision": {"Tail": d.tail.0, "Accepted": d.accepted}, "decision_err": err_of(&decision)})
}

fn spell_candidate(input: &Value) -> Value {
    let cands: Vec<_> = input["candidates"].as_array().unwrap().iter().map(spell_candidate_from).collect();
    let (best, ok) = select_native_ai_spell_candidate(&cands);
    json!({"best": spell_candidate_val(&best), "ok": ok})
}

fn item_score(ctx: &Ctx, input: &Value) -> Value {
    let book: &[NativeCommandRecord] = if b(input, "book") { &ctx.book } else { &[] };
    let got = score_native_ai_item_command_targets(
        &unhex(s(input, "records")),
        &bytes_of(&input["targets"]),
        i(input, "item_type"),
        i(input, "item_word"),
        book,
    );
    json!({"score": got.as_ref().copied().unwrap_or(0), "err": err_of(&got)})
}

fn group_score(input: &Value) -> Value {
    let groups: Vec<_> = input["groups"]
        .as_array()
        .unwrap()
        .iter()
        .map(|g| NativeAIScoredCommandCandidateGroup {
            destination: cell_from(&g["Destination"]),
            target_indices: bytes_of(&g["TargetIndices"]),
        })
        .collect();
    let m = i(input, "skip_mod");
    let mut gate = skip_gate(m.max(1));
    let got = score_native_ai_scored_command_groups(
        &unhex(s(input, "records")),
        i(input, "command_id"),
        i(input, "value"),
        &groups,
        (m != 0).then_some(&mut gate as &mut dyn FnMut(i64) -> bool),
    );
    let scores = got.as_ref().map(|v| v.iter().map(group_score_val).collect::<Vec<_>>()).unwrap_or_default();
    json!({"scores": scores, "err": err_of(&got)})
}

fn spell_score(input: &Value) -> Value {
    let (records, targets) = (unhex(s(input, "records")), bytes_of(&input["targets"]));
    let id = i(input, "id");
    let m = i(input, "skip_mod");
    let mut gate = skip_gate(m.max(1));
    let got = match s(input, "fn") {
        "attack" => score_native_ai_spell_attack(
            &records,
            &targets,
            id,
            i(input, "value"),
            (m != 0).then_some(&mut gate as &mut dyn FnMut(i64) -> bool),
        ),
        "recovery" => score_native_ai_spell_recovery(&records, &targets, id),
        "flag" => score_native_ai_spell_flag(&records, &targets, id),
        "zero_flag" => score_native_ai_spell_zero_flag(&records, &targets, id),
        _ => score_native_ai_spell22(&records, &targets),
    };
    json!({"score": got.as_ref().copied().unwrap_or(0), "err": err_of(&got)})
}

fn idle(input: &Value) -> Value {
    let mut records = unhex(s(input, "records"));
    let (count, unit) = (i(input, "count"), i(input, "unit"));
    let mut res = serde_json::Map::new();
    match s(input, "mode") {
        "plan" => {
            let d = plan_native_ai_idle_recovery(&records, count, unit);
            // Go 在錯誤時仍回傳只填了 Unit 的 decision；Rust 的 Err 不帶值，依 Go 的形狀補出。
            let shown = d.as_ref().copied().unwrap_or(NativeAIIdleRecoveryDecision { unit, ..Default::default() });
            res.insert("decision".into(), decision_val(&shown));
            res.insert("err".into(), err_of(&d));
            if let Ok(d) = d {
                let p = build_native_ai_idle_recovery_presentation(d);
                res.insert("presentation".into(), presentation_val(&p.clone().unwrap_or_default()));
                res.insert("presentation_err".into(), err_of(&p));
            }
        }
        "apply" => {
            let ok = apply_native_ai_idle_recovery(&mut records, count, unit);
            res.insert("ok".into(), json!(*ok.as_ref().unwrap_or(&false)));
            res.insert("err".into(), err_of(&ok));
        }
        mode => {
            let mut calls = 0;
            let ok = {
                let mut ok_fn = |_: NativeAIIdleRecoveryPresentation| -> Result<(), BattleError> {
                    calls += 1;
                    if mode == "present_err" { Err(BattleError::new("renderer unavailable")) } else { Ok(()) }
                };
                let present: Option<&mut dyn FnMut(NativeAIIdleRecoveryPresentation) -> Result<(), BattleError>> =
                    (mode != "present_nil").then_some(&mut ok_fn);
                apply_native_ai_idle_recovery_with_presentation(&mut records, count, unit, present)
            };
            res.insert("ok".into(), json!(*ok.as_ref().unwrap_or(&false)));
            res.insert("err".into(), err_of(&ok));
            res.insert("calls".into(), json!(calls));
        }
    }
    res.insert("records".into(), json!(hex(&records)));
    Value::Object(res)
}

fn mode11(input: &Value) -> Value {
    let (c23, c4f) = (i(input, "c23") as i32, i(input, "c4f") as i32);
    let (h23, h4f) = (b(input, "h23"), b(input, "h4f"));
    let selected = select_native_ai_mode11_transaction(c23, c4f, h23, h4f);
    let tx = NativeAIMode11Transaction {
        first_route: NativeAIMode11Route(i(&input["tx"], "FirstRoute") as u8),
        second_route: NativeAIMode11Route(i(&input["tx"], "SecondRoute") as u8),
    };
    let stages = tx.stages();
    let fail_at = i(input, "fail_at");
    let mut calls = Vec::new();
    let exec = {
        let mut execute = |stage: NativeAIMode11Stage| {
            calls.push(stage.ordinal * 10 + stage.route.0 as i64);
            if stage.ordinal == fail_at { Err(BattleError::new("presentation owner unavailable")) } else { Ok(()) }
        };
        let f: Option<&mut dyn FnMut(NativeAIMode11Stage) -> Result<(), BattleError>> =
            b(input, "execute").then_some(&mut execute);
        execute_native_ai_mode11_transaction(tx, f)
    };
    json!({"select": tx_val(&selected.clone().unwrap_or_default()), "select_err": err_of(&selected),
        "stages": stages.as_ref().map(|v| v.iter().map(stage_val).collect::<Vec<_>>()).unwrap_or_default(),
        "stages_err": err_of(&stages), "calls": calls, "exec_err": err_of(&exec)})
}

fn mode_record(input: &Value) -> Value {
    let unit = unit_from(&input["unit"]);
    let got = native_ai_mode_record_for_unit((!b(input, "nil")).then_some(&unit));
    json!({"record": hex(got.as_deref().unwrap_or_default()), "err": err_of(&got)})
}

fn mode_write(input: &Value) -> Value {
    let mut records = unhex(s(input, "records"));
    let (count, first, last, mode) = (i(input, "count"), i(input, "first"), i(input, "last"), i(input, "mode") as u8);
    let got = if b(input, "byte") {
        set_native_ai_mode_byte(&mut records, count, first, mode)
    } else {
        set_native_ai_mode_range(&mut records, count, first, last, mode)
    };
    json!({"err": err_of(&got), "records": hex(&records)})
}

fn physical_score(input: &Value) -> Value {
    let got = score_native_physical_attack_candidate(score_input_from(&input["in"]));
    let (sc, ok) = got.as_ref().copied().unwrap_or_default();
    json!({"score": score_val(&sc), "ok": ok, "err": err_of(&got)})
}

fn physical_select(input: &Value) -> Value {
    let cands: Vec<_> = input["candidates"].as_array().unwrap().iter().map(candidate_from).collect();
    let got = select_native_physical_attack_candidate(&cands);
    let (sel, ok) = got.as_ref().copied().unwrap_or_default();
    json!({"selection": selection_val(&sel), "ok": ok, "err": err_of(&got)})
}

// ---- 以戰場底稿為輸入 ----

fn score1598a(ctx: &Ctx, input: &Value) -> Value {
    let k = i(input, "fixture") as usize;
    let fx = &ctx.fixtures[k];
    let units = ctx.units(k, &input["edits"]);
    let records = native_ai_scoring_records(&units).unwrap();
    let base = native_composition_base_flags(fx.w, fx.h, &fx.events).unwrap();
    let actor = i(input, "actor");
    let m = i(input, "skip_mod");
    let mut gate = skip_gate(m.max(1));
    let got = score_native_ai1598_a(
        fx.w,
        fx.h,
        &records,
        units.len() as i64,
        actor,
        i(input, "selector"),
        Some(&units[actor as usize]),
        &ctx.book,
        &base,
        &fx.terrain,
        &ctx.cost_rows[i(input, "row") as usize],
        (m != 0).then_some(&mut gate as &mut dyn FnMut(i64) -> bool),
    );
    json!({"result": r1598a_val(&got.clone().unwrap_or_default()), "err": err_of(&got)})
}

fn score1567e(ctx: &Ctx, input: &Value) -> Value {
    let k = i(input, "fixture") as usize;
    let fx = &ctx.fixtures[k];
    let units = ctx.units(k, &input["edits"]);
    let records = native_ai_scoring_records(&units).unwrap();
    let base = native_composition_base_flags(fx.w, fx.h, &fx.events).unwrap();
    let got = score_native_ai1567_e(
        fx.w,
        fx.h,
        &records,
        units.len() as i64,
        i(input, "actor"),
        i(input, "selector"),
        &ctx.item_rows,
        &ctx.book,
        &base,
    );
    json!({"result": r1567e_val(&got.clone().unwrap_or_default()), "err": err_of(&got)})
}

fn item_source(ctx: &Ctx, input: &Value) -> Value {
    let k = i(input, "fixture") as usize;
    let units = ctx.units(k, &input["edits"]);
    let records = native_ai_scoring_records(&units).unwrap();
    let actor = i(input, "actor") as usize;
    let got = resolve_native_ai_physical_item_source(&records[actor * 0x50..], &ctx.item_rows);
    let (src, found) = got.clone().unwrap_or_default();
    json!({"source": item_source_val(&src), "found": found, "err": err_of(&got)})
}

/// Go 黃金工具的確定性 score-input 來源（`resolver`）。
fn resolver(
    fail_index: i64,
) -> impl FnMut(NativeAIPhysicalAttackRawCandidate) -> Result<NativePhysicalAttackScoreInput, BattleError> {
    move |raw| {
        if raw.target_index == fail_index {
            return Err(BattleError::new(format!("resolver refused {}", raw.target_index)));
        }
        let le16 = |r: &[u8], off: usize| u16::from_le_bytes([r[off], r[off + 1]]) as i64;
        let helper = if (raw.destination.x + raw.destination.y + raw.target_index) % 2 == 0 { 1 } else { -1 };
        Ok(NativePhysicalAttackScoreInput {
            actor_word48: le16(&raw.actor_record, 0x48),
            actor_word4_a: le16(&raw.actor_record, 0x4a),
            target_word48: le16(&raw.target_record, 0x48),
            target_word4_a: le16(&raw.target_record, 0x4a),
            target_word40: le16(&raw.target_record, 0x40),
            raw_target_byte8: raw.target_record[8],
            raw_helper1_debe_result: helper,
        })
    }
}

fn physical_build(ctx: &Ctx, input: &Value) -> Value {
    let k = i(input, "fixture") as usize;
    let fx = &ctx.fixtures[k];
    let units = ctx.units(k, &Value::Null);
    let records = fx.records.clone().unwrap();
    let base = native_composition_base_flags(fx.w, fx.h, &fx.events).unwrap();
    let selector = i(input, "selector");
    let movement = if b(input, "runtime") {
        native_command_runtime_flags(fx.w, fx.h, &fx.events, &units, selector as u8).unwrap()
    } else {
        base.clone()
    };
    let g = &input["geometry"];
    let geometry = NativeAIPhysicalAttackGeometry {
        target_mode: i(g, "TargetMode"),
        target_inner_mark: i(g, "TargetInnerMark"),
        target_code: i(g, "TargetCode"),
    };
    let mut resolve = resolver(i(input, "fail_index"));
    let got = build_native_ai_physical_attack_candidates(
        fx.w,
        fx.h,
        &records,
        units.len() as i64,
        i(input, "actor"),
        selector,
        i(input, "budget"),
        geometry,
        &movement,
        &base,
        &fx.terrain,
        &ctx.cost_rows[i(input, "row") as usize],
        b(input, "resolver").then_some(&mut resolve as &mut NativeAIPhysicalAttackScoreResolver<'_>),
    );
    let mut res = json!({
        "candidates": got.as_ref().map(|v| v.iter().map(candidate_val).collect::<Vec<_>>()).unwrap_or_default(),
        "err": err_of(&got),
    });
    if let Ok(cands) = &got {
        let sel = select_native_physical_attack_candidate(cands);
        let (x, ok) = sel.as_ref().copied().unwrap_or_default();
        res["selection"] = selection_val(&x);
        res["ok"] = json!(ok);
        res["select_err"] = err_of(&sel);
    }
    res
}

fn phase(ctx: &Ctx, input: &Value) -> Value {
    let st = ctx.state(&input["state"]);
    let inputs: Vec<_> = input["inputs"]
        .as_array()
        .unwrap()
        .iter()
        .map(|x| {
            let m = i(x, "skip_mod");
            NativeAIPhaseZeroInput {
                unit_index: i(x, "unit"),
                cost_row: ctx.cost_rows[i(x, "row") as usize].clone(),
                skip: (m != 0).then(|| Callback::new(std::sync::Arc::new(move |index: i64| index % m == 0) as _)),
            }
        })
        .collect();
    let got = build_native_ai_phase_diagnostic_plan(Some(&st), &ctx.book, &ctx.item_rows, &inputs);
    let (plan, diags) = got.clone().unwrap_or_default();
    json!({"plan": phase_plan_val(&plan), "diagnostics": diags.iter().map(diagnostic_val).collect::<Vec<_>>(),
        "err": err_of(&got)})
}

fn mode5(ctx: &Ctx, input: &Value) -> Value {
    let mut st = ctx.state(&input["state"]);
    let (event, dest) = (i(input, "event") as u8, pair(&input["dest"]));
    let mut res = serde_json::Map::new();
    let mut lookups = Vec::new();
    for id in [event as i64, i(input, "lookup2")] {
        let c = st.native_ai_mode5_event_cell(id as u8);
        let cell = c.as_ref().copied().unwrap_or_default();
        lookups.push(json!({"id": id, "cell": [cell.x, cell.y], "err": err_of(&c)}));
    }
    res.insert("lookups".into(), json!(lookups));
    let (tiles, ok) = st.native_map_draw_tiles();
    res.insert("draw_before".into(), json!(tiles));
    res.insert("draw_ok".into(), json!(ok));
    let before = snapshot(&st.units);
    let target = (!b(input, "nil")).then_some(i(input, "unit") as usize);
    let mut cues = Vec::new();
    let err = if b(input, "audio") {
        let mut emit = |cue: NativeAIMode5AudioCue| cues.push(cue_val(&cue));
        st.apply_native_ai_mode5_event_with_audio_cue(target, event, dest, Some(&mut emit))
    } else {
        st.apply_native_ai_mode5_event(target, event, dest)
    };
    res.insert("err".into(), err_of(&err));
    res.insert("cues".into(), json!(cues));
    res.insert("changed".into(), changed(&before, &st.units));
    res.insert("event_state".into(), json!(st.native_event_state.to_vec()));
    res.insert("grid".into(), json!(hex(&st.native_map_event_grid)));
    let opened: Vec<_> = st.opened_treasure.iter().flatten().map(|(slot, v)| json!([slot, *v as i64])).collect();
    res.insert("opened".into(), json!(opened));
    let (tiles, ok) = st.native_map_draw_tiles();
    res.insert("draw_after".into(), json!(tiles));
    res.insert("draw_after_ok".into(), json!(ok));
    let go_at = case_tile_at(input);
    let at: Vec<_> = go_at
        .iter()
        .map(|&(x, y)| {
            let (tile, ok) = st.native_map_tile_at(x, y);
            json!({"x": x, "y": y, "tile": tile, "ok": ok})
        })
        .collect();
    res.insert("tile_at".into(), json!(at));
    Value::Object(res)
}

/// mode5 的 `tile_at` 查詢座標放在結果裡（Go 產生器在呼叫前抽樣）；Rust 從輸入的 `tile_at_query` 取。
fn case_tile_at(input: &Value) -> Vec<(i64, i64)> {
    input["tile_at_query"]
        .as_array()
        .unwrap()
        .iter()
        .map(|p| (p[0].as_i64().unwrap(), p[1].as_i64().unwrap()))
        .collect()
}

fn player_move(ctx: &Ctx, input: &Value) -> Value {
    let st = ctx.state(&input["state"]);
    let u = (!b(input, "nil")).then_some(i(input, "actor") as usize);
    let got = st.plan_native_player_movement(u);
    let mut res = json!({"err": err_of(&got)});
    let mut paths = Vec::new();
    if let Ok(p) = &got {
        let reach: Vec<_> = {
            let mut cells: Vec<Cell> = p.reach.keys().copied().collect();
            cells.sort_by_key(|c| (c.y, c.x));
            cells.iter().map(|c| json!([c.x, c.y])).collect()
        };
        res["field"] = json!(hex(&p.field));
        res["reach"] = json!(reach);
        for q in input["path_query"].as_array().unwrap() {
            let dest = pair(q);
            let path = p.path(dest);
            let cells: Vec<_> = path.as_deref().unwrap_or_default().iter().map(|c| json!([c.x, c.y])).collect();
            paths.push(json!({"dest": [dest.x, dest.y], "path": cells, "err": err_of(&path)}));
        }
    }
    res["paths"] = json!(paths);
    res
}

fn derived_damage(input: &Value) -> Value {
    let mut rng = GoMathRand::new(i(input, "seed"));
    let got = resolve_native_command_derived_strike_damage(
        i(input, "ap"),
        i(input, "dp"),
        i(input, "mult"),
        (!b(input, "nil_rng")).then_some(&mut rng as &mut dyn GoRand),
    );
    let (amount, damage) = got.as_ref().copied().unwrap_or_default();
    json!({"amount": amount, "damage": damage, "err": err_of(&got)})
}

fn command24(ctx: &Ctx, input: &Value) -> Value {
    let mut st = ctx.state(&input["state"]);
    let mut rng = GoMathRand::new(i(input, "seed"));
    let actor = (!b(input, "actor_nil")).then_some(i(input, "actor") as usize);
    let confirmed = usize::try_from(i(input, "confirmed")).ok();
    let (cursor, saved, id) = (pair(&input["cursor"]), pair(&input["saved"]), i(input, "id"));
    let before = snapshot(&st.units);
    let mut res = serde_json::Map::new();
    let mut plan: Option<NativeCommandDerivedStrikePlan> = None;
    let err = match s(input, "kind") {
        "plan" => st.plan_native_command_derived_strike(actor, confirmed, id, Some(&mut rng)).map(|p| plan = Some(p)),
        "plan29" => st.plan_native_command29_at_cursor(actor, cursor, Some(&mut rng)).map(|p| plan = Some(p)),
        "plan30" => st.plan_native_command30(actor, saved, cursor, Some(&mut rng)).map(|p| plan = Some(p)),
        "execute" => st.execute_native_command_derived_strike(actor, confirmed, id, Some(&mut rng)).map(|r| {
            res.insert("results".into(), json!(r.iter().map(damage24_val).collect::<Vec<_>>()));
        }),
        "execute30" => st.execute_native_command30(actor, saved, cursor, Some(&mut rng)).map(|r| {
            res.insert("results".into(), json!(r.iter().map(damage24_val).collect::<Vec<_>>()));
        }),
        "targets29" => st.native_command29_effect_targets(actor, cursor).map(|t| {
            res.insert("targets".into(), json!(t));
        }),
        "targets30" => st.native_command30_effect_targets(actor, saved, cursor).map(|t| {
            res.insert("targets".into(), json!(t));
        }),
        other => panic!("未知的 command24 kind {other}"),
    };
    // Go 的 execute／targets 失敗時仍輸出 nil 結果（null），Rust 補上同樣的鍵。
    match s(input, "kind") {
        "execute" | "execute30" => {
            res.entry("results").or_insert(Value::Null);
        }
        "targets29" | "targets30" => {
            res.entry("targets").or_insert(Value::Null);
        }
        _ => {}
    }
    res.insert("err".into(), err_of(&err));
    res.insert("plan".into(), plan.as_ref().map_or(Value::Null, strike_plan_val));
    res.insert("plan_nil".into(), json!(plan.is_none()));
    res.insert("rng_after".into(), json!(rng.int63()));
    res.insert("changed_plan".into(), changed(&before, &st.units));
    let mut steps = Vec::new();
    if let Some(plan) = plan.as_mut() {
        for step in input["script"].as_array().unwrap() {
            let (index, stage) = (i(step, "index"), i(step, "stage"));
            let mut projection = Value::Null;
            let serr = match s(step, "op") {
                "mp" => apply_native_command_derived_strike_mp(&mut st.units, Some(plan)),
                "target" => apply_native_command_derived_strike_target(&mut st.units, Some(plan), index),
                "stage" => apply_native_command28_target_stage(&mut st.units, Some(plan), index, stage),
                "complete" => complete_native_command_derived_strike(&mut st.units, Some(plan)),
                _ => {
                    restore_native_command30_targets(&mut st.units, Some(plan));
                    Ok(())
                }
            };
            if s(step, "op") == "stage" {
                let r = native_command28_stage_result(&st.units, Some(plan), index, stage);
                projection =
                    json!({"result": damage24_val(&r.as_ref().copied().unwrap_or_default()), "err": err_of(&r)});
            }
            steps.push(json!({"err": err_of(&serr), "projection": projection}));
        }
    }
    res.insert("steps".into(), json!(steps));
    res.insert("changed".into(), changed(&before, &st.units));
    Value::Object(res)
}
