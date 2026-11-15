//! 與凍結 Go 版 `remake/cmd/fd2/rust_golden_save_test.go` 的黃金差分：F 組的存檔、設定與能力設定附檔。
//!
//! 由 `tools/rust_golden_save.sh` 產生黃金 JSON、設定 `FD2_GOLDEN_SAVE` 並以 `--ignored` 執行；設
//! `FD2_REQUIRE_GOLDEN=1` 時黃金檔缺少即失敗。`fd2-game` 的函式與型別是 `pub(crate)`，整合測試（`tests/`）看不到，所以黃金比對
//! 放在 crate 內的單元測試（見 PORTING.md 的 F 組段落）。
//!
//! 每個案例的比對：
//! - `save`：Go 寫出（或手寫、真實檔）的位元組讀成 [`SaveData`]，是否接受、章節範圍、隊伍拓撲與能力
//!   設定的驗證結果、逐欄讀回結果都要與 Go 相同；Rust 重新寫出的位元組必須與 Go 重新寫出的逐位元組
//!   相同。拿掉 `party_roster` 與 `battle_snapshot`（battle 套件的型別）後的外層位元組另外比一次，
//!   讓失敗訊息分得出差異在哪一層。
//! - `settings`：零值起點與 `loadSettings` 起點的解碼結果、重新寫出的位元組逐位元組相同。
//!   `loadSettings` 正規化後的結果在另一個測試（要等 A 組的正規化函式）。
//! - `sidecar`：`readBalanceSidecar` 的錯誤（本套件的錯誤逐字比，JSON 解碼錯誤只比失敗）、讀回結果與
//!   重新寫出的位元組逐位元組相同。
//! - `balance_key`：`balanceSaveKey` 的結果或錯誤逐字相同。
//!
//! 比對時的表示法正規化列在 [`NORMALIZATIONS`]，只剩讀回結果的 `party_join_order`；刻意不照做的 Go
//! 行為列在 [`KNOWN_DIFFERENCES`]（與 `SAVE_JSON_KNOWN_DIFFERENCES` 相同）。

use super::*;
use base64::Engine as _;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

/// 刻意不照做的 Go 行為。黃金案例不落在這些輸入上，或案例本身標明（設定檔的型別錯誤）。
const KNOWN_DIFFERENCES: &[&str] = SAVE_JSON_KNOWN_DIFFERENCES;

/// 比對讀回結果前的表示法正規化。battle 套件型別的子樹以 Go 的寫法（[`to`]）比對，不正規化（#319）。
const NORMALIZATIONS: &[&str] = &[
    // party_join_order 有 omitempty：Go 讀到 [] 時是非 nil 空切片、讀到缺欄時是 nil，寫出都省略，
    // 位元組相同；Rust 的欄位不分兩者，只在讀回結果的比對上視為相同。
    "party_join_order：null 與 [] 相同",
];

fn b64(s: &Value) -> Vec<u8> {
    base64::engine::general_purpose::STANDARD.decode(s.as_str().expect("base64 欄位不是字串")).expect("base64 解碼")
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

/// `party_join_order` 的正規化（見 [`NORMALIZATIONS`]）：`null` 與 `[]` 相同。
fn norm_join_order(v: &Value) -> Value {
    match v {
        Value::Array(a) if a.is_empty() => Value::Null,
        other => other.clone(),
    }
}

fn first_diff(path: &str, a: &Value, b: &Value) -> String {
    match (a, b) {
        (Value::Object(x), Value::Object(y)) => {
            for k in x.keys().chain(y.keys()) {
                let (p, q) = (x.get(k).unwrap_or(&Value::Null), y.get(k).unwrap_or(&Value::Null));
                if p != q {
                    return first_diff(&format!("{path}.{k}"), p, q);
                }
            }
            format!("{path}：{a} ≠ {b}")
        }
        (Value::Array(x), Value::Array(y)) if x.len() == y.len() => {
            for (i, (p, q)) in x.iter().zip(y).enumerate() {
                if p != q {
                    return first_diff(&format!("{path}[{i}]"), p, q);
                }
            }
            format!("{path}：{a} ≠ {b}")
        }
        _ => {
            let (sa, sb) = (a.to_string(), b.to_string());
            format!("{path}：Rust {} ≠ Go {}", &sa[..sa.len().min(200)], &sb[..sb.len().min(200)])
        }
    }
}

fn same(path: &str, rust: &Value, go: &Value) -> std::result::Result<(), String> {
    if rust == go { Ok(()) } else { Err(first_diff(path, rust, go)) }
}

/// 數字只比數值（`font_scale` 這類 Go `float64` 欄位在 Value 裡可能是整數或小數）。
fn same_number(path: &str, rust: &Value, go: &Value) -> std::result::Result<(), String> {
    if rust.as_f64() == go.as_f64() && rust.as_f64().is_some() {
        Ok(())
    } else {
        Err(format!("{path}：Rust {rust} ≠ Go {go}"))
    }
}

fn int_pairs<V: serde::Serialize>(m: &BTreeMap<i64, V>) -> Value {
    Value::Array(m.iter().map(|(k, v)| json!([k, to(v)])).collect())
}

/// 以 Go `json.Marshal` 的寫法（[`go_json_marshal_indent`]）轉成 `Value`，數字的整數與小數形式和 Go 的 view 一致。
fn to<T: serde::Serialize>(v: &T) -> Value {
    serde_json::from_slice(&go_json_marshal_indent(v).expect("序列化")).expect("讀回")
}

fn save_view(d: &SaveData) -> Value {
    json!({
        "balance": to(&d.balance),
        "node": d.node,
        "flags_nil": d.flags.is_none(),
        "flags": to(&d.flags),
        "gold": d.gold,
        "items_nil": d.items.is_none(),
        "items": to(&d.items),
        "party_members": int_pairs(&d.party_members),
        "party_join_order": to(&d.party_join_order),
        "party_deploy": int_pairs(&d.party_deploy),
        "party_roster": int_pairs(&d.party_roster),
        "chapter": d.chapter,
        "native_hud_gate_a": d.native_hud_gate_a,
        "native_raw_53af9": d.native_raw_53af9,
        "native_raw_51e61": d.native_raw_51e61,
        "native_raw_51e62": d.native_raw_51e62,
        "battle_snapshot": to(&d.battle_snapshot),
    })
}

/// 逐欄比對讀回結果：除 `party_join_order` 外嚴格相等。
fn compare_save_view(id: &str, rust: &Value, go: &Value) -> std::result::Result<(), String> {
    let (Value::Object(r), Value::Object(g)) = (rust, go) else {
        return Err(format!("{id}：view 不是物件"));
    };
    let mut keys: Vec<&String> = g.keys().collect();
    keys.sort();
    let mut rkeys: Vec<&String> = r.keys().collect();
    rkeys.sort();
    if keys != rkeys {
        return Err(format!("{id}：view 欄位集合不同 {rkeys:?} ≠ {keys:?}"));
    }
    for key in keys {
        let path = format!("{id}.view.{key}");
        match key.as_str() {
            "party_join_order" => same(&path, &norm_join_order(&r[key]), &norm_join_order(&g[key]))?,
            _ => same(&path, &r[key], &g[key])?,
        }
    }
    Ok(())
}

#[derive(Default)]
struct Stats {
    cases: usize,
    save: usize,
    save_accepted: usize,
    save_bytes_exact: usize,
    settings: usize,
    sidecar: usize,
    balance_key: usize,
    real: usize,
}

fn check_save(c: &Value, st: &mut Stats) -> std::result::Result<(), String> {
    let id = c["id"].as_str().unwrap_or("?");
    let raw = b64(&c["input"]);
    st.save += 1;
    let decoded = go_json_unmarshal_zero::<SaveData>(&raw);
    let accepted = c["accepted"].as_bool().expect("accepted");
    if decoded.is_ok() != accepted {
        return Err(format!("{id}：接受與否 Rust {} ≠ Go {accepted}（{:?}）", decoded.is_ok(), decoded.err()));
    }
    let Ok(d) = decoded else {
        return Ok(());
    };
    st.save_accepted += 1;
    let in_range = d.chapter >= 0 && d.chapter <= 0xff;
    if Some(in_range) != c["chapter_in_range"].as_bool() {
        return Err(format!("{id}：章節範圍 {in_range}"));
    }
    let topology = validate_save_party_topology(&d).err().map(|e| e.message().to_string()).unwrap_or_default();
    let options: Vec<&str> =
        c["topology_errors"].as_array().expect("topology_errors").iter().filter_map(Value::as_str).collect();
    if !options.contains(&topology.as_str()) {
        return Err(format!("{id}：隊伍拓撲錯誤 Rust {topology:?} 不在 Go {options:?}"));
    }
    let balance =
        d.balance.as_ref().and_then(|b| b.validate().err()).map(|e| e.message().to_string()).unwrap_or_default();
    if Some(balance.as_str()) != c["balance_error"].as_str() {
        return Err(format!("{id}：能力設定驗證 Rust {balance:?} ≠ Go {}", c["balance_error"]));
    }
    compare_save_view(id, &save_view(&d), &c["view"])?;
    let rewritten = go_json_marshal_indent(&d).map_err(|e| e.to_string())?;
    let go_rewritten = b64(&c["rewritten"]);
    let mut outer = d.clone();
    outer.party_roster.clear();
    outer.battle_snapshot = None;
    let outer = go_json_marshal_indent(&outer).map_err(|e| e.to_string())?;
    let go_outer = b64(&c["outer"]);
    if outer != go_outer {
        return Err(format!(
            "{id}：外層位元組不同\nRust：{}\nGo：  {}",
            String::from_utf8_lossy(&outer),
            String::from_utf8_lossy(&go_outer)
        ));
    }
    if rewritten != go_rewritten {
        let (rs, gs) = (String::from_utf8_lossy(&rewritten), String::from_utf8_lossy(&go_rewritten));
        let (line, (rl, gl)) = rs
            .lines()
            .zip(gs.lines())
            .enumerate()
            .find(|(_, (a, b))| a != b)
            .unwrap_or((0, ("（長度不同）", "（長度不同）")));
        return Err(format!("{id}：重新寫出的位元組不同，第 {} 行 Go {} ／ Rust {}", line + 1, gl.trim(), rl.trim()));
    }
    st.save_bytes_exact += 1;
    Ok(())
}

fn settings_view(s: &Settings) -> Value {
    json!({
        "bgm_source": s.bgm_source, "locale_id": s.locale_id, "font_scale": s.font_scale,
        "interface_mode": s.interface_mode, "theme_id": s.theme_id, "game_speed": s.game_speed,
        "battle_animation_speed": s.battle_animation_speed, "hd_texture_quality": s.hd_texture_quality,
        "poster": s.poster.as_ref().map(|p| json!({"set": p.set, "edge": p.edge, "fade_ms": p.fade_ms})),
    })
}

fn compare_settings_view(path: &str, rust: &Value, go: &Value) -> std::result::Result<(), String> {
    same_number(&format!("{path}.font_scale"), &rust["font_scale"], &go["font_scale"])?;
    let strip = |v: &Value| {
        let mut v = v.clone();
        v.as_object_mut().expect("settings view").remove("font_scale");
        v
    };
    same(path, &strip(rust), &strip(go))
}

/// Go 的設定檔是否落在「型別錯誤時的部分填入」（KNOWN_DIFFERENCES）：零值解碼失敗、但不是語法錯誤。
fn settings_type_error(c: &Value) -> bool {
    c["accepted"] == Value::Bool(false) && c["syntax_error"] == Value::Bool(false)
}

fn check_settings(c: &Value, st: &mut Stats) -> std::result::Result<(), String> {
    let id = c["id"].as_str().unwrap_or("?");
    let raw = b64(&c["input"]);
    st.settings += 1;
    let decoded = go_json_unmarshal_settings(&Settings::default(), &raw);
    let accepted = c["accepted"].as_bool().expect("accepted");
    if decoded.is_ok() != accepted {
        return Err(format!("{id}：接受與否 Rust {} ≠ Go {accepted}（{:?}）", decoded.is_ok(), decoded.err()));
    }
    if let Ok(s) = &decoded {
        compare_settings_view(&format!("{id}.view"), &settings_view(s), &c["view"])?;
        let rewritten = go_json_marshal_indent(s).map_err(|e| e.to_string())?;
        let go = b64(&c["rewritten"]);
        if rewritten != go {
            return Err(format!(
                "{id}：重新寫出的位元組不同\nRust：{}\nGo：  {}",
                String::from_utf8_lossy(&rewritten),
                String::from_utf8_lossy(&go)
            ));
        }
    }
    let base = settings_load_base();
    let loaded = decode_settings_onto(&base, &raw).unwrap_or_else(|| base.clone());
    if settings_type_error(c) {
        // KNOWN_DIFFERENCES「型別錯誤時的部分填入」：Rust 整份拒收，保留起點。
        if loaded != base {
            return Err(format!("{id}：型別錯誤的設定檔沒有保留起點"));
        }
    } else {
        compare_settings_view(&format!("{id}.base_view"), &settings_view(&loaded), &c["base_view"])?;
    }
    Ok(())
}

fn check_settings_loaded(c: &Value) -> std::result::Result<(), String> {
    let id = c["id"].as_str().unwrap_or("?");
    if settings_type_error(c) {
        return Ok(());
    }
    let raw = b64(&c["input"]);
    let base = settings_load_base();
    let loaded = normalize_loaded_settings(decode_settings_onto(&base, &raw).unwrap_or(base));
    compare_settings_view(&format!("{id}.loaded_view"), &settings_view(&loaded), &c["loaded_view"])
}

fn sidecar_view(d: &BalanceSidecar) -> Value {
    let entries: Vec<Value> = d
        .entries
        .iter()
        .map(|(k, v)| {
            json!([k, {"options": serde_json::to_value(&v.options).expect("序列化"), "enemy_bases": int_pairs(&v.enemy_bases)}])
        })
        .collect();
    // Go readBalanceSidecar 最後把 nil 換回空 map，回傳值不會是 nil。
    json!({"version": d.version, "entries_nil": false, "entries": entries})
}

fn check_sidecar(c: &Value, dir: &std::path::Path, st: &mut Stats) -> std::result::Result<(), String> {
    let id = c["id"].as_str().unwrap_or("?");
    st.sidecar += 1;
    let path = dir.join(format!("{id}-FD2.SAV"));
    let path_str = path.to_string_lossy().into_owned();
    if c["absent"] != Value::Bool(true) {
        let mut raw = b64(&c["input"]);
        raw.extend(std::iter::repeat_n(b' ', c["pad_spaces"].as_u64().unwrap_or(0) as usize));
        std::fs::write(format!("{path_str}.fd2-options.json"), raw).map_err(|e| e.to_string())?;
    }
    let errors: Vec<&str> = c["errors"].as_array().expect("errors").iter().filter_map(Value::as_str).collect();
    let kinds: Vec<&str> = c["error_kinds"].as_array().expect("error_kinds").iter().filter_map(Value::as_str).collect();
    match read_balance_sidecar(&path_str) {
        Err(err) => {
            if errors.contains(&"") {
                return Err(format!("{id}：Rust 拒收（{}），Go 接受", err.message()));
            }
            // 本套件的錯誤逐字比；JSON 解碼與 os 錯誤只比失敗（KNOWN_DIFFERENCES）。
            let contract: Vec<&str> =
                errors.iter().zip(&kinds).filter(|(_, k)| **k == "contract").map(|(m, _)| *m).collect();
            if !contract.is_empty() && !contract.contains(&err.message()) {
                return Err(format!("{id}：錯誤 Rust {:?} 不在 Go {contract:?}", err.message()));
            }
        }
        Ok(d) => {
            if errors != [""] {
                return Err(format!("{id}：Rust 接受，Go 回 {errors:?}"));
            }
            same(&format!("{id}.view"), &sidecar_view(&d), &c["view"])?;
            let rewritten = go_json_marshal_indent(&d).map_err(|e| e.to_string())?;
            let go = b64(&c["rewritten"]);
            if rewritten != go {
                return Err(format!(
                    "{id}：重新寫出的位元組不同\nRust：{}\nGo：  {}",
                    String::from_utf8_lossy(&rewritten),
                    String::from_utf8_lossy(&go)
                ));
            }
        }
    }
    Ok(())
}

fn check_balance_key(c: &Value, plains: &[Vec<u8>], st: &mut Stats) -> std::result::Result<(), String> {
    let id = c["id"].as_str().unwrap_or("?");
    st.balance_key += 1;
    let plain = &plains[c["plain"].as_u64().expect("plain") as usize];
    let slot = c["slot"].as_i64().expect("slot");
    let (key, err) = match balance_save_key(plain, slot) {
        Ok(key) => (key, String::new()),
        Err(err) => (String::new(), err.message().to_string()),
    };
    if Some(key.as_str()) != c["key"].as_str() || Some(err.as_str()) != c["error"].as_str() {
        return Err(format!("{id}：Rust ({key:?}, {err:?}) ≠ Go ({}, {})", c["key"], c["error"]));
    }
    Ok(())
}

fn load_golden() -> Option<Value> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let Some(path) = std::env::var_os("FD2_GOLDEN_SAVE") else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有設定 FD2_GOLDEN_SAVE");
        eprintln!("略過：沒有設定 FD2_GOLDEN_SAVE（由 tools/rust_golden_save.sh 產生）");
        return None;
    };
    let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀取黃金檔 {path:?}：{e}"));
    eprintln!("黃金檔 {path:?} SHA-256 {}", hex(&Sha256::digest(&raw)));
    let golden: Value = serde_json::from_slice(&raw).expect("黃金檔 JSON");
    assert_eq!(golden["schema"], "fd2-rust-golden-save-v1");
    Some(golden)
}

fn run_all(golden: &Value, dir: &std::path::Path) -> std::result::Result<Stats, String> {
    let mut st = Stats::default();
    let plains: Vec<Vec<u8>> = golden["plains"].as_array().expect("plains").iter().map(b64).collect();
    let cases = golden["cases"].as_array().expect("cases");
    for c in cases {
        st.cases += 1;
        match c["kind"].as_str().expect("kind") {
            "save" => check_save(c, &mut st)?,
            "settings" => check_settings(c, &mut st)?,
            "sidecar" => check_sidecar(c, dir, &mut st)?,
            "balance_key" => check_balance_key(c, &plains, &mut st)?,
            other => return Err(format!("未知的案例種類 {other}")),
        }
    }
    // 真實檔：案例裡的位元組與黃金檔記錄的大小、SHA-256 相符。
    for real in golden["real_inputs"].as_array().expect("real_inputs") {
        let name = real["name"].as_str().expect("name");
        let c = cases
            .iter()
            .find(|c| c["id"].as_str() == Some(&format!("real-{name}")))
            .ok_or_else(|| format!("真實檔 {name} 沒有案例"))?;
        let raw = b64(&c["input"]);
        if Some(raw.len() as u64) != real["size"].as_u64()
            || Some(hex(&Sha256::digest(&raw)).as_str()) != real["sha256"].as_str()
        {
            return Err(format!("真實檔 {name} 的大小或雜湊不符"));
        }
        st.real += 1;
    }
    Ok(st)
}

/// 不設黃金檔時沒有東西可比；與 fd2-rules／fd2-campaign 放在 `tests/` 的黃金一樣不進一般的
/// `cargo test --lib`（`tools/rust_game_test.sh`），由 `tools/rust_golden_save.sh` 以 `--ignored` 執行。
#[test]
#[ignore = "黃金差分：由 tools/rust_golden_save.sh 執行"]
fn golden_save_matches_go() {
    let Some(golden) = load_golden() else {
        return;
    };
    let _ = (KNOWN_DIFFERENCES, NORMALIZATIONS);
    let dir = crate::test_support::temp_dir("golden-save");
    let st = run_all(&golden, &dir).unwrap_or_else(|e| panic!("{e}"));
    eprintln!(
        "案例 {}：存檔 {}（接受 {}，重新寫出逐位元組相同 {}）、設定 {}、附檔 {}、附檔鍵 {}、真實檔 {}",
        st.cases, st.save, st.save_accepted, st.save_bytes_exact, st.settings, st.sidecar, st.balance_key, st.real
    );
    assert_eq!(st.save_bytes_exact, st.save_accepted, "接受的存檔都要逐位元組相同");
    assert!(st.save >= 150 && st.settings >= 60 && st.sidecar >= 30 && st.balance_key >= 20 && st.real >= 5);

    // 竄改反向檢查：每一類比對都要抓得到單一欄位的改動。
    let tampers: Vec<(&str, Box<dyn Fn(&mut Value) -> bool>)> = vec![
        (
            "存檔讀回的 gold",
            Box::new(|c: &mut Value| {
                if c["kind"] == "save" && c["accepted"] == true {
                    c["view"]["gold"] = json!(c["view"]["gold"].as_i64().unwrap() + 1);
                    return true;
                }
                false
            }),
        ),
        (
            "存檔名冊單位的 HP",
            Box::new(|c: &mut Value| {
                if c["kind"] == "save"
                    && let Some(first) = c["view"]["party_roster"].as_array_mut().and_then(|a| a.first_mut())
                {
                    first[1]["HP"] = json!(first[1]["HP"].as_i64().unwrap_or(0) + 1);
                    return true;
                }
                false
            }),
        ),
        (
            "存檔外層位元組",
            Box::new(|c: &mut Value| {
                if c["kind"] == "save" && c["accepted"] == true {
                    let mut raw = b64(&c["outer"]);
                    let last = raw.len() - 2;
                    raw[last] ^= 1;
                    c["outer"] = json!(base64::engine::general_purpose::STANDARD.encode(raw));
                    return true;
                }
                false
            }),
        ),
        (
            // #319 修掉的差異：Go 寫 `"OffX": 0`，舊的 fd2-rules serde 寫 `0.0`；改回舊寫法要被位元組比對抓到。
            "存檔名冊單位的浮點寫法",
            Box::new(|c: &mut Value| {
                if c["kind"] == "save" && c["accepted"] == true {
                    let raw = String::from_utf8(b64(&c["rewritten"])).unwrap();
                    if raw.contains("\"OffX\": 0,") {
                        let raw = raw.replacen("\"OffX\": 0,", "\"OffX\": 0.0,", 1);
                        c["rewritten"] = json!(base64::engine::general_purpose::STANDARD.encode(raw));
                        return true;
                    }
                }
                false
            }),
        ),
        (
            "存檔名冊單位的 nil 切片",
            Box::new(|c: &mut Value| {
                if c["kind"] == "save" && c["accepted"] == true {
                    let raw = String::from_utf8(b64(&c["rewritten"])).unwrap();
                    if raw.contains("\"Spells\": null,") {
                        let raw = raw.replacen("\"Spells\": null,", "\"Spells\": [],", 1);
                        c["rewritten"] = json!(base64::engine::general_purpose::STANDARD.encode(raw));
                        return true;
                    }
                }
                false
            }),
        ),
        (
            "設定重新寫出的位元組",
            Box::new(|c: &mut Value| {
                if c["kind"] == "settings" && c["accepted"] == true {
                    let mut raw = b64(&c["rewritten"]);
                    raw.push(b' ');
                    c["rewritten"] = json!(base64::engine::general_purpose::STANDARD.encode(raw));
                    return true;
                }
                false
            }),
        ),
        (
            "設定讀回的 locale_id",
            Box::new(|c: &mut Value| {
                if c["kind"] == "settings" && !settings_type_error(c) {
                    c["base_view"]["locale_id"] = json!("tampered");
                    return true;
                }
                false
            }),
        ),
        (
            "附檔錯誤文字",
            Box::new(|c: &mut Value| {
                if c["kind"] == "sidecar" && c["error_kinds"][0] == "contract" && c["errors"][0] != "" {
                    c["errors"][0] = json!("竄改的錯誤");
                    return true;
                }
                false
            }),
        ),
        (
            "附檔鍵",
            Box::new(|c: &mut Value| {
                if c["kind"] == "balance_key" && c["error"] == "" {
                    c["key"] = json!(format!("{}0", c["key"].as_str().unwrap()));
                    return true;
                }
                false
            }),
        ),
    ];
    for (name, tamper) in tampers {
        let mut copy = golden.clone();
        let cases = copy["cases"].as_array_mut().unwrap();
        let hit = cases.iter_mut().find_map(|c| tamper(c).then_some(()));
        assert!(hit.is_some(), "竄改檢查「{name}」找不到可改的案例");
        let dir = crate::test_support::temp_dir("golden-save-tamper");
        match run_all(&copy, &dir) {
            Ok(_) => panic!("竄改「{name}」後比對仍然通過"),
            Err(e) => eprintln!("竄改「{name}」被抓到：{}", e.lines().next().unwrap_or("")),
        }
    }
}

/// `loadSettings` 正規化後的結果；正規化呼叫 A 組的 `normalizePlayerSpeed` 等函式。A 合併後在
/// `tools/rust_golden_save.sh` 的測試清單加上它。
#[test]
#[ignore = "黃金差分：由 tools/rust_golden_save.sh 執行"]
fn golden_save_settings_loaded_matches_go() {
    let Some(golden) = load_golden() else {
        return;
    };
    let mut n = 0;
    for c in golden["cases"].as_array().unwrap().iter().filter(|c| c["kind"] == "settings") {
        check_settings_loaded(c).unwrap_or_else(|e| panic!("{e}"));
        n += 1;
    }
    assert!(n >= 60);
}
