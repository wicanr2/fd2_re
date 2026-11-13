//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-c` 的黃金差分：fd2-rules C 組（物品效果與面板繪製）。
//!
//! 由 `tools/rust_golden_rules_c.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_C`；真實輸入掛在
//! `FD2_GOLDEN_REPO`（受版控的 `remake/assets`）、`FD2_GOLDEN_FLAME2`（原版 FLAME2）與
//! `FD2_GOLDEN_PACK`（分離素材包）底下，先逐檔核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依 `op` 在 Rust 端重做同一串呼叫，結果與 Go 逐欄比對；Go 結構以 Go 欄位名的
//! JSON 比對，繪製結果比 320×200 目的緩衝的 SHA-256。C 組自己的錯誤文字必須逐字相同（含內嵌的格式層
//! 錯誤）；Go 標準函式庫產生的錯誤（讀檔、JSON 解碼、hex 解碼）只比接受或拒收。比對時的表示法正規化
//! 列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。

use fd2_formats::{fdicon, fdother, fdtxt};
use fd2_rules::battle::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為（PORTING.md 決定 2、`rust-port-no-go-stdlib`）。黃金案例不依賴這些差異。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文（native_command_scene／palette_flash／item rows）。
    "JSON 欄位名大小寫",
    // Go 把 JSON 陣列讀進 [3]byte 時多的元素丟掉、少的補零；serde 的 [u8; 3] 要求剛好三個元素。
    "palette flash entries 的 [3]byte 長度",
    // Go 讀 null 到整數、布林欄位時保留零值；Rust 只在切片、字串與整份文件接受 null。
    "純量欄位的 null",
    // hex 解碼錯誤的內層文字屬 Go encoding/hex；Rust 只保留「decode native item effect row N: 」前綴。
    "hex 解碼錯誤文字",
    // Go 以 nil 指標呼叫 ApplyNativeItemCommandDamage 的 targets 元素會回報 nil target；Rust 的 targets
    // 是單位本體的切片，沒有 nil，這個分支不存在。
    "nil 的道具目標單位",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列、空字串相同",
    // Go 把整數值的 float64（Unit.Exp）寫成 73，serde_json 寫成 73.0；只比數值。
    "數字只比數值",
    // Go 讀檔、JSON 與 hex 解碼錯誤屬標準函式庫，只比接受或拒收。
    "非 C 組自己的錯誤只比接受或拒收",
    // 內嵌的 os 開檔錯誤（`open <路徑>: <原因>`）只比到路徑，原因文字屬 Go os 套件。
    "os 開檔錯誤只比到路徑",
];

const PANEL: usize = 320 * 200;

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

fn units_val(units: &[Unit]) -> Value {
    Value::Array(units.iter().map(|u| serde_json::to_value(u).unwrap()).collect())
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

/// 依 [`NORMALIZATIONS`]：字串裡的 `: open <路徑>: <原因>` 截到路徑為止。
fn trim_open_error(v: &Value) -> Value {
    match v {
        Value::String(text) => match text.find(": open ") {
            Some(at) => {
                let rest = &text[at + ": open ".len()..];
                let end = rest.find(": ").map_or(text.len(), |k| at + ": open ".len() + k);
                Value::String(text[..end].to_string())
            }
            None => v.clone(),
        },
        Value::Array(a) => Value::Array(a.iter().map(trim_open_error).collect()),
        Value::Object(o) => Value::Object(o.iter().map(|(k, x)| (k.clone(), trim_open_error(x))).collect()),
        _ => v.clone(),
    }
}

fn check(ctx: &str, go: &Value, rust: &Value) {
    let (go, rust) = (&trim_open_error(go), &trim_open_error(rust));
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

/// C 組自己的錯誤文字以這些前綴開頭；其他（Go 標準函式庫）只比接受或拒收。
fn own(g: &str) -> bool {
    ["native ", "battle: ", "ch28 post"].iter().any(|p| g.starts_with(p))
}

/// 把 Rust 結果物件裡的錯誤欄位換成 Go 的值（Go 端錯誤屬標準函式庫且雙方都拒收時），再整體比對。
fn check_with_errs(ctx: &str, go: &Value, rust: &mut Value, err_keys: &[&str]) {
    for key in err_keys {
        let (g, r) = (&go[*key], &rust[*key]);
        if let (Some(gs), Some(_)) = (g.as_str(), r.as_str())
            && !own(gs)
        {
            rust[*key] = g.clone();
        }
    }
    check(ctx, go, rust);
}

// ---- Go 結構的 JSON（沒有 json 標籤，鍵是 Go 欄位名）----

fn point_val(p: &NativeItemPanelPoint) -> Value {
    json!({"X": p.x, "Y": p.y})
}

fn region_val(r: &NativeItemPanelRegion) -> Value {
    json!({"Enabled": r.enabled, "SourceX": r.source_x, "SourceY": r.source_y, "DestX": r.dest_x, "DestY": r.dest_y,
        "Width": r.width, "Height": r.height})
}

fn frame_val(f: &NativeItemPanelFrame) -> Value {
    json!({"Frame": f.frame, "Left": region_val(&f.left), "Upper": region_val(&f.upper), "Bottom": region_val(&f.bottom)})
}

fn icon_val(c: &NativeItemPanelIconCall) -> Value {
    json!({"Destination": point_val(&c.destination), "FDOTHEREntry": c.fdother_entry,
        "AlternateFDOTHEREntry": c.alternate_fdother_entry, "FlagOffset": c.flag_offset,
        "UseAlternateWhenFlagZero": c.use_alternate_when_flag_zero, "DrawWhenFlagNonzero": c.draw_when_flag_nonzero})
}

fn plan_val(p: &NativeItemPanelDataPlan) -> Value {
    json!({
        "Bars": p.bars.iter().map(|c| json!({"Destination": point_val(&c.destination),
            "FDOTHERBaseEntry": c.fdother_base_entry, "CurrentOffset": c.current_offset,
            "MaximumOffset": c.maximum_offset})).collect::<Vec<_>>(),
        "ComparedNumbers": p.compared_numbers.iter().map(|c| json!({"Destination": point_val(&c.destination),
            "ValueOffset": c.value_offset, "CompareOffset": c.compare_offset, "Width": c.width})).collect::<Vec<_>>(),
        "RawNumbers": p.raw_numbers.iter().map(|c| json!({"Destination": point_val(&c.destination),
            "ValueOffset": c.value_offset, "ValueBytes": c.value_bytes, "Color": c.color,
            "AlternateColor": c.alternate_color, "AlternateFlagOffset": c.alternate_flag_offset,
            "Width": c.width})).collect::<Vec<_>>(),
        "Text": p.text.iter().map(|c| json!({"Destination": point_val(&c.destination),
            "RecordOffset": c.record_offset, "FDTXTBase": c.fdtxt_base})).collect::<Vec<_>>(),
        "BaseIcon": icon_val(&p.base_icon),
        "FlagIcons": p.flag_icons.iter().map(icon_val).collect::<Vec<_>>(),
    })
}

fn layout_val(l: &NativeItemPanelBaseLayout) -> Value {
    json!({"Stride": l.stride, "FDOTHERResource": l.fdother_resource, "GridOrigin": point_val(&l.grid_origin),
        "GridColumns": l.grid_columns, "GridRows": l.grid_rows, "PortraitRecordOffset": l.portrait_record_offset,
        "PortraitDestination": point_val(&l.portrait_destination), "UpperDirectoryEntry": l.upper_directory_entry,
        "UpperDestination": point_val(&l.upper_destination), "BottomDirectoryEntry": l.bottom_directory_entry,
        "BottomDestination": point_val(&l.bottom_destination)})
}

fn cell_val(c: &NativeItemSelectorCell) -> Value {
    json!({"RawSlot": c.raw_slot, "DisplayIndex": c.display_index, "ItemID": c.item_id, "Equipped": c.equipped,
        "Selected": c.selected, "LabelX": c.label_x, "LabelY": c.label_y, "CategoryIcon": c.category_icon,
        "StatIcon": c.stat_icon, "StatValue": c.stat_value, "HasStatValue": c.has_stat_value})
}

fn restore_val(r: &NativeRawRestoreResult) -> Value {
    json!({"UnitIndex": r.unit_index, "Rolled": r.rolled, "Actual": r.actual, "Score": r.score, "RNGState": r.rng_state})
}

fn damage_val(d: &NativeRawDamageResult) -> Value {
    json!({"UnitIndex": d.unit_index, "Rolled": d.rolled, "Actual": d.actual, "RNGState": d.rng_state})
}

fn digits_val(q: &[NativePresentationDigit]) -> Value {
    Value::Array(
        q.iter().map(|d| json!({"PositionCode": d.position_code, "Target": d.target, "Digit": d.digit})).collect(),
    )
}

fn digits_from(v: &Value) -> Vec<NativePresentationDigit> {
    v.as_array()
        .map(|a| {
            a.iter()
                .map(|d| NativePresentationDigit {
                    position_code: i(d, "PositionCode"),
                    target: i(d, "Target"),
                    digit: i(d, "Digit"),
                })
                .collect()
        })
        .unwrap_or_default()
}

// ---- 路由 ----

fn route_case(name: &str, input: &Value, result: &Value) {
    let (typ, word) = (i(input, "type") as u8, i(input, "word") as u16);
    let entry = |(route, ok): (Value, bool)| json!({"ok": ok, "route": route});
    let (r, ok) = native_item_effect_route_for_type(typ, word);
    let dispatch = json!({"ItemType": r.item_type, "Word": r.word, "Primary": r.primary, "Secondary": r.secondary,
        "Arg2": r.arg2, "Arg5": r.arg5, "Arg7": r.arg7});
    let (a, ok2) = native_item_apdp_step_route_for_type(typ);
    let (h, ok3) = native_item_hitev_step_route_for_type(typ);
    let (c, ok4) = native_item_capacity_step_route_for_type(typ, word);
    let (hp, ok5) = native_item_hp_restore_route_for_type(typ, word);
    let (mp, ok6) = native_item_mp_restore_route_for_type(typ, word);
    let (m, ok7) = native_item_marker_application_route_for_type(typ);
    let (mc, ok8) = native_item_marker_clear_restore_route_for_type(typ);
    let (cd, ok9) = native_item_command_damage_route_for_type(typ, word);
    let (rl, ok10) = native_item_relocation_route_for_type(typ, word);
    let got = json!({
        "dispatch": entry((dispatch, ok)),
        "apdp": entry((apdp_val(&a), ok2)),
        "hitev": entry((json!({"ItemType": h.item_type, "Increment": h.increment, "ConsumesSource": h.consumes_source}), ok3)),
        "capacity": entry((capacity_val(&c), ok4)),
        "hp": entry((json!({"ItemType": hp.item_type, "Amount": hp.amount, "ConsumesSource": hp.consumes_source}), ok5)),
        "mp": entry((json!({"ItemType": mp.item_type, "Amount": mp.amount}), ok6)),
        "marker": entry((json!({"ItemType": m.item_type, "PresentationCode": m.presentation_code,
            "MarkerOffset": m.marker_offset, "HPDamageBase": m.hp_damage_base, "ConsumesSource": m.consumes_source}), ok7)),
        "marker_clear": entry((json!({"ItemType": mc.item_type, "PresentationCode": mc.presentation_code,
            "MarkerOffset": mc.marker_offset, "HPRestoreBase": mc.hp_restore_base,
            "ConsumesSource": mc.consumes_source}), ok8)),
        "command_damage": entry((json!({"ItemType": cd.item_type, "CommandID": cd.command_id,
            "Presentation": cd.presentation, "ConsumesSource": cd.consumes_source}), ok9)),
        "relocation": entry((json!({"ItemType": rl.item_type, "RowWord": rl.row_word, "CommandID": rl.command_id,
            "RequiredIdentity": rl.required_identity, "RequiredMaxMP": rl.required_max_mp,
            "Presentation": rl.presentation, "ConsumesSource": rl.consumes_source}), ok10)),
    });
    check(name, result, &got);
}

fn apdp_val(a: &NativeItemAPDPStepRoute) -> Value {
    json!({"ItemType": a.item_type, "DerivedStat": a.derived_stat.0, "MarkerOffset": a.marker_offset,
        "WordOffset": a.word_offset, "ConsumesSource": a.consumes_source})
}

fn capacity_val(c: &NativeItemCapacityStepRoute) -> Value {
    json!({"ItemType": c.item_type, "Stat": c.stat.0, "FieldOffset": c.field_offset, "Amount": c.amount,
        "PresentationCode": c.presentation_code, "ConsumesSource": c.consumes_source})
}

/// Go `bookVariant` 的 Rust 版。
fn book_variant(book: &[NativeCommandRecord], input: &Value) -> Vec<NativeCommandRecord> {
    let mut book = book.to_vec();
    match i(input, "book_variant") {
        0 => {}
        1 => book[23].mp_cost = i(input, "mp_cost"),
        2 => book.truncate(35),
        3 => book[23].id = 22,
        v => panic!("未知的 book_variant {v}"),
    }
    book
}

fn item_step(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let mut records = unhex(s(input, "records"));
    let indices = byte_list(&input["indices"]);
    let rng = i(input, "rng") as u16;
    let (source_unit, source_slot) = (i(input, "source_unit"), i(input, "source_slot"));
    let item_type = i(input, "item_type") as u8;
    let amount = i(input, "amount") as u16;
    let t = i(input, "tamper");
    let mut got = serde_json::Map::new();
    let mut put = |k: &str, v: Value| {
        got.insert(k.into(), v);
    };
    match s(input, "kind") {
        "apdp" => {
            let mut route = native_item_apdp_step_route_for_type(item_type).0;
            match t {
                1 => route.marker_offset += 1,
                2 => route.word_offset += 1,
                3 => route.consumes_source = true,
                4 => route.derived_stat = NativeItemDerivedStat("hp"),
                5 => route.item_type = route.item_type.wrapping_add(1),
                _ => {}
            }
            let r = apply_native_item_apdp_step(&mut records, &indices, route, rng);
            put("err", err_of(&r));
            let (results, next, score) = r.unwrap_or((Vec::new(), rng, 0));
            put(
                "results",
                Value::Array(
                    results
                        .iter()
                        .map(|x| {
                            json!({"UnitIndex": x.unit_index,
                "Processed": x.processed, "Marker": x.marker, "Delta": x.delta})
                        })
                        .collect(),
                ),
            );
            put("rng", json!(next));
            put("score", json!(score));
        }
        "hitev" => {
            let mut route = native_item_hitev_step_route_for_type(item_type).0;
            match t {
                1 | 4 => route.increment = route.increment.wrapping_add(1),
                2 | 5 => route.consumes_source = true,
                3 => route.item_type = route.item_type.wrapping_add(1),
                _ => {}
            }
            let r = apply_native_item_hitev_step(&mut records, &indices, route, rng);
            put("err", err_of(&r));
            let (results, next, score) = r.unwrap_or((Vec::new(), rng, 0));
            put(
                "results",
                Value::Array(
                    results
                        .iter()
                        .map(|x| {
                            json!({"UnitIndex": x.unit_index,
                "Processed": x.processed, "Marker": x.marker})
                        })
                        .collect(),
                ),
            );
            put("rng", json!(next));
            put("score", json!(score));
        }
        "capacity" => {
            let mut route = native_item_capacity_step_route_for_type(item_type, amount).0;
            match t {
                1 => route.field_offset += 1,
                2 => route.amount = route.amount.wrapping_add(1),
                3 => route.presentation_code += 1,
                4 => route.stat = NATIVE_ITEM_MV,
                5 => route.consumes_source = false,
                _ => {}
            }
            let r = apply_native_item_capacity_step(&mut records, i(input, "target"), route, source_unit, source_slot);
            put("err", err_of(&r));
            put("value", json!(r.unwrap_or(0)));
        }
        "hp" => {
            let mut route = native_item_hp_restore_route_for_type(item_type, amount).0;
            match t {
                1 | 4 => route.amount = route.amount.wrapping_add(1),
                2 | 5 => route.consumes_source = !route.consumes_source,
                3 => route.item_type = 6,
                _ => {}
            }
            let r = apply_native_item_hp_restore(&mut records, &indices, route, rng, source_unit, source_slot);
            put("err", err_of(&r));
            let batch = r.unwrap_or(NativeRawRestoreBatch { rng_state: rng, ..Default::default() });
            put(
                "batch",
                json!({"Results": batch.results.iter().map(restore_val).collect::<Vec<_>>(),
                "RNGState": batch.rng_state, "Score": batch.score}),
            );
        }
        "mp" => {
            let mut route = native_item_mp_restore_route_for_type(item_type, amount).0;
            match t {
                1 | 3 | 5 => route.amount = route.amount.wrapping_add(1),
                2 | 4 => route.item_type = 12,
                _ => {}
            }
            let r = apply_native_item_mp_restore(&mut records, &indices, route, rng, source_unit, source_slot);
            put("err", err_of(&r));
            let batch = r.unwrap_or(NativeItemMPRestoreBatch { rng_state: rng, ..Default::default() });
            put(
                "batch",
                json!({"Results": batch.results.iter().map(restore_val).collect::<Vec<_>>(),
                "SkippedTargets": batch.skipped_targets, "RNGState": batch.rng_state, "Score": batch.score}),
            );
        }
        "marker" => {
            let mut route = native_item_marker_application_route_for_type(item_type).0;
            match t {
                1 => route.presentation_code += 1,
                2 => route.marker_offset += 1,
                3 => route.hp_damage_base += 1,
                4 => route.consumes_source = true,
                5 => route.item_type = 15,
                _ => {}
            }
            let r = apply_native_item_marker_application(&mut records, &indices, route, rng);
            put("err", err_of(&r));
            let (results, next, score) = r.unwrap_or((Vec::new(), rng, 0));
            put(
                "results",
                Value::Array(
                    results
                        .iter()
                        .map(|x| {
                            json!({"TargetIndex": x.target_index,
                "Applied": x.applied, "Damage": damage_val(&x.damage), "Marker": x.marker})
                        })
                        .collect(),
                ),
            );
            put("rng", json!(next));
            put("score", json!(score));
        }
        "marker_clear" => {
            let mut route = native_item_marker_clear_restore_route_for_type(item_type).0;
            match t {
                1 => route.presentation_code += 1,
                2 => route.marker_offset += 1,
                3 => route.hp_restore_base += 1,
                4 => route.consumes_source = false,
                5 => route.item_type = 8,
                _ => {}
            }
            let r =
                apply_native_item_marker_clear_restore(&mut records, &indices, route, rng, source_unit, source_slot);
            put("err", err_of(&r));
            let (results, next, score) = r.unwrap_or((Vec::new(), rng, 0));
            put(
                "results",
                Value::Array(
                    results
                        .iter()
                        .map(|x| {
                            json!({"TargetIndex": x.target_index,
                "Applied": x.applied, "Restore": restore_val(&x.restore)})
                        })
                        .collect(),
                ),
            );
            put("rng", json!(next));
            put("score", json!(score));
        }
        kind @ ("relocation" | "command23") => {
            let book = book_variant(&ctx.book, input);
            let actor = i(input, "actor");
            let (dx, dy) = (i(input, "dx") as u8, i(input, "dy") as u8);
            let r = if kind == "relocation" {
                let mut route = native_item_relocation_route_for_type(item_type, amount).0;
                match t {
                    1 => route.command_id += 1,
                    2 => route.required_identity = route.required_identity.wrapping_add(1),
                    3 => route.required_max_mp = route.required_max_mp.wrapping_add(1),
                    4 => route.presentation += 1,
                    5 => route.consumes_source = true,
                    _ => {}
                }
                apply_native_item_relocation(&mut records, actor, &indices, dx, dy, route, &book)
            } else {
                apply_native_command23_relocation(&mut records, actor, &indices, dx, dy, &book)
            };
            put("err", err_of(&r));
            let g = r.unwrap_or_default();
            put(
                "result",
                json!({"TargetUnit": g.target_unit, "FromX": g.from_x, "FromY": g.from_y,
                "DestinationX": g.destination_x, "DestinationY": g.destination_y, "RemainingMP": g.remaining_mp,
                "AccumulatorDelta": g.accumulator_delta}),
            );
        }
        other => panic!("{name}：未知的 kind {other}"),
    }
    got.insert("records".into(), json!(hex(&records)));
    check(name, result, &Value::Object(got));
}

fn unit_item(name: &str, input: &Value, result: &Value) {
    let mut units: Vec<Unit> = input["units"].as_array().unwrap().iter().map(unit_from).collect();
    let idx = |k: &str| usize::try_from(i(input, k)).ok();
    let (target, source, slot) = (idx("target"), idx("source"), i(input, "slot"));
    let amount = i(input, "amount") as u16;
    let tampered = i(input, "tamper") != 0;
    let r = if s(input, "kind") == "capacity" {
        let mut route = native_item_capacity_step_route_for_type(i(input, "item_type") as u8, amount).0;
        if tampered {
            route.field_offset += 1;
        }
        apply_native_item_capacity_to_unit(&mut units, target, source, route, slot)
    } else {
        let mut route = native_item_word_delta_route_for_type(i(input, "item_type")).0;
        if tampered {
            route.presentation_code += 1;
        }
        apply_native_item_base_stat_delta_to_unit(&mut units, target, source, route, amount, slot)
    };
    let got = json!({"value": r.as_ref().copied().unwrap_or(0), "err": err_of(&r), "units": units_val(&units)});
    check(name, result, &got);
}

// ---- 載入器 ----

fn loader_case(ctx: &Ctx, name: &str, op: &str, input: &Value, result: &Value, index: i64) {
    let path = ctx.tmp.join(format!("{op}-{index}.json"));
    std::fs::create_dir_all(&ctx.tmp).unwrap();
    std::fs::write(&path, s(input, "text")).unwrap();
    let path = path.to_string_lossy().into_owned();
    let mut got = match op {
        "item_table" => {
            let r = load_native_item_effect_row_prefix(&path);
            json!({"err": err_of(&r), "table": hex(&r.unwrap_or_default())})
        }
        "scene" => {
            let r = load_native_command_scene_table(&path);
            let mut v = json!({"err": err_of(&r)});
            if let Ok(t) = r {
                v["table"] = serde_json::to_value(t).unwrap();
            }
            v
        }
        "flash" => {
            let r = load_native_command_palette_flash_table(&path);
            let mut v = json!({"err": err_of(&r)});
            if let Ok(t) = r {
                v["table"] = serde_json::to_value(t).unwrap();
            }
            v
        }
        _ => unreachable!(),
    };
    check_with_errs(name, result, &mut got, &["err"]);
}

fn real_loaders(ctx: &Ctx, name: &str, op: &str, input: &Value, result: &Value) {
    let path = ctx.path(s(input, "path"));
    let got = match op {
        "item_table_real" => {
            let r = load_native_item_effect_row_prefix(&path);
            let table = r.as_ref().cloned().unwrap_or_default();
            json!({"err": err_of(&r), "len": table.len(), "sha256": digest(&table)})
        }
        "scene_real" => {
            let r = load_native_command_scene_table(&path);
            let mut v = json!({"err": err_of(&r)});
            if let Ok(t) = &r {
                let bg: Vec<Value> = (-1..=31)
                    .map(|c| {
                        let r = t.initial_background(c);
                        json!([r.as_ref().copied().unwrap_or(0), err_of(&r)])
                    })
                    .collect();
                v["table"] = serde_json::to_value(t).unwrap();
                v["backgrounds"] = Value::Array(bg);
            }
            v
        }
        "flash_real" => {
            let r = load_native_command_palette_flash_table(&path);
            let mut v = json!({"err": err_of(&r)});
            if let Ok(t) = &r {
                let phases: Vec<Value> = (-1..=37)
                    .map(|id| {
                        let r = t.native_command_palette_flash_phases(id);
                        json!([r.as_ref().cloned().unwrap_or_default(), err_of(&r)])
                    })
                    .collect();
                v["table"] = serde_json::to_value(t).unwrap();
                v["phases"] = Value::Array(phases);
            }
            v
        }
        _ => unreachable!(),
    };
    check(name, result, &got);
}

// ---- 選擇器、命令格、呈現佇列 ----

fn selector_advance(name: &str, input: &Value, result: &Value) {
    let scans = ints(&input["scans"]);
    let mut results = Vec::new();
    for selection in -1..=9 {
        for count in -1..=9 {
            for &scan in &scans {
                for battle_use in [false, true] {
                    for effect in [0u8, 5] {
                        let r = advance_native_item_selector(selection, count, scan, battle_use, effect);
                        let (got, res) = match &r {
                            Ok((g, res)) => (*g, res.0),
                            Err(_) => (selection, NATIVE_ITEM_SELECTOR_WAIT.0),
                        };
                        results.push(json!([got, res, err_of(&r)]));
                    }
                }
            }
        }
    }
    check(name, result, &json!({"results": results}));
}

fn selector_cells(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let records = unhex(s(input, "records"));
    let rows = &ctx.item_rows[..i(input, "rows_len") as usize];
    let r = native_item_selector_cells(&records, i(input, "unit"), i(input, "selected"), rows);
    let cells: Vec<Value> = r.as_ref().map(|c| c.iter().map(cell_val).collect()).unwrap_or_default();
    check(name, result, &json!({"cells": cells, "err": err_of(&r)}));
}

fn panel_layout(name: &str, result: &Value) {
    let frames: Vec<Value> = (-1..=12)
        .map(|f| {
            let r = native_item_panel_frame_for(f);
            json!([frame_val(&r.as_ref().copied().unwrap_or_default()), err_of(&r)])
        })
        .collect();
    let opening = native_item_panel_schedule(true);
    let closing = native_item_panel_schedule(false);
    let sched = |r: &Result<Vec<NativeItemPanelFrame>, BattleError>| {
        Value::Array(r.as_ref().map(|v| v.iter().map(frame_val).collect()).unwrap_or_default())
    };
    let got = json!({
        "frames": frames, "opening": sched(&opening), "closing": sched(&closing),
        "errs": [err_of(&opening), err_of(&closing)],
        "base": layout_val(&native_item_panel_base_layout_for()), "plan": plan_val(&native_item_panel_data_plan_for()),
    });
    check(name, result, &got);
}

fn grid(name: &str, input: &Value, result: &Value) {
    let cells: Vec<Value> = native_command_grid(&ints(&input["ids"]), i(input, "selected"))
        .iter()
        .map(|c| {
            json!({"CommandID": c.command_id, "Column": c.column, "Row": c.row, "X": c.x, "Y": c.y,
            "Selected": c.selected})
        })
        .collect();
    check(name, result, &json!({"cells": cells}));
}

fn grid_move(name: &str, result: &Value) {
    let mut moves = Vec::new();
    for index in -1..=9 {
        for count in -1..=9 {
            for dir in -1..=4 {
                moves.push(native_command_grid_move(index, count, dir));
            }
        }
    }
    check(name, result, &json!({"moves": moves}));
}

fn presentation(name: &str, input: &Value, result: &Value) {
    let queue = digits_from(&input["queue"]);
    let target = i(input, "target");
    let got = if b(input, "miss") {
        json!({"queue": digits_val(&append_native_presentation_miss(&queue, target))})
    } else {
        let r = append_native_presentation_digits(
            &queue,
            i(input, "value"),
            i(input, "bias"),
            target,
            b(input, "in_camera"),
        );
        json!({"queue": digits_val(r.as_deref().unwrap_or_default()), "err": err_of(&r)})
    };
    check(name, result, &got);
}

fn opt_unit(v: &Value) -> Option<Unit> {
    (!v.is_null()).then(|| unit_from(v))
}

fn physical_scene(name: &str, input: &Value, result: &Value) {
    let (actor, target) = (opt_unit(&input["actor"]), opt_unit(&input["target"]));
    let control = |k: &str| -> [u8; 4] { byte_list(&input[k]).try_into().unwrap() };
    let r = select_native_physical_scene(
        i(input, "initial") as u8,
        actor.as_ref(),
        target.as_ref(),
        control("actor_control"),
        control("target_control"),
        i(input, "header") as u8,
    );
    let sel = r.as_ref().copied().unwrap_or_default();
    let gate = native_command_background_gate(actor.as_ref());
    let got = json!({
        "selection": {"PrimaryBG": sel.primary_bg, "SecondaryBG": sel.secondary_bg, "BaseBG": sel.base_bg,
            "TAI": sel.tai, "HasSeparateBackgrounds": sel.has_separate_backgrounds},
        "err": err_of(&r), "gate": gate.as_ref().copied().unwrap_or(false), "gate_err": err_of(&gate),
    });
    check(name, result, &got);
}

// ---- ch28 ----

fn ch28_state(input: &Value) -> State {
    let mut st = State::default();
    for u in input["units"].as_array().unwrap() {
        let u = u.as_array().unwrap();
        let n = |k: usize| u[k].as_i64().unwrap();
        let mut unit = Unit::default();
        unit.group = n(0);
        unit.hp = n(1);
        unit.native_map_presentation =
            NativeMapPresentationState { x: n(2) as u8, y: n(3) as u8, pose: n(4) as u8, motion: 0 };
        unit.native_record_byte5 = n(5) as u8;
        unit.has_native_map_presentation = u[6].as_bool().unwrap();
        unit.has_native_record_byte5 = u[7].as_bool().unwrap();
        st.units.push(unit);
    }
    for g in ints(&input["roster"]) {
        let mut unit = Unit::default();
        unit.group = g;
        st.roster.push(unit);
    }
    st.has_native_runtime_unit_projection = b(input, "projection");
    for r in input["records"].as_array().unwrap() {
        let mut rec = NativeRuntimeRecordState::default();
        rec.raw.copy_from_slice(&unhex(r.as_str().unwrap()));
        st.native_runtime_records.push(rec);
    }
    st
}

fn ch28_summary(st: &State) -> Value {
    let units: Vec<Value> = st
        .units
        .iter()
        .map(|u| {
            json!([
                u.hp,
                u.battle_fig,
                u.has_battle_fig,
                u.native_record_byte8,
                u.has_native_record_byte8,
                u.native_map_presentation.x,
                u.native_map_presentation.y,
                u.native_map_presentation.pose,
                u.dir,
                u.native_record_byte5
            ])
        })
        .collect();
    let records: Vec<Value> = st.native_runtime_records.iter().map(|r| json!(hex(&r.raw))).collect();
    json!({"units": units, "records": records})
}

fn ch28_prelude(name: &str, input: &Value, result: &Value) {
    let mut st = ch28_state(input);
    let validate = validate_native_ch28_post_frontier(Some(&st));
    let prelude = apply_native_ch28_post_raw_prelude(Some(&mut st));
    let got = json!({"validate": err_of(&validate), "prelude": err_of(&prelude), "after": ch28_summary(&st)});
    check(name, result, &got);
}

fn ch28_present(name: &str, input: &Value, result: &Value) {
    let mut st = ch28_state(input);
    let plan = plan_native_ch28_post_presentation(Some(&st), i(input, "camera_x"), i(input, "camera_y"));
    let p = plan.as_ref().cloned().unwrap_or_default();
    let targets: Vec<Value> = p.targets.iter().map(|t| json!({"Slot": t.slot, "Origin": t.origin})).collect();
    let pose = apply_native_ch28_post_pose_frame(Some(&mut st), i(input, "frame"));
    let after_pose = ch28_summary(&st);
    let inactive = apply_native_ch28_post_inactive_mark(Some(&mut st));
    let got = json!({
        "plan": {"CameraX": p.camera_x, "CameraY": p.camera_y, "Targets": targets}, "plan_err": err_of(&plan),
        "pose_err": err_of(&pose), "after_pose": after_pose,
        "inactive_err": err_of(&inactive), "after_inactive": ch28_summary(&st),
    });
    check(name, result, &got);
}

// ---- 面板繪製 ----

/// Go `syntheticAssets`（與 Go 測試 `nativeItemPanelTestAssets` 相同，另加 149×42 的戰鬥面板框）。
fn synthetic_assets(text_word: u16) -> NativeItemPanelDataAssets {
    let raw: BTreeMap<i64, fdother::RawCell> =
        (0..=92).map(|k| (k, fdother::RawCell { width: 1, height: 1, pixels: vec![k as u8] })).collect();
    let frames: BTreeMap<i64, fdother::Frame> =
        (0..=140).map(|k| (k, fdother::Frame::raw(0, 0, 1, 1, vec![1, 0, 1, 0, 0, k as u8]))).collect();
    const COUNT: usize = 500;
    let mut data = vec![0u8; COUNT * 2 + COUNT * 4];
    for index in 0..COUNT {
        let offset = COUNT * 2 + index * 4;
        data[index * 2..index * 2 + 2].copy_from_slice(&(offset as u16).to_le_bytes());
        data[offset..offset + 2].copy_from_slice(&text_word.to_le_bytes());
        data[offset + 2..offset + 4].copy_from_slice(&fdtxt::STRING_END.to_le_bytes());
    }
    let mut font_data = vec![0u8; fdtxt::GLYPH_BYTES];
    font_data[0] = 0x80;
    NativeItemPanelDataAssets {
        raw_cells: Some(raw),
        frames: Some(frames),
        strings: Some(fdtxt::parse(&data).unwrap()),
        font: Some(fdtxt::parse_font(&font_data).unwrap()),
        battle_panel: fdother::LMI1Entry {
            width: 149,
            height: 42,
            pixels: (0..149 * 42).map(|k| (k % 7) as u8).collect(),
        },
        ..Default::default()
    }
}

struct Ctx {
    repo: PathBuf,
    flame: PathBuf,
    pack: PathBuf,
    tmp: PathBuf,
    item_rows: Vec<u8>,
    book: Vec<NativeCommandRecord>,
    assets: BTreeMap<&'static str, NativeItemPanelDataAssets>,
}

impl Ctx {
    /// 黃金檔記錄的檔案標籤 → 掛載路徑。
    fn path(&self, label: &str) -> String {
        let (root, rest) = label.split_once('/').unwrap();
        let base = match root {
            "remake" => self.repo.join("remake"),
            "flame2" => self.flame.clone(),
            "pack" => self.pack.clone(),
            other => panic!("未知的檔案標籤 {other}"),
        };
        base.join(rest).to_string_lossy().into_owned()
    }

    fn assets(&self, name: &str) -> NativeItemPanelDataAssets {
        if name == "nil_maps" {
            let mut a = synthetic_assets(0);
            (a.raw_cells, a.frames) = (None, None);
            return a;
        }
        self.assets.get(name).cloned().unwrap_or_else(|| panic!("未知的素材 {name}"))
    }

    fn fdother(&self) -> String {
        self.flame.join("FDOTHER.DAT").to_string_lossy().into_owned()
    }

    fn fdtxt(&self) -> String {
        self.flame.join("FDTXT.DAT").to_string_lossy().into_owned()
    }

    fn pack_root(&self) -> String {
        self.pack.to_string_lossy().into_owned()
    }

    fn portraits(&self) -> String {
        self.pack.join("portraits").to_string_lossy().into_owned()
    }
}

fn dst(input: &Value) -> Vec<u8> {
    vec![i(input, "fill") as u8; PANEL]
}

fn raw_cell(v: &Value) -> fdother::RawCell {
    fdother::RawCell { width: i(v, "width"), height: i(v, "height"), pixels: unhex(s(v, "pixels")) }
}

fn render_result<T>(r: &Result<T, BattleError>, dst: &[u8]) -> Value {
    json!({"err": err_of(r), "sha256": digest(dst)})
}

fn render(ctx: &Ctx, name: &str, op: &str, input: &Value, result: &Value) {
    let got = match op {
        "render_panel_data" => {
            let assets = ctx.assets(s(input, "assets"));
            let mut d = dst(input);
            let r = render_native_item_panel_data(&assets, &unhex(s(input, "record")), &mut d);
            render_result(&r, &d)
        }
        "render_panel_rows" => {
            let assets = ctx.assets(s(input, "assets"));
            let mut d = dst(input);
            let rows = &ctx.item_rows[..i(input, "rows_len") as usize];
            let r =
                render_native_item_panel_rows(&assets, &unhex(s(input, "record")), i(input, "selected"), rows, &mut d);
            render_result(&r, &d)
        }
        "render_panel_resources" => {
            let record = unhex(s(input, "record"));
            let (mut d, mut oracle) = (dst(input), dst(input));
            let r = render_native_item_panel_resources(&ctx.pack_root(), &ctx.portraits(), &record, &mut d);
            let ra = render_native_item_panel_resources_archive(
                &ctx.fdother(),
                &ctx.fdtxt(),
                &ctx.portraits(),
                &record,
                &mut oracle,
            );
            json!({"err": err_of(&r), "sha256": digest(&d), "archive_err": err_of(&ra), "archive_sha256": digest(&oracle)})
        }
        "render_panel_base" => {
            let portrait = i(input, "portrait");
            let (mut d, mut oracle) = (dst(input), dst(input));
            let r = render_native_item_panel_base_resources(&ctx.pack_root(), &ctx.portraits(), portrait, &mut d);
            let ra = render_native_item_panel_base_resources_archive(
                &ctx.fdother(),
                &ctx.portraits(),
                portrait,
                &mut oracle,
            );
            json!({"err": err_of(&r), "sha256": digest(&d), "archive_err": err_of(&ra), "archive_sha256": digest(&oracle)})
        }
        "render_battle_panel" => {
            let assets = ctx.assets(s(input, "assets"));
            let record = unhex(s(input, "record"));
            let (unit, chapter) = (i(input, "unit"), i(input, "chapter"));
            let mut d = dst(input);
            let r = if b(input, "name") {
                render_native_battle_panel(&assets, &record, &mut d, unit, chapter)
            } else {
                render_native_battle_panel_without_name(&assets, &record, &mut d, unit, chapter)
            };
            let o = native_battle_panel_origin(&record, unit, chapter);
            let (x, y) = o.as_ref().copied().unwrap_or((0, 0));
            json!({"err": err_of(&r), "sha256": digest(&d), "origin": [x, y, err_of(&o)]})
        }
        "render_values_at" => {
            let assets = ctx.assets(s(input, "assets"));
            let v = &input["values"];
            let values = NativeBattlePanelValues {
                level: i(v, "Level"),
                hp: i(v, "HP"),
                max_hp: i(v, "MaxHP"),
                mp: i(v, "MP"),
                max_mp: i(v, "MaxMP"),
            };
            let mut d = dst(input);
            let r = render_native_battle_panel_values_at(&assets, &mut d, i(input, "x"), i(input, "y"), values);
            render_result(&r, &d)
        }
        "render_map_unit_window" => {
            let assets = ctx.assets(s(input, "assets"));
            let (stride, rows, cursor) = (i(input, "stride"), i(input, "rows"), i(input, "cursor"));
            let mut d = vec![i(input, "fill") as u8; (stride * rows) as usize];
            let r = render_native_map_unit_window(&assets, &unhex(s(input, "record")), &mut d, stride, cursor);
            let (ox, oy) = native_map_unit_window_origin(cursor);
            json!({"err": err_of(&r), "sha256": digest(&d), "origin": [ox, oy]})
        }
        "render_revive" => {
            let assets = ctx.assets(s(input, "assets"));
            let cell = raw_cell(&input["cell"]);
            let rows: Vec<NativeReviveRow> = input["rows"]
                .as_array()
                .unwrap()
                .iter()
                .map(|r| NativeReviveRow {
                    sprite: fdicon::Sprite {
                        pixels: unhex(s(r, "pixels")),
                        mask: unhex(s(r, "mask")),
                        remap_mask: Vec::new(),
                    },
                    name_text_index: i(r, "name"),
                    race_text_index: i(r, "race"),
                    class_text_index: i(r, "class"),
                    fee: i(r, "fee"),
                })
                .collect();
            let mut d = dst(input);
            let r = render_native_revive_rows(&assets, &cell, &rows, i(input, "selected"), &mut d);
            render_result(&r, &d)
        }
        "render_transfer" => {
            let assets = ctx.assets(s(input, "assets"));
            let cell = raw_cell(&input["cell"]);
            let ids = ints(&input["ids"]);
            let (start, selected) = (i(input, "start"), i(input, "selected"));
            let mode = NativeFacilityPriceMode(i(input, "mode") as u8);
            let mut d = dst(input);
            let rows = &ctx.item_rows;
            let r = match i(input, "variant") {
                0 => render_native_transfer_item_rows(&assets, &cell, &ids, start, selected, rows, &mut d),
                1 => render_native_facility_item_rows(&assets, &cell, &ids, start, selected, rows, mode, &mut d),
                _ => render_native_facility_item_rows_without_names(
                    &assets, &cell, &ids, start, selected, rows, mode, &mut d,
                ),
            };
            render_result(&r, &d)
        }
        "render_number" => {
            let assets = ctx.assets(s(input, "assets"));
            let mut d = dst(input);
            let r = render_native_facility_number(
                &assets,
                &mut d,
                i(input, "x"),
                i(input, "y"),
                i(input, "value"),
                i(input, "base"),
                i(input, "width"),
            );
            render_result(&r, &d)
        }
        "render_command_overlay" => {
            let assets = ctx.assets(s(input, "assets"));
            let book = book_variant(&ctx.book, input);
            let mut d = dst(input);
            let r = render_native_command_overlay(&assets, &ints(&input["ids"]), &book, i(input, "selected"), &mut d);
            render_result(&r, &d)
        }
        _ => unreachable!(),
    };
    check(name, result, &got);
}

fn facility_price(ctx: &Ctx, name: &str, result: &Value) {
    let mut prices = Vec::new();
    for item in [-1, 0, 1, 57, 100, 0xd6, 0xd7, 0x100] {
        for mode in 0..=2u8 {
            let r = native_facility_item_list_price(&ctx.item_rows, item, NativeFacilityPriceMode(mode));
            prices.push(json!([r.as_ref().copied().unwrap_or(0), err_of(&r)]));
        }
    }
    check(name, result, &json!({"prices": prices}));
}

fn battle_panel_record(name: &str, input: &Value, result: &Value) {
    let unit = opt_unit(&input["unit"]);
    let r = native_battle_panel_record_for_unit(unit.as_ref());
    check(name, result, &json!({"record": hex(r.as_deref().unwrap_or_default()), "err": err_of(&r)}));
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_C").filter(|p| Path::new(p).exists());
    let dir = |var: &str, probe: &str| std::env::var_os(var).map(PathBuf::from).filter(|p| p.join(probe).exists());
    let repo = dir("FD2_GOLDEN_REPO", "remake/assets");
    let flame = dir("FD2_GOLDEN_FLAME2", "FDOTHER.DAT");
    let pack = dir("FD2_GOLDEN_PACK", "ui/fdother_005_item_panel");
    let (Some(path), Some(repo), Some(flame), Some(pack)) = (path, repo, flame, pack) else {
        assert!(
            !require,
            "FD2_REQUIRE_GOLDEN=1 但缺少 FD2_GOLDEN_RULES_C／FD2_GOLDEN_REPO／FD2_GOLDEN_FLAME2／FD2_GOLDEN_PACK"
        );
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_C 或真實輸入");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let tmp = std::env::temp_dir().join(format!("fd2-golden-rules-c-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&tmp);
    let mut ctx = Ctx { repo, flame, pack, tmp, item_rows: Vec::new(), book: Vec::new(), assets: BTreeMap::new() };
    ctx.item_rows =
        load_native_item_effect_row_prefix(&ctx.path("remake/assets/data/native_item_effect_rows.json")).unwrap();
    ctx.book = load_native_command_records(&ctx.path("remake/assets/spells.json")).unwrap();
    let mut no_text = synthetic_assets(0);
    (no_text.strings, no_text.font) = (None, None);
    let assets = [
        ("synthetic", synthetic_assets(0)),
        ("synthetic_ctrl", synthetic_assets(0xfffe)),
        ("separated", load_native_item_panel_data_assets(&ctx.pack_root()).unwrap()),
        ("archive", load_native_item_panel_data_assets_archive(&ctx.fdother(), &ctx.fdtxt()).unwrap()),
        ("values", load_native_battle_panel_value_assets(&ctx.pack_root()).unwrap()),
        ("values_archive", load_native_battle_panel_value_assets_archive(&ctx.fdother()).unwrap()),
        ("synthetic_no_text", no_text),
    ];
    ctx.assets = assets.into_iter().collect();
    Some((doc, ctx))
}

#[test]
fn golden_rules_c() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-c/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    let files = doc["files"].as_array().unwrap();
    for file in files {
        let path = ctx.path(s(file, "path"));
        let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀不到 {path}：{e}"));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    let mut counts = BTreeMap::<String, usize>::new();
    for case in doc["cases"].as_array().unwrap() {
        let op = s(case, "op");
        let index = case["index"].as_i64().unwrap();
        let name = format!("case {index} {op}");
        let (input, result) = (&case["input"], &case["result"]);
        match op {
            "routes" => route_case(&name, input, result),
            "item_step" => item_step(&ctx, &name, input, result),
            "unit_item" => unit_item(&name, input, result),
            "item_table" | "scene" | "flash" => loader_case(&ctx, &name, op, input, result, index),
            "item_table_real" | "scene_real" | "flash_real" => real_loaders(&ctx, &name, op, input, result),
            "selector_advance" => selector_advance(&name, input, result),
            "selector_cells" => selector_cells(&ctx, &name, input, result),
            "panel_layout" => panel_layout(&name, result),
            "grid" => grid(&name, input, result),
            "grid_move" => grid_move(&name, result),
            "presentation" => presentation(&name, input, result),
            "physical_scene" => physical_scene(&name, input, result),
            "ch28_prelude" => ch28_prelude(&name, input, result),
            "ch28_present" => ch28_present(&name, input, result),
            "facility_price" => facility_price(&ctx, &name, result),
            "battle_panel_record" => battle_panel_record(&name, input, result),
            op if op.starts_with("render_") => render(&ctx, &name, op, input, result),
            _ => panic!("{name}：未知的 op"),
        }
        *counts.entry(op.to_string()).or_default() += 1;
    }
    let _ = std::fs::remove_dir_all(&ctx.tmp);
    let total: usize = counts.values().sum();
    eprintln!("golden_rules_c：{} 個輸入檔、{total} 個案例全部相同：{counts:?}", files.len());
}
