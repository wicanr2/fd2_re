//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-a` 的黃金差分：fd2-rules A 組（單位、記錄與物品欄基礎）。
//!
//! 由 `tools/rust_golden_rules_a.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_A`；真實輸入（受版控的
//! `remake/assets` 與 `docs/data/exe_tables`）掛在 `FD2_GOLDEN_REPO` 底下，先核對黃金檔記錄的
//! SHA-256。設定 `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依 `op` 在 Rust 端重做同一串呼叫，結果與 Go 逐欄比對；單位以 Go 欄位名的 JSON
//! 比對（`Unit` 的 serde 名稱與 Go 相同）。A 組自己的錯誤文字必須逐字相同；Go 標準函式庫產生的錯誤
//! （讀檔、JSON 解碼）只比接受或拒收。比對時的表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的
//! 行為列在 [`KNOWN_DIFFERENCES`]。

use fd2_rules::battle::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為（PORTING.md 決定 2、`rust-port-no-go-stdlib`）。黃金案例都不落在這些輸入上，
/// 差分不需要為它們特別處理；列在這裡讓後續分組知道邊界。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文（`Cell` 另收 `x`／`y`）。
    "JSON 欄位名大小寫",
    // Go 把 null 讀進整數、布林等欄位時保留零值；Rust 只在切片、字串、指標（Option）欄位接受 null。
    "純量欄位的 null",
    // Go 重複的鍵以最後一個為準；serde 對具名欄位重複直接拒收。
    "重複的 JSON 鍵",
    // Go 的非 nil 空 []byte 寫成 ""；Rust 的空 Vec<u8> 寫成 null（需要區分時用 go_bytes::option）。
    "空 []byte 的輸出",
    // loadNativeTurnEventControls 的目錄名解析只實作 Sscanf("map%d") 的常見寫法（空白、正負號、十進位）。
    "map%d 目錄名的 Sscanf 細節",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列、空字串相同",
    // Go 把整數值的 float64 寫成 255，serde_json 寫成 255.0；只比數值。
    "數字只比數值",
    // Go JSON 解碼與讀檔錯誤屬標準函式庫（rust-port-no-go-stdlib），只比接受或拒收。
    "非 A 組自己的錯誤只比接受或拒收",
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

fn byte_list(v: &Value) -> Vec<u8> {
    ints(v).into_iter().map(|x| x as u8).collect()
}

fn unit_from(v: &Value) -> Unit {
    serde_json::from_value(v.clone()).unwrap_or_else(|e| panic!("Unit JSON 讀不進來：{e}\n{v}"))
}

fn unit_val(u: &Unit) -> Value {
    serde_json::to_value(u).unwrap()
}

fn units_val(units: &[Unit]) -> Value {
    Value::Array(units.iter().map(unit_val).collect())
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

/// A 組自己的錯誤逐字比對；`own` 為假時（Go 標準函式庫的錯誤）只比接受或拒收。
fn check_err(ctx: &str, go: &Value, rust: &Value, own: impl Fn(&str) -> bool) {
    match (go.as_str(), rust.as_str()) {
        (None, None) => {}
        (Some(g), Some(r)) if !own(g) => {
            let _ = r;
        }
        _ => check(ctx, go, rust),
    }
}

fn all_own(_: &str) -> bool {
    true
}

// ---- Go 結構的 JSON（沒有 json 標籤，鍵是 Go 欄位名）----

fn damage_val(d: &NativeRawDamageResult) -> Value {
    json!({"UnitIndex": d.unit_index, "Rolled": d.rolled, "Actual": d.actual, "RNGState": d.rng_state})
}

fn restore_val(r: &NativeRawRestoreResult) -> Value {
    json!({"UnitIndex": r.unit_index, "Rolled": r.rolled, "Actual": r.actual, "Score": r.score, "RNGState": r.rng_state})
}

fn base_val(x: &NativeFutureConstructorBase) -> Value {
    json!({"RecordRace": x.record_race, "RecordClass": x.record_class, "BaseAP": x.base_ap, "BaseDP": x.base_dp,
        "BaseDX": x.base_dx, "Mobility": x.mobility, "MaxHP": x.max_hp, "MaxMP": x.max_mp})
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

fn layer_val(e: &fd2_formats_layer::Unit) -> Value {
    json!({"X": e.x, "Y": e.y, "Slot": e.slot, "Pose": e.pose, "MotionOffset": e.motion_offset,
        "Flags": e.flags, "ForceBase": e.force_base, "Inactive": e.inactive})
}

fn fg_val(e: &fd2_formats_layer::Foreground) -> Value {
    json!({"X": e.x, "Y": e.y, "Pose": e.pose, "MotionOffset": e.motion_offset, "Inactive": e.inactive,
        "Unit7": e.unit7, "Race": e.race, "Class": e.class})
}

mod fd2_formats_layer {
    pub use fd2_formats::fdicon::{NativeForegroundLayerEntry as Foreground, NativeUnitLayerEntry as Unit};
}

fn hud_runtime_val(r: &NativeMapHUDRuntimeState) -> Value {
    json!({"DisplayGateA": r.display_gate_a, "DisplayGateB": r.display_gate_b, "AnchorX": r.anchor_x})
}

fn hud_persistent_val(p: &NativeMapHUDPersistentState) -> Value {
    json!({"DisplayGateA": p.display_gate_a, "AnchorX": p.anchor_x, "HasDisplayGateA": p.has_display_gate_a,
        "HasAnchorX": p.has_anchor_x})
}

fn hud_persistent_from(v: &Value) -> NativeMapHUDPersistentState {
    NativeMapHUDPersistentState {
        display_gate_a: i(v, "DisplayGateA") as u8,
        anchor_x: i(v, "AnchorX"),
        has_display_gate_a: b(v, "HasDisplayGateA"),
        has_anchor_x: b(v, "HasAnchorX"),
    }
}

/// Go `stateSummary` 的 Rust 版。
fn state_summary(st: &State) -> Value {
    let mut cells: Vec<(&Cell, &Treasure)> = st.treasures.iter().flatten().collect();
    cells.sort_by_key(|(c, _)| (c.y, c.x));
    let treasures: Vec<Value> = cells
        .iter()
        .map(|(c, t)| {
            json!({"x": c.x, "y": c.y, "slot": t.slot, "kind": t.kind, "native_type": t.native_type,
                "value": t.value, "hidden": t.hidden})
        })
        .collect();
    let camps = [OWN, ALLY, ENEMY];
    json!({
        "w": st.w, "h": st.h, "turn": st.turn, "native_round_counter": st.native_round_counter,
        "own_deploy": st.own_deploy.iter().map(|c| [c.x, c.y]).collect::<Vec<_>>(),
        "cost": st.cost,
        "composition": hex(&st.native_composition_event_bytes),
        "has_event_grid": st.has_native_map_event_grid, "event_grid": hex(&st.native_map_event_grid),
        "has_turn_controls": st.has_native_turn_event_control_state,
        "turn_controls": st.native_turn_event_controls.iter().map(|c| [c.turn, c.event_id, c.raw_camp]).collect::<Vec<_>>(),
        "field_event_slots": st.native_field_event_slots,
        "field_events": st.native_field_events.iter().map(|e| [e.event_id, e.selector]).collect::<Vec<_>>(),
        "field_event_rules": serde_json::to_value(&st.native_field_event_rules).unwrap(),
        "tile_blit_modes": hex(&st.native_tile_blit_modes), "terrain_control": hex(&st.native_terrain_control),
        "terrain_move_codes": hex(&st.native_terrain_move_codes),
        "treasures": treasures,
        "treasure_rules": serde_json::to_value(&st.native_treasure_event_rules).unwrap(),
        "has_chests": st.has_native_chest_control_state,
        "chests": st.native_chest_controls.iter().map(|c| json!([c.raw_type, c.value])).collect::<Vec<_>>(),
        "alive": camps.iter().map(|&c| st.alive_count(c)).collect::<Vec<_>>(),
        "pending": camps.iter().map(|&c| st.pending_count(c)).collect::<Vec<_>>(),
        "units": units_val(&st.units),
    })
}

struct Ctx {
    repo: PathBuf,
    item_rows: Vec<u8>,
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
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_A").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let (Some(path), Some(repo)) = (path, repo) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_RULES_A 或 FD2_GOLDEN_REPO");
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_A／FD2_GOLDEN_REPO");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let tmp = std::env::temp_dir().join(format!("fd2-golden-rules-a-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&tmp);
    let item_rows = unhex(s(&doc["real"], "item_rows"));
    Some((doc, Ctx { repo, item_rows, tmp }))
}

#[test]
fn golden_rules_a() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-a/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    for file in doc["real"]["files"].as_array().unwrap() {
        let raw = std::fs::read(ctx.real(s(file, "path"))).unwrap_or_else(|e| panic!("讀不到 {}：{e}", file["path"]));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    let mut counts = std::collections::BTreeMap::<String, usize>::new();
    for case in doc["cases"].as_array().unwrap() {
        let op = s(case, "op");
        let ctx_name = format!("case {} {op}", case["index"]);
        let (input, result) = (&case["input"], &case["result"]);
        match op {
            "load_map" => load_map(&ctx, &ctx_name, input, result),
            "selector_batch" => selector_batch(&ctx, &ctx_name, input, result),
            "spawn_groups" => spawn_groups(&ctx, &ctx_name, input, result),
            "load_command_records" | "load_movement_rows" | "load_command_learn" | "load_command_learn_selectors" => {
                loader(&ctx, &ctx_name, op, input, result, case["index"].as_i64().unwrap())
            }
            "unit_inventory" => unit_inventory(&ctx_name, input, result),
            "inventory_pure" => inventory_pure(&ctx_name, input, result),
            "records" => records(&ctx_name, input, result),
            "equipment" => equipment(&ctx_name, input, result),
            "constructor_base" => constructor_base(&ctx_name, input, result),
            "materialize_constructor" => materialize_constructor(&ctx, &ctx_name, input, result),
            "commands" => commands(&ctx_name, input, result),
            "tables" => tables(&ctx_name, input, result),
            "destinations" => destinations(&ctx_name, input, result),
            "panel_records" => panel_records(&ctx_name, input, result),
            "transient" => transient(&ctx_name, input, result),
            "map_presentation" => map_presentation(&ctx_name, input, result),
            "hud" => hud(&ctx_name, input, result),
            "load_synthetic" => load_synthetic(&ctx, &ctx_name, input, result, case["index"].as_i64().unwrap()),
            _ => panic!("{ctx_name}：未知的 op"),
        }
        *counts.entry(op.to_string()).or_default() += 1;
    }
    let _ = std::fs::remove_dir_all(&ctx.tmp);
    let total: usize = counts.values().sum();
    eprintln!("rules-a 黃金差分：{total} 案例全部相符 {counts:?}");
}

fn load_map(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let r = load(&ctx.real(s(input, "path")));
    check_err(name, &result["err"], &err_of(&r), all_own);
    let Ok(mut st) = r else { return };
    check(&format!("{name} state"), &result["state"], &state_summary(&st));
    let bind = st.bind_native_future_item_rows(&ctx.item_rows);
    check(&format!("{name} bind_err"), &result["bind_err"], &err_of(&bind));
    check(&format!("{name} units_after_bind"), &result["units_after_bind"], &units_val(&st.units));
}

fn selector_batch(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let fresh = load(&ctx.real(s(input, "path"))).unwrap();
    let mut st = State::default();
    (st.w, st.h) = (fresh.w, fresh.h);
    st.append_native_map_selector_batch_or_legacy(fresh.units);
    let error = st.native_map_selector_error.as_ref().map(|e| Value::String(e.to_string())).unwrap_or(Value::Null);
    check(&format!("{name} error"), &result["error"], &error);
    check(&format!("{name} units"), &result["units"], &units_val(&st.units));
    check(&format!("{name} has_cycle"), &result["has_cycle"], &json!(st.has_native_map_cycle_state));
    let frame = st.native_map_frame_roster();
    check(&format!("{name} frame_err"), &result["frame_err"], &err_of(&frame));
    if let Ok(frame) = frame {
        check(&format!("{name} frame_units"), &result["frame_units"], &frame.units.iter().map(layer_val).collect());
        check(
            &format!("{name} frame_foreground"),
            &result["frame_foreground"],
            &frame.foreground.iter().map(fg_val).collect(),
        );
    }
    let keys: Vec<i64> = (0..st.units.len())
        .map(|k| {
            let (key, ok) = st.native_map_sprite_key(Some(k));
            if ok { key } else { -1 }
        })
        .collect();
    check(&format!("{name} sprite_keys"), &result["sprite_keys"], &json!(keys));
}

fn spawn_groups(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let spawn = load(&ctx.real(s(input, "path"))).unwrap();
    let min_group = i(input, "min_group");
    let mut order: Vec<i64> = spawn.units.iter().map(|u| u.group).collect();
    order.sort_unstable();
    order.dedup();
    let mut st = State::default();
    (st.w, st.h) = (spawn.w, spawn.h);
    let mut first = Vec::new();
    for mut u in spawn.units {
        if u.group == min_group {
            first.push(u);
        } else {
            u.on_field = false;
            st.pending_groups.insert(u.group, true);
            st.roster.push(u);
        }
    }
    if b(input, "native_attempt") {
        st.append_native_map_selector_batch_or_legacy(first);
    } else {
        for u in first {
            st.add_unit(u);
        }
    }
    let mut steps = Vec::new();
    for (k, &g) in order.iter().enumerate() {
        if g == min_group {
            continue;
        }
        let camp = Camp(k as i64 % 3);
        let (change, act) = (k % 2 == 0, k % 3 != 0);
        let pending_before = [st.pending_count(OWN), st.pending_count(ALLY), st.pending_count(ENEMY)];
        let n = st.spawn_group(g, camp, change, act);
        st.pending_groups.remove(&g);
        steps.push(json!({"group": g, "camp": camp.0, "change": change, "act": act, "n": n,
            "pending_before": pending_before, "roster_left": st.roster.len()}));
    }
    check(&format!("{name} steps"), &result["steps"], &Value::Array(steps));
    let error = st.native_map_selector_error.as_ref().map(|e| Value::String(e.to_string())).unwrap_or(Value::Null);
    check(&format!("{name} error"), &result["error"], &error);
    check(&format!("{name} units"), &result["units"], &units_val(&st.units));
}

fn loader(ctx: &Ctx, name: &str, op: &str, input: &Value, result: &Value, index: i64) {
    let path = match input.get("path") {
        Some(p) => ctx.real(p.as_str().unwrap()),
        None => ctx.temp_file(&format!("loader-{index}.json"), s(input, "text")),
    };
    // A 組自己的拒收訊息以這些前綴開頭；其餘是 Go JSON 解碼錯誤。
    let own = |m: &str| {
        ["command learn", "native command", "native movement"].iter().any(|p| m.starts_with(p)) && !m.contains("json:")
    };
    match op {
        "load_command_records" => {
            let r = load_native_command_records(&path);
            check_err(name, &result["err"], &err_of(&r), own);
            let records = r.map(|v| v.iter().map(command_val).collect::<Vec<_>>());
            check(name, &result["records"], &records.map(Value::Array).unwrap_or(Value::Null));
        }
        "load_movement_rows" => {
            let r = load_native_movement_cost_rows(&path);
            check_err(name, &result["err"], &err_of(&r), own);
            let rows: Vec<String> = r.map(|v| v.iter().map(|row| hex(row)).collect()).unwrap_or_default();
            check(name, &result["rows"], &json!(rows));
        }
        "load_command_learn" => {
            let r = load_command_learn(&path);
            check_err(name, &result["err"], &err_of(&r), own);
            check(name, &result["table"], &r.map(|t| serde_json::to_value(t).unwrap()).unwrap_or(Value::Null));
        }
        _ => {
            let r = load_command_learn_selectors(&path);
            check_err(name, &result["err"], &err_of(&r), own);
            check(name, &result["selectors"], &r.map(|t| serde_json::to_value(t).unwrap()).unwrap_or(Value::Null));
        }
    }
}

fn unit_inventory(name: &str, input: &Value, result: &Value) {
    let mut units = vec![unit_from(&input["unit"]), unit_from(&input["other"])];
    for (k, op) in result["ops"].as_array().unwrap().iter().enumerate() {
        let ctx = format!("{name} op {k} {}", op["op"]);
        let got = match s(op, "op") {
            "add" => json!({"ok": units[0].add_inventory_item(i(op, "id"), b(op, "equipped"))}),
            "remove_index" => json!({"ok": units[0].remove_inventory_index(i(op, "i"))}),
            "remove_compact" => {
                json!({"err": err_of(&remove_native_compact_inventory(Some(&mut units[0]), i(op, "i")))})
            }
            "replace_reward" => json!({"err": err_of(&replace_native_full_inventory_reward(
                Some(&mut units[0]),
                i(op, "sel"),
                i(op, "item")
            ))}),
            "validate" => json!({"err": err_of(&validate_native_inventory_projection(Some(&units[0])))}),
            "records" => {
                let r = native_inventory_records(&units[..1], 1);
                json!({"err": err_of(&r), "buf": r.map(|b| hex(&b)).unwrap_or_default()})
            }
            "transfer" => {
                json!({"err": err_of(&transfer_native_inventory_item(&mut units, Some(0), i(op, "i"), Some(1)))})
            }
            _ => json!({"err": err_of(&transfer_native_inventory_item(&mut units, Some(0), i(op, "i"), Some(0)))}),
        };
        for (key, value) in got.as_object().unwrap() {
            check(&format!("{ctx} {key}"), &op[key], value);
        }
        check(&format!("{ctx} unit"), &op["unit"], &unit_val(&units[0]));
        check(&format!("{ctx} other"), &op["other"], &unit_val(&units[1]));
    }
}

fn inventory_pure(name: &str, input: &Value, result: &Value) {
    let (source, flags, slots) = (ints(&input["source"]), ints(&input["flags"]), ints(&input["slots"]));
    let r = native_inventory_flags_from_source(&source);
    check(&format!("{name} from_source_err"), &result["from_source_err"], &err_of(&r));
    check(&format!("{name} from_source"), &result["from_source"], &json!(r.unwrap_or_default()));
    let r = native_inventory_compact_eligible(&flags, &slots, i(input, "idx"));
    check(&format!("{name} eligible_err"), &result["eligible_err"], &err_of(&r));
    check(&format!("{name} eligible"), &result["eligible"], &json!(r.unwrap_or(false)));
    let r = native_inventory_available_count(&flags);
    check(&format!("{name} count_err"), &result["count_err"], &err_of(&r));
    check(&format!("{name} count"), &result["count"], &json!(r.unwrap_or(0)));
    let r = native_equipped_inventory_slot(&flags, &slots, i(input, "a2"));
    check(&format!("{name} slot_err"), &result["slot_err"], &err_of(&r));
    check(&format!("{name} slot"), &result["slot"], &json!(r.unwrap_or(-1)));
}

fn records(name: &str, input: &Value, result: &Value) {
    let mut buf = unhex(s(input, "buf"));
    let rng = input.get("rng").and_then(Value::as_i64).unwrap_or(0) as u16;
    let indices = byte_list(&input["indices"]);
    let mut got = serde_json::Map::new();
    let mut put = |k: &str, v: Value| {
        got.insert(k.to_string(), v);
    };
    match s(input, "op") {
        "assign_reserved" => {
            let r = assign_native_reserved_item(&mut buf, i(input, "unit"), i(input, "item") as u8);
            put("err", err_of(&r));
            put("ok", json!(r.unwrap_or(false)));
        }
        "remove_slot" => {
            put("err", err_of(&remove_native_inventory_slot(&mut buf, i(input, "unit"), i(input, "slot"))))
        }
        "find_item" => {
            let r = find_native_inventory_item(&buf, i(input, "item") as u8);
            put("err", err_of(&r));
            let (u, sl) = r.unwrap_or((-1, -1));
            put("unit", json!(u));
            put("slot", json!(sl));
        }
        "find_item_in_unit" => {
            let r = find_native_inventory_item_in_unit(&buf, i(input, "unit"), i(input, "item") as u8);
            put("err", err_of(&r));
            put("slot", json!(r.unwrap_or(-1)));
        }
        "occupied_count" => {
            let r = native_inventory_occupied_count(&buf, i(input, "unit"));
            put("err", err_of(&r));
            put("count", json!(r.unwrap_or(0)));
        }
        "set_coordinates" => put(
            "err",
            err_of(&set_native_unit_coordinate_bytes(
                &mut buf,
                i(input, "unit"),
                i(input, "x") as u8,
                i(input, "y") as u8,
            )),
        ),
        "rewrite_marker" => put("err", err_of(&rewrite_native_marker1_f(&mut buf, i(input, "count")))),
        "set_byte5" => put("err", err_of(&set_native_record_byte5_one(&mut buf, i(input, "index")))),
        "byte5_bit0" => {
            let r = native_record_byte5_bit0(&buf, i(input, "index"));
            put("err", err_of(&r));
            put("value", json!(r.unwrap_or(0)));
        }
        "set_bit7" => put("err", err_of(&set_native_record_bit7(&mut buf, i(input, "index")))),
        "clear_bit7" => put("err", err_of(&clear_native_record_bit7_all(&mut buf, i(input, "count")))),
        "clear_unit_byte" => {
            let r = clear_native_unit_byte(&mut buf, &indices, i(input, "offset"));
            put("err", err_of(&r));
            let (results, total) = r.unwrap_or_default();
            let results: Vec<Value> = results
                .iter()
                .map(|x| json!({"UnitIndex": x.unit_index, "Cleared": x.cleared, "RawValue": x.raw_value}))
                .collect();
            put("results", Value::Array(results));
            put("total", json!(total));
        }
        "word_delta_remove" => {
            let r = apply_native_word_delta_and_remove(
                &mut buf,
                i(input, "target"),
                i(input, "offset"),
                i(input, "delta"),
                i(input, "removal_unit"),
                i(input, "removal_slot"),
            );
            put("err", err_of(&r));
            put("value", json!(r.unwrap_or(0)));
        }
        "word_subtract" => {
            let r = apply_native_raw_word_subtract(&mut buf, i(input, "unit"), i(input, "offset"), i(input, "amount"));
            put("err", err_of(&r));
            put("value", json!(r.unwrap_or(0)));
        }
        "hp_damage" => {
            let r = apply_native_raw_hp_damage(&mut buf, i(input, "unit"), i(input, "amount"), rng);
            put("err", err_of(&r));
            put("result", damage_val(&r.unwrap_or_default()));
        }
        "hp_restore" | "mp_restore" => {
            let r = if s(input, "op") == "hp_restore" {
                apply_native_raw_hp_restore(&mut buf, i(input, "unit"), i(input, "amount"), rng)
            } else {
                apply_native_raw_mp_restore(&mut buf, i(input, "unit"), i(input, "amount"), rng)
            };
            put("err", err_of(&r));
            put("result", restore_val(&r.unwrap_or_default()));
        }
        "hp_restore_list" => {
            let r = apply_native_raw_hp_restore_list(&mut buf, &indices, i(input, "amount"), rng);
            put("err", err_of(&r));
            // Go 失敗時回 NativeRawRestoreBatch{RNGState: rngState}。
            let batch = r.unwrap_or(NativeRawRestoreBatch { rng_state: rng, ..Default::default() });
            put(
                "result",
                json!({"Results": batch.results.iter().map(restore_val).collect::<Vec<_>>(),
                    "RNGState": batch.rng_state, "Score": batch.score}),
            );
        }
        "application" => {
            let r = apply_native_raw_application(&mut buf, &indices, i(input, "offset"), rng);
            put("err", err_of(&r));
            let (results, state, total) = r.unwrap_or((Vec::new(), rng, 0));
            let results: Vec<Value> = results
                .iter()
                .map(|x| {
                    json!({"TargetIndex": x.target_index, "Applied": x.applied, "Damage": damage_val(&x.damage),
                        "Marker": x.marker})
                })
                .collect();
            put("results", Value::Array(results));
            put("rng", json!(state));
            put("total", json!(total));
        }
        "flag_restore" => {
            let r = apply_native_raw_flag_restore(&mut buf, &indices, i(input, "offset"), rng);
            put("err", err_of(&r));
            let (results, state, total) = r.unwrap_or((Vec::new(), rng, 0));
            let results: Vec<Value> = results
                .iter()
                .map(|x| json!({"TargetIndex": x.target_index, "Applied": x.applied, "Restore": restore_val(&x.restore)}))
                .collect();
            put("results", Value::Array(results));
            put("rng", json!(state));
            put("total", json!(total));
        }
        "pair_step" => {
            let r = apply_native_raw_pair_step(&mut buf, &indices, rng);
            put("err", err_of(&r));
            let (results, state, total) = r.unwrap_or((Vec::new(), rng, 0));
            let results: Vec<Value> = results
                .iter()
                .map(|x| json!({"UnitIndex": x.unit_index, "Processed": x.processed, "Marker": x.marker}))
                .collect();
            put("results", Value::Array(results));
            put("rng", json!(state));
            put("total", json!(total));
        }
        "word_step" => {
            let r =
                apply_native_raw_word_step_at_offsets(&mut buf, &indices, rng, i(input, "marker"), i(input, "word"));
            put("err", err_of(&r));
            let (results, state, total) = r.unwrap_or((Vec::new(), rng, 0));
            let results: Vec<Value> = results
                .iter()
                .map(|x| {
                    json!({"UnitIndex": x.unit_index, "Processed": x.processed, "Marker": x.marker, "Delta": x.delta})
                })
                .collect();
            put("results", Value::Array(results));
            put("rng", json!(state));
            put("total", json!(total));
        }
        "persistent_copy" => {
            let mut persistent = unhex(s(input, "persistent"));
            let r = apply_native_persistent_record_copy(
                &buf,
                &mut persistent,
                i(input, "runtime_index"),
                i(input, "persistent_index"),
            );
            put("err", err_of(&r));
            put("persistent", json!(hex(&persistent)));
        }
        "persistent_identity" => {
            let r = find_native_persistent_identity(&buf, i(input, "count"), i(input, "identity") as u8);
            put("err", err_of(&r));
            let (index, ok) = r.unwrap_or((-1, false));
            put("index", json!(index));
            put("ok", json!(ok));
        }
        "relocation_allowed" => {
            // Go 用 assets/data/native_movement_cost_rows.json 的表；Rust 端讀同一份（雜湊已核對）。
            let rows = relocation_rows();
            let rows = if b(input, "short_rows") { &rows[..28] } else { &rows[..] };
            let r = native_relocation_destination_allowed(
                &buf,
                i(input, "count"),
                i(input, "target"),
                i(input, "x") as u8,
                i(input, "y") as u8,
                i(input, "terrain"),
                rows,
            );
            put("err", err_of(&r));
            put("ok", json!(r.unwrap_or(false)));
        }
        other => panic!("{name}：未知的 records op {other}"),
    }
    got.insert("buf".into(), json!(hex(&buf)));
    check(name, result, &Value::Object(got));
}

fn relocation_rows() -> Vec<Vec<u8>> {
    static ROWS: std::sync::OnceLock<Vec<Vec<u8>>> = std::sync::OnceLock::new();
    ROWS.get_or_init(|| {
        let repo = PathBuf::from(std::env::var_os("FD2_GOLDEN_REPO").unwrap());
        let path = repo.join("remake/assets/data/native_movement_cost_rows.json");
        load_native_movement_cost_rows(path.to_str().unwrap()).unwrap()
    })
    .clone()
}

fn equipment(name: &str, input: &Value, result: &Value) {
    let mut record = unhex(s(input, "record"));
    let table = unhex(s(input, "table"));
    let r = if b(input, "runtime") {
        apply_native_runtime_equipment_recalc(&mut record, &table)
    } else {
        apply_native_equipment_recalc(&mut record, &table)
    };
    check(name, result, &json!({"err": err_of(&r), "record": hex(&record)}));
}

fn constructor_base(name: &str, input: &Value, result: &Value) {
    let table: Option<NativeConstructorTable> = serde_json::from_value(input["table"].clone()).unwrap();
    let r = decode_native_future_constructor_base(table.as_ref(), i(input, "level") as u8);
    check(name, result, &json!({"err": err_of(&r), "base": base_val(&r.unwrap_or_default())}));
}

fn materialize_constructor(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let mut unit = unit_from(&input["unit"]);
    let rows = &ctx.item_rows[..i(input, "rows_len") as usize];
    let r = materialize_native_future_constructor(Some(&mut unit), rows);
    check(name, result, &json!({"err": err_of(&r), "unit": unit_val(&unit)}));
}

fn commands(name: &str, input: &Value, result: &Value) {
    let mut u = unit_from(&input["unit"]);
    let book: Vec<NativeCommandRecord> = input["book"].as_array().unwrap().iter().map(command_from).collect();
    let available: Vec<bool> = (-1..=40).map(|id| native_command_available(Some(&u), &book, id)).collect();
    let mut got = json!({
        "available": available,
        "ids": native_available_command_ids(Some(&u), &book),
        "ai_ids": native_available_ai_command_ids(Some(&u), &book),
        "scored": native_available_ai_scored_command_ids(Some(&u), &book),
        "native": u.native_command_ids(),
    });
    let cost = i(result, "spend_cost");
    got["spend_cost"] = json!(cost);
    got["spend_ok"] = json!(spend_native_command_mp(Some(&mut u), cost));
    got["mp_after"] = json!(u.mp);
    let raw = unhex(s(result, "mask_raw"));
    got["mask_raw"] = json!(hex(&raw));
    got["mask_err"] = err_of(&u.set_initial_command_mask(&raw));
    let enabled: Vec<Value> = result["enabled"]
        .as_array()
        .unwrap()
        .iter()
        .map(|e| {
            let id = i(e, "id");
            json!({"id": id, "ok": u.enable_native_command(id)})
        })
        .collect();
    got["enabled"] = Value::Array(enabled);
    got["mask_after"] = json!(u.native_command_mask);
    got["ids_after"] = json!(u.native_command_ids());
    check(name, result, &got);
}

fn tables(name: &str, input: &Value, result: &Value) {
    let off = native_class_compatibility_row_offset(i(input, "class"));
    let compat = native_class_item_compatible(i(input, "probe") as u8, &unhex(s(input, "row")));
    let row_off = native_item_effect_row_offset(i(input, "item"));
    let (route, ok) = native_item_word_delta_route_for_type(i(input, "type"));
    let got = json!({
        "class_off": off.clone().unwrap_or(0), "class_err": err_of(&off),
        "compat": compat.clone().unwrap_or(false), "compat_err": err_of(&compat),
        "row_off": row_off.clone().unwrap_or(0), "row_err": err_of(&row_off),
        "route": {"ItemType": route.item_type, "BaseStat": route.base_stat.0, "FieldOffset": route.field_offset,
            "PresentationCode": route.presentation_code},
        "route_ok": ok,
    });
    check(name, result, &got);
}

fn destinations(name: &str, input: &Value, result: &Value) {
    let r = native_ai_physical_destinations(
        i(input, "w"),
        i(input, "h"),
        &unhex(s(input, "records")),
        i(input, "count"),
        i(input, "actor"),
        i(input, "selector"),
        i(input, "budget"),
        &unhex(s(input, "flags")),
        &unhex(s(input, "codes")),
        &unhex(s(input, "cost_row")),
    );
    let cells: Vec<[i64; 2]> = r.clone().unwrap_or_default().iter().map(|c| [c.x, c.y]).collect();
    check(name, result, &json!({"cells": cells, "err": err_of(&r)}));
}

fn panel_records(name: &str, input: &Value, result: &Value) {
    let units: Vec<Unit> = input["units"].as_array().unwrap().iter().map(unit_from).collect();
    let record = native_item_panel_record_for_unit(Some(&units[0]));
    let scoring = native_ai_scoring_records(&units);
    let got = json!({
        "record": hex(&record.clone().unwrap_or_default()), "record_err": err_of(&record),
        "scoring": hex(&scoring.clone().unwrap_or_default()), "scoring_err": err_of(&scoring),
    });
    check(name, result, &got);
}

fn transient(name: &str, input: &Value, result: &Value) {
    let units: Vec<Unit> = input["units"].as_array().unwrap().iter().map(unit_from).collect();
    let records: Vec<NativeRuntimeRecordState> = input["raw"]
        .as_array()
        .unwrap()
        .iter()
        .map(|r| {
            let mut state = NativeRuntimeRecordState::default();
            state.raw.copy_from_slice(&unhex(r.as_str().unwrap()));
            state
        })
        .collect();
    // State 有 crate 內部欄位，外部不能用結構字面值建構。
    let mut st = State::default();
    st.units = units;
    st.native_runtime_records = records;
    st.has_native_runtime_unit_projection = b(input, "has_projection");
    let selector = i(input, "selector") as u8;
    let expiry = |list: &[NativeTransientExpiry]| -> Value {
        Value::Array(list.iter().map(|x| json!([x.unit.map_or(-1, |u| u as i64), x.offset])).collect())
    };
    let mut got = serde_json::Map::new();
    match s(input, "mode") {
        "tick" => {
            let expired = st.tick_native_transients_raw(selector);
            got.insert("expired".into(), expiry(&expired));
        }
        mode => {
            let r = if mode == "typed" {
                st.advance_native_transient_phase_typed(selector)
            } else {
                st.advance_native_transient_phase_raw(selector)
            };
            got.insert("err".into(), err_of(&r));
            let r = r.unwrap_or_default();
            let damage: Vec<[i64; 4]> = r.damage.iter().map(|d| [d.slot, d.amount, d.before, d.after]).collect();
            got.insert("damage".into(), json!(damage));
            got.insert("inactive".into(), json!(r.inactive_slots));
            got.insert("expired".into(), expiry(&r.expired));
        }
    }
    got.insert("units".into(), units_val(&st.units));
    got.insert("raw".into(), json!(st.native_runtime_records.iter().map(|r| hex(&r.raw)).collect::<Vec<_>>()));
    check(name, result, &Value::Object(got));
}

fn map_presentation(name: &str, input: &Value, result: &Value) {
    let mut u = unit_from(&input["unit"]);
    for (k, op) in result["ops"].as_array().unwrap().iter().enumerate() {
        let ctx = format!("{name} op {k} {}", op["op"]);
        let got = match s(op, "op") {
            "materialize" => json!({"err": err_of(&u.materialize_native_map_presentation())}),
            "pose" => json!({"ok": u.set_map_pose(i(op, "a"))}),
            "placement" => json!({"ok": u.set_map_placement(i(op, "a"), i(op, "b"), i(op, "c"))}),
            "coords_raw" => json!({"ok": u.set_native_map_coordinates_raw(i(op, "a"), i(op, "b"))}),
            "grid_motion" => json!({"ok": u.set_native_map_grid_motion(i(op, "a"), i(op, "b"))}),
            "finish_step" => json!({"ok": u.finish_native_map_grid_step(i(op, "a"), i(op, "b"), i(op, "c"))}),
            _ => {
                let (entry, ok) = u.native_unit_layer_entry();
                let (fg, ok_fg) = u.native_foreground_layer_entry();
                json!({"entry": layer_val(&entry), "ok": ok, "fg": fg_val(&fg), "ok_fg": ok_fg})
            }
        };
        for (key, value) in got.as_object().unwrap() {
            check(&format!("{ctx} {key}"), &op[key], value);
        }
        check(&format!("{ctx} unit"), &op["unit"], &unit_val(&u));
    }
}

fn hud(name: &str, input: &Value, result: &Value) {
    let (gate_a, gate_b, anchor) = (i(input, "gate_a") as u8, i(input, "gate_b") as u8, i(input, "anchor"));
    let mut st = State::default();
    let mut got = serde_json::Map::new();
    got.insert("advance_before".into(), json!(st.advance_native_map_hud_anchor(2, 6)));
    got.insert("materialized".into(), json!(st.materialize_native_map_hud_state(gate_a, gate_b, anchor)));
    let steps: Vec<Value> = result["steps"]
        .as_array()
        .unwrap()
        .iter()
        .map(|step| {
            let (x, y) = (i(step, "x"), i(step, "y"));
            let ok = st.advance_native_map_hud_anchor(x, y);
            json!({"x": x, "y": y, "ok": ok, "anchor": st.native_map_hud_state.anchor_x})
        })
        .collect();
    got.insert("steps".into(), Value::Array(steps));
    let mut p = hud_persistent_from(&result["persistent_before"]);
    got.insert("persistent_before".into(), hud_persistent_val(&p));
    got.insert("captured".into(), json!(p.capture_native_map_hud(st.native_map_hud_state)));
    got.insert("restored".into(), json!(p.restore_saved_gate_a(gate_b)));
    let (runtime, ok) = p.materialize_runtime(gate_a);
    got.insert("persistent".into(), hud_persistent_val(&p));
    got.insert("runtime".into(), hud_runtime_val(&runtime));
    got.insert("runtime_ok".into(), json!(ok));
    got.insert("state".into(), hud_runtime_val(&st.native_map_hud_state));
    check(name, result, &Value::Object(got));
}

fn load_synthetic(ctx: &Ctx, name: &str, input: &Value, result: &Value, index: i64) {
    let path = ctx.temp_file(&format!("load-{index}/units.json"), s(input, "units_json"));
    if let Some(map_json) = input["map_json"].as_str() {
        ctx.temp_file(&format!("load-{index}/map.json"), map_json);
    }
    let r = load(&path);
    // Load 自己的拒收訊息都以 "battle:" 開頭；其餘是 Go JSON 解碼錯誤。
    check_err(name, &result["err"], &err_of(&r), |m| m.starts_with("battle:"));
    let Ok(mut st) = r else { return };
    check(&format!("{name} state"), &result["state"], &state_summary(&st));
    let bind = st.bind_native_future_item_rows(&ctx.item_rows);
    check(&format!("{name} bind_err"), &result["bind_err"], &err_of(&bind));
    check(&format!("{name} units_after_bind"), &result["units_after_bind"], &units_val(&st.units));
    // Go 工具在領取前關掉事件格網（D 組 native_ai_mode5 的範圍），這裡照做。
    st.has_native_map_event_grid = false;
    let go_claims = result["claims"].as_array().unwrap();
    let mut claims = Vec::new();
    for y in 0..st.h {
        for x in 0..st.w {
            if !st.treasure_at(x, y).1 {
                continue;
            }
            let full = go_claims.get(claims.len()).is_some_and(|c| b(c, "full"));
            let mut claimer = Unit::default();
            (claimer.hp, claimer.max_hp, claimer.on_field, claimer.x, claimer.y) = (1, 1, true, x, y);
            if full {
                claimer.inventory = vec![1, 2, 3, 4, 5, 6, 7];
            }
            st.add_unit(claimer);
            let u = st.units.len() - 1;
            let (t, ok) = st.claim_treasure(Some(u), x, y);
            claims.push(json!({"x": x, "y": y, "full": full, "ok": ok, "slot": t.slot, "kind": t.kind,
                "value": t.value, "inventory": st.units[u].inventory, "event_state": st.native_event_state}));
        }
    }
    check(&format!("{name} claims"), &result["claims"], &Value::Array(claims));
}
