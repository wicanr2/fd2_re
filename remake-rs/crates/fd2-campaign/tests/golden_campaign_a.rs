//! 與凍結 Go 版 `remake/cmd/fd2-golden-campaign-a` 的黃金差分：campaign A 組（劇本資料與處理器編譯）與
//! editorcanonical。
//!
//! 由 `tools/rust_golden_campaign_a.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_CAMPAIGN_A`；真實輸入是全部
//! `remake/assets` 劇本（戰役圖、處理器腳本與綁定、演出資源、故事索引、editor-canonical），掛在
//! `FD2_GOLDEN_ASSETS`（與 Go 端相同的 `/assets`，錯誤訊息裡的路徑字樣因此相同）。先核對黃金檔記錄的
//! 每個輸入的大小與 SHA-256。設定 `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依 `op` 在 Rust 端重做同一串呼叫，結構以 serde 輸出的 JSON（Go 標籤名）逐欄比對；
//! `CompileCampaignJSON` 的輸出逐位元組比對。A 組自己的錯誤文字必須逐字相同；Go 標準函式庫產生的錯誤
//! （讀檔、JSON 解碼）只比接受或拒收。表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在
//! [`KNOWN_DIFFERENCES`]。
//!
//! 竄改反向檢查：設 `FD2_GOLDEN_TAMPER=<案例 index>|<JSON pointer>`，比對前把 Go 值在該處換成字串
//! `"tampered"`；差分必須在該欄位失敗。

use fd2_campaign::campaign::{self, Beat, HandlerBindings, HandlerCompileIssue, NativeDialogueLayout};
use fd2_campaign::editorcanonical;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

/// 刻意不照做 Go 的行為（`rust-port-no-go-stdlib` 與 PORTING.md「Go 寫法難對應處」）。黃金案例都不落在
/// 這些輸入上；列在這裡讓後續分組知道邊界。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文。
    "JSON 欄位名大小寫",
    // Go 把 null 讀進整數、布林等欄位時保留零值；Rust 只在切片、字串、map、指標（Option）欄位接受 null。
    "純量欄位的 null",
    // Go 重複的鍵以最後一個為準；serde 對具名欄位重複直接拒收。
    "重複的 JSON 鍵",
    // Go `[]*NativeDialogueLayout` 可含 nil 元素並在 Decode 報「為空」；Rust 元素不是 Option，JSON null
    // 元素在解碼時就拒收（兩者都拒收，錯誤文字不同）。
    "native_dialogues 的 null 元素",
    // attachNativeDialogueLayout 以 %v 印 *bool：Go 印指標位址（每次執行不同），Rust 印布林值。
    "native dialogue control 不一致錯誤裡的 upper 值",
    // LoadStoryIndexMap／validateStoryIndexMapping 以 %#v 印整個結構（Go 語法傾印）；Rust 印 Debug。
    "story index 錯誤裡的 %#v 結構傾印",
    // filepath.Join 會做字面清理（`a/b/../c` → `a/c`）；Rust 以 std::path 組路徑，指向同一檔案，
    // 只有錯誤訊息裡的路徑字樣可能不同。
    "filepath.Join 的路徑清理",
    // hex.DecodeString 的錯誤細節屬標準函式庫；Rust 只保留 `item N raw: ` 前綴。
    "LoadAICommandSpellMap 的 hex 錯誤細節",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片／map 寫成 null、空的寫成 []／{}；Rust 的 Vec／BTreeMap 不分 nil。null、[]、{}、"" 視為相同，
    // 一方缺席、另一方是這些空值也視為相同（Go `omitempty` 省略空 map，Rust 的 Option<map> 為 Some({}) 時寫 {}）。
    "null 與空陣列、空物件、空字串相同",
    // Go 把整數值的 float64 寫成 255，serde_json 可能寫成 255.0；只比數值。
    "數字只比數值",
    // Go JSON 解碼與讀檔錯誤屬標準函式庫（rust-port-no-go-stdlib），只比接受或拒收。
    "標準函式庫的錯誤只比接受或拒收",
];

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn digest(b: &[u8]) -> String {
    hex(&Sha256::digest(b))
}

fn is_empty_like(v: &Value) -> bool {
    match v {
        Value::Null => true,
        Value::Array(a) => a.is_empty(),
        Value::Object(o) => o.is_empty(),
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
                _ => a.as_f64() == b.as_f64(),
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
                    None if is_empty_like(x) => {}
                    None => return Some(format!("{path}.{k}: Rust 沒有這個欄位（go={x}）")),
                }
            }
            b.iter()
                .find(|(k, y)| !a.contains_key(*k) && !is_empty_like(y))
                .map(|(k, y)| format!("{path}.{k}: Go 沒有這個欄位（rust={y}）"))
        }
        _ if is_empty_like(go) && is_empty_like(rust) => None,
        _ => (go != rust).then(|| format!("{path}: go={go} rust={rust}")),
    }
}

/// Go 標準函式庫的錯誤（讀檔、JSON 解碼）：只比接受或拒收。
fn is_std_error(msg: &str) -> bool {
    ["json: ", "invalid character", "unexpected end of JSON", "no such file", "open ", "cannot unmarshal"]
        .iter()
        .any(|m| msg.contains(m))
}

struct Checker {
    failures: Vec<String>,
    compared: usize,
}

impl Checker {
    fn value(&mut self, ctx: &str, go: &Value, rust: &Value) {
        self.compared += 1;
        if let Some(d) = diff("", go, rust) {
            self.failures.push(format!("{ctx}{d}"));
        }
    }

    fn error(&mut self, ctx: &str, go: &Value, rust: Option<String>) {
        self.compared += 1;
        match (go.as_str(), rust) {
            (None, None) => {}
            (Some(g), Some(r)) => {
                if !is_std_error(g) && g != r {
                    self.failures.push(format!("{ctx}錯誤文字 go={g:?} rust={r:?}"));
                }
            }
            (g, r) => self.failures.push(format!("{ctx}接受與否不同 go={g:?} rust={r:?}")),
        }
    }
}

fn val<T: serde::Serialize>(v: &T) -> Value {
    serde_json::to_value(v).expect("serde 輸出")
}

fn err_text<T>(r: &Result<T, campaign::CampaignError>) -> Option<String> {
    r.as_ref().err().map(|e| e.to_string())
}

fn issues_val(issues: &[HandlerCompileIssue]) -> Value {
    Value::Array(
        issues
            .iter()
            .map(|i| json!({"Beat": i.beat, "Op": i.op, "Source": val(&i.source), "Reason": i.reason}))
            .collect(),
    )
}

fn layouts_of<'a>(beats: &'a [Beat], out: &mut Vec<&'a NativeDialogueLayout>) {
    for beat in beats {
        if let Some(layout) = &beat.native_dialogue {
            out.push(layout);
        }
        out.extend(beat.native_dialogues.iter());
        layouts_of(&beat.then, out);
        layouts_of(&beat.r#else, out);
    }
}

fn glyph_summary(layout: &NativeDialogueLayout) -> Value {
    let geometry = campaign::native_story_dialogue_text_geometry(&layout.control);
    let (x, y, width, step, rows) = geometry.as_ref().copied().unwrap_or_default();
    let mut pages = Vec::new();
    for page in 0..=layout.pages.len() as i64 {
        let steps = campaign::native_story_dialogue_glyph_steps(layout, page);
        let mut entry = json!({"page": page, "error": err_text(&steps)});
        if let Ok(steps) = &steps {
            let clock: String = steps.iter().map(|s| if *s { '1' } else { '0' }).collect();
            entry["steps"] = json!(clock);
            let n = steps.len() as i64;
            for progress in [0, (n - 1) / 2, n - 1, n] {
                let glyphs = campaign::native_story_dialogue_visible_glyphs(layout, page, progress);
                let list = glyphs.as_ref().map(|g| g.as_slice()).unwrap_or_default();
                let text: String =
                    list.iter().map(|g| format!("{}|{}|{}|{}\n", g.token, g.x, g.y, g.glyph_id)).collect();
                entry[format!("visible_{progress}")] =
                    json!({"error": err_text(&glyphs), "count": list.len(), "sha256": digest(text.as_bytes())});
            }
        }
        pages.push(entry);
    }
    json!({
        "geometry": [x, y, width, step, rows],
        "geometry_error": err_text(&geometry),
        "validate": err_text(&layout.validate()),
        "pages": pages,
    })
}

fn expand_val(beats: &[Beat]) -> Value {
    let expanded = campaign::expand_native_dialogue_groups(beats);
    let mut result = json!({"error": err_text(&expanded)});
    if let Ok(expanded) = &expanded {
        result["beats"] = val(expanded);
        let mut layouts = Vec::new();
        layouts_of(expanded, &mut layouts);
        result["glyphs"] = Value::Array(layouts.into_iter().map(glyph_summary).collect());
    }
    result
}

const OUTCOMES: &[&str] = &[
    "",
    "win",
    "lose",
    "next",
    "opt0",
    "opt1",
    "opt2",
    "opt3",
    "opt-1",
    "opt 1",
    "opt01x",
    "optx",
    "present",
    "missing",
    "crafted",
    "insufficient",
    "cancel",
    "rumor",
];

fn runner_val(c: &campaign::Campaign) -> Value {
    let mut nodes = serde_json::Map::new();
    for (id, node) in &c.nodes {
        let mut advance = serde_json::Map::new();
        for outcome in OUTCOMES {
            let mut r = campaign::new_runner(c);
            r.cur = id.clone();
            let next = r.advance(outcome);
            advance.insert(outcome.to_string(), json!({"next": next, "cur": r.cur, "flags": val(&r.flags)}));
        }
        let mut r = campaign::new_runner(c);
        r.cur = id.clone();
        let mut entry = json!({"advance": advance, "shop_goods": val(&r.shop_goods()), "node_id": r.node_id()});
        if node.r#type == "choice" || node.r#type == "town" {
            entry["visible"] = val(&r.visible());
        }
        let mut scans = vec![0x3b, 0x54, 0x71];
        if let Some(gate) = &node.native_secret_gate {
            scans.push(gate.scan_code);
        }
        let mut matched = Vec::new();
        for selection in -1..=5 {
            for &scan in &scans {
                if r.match_native_town_secret(selection, scan) {
                    matched.push(json!({"selection": selection, "scan": scan}));
                }
            }
        }
        entry["secret_match"] = Value::Array(matched);
        let mut confirm = serde_json::Map::new();
        for selection in [4, 5] {
            let mut r = campaign::new_runner(c);
            r.cur = id.clone();
            let ok = r.confirm_native_town_secret(selection);
            confirm.insert(selection.to_string(), json!({"ok": ok, "cur": r.cur}));
        }
        entry["secret_confirm"] = Value::Object(confirm);
        nodes.insert(id.clone(), entry);
    }
    let r = campaign::new_runner(c);
    json!({"start": r.cur, "flags": val(&r.flags), "nodes": nodes})
}

fn s<'a>(v: &'a Value, key: &str) -> &'a str {
    v[key].as_str().unwrap_or_else(|| panic!("欄位 {key} 不是字串：{}", v[key]))
}

fn campaign_load(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let loaded = campaign::load(s(input, "path"));
    ck.error(ctx, &result["error"], err_text(&loaded));
    let Ok(c) = loaded else {
        return;
    };
    ck.value(&format!("{ctx}campaign"), &result["campaign"], &val(&c));
    for (id, node) in &c.nodes {
        if !node.beats.is_empty() {
            let go = &result["expanded"][id];
            let rust = expand_val(&node.beats);
            ck.error(&format!("{ctx}expanded[{id}] "), &go["error"], rust["error"].as_str().map(String::from));
            ck.value(&format!("{ctx}expanded[{id}]"), go, &rust);
        }
    }
    ck.value(&format!("{ctx}runner"), &result["runner"], &runner_val(&c));
}

fn handler_script(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let loaded = campaign::load_handler_script(s(input, "path"));
    ck.error(ctx, &result["error"], err_text(&loaded));
    let Ok(script) = loaded else {
        return;
    };
    ck.value(&format!("{ctx}script"), &result["script"], &val(&script));
    let (beats, issues) = campaign::compile_handler_script(Some(&script), &HandlerBindings::default());
    ck.value(&format!("{ctx}beats"), &result["beats"], &val(&beats));
    ck.value(&format!("{ctx}issues"), &result["issues"], &issues_val(&issues));
}

fn handler_binding(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let path = s(input, "path");
    let loaded = campaign::load_handler_binding(path);
    ck.error(&format!("{ctx}load "), &result["load_error"], err_text(&loaded));
    if let Ok(binding) = &loaded {
        ck.value(&format!("{ctx}binding"), &result["binding"], &val(binding));
    }
    let compiled = campaign::compile_handler_binding(path);
    ck.error(ctx, &result["error"], err_text(&compiled));
    let Ok((beats, issues)) = compiled else {
        return;
    };
    ck.value(&format!("{ctx}beats"), &result["beats"], &val(&beats));
    ck.value(&format!("{ctx}issues"), &result["issues"], &issues_val(&issues));
    let expanded = expand_val(&beats);
    ck.error(&format!("{ctx}expanded "), &result["expanded"]["error"], expanded["error"].as_str().map(String::from));
    ck.value(&format!("{ctx}expanded"), &result["expanded"], &expanded);
}

fn acting_set(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let loaded = campaign::load_acting_resource_set(s(input, "path"));
    ck.error(ctx, &result["error"], err_text(&loaded));
    if let Ok(resources) = loaded {
        ck.value(&format!("{ctx}resources"), &result["resources"], &val(&resources));
    }
}

fn story_index(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let loaded = campaign::load_story_index_map(s(input, "path"));
    ck.error(ctx, &result["error"], err_text(&loaded));
    let Ok(index) = loaded else {
        return;
    };
    ck.value(&format!("{ctx}index"), &result["index"], &val(&index));
    let mut lookups = Vec::new();
    for resource in &index.resources {
        for mapping in &resource.script_mappings {
            for i in -1..=mapping.mappings.len() as i64 {
                let (targets, ok) = index.lookup(&resource.source_dat, &mapping.script, i);
                lookups.push(json!({"source_dat": resource.source_dat, "script": mapping.script, "index": i,
                    "ok": ok, "targets": val(&targets)}));
            }
            let missing = format!("{}.missing", mapping.script);
            let (targets, ok) = index.lookup(&resource.source_dat, &missing, 0);
            lookups.push(json!({"source_dat": resource.source_dat, "script": missing, "index": 0,
                "ok": ok, "targets": val(&targets)}));
        }
    }
    ck.value(&format!("{ctx}lookups"), &result["lookups"], &Value::Array(lookups));
}

fn editor_canonical(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let root = s(input, "root");
    let summary = editorcanonical::validate_bundle(root);
    ck.error(&format!("{ctx}validate "), &result["validate_error"], err_text(&summary));
    if let Ok(summary) = &summary {
        ck.value(&format!("{ctx}summary"), &result["summary"], &val(summary));
    }
    let compiled = editorcanonical::compile_campaign_json(root);
    ck.error(&format!("{ctx}compile "), &result["compile_error"], err_text(&compiled));
    let Ok(compiled) = compiled else {
        return;
    };
    // 逐位元組比對：鍵序、RawMessage 壓縮與 HTML 跳脫都要與 Go json.Marshal 相同。
    ck.compared += 1;
    if result["compiled_sha256"].as_str() != Some(digest(&compiled).as_str())
        || result["compiled"].as_str().map(str::as_bytes) != Some(compiled.as_slice())
    {
        let go = result["compiled"].as_str().unwrap_or_default().as_bytes();
        let at = go.iter().zip(&compiled).position(|(a, b)| a != b).unwrap_or(go.len().min(compiled.len()));
        let window = |b: &[u8]| String::from_utf8_lossy(&b[at.saturating_sub(40)..(at + 40).min(b.len())]).into_owned();
        ck.failures.push(format!(
            "{ctx}compiled 位元組不同（go {} bytes、rust {} bytes，第 {at} 位元組起）\ngo  ={}\nrust={}",
            go.len(),
            compiled.len(),
            window(go),
            window(&compiled)
        ));
    }
    let decoded = campaign::decode(&compiled);
    ck.error(&format!("{ctx}decode "), &result["decode_error"], err_text(&decoded));
    if let Ok(c) = decoded {
        ck.value(&format!("{ctx}campaign"), &result["campaign"], &val(&c));
    }
}

/// 合成的 canonical 文件：寫進暫存目錄的 `campaign/campaign_full.json` 再編譯（Go 端同樣做法）。
fn editor_canonical_synthetic(ck: &mut Checker, ctx: &str, input: &Value, result: &Value) {
    let root = std::env::temp_dir().join(format!("fd2-golden-campaign-a-{}-{}", std::process::id(), s(input, "name")));
    let _ = std::fs::remove_dir_all(&root);
    std::fs::create_dir_all(root.join("campaign")).expect("建立暫存目錄");
    std::fs::write(root.join("campaign").join("campaign_full.json"), s(input, "document")).expect("寫入合成文件");
    let compiled = editorcanonical::compile_campaign_json(&root.to_string_lossy());
    let _ = std::fs::remove_dir_all(&root);
    ck.error(&format!("{ctx}compile "), &result["compile_error"], err_text(&compiled));
    let Ok(compiled) = compiled else {
        return;
    };
    ck.compared += 1;
    if result["compiled"].as_str().map(str::as_bytes) != Some(compiled.as_slice())
        || result["compiled_sha256"].as_str() != Some(digest(&compiled).as_str())
    {
        ck.failures.push(format!(
            "{ctx}compiled 位元組不同\ngo  ={}\nrust={}",
            result["compiled"].as_str().unwrap_or_default(),
            String::from_utf8_lossy(&compiled)
        ));
    }
    ck.error(&format!("{ctx}decode "), &result["decode_error"], err_text(&campaign::decode(&compiled)));
}

#[test]
fn golden_campaign_a() {
    let _ = (KNOWN_DIFFERENCES, NORMALIZATIONS);
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let (Some(path), Some(assets)) = (std::env::var_os("FD2_GOLDEN_CAMPAIGN_A"), std::env::var_os("FD2_GOLDEN_ASSETS"))
    else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_CAMPAIGN_A 或 FD2_GOLDEN_ASSETS");
        eprintln!("略過：沒有 FD2_GOLDEN_CAMPAIGN_A／FD2_GOLDEN_ASSETS");
        return;
    };
    let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀黃金檔 {}: {e}", path.to_string_lossy()));
    println!("黃金檔 {} SHA-256 {}", path.to_string_lossy(), digest(&raw));
    let mut doc: Value = serde_json::from_slice(&raw).expect("黃金檔 JSON");
    assert_eq!(doc["schema"], "fd2-golden-campaign-a/1");
    let assets = assets.to_string_lossy().into_owned();
    assert_eq!(doc["assets"].as_str(), Some(assets.as_str()), "黃金檔的素材根目錄與 FD2_GOLDEN_ASSETS 不同");

    // 真實輸入：逐一核對大小與 SHA-256。
    let inputs = doc["inputs"].as_array().expect("inputs").clone();
    for input in &inputs {
        let rel = s(input, "path");
        let file = std::fs::read(format!("{assets}/{rel}")).unwrap_or_else(|e| panic!("讀輸入 {rel}: {e}"));
        assert_eq!(file.len() as u64, input["size"].as_u64().unwrap(), "輸入 {rel} 大小不同");
        assert_eq!(digest(&file), s(input, "sha256"), "輸入 {rel} SHA-256 不同");
    }

    if let Ok(spec) = std::env::var("FD2_GOLDEN_TAMPER") {
        let (index, pointer) = spec.split_once('|').expect("FD2_GOLDEN_TAMPER=<index>|<pointer>");
        let index: usize = index.parse().expect("竄改案例 index");
        let target = doc["cases"][index].pointer_mut(pointer).unwrap_or_else(|| panic!("竄改位置 {pointer} 不存在"));
        println!("竄改：案例 {index} {pointer}：{target} → \"tampered\"");
        *target = json!("tampered");
    }

    let mut ck = Checker { failures: Vec::new(), compared: 0 };
    let cases = doc["cases"].as_array().expect("cases");
    let mut ops = std::collections::BTreeMap::<String, usize>::new();
    for case in cases {
        let op = s(case, "op");
        *ops.entry(op.to_string()).or_default() += 1;
        let (input, result) = (&case["input"], &case["result"]);
        let name = input["path"].as_str().or(input["name"].as_str()).unwrap_or("");
        let ctx = format!("案例 {} {op} {name}：", case["index"]);
        match op {
            "campaign_load" => campaign_load(&mut ck, &ctx, input, result),
            "handler_script" => handler_script(&mut ck, &ctx, input, result),
            "handler_binding" => handler_binding(&mut ck, &ctx, input, result),
            "acting_set" => acting_set(&mut ck, &ctx, input, result),
            "story_index" => story_index(&mut ck, &ctx, input, result),
            "editor_canonical" => editor_canonical(&mut ck, &ctx, input, result),
            "editor_canonical_synthetic" => editor_canonical_synthetic(&mut ck, &ctx, input, result),
            other => panic!("未知的案例 op {other}"),
        }
    }
    println!("案例 {}（{ops:?}），輸入檔 {}，比對 {} 項", cases.len(), inputs.len(), ck.compared);
    if !ck.failures.is_empty() {
        let shown: Vec<&String> = ck.failures.iter().take(20).collect();
        panic!(
            "{} 項不同（列前 20 項）：\n{}",
            ck.failures.len(),
            shown.iter().map(|s| s.as_str()).collect::<Vec<_>>().join("\n")
        );
    }
}
