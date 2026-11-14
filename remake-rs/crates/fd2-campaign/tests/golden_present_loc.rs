//! 與凍結 Go 版 `remake/cmd/fd2-golden-present-loc` 的黃金差分：fd2-campaign 的 `localization` 與
//! `musiccatalog`（R3 present 組）。
//!
//! 由 `tools/rust_golden_present_loc.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_PRESENT_LOC`；真實輸入掛在
//! `FD2_GOLDEN_ASSETS`（與 Go 端同一個路徑，錯誤訊息裡的路徑兩邊相同），先逐檔核對黃金檔記錄的大小與
//! SHA-256。設定 `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依 `op` 在 Rust 端重做同一串呼叫，結果逐欄比對。變造案例照黃金檔記錄的操作改寫同一份
//! JSON 樹，寫進自己的暫存根目錄下同名的子目錄。兩套件自己的錯誤文字必須逐字相同；Go 標準函式庫產生的
//! 錯誤只比本套件的前綴。表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在
//! [`KNOWN_DIFFERENCES`]。

use fd2_campaign::localization::{self, FormatArg};
use fd2_campaign::musiccatalog;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為（`rust-port-no-go-stdlib`）。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Format 的 fmt.Sprintf 只實作 %d %v %x %X %c（整數）、%s %v %q %x %X（字串）、%f %F（浮點數）；Go 支援但
    // 這裡不支援的組合（例如 %e、%g、%b、%o、%T、%U、浮點數的 %v、%x、整數的 %q）在 Rust 回
    // 「incompatible argument types」。黃金案例中這類「Go 成功、Rust 不支援」的組合跳過並計數。
    "Format 不支援的動詞與型別組合",
    // Go 解碼時欄位名不分大小寫、重複的欄位取後者；serde 只認標籤原文，重複欄位拒收。黃金案例不含這兩種輸入。
    "JSON 欄位名大小寫與重複欄位",
    // entities.json 的驗證在 Go 以 map 的隨機順序走訪；多筆不合法時回報哪一筆、兩個鍵字串轉成同一個 ID 時
    // 留下哪個名稱由順序決定。黃金案例每次只放一個不合法項目，也不含同 ID 的兩個鍵。
    "entities.json 的 map 走訪順序",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // 暫存目錄的前綴兩邊不同，錯誤訊息與 Resolve 的路徑裡換成 <tmp>。
    "暫存目錄前綴換成 <tmp>",
    // Go 標準函式庫的錯誤文字（os 讀檔、encoding/json 解碼）只比到本套件的前綴：
    // `read locale ... "<路徑>": `、`decode locale ... "<路徑>": `、`... trailing JSON: `、
    // `music catalog: `、`music catalog decode: `、`... profile fm: open <路徑>: `。
    "標準函式庫錯誤只比本套件前綴",
    // StoryText 的文字以 SHA-256 前 16 個十六進位字元比對（黃金檔不存全文）。
    "官方對白文字比雜湊",
    // Go 複製 Variables 用 `append([]string(nil), ...)`，空清單寫成 null；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列相同",
];

fn is_empty_like(v: &Value) -> bool {
    matches!(v, Value::Null) || v.as_array().is_some_and(Vec::is_empty)
}

/// 依 [`NORMALIZATIONS`] 比對兩個 JSON 值，回傳第一個不同的位置。
fn diff(path: &str, go: &Value, rust: &Value) -> Option<String> {
    match (go, rust) {
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

fn check_json(label: &str, go: &Value, rust: &Value) {
    if let Some(d) = diff("", go, rust) {
        panic!("{label}: {d}");
    }
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn digest(b: &[u8]) -> String {
    hex(&Sha256::digest(b))
}

fn short(s: &str) -> String {
    digest(s.as_bytes())[..16].to_string()
}

fn require_golden() -> bool {
    std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1")
}

struct Ctx {
    assets: String,
    tmp: PathBuf,
    tmp_str: String,
}

impl Ctx {
    fn errv(&self, e: Option<String>) -> Value {
        match e {
            None => Value::Null,
            Some(s) => Value::String(s.replace(&self.tmp_str, "<tmp>")),
        }
    }

    fn dir(&self, name: &str) -> PathBuf {
        let dir = self.tmp.join(name);
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    fn locales(&self) -> String {
        format!("{}/locales", self.assets)
    }
}

/// Go 錯誤文字裡屬於標準函式庫的部分從哪裡開始；`None` 表示整段都要逐字相同。
fn std_prefix(go: &str) -> Option<usize> {
    for head in [
        "read locale pack ",
        "read locale content ",
        "read locale entities ",
        "decode locale pack ",
        "decode locale content ",
        "decode locale entities ",
    ] {
        if let Some(rest) = go.strip_prefix(head) {
            // `<head>"<路徑>": <detail>`；路徑是 %q，裡面沒有未跳脫的 `"`。
            let close = rest[1..].find('"').map(|i| i + 2)?;
            let after = &rest[close..];
            let base = head.len() + close + 2;
            if head.starts_with("decode") {
                if after == ": trailing JSON" {
                    return None;
                }
                if after.starts_with(": trailing JSON: ") {
                    return Some(base + "trailing JSON: ".len());
                }
            }
            return Some(base);
        }
    }
    if go.starts_with("music catalog decode: ") {
        return Some("music catalog decode: ".len());
    }
    if go.starts_with("music catalog: ") {
        return Some("music catalog: ".len());
    }
    if go.starts_with("music catalog track ")
        && let Some(at) = go.find(": open ")
    {
        let rest = &go[at + ": open ".len()..];
        return rest.find(": ").map(|i| at + ": open ".len() + i + 2);
    }
    None
}

fn same_error(go: &Value, rust: &Value) -> bool {
    match (go, rust) {
        (Value::Null, Value::Null) => true,
        (Value::String(g), Value::String(r)) => match std_prefix(g) {
            None => g == r,
            Some(n) => r.len() >= n && g.as_bytes()[..n] == r.as_bytes()[..n],
        },
        _ => false,
    }
}

fn check_error(label: &str, go: &Value, rust: &Value) {
    assert!(same_error(go, rust), "{label}: 錯誤不同\n  go={go}\n  rust={rust}");
}

fn catalog_json(c: &localization::Catalog) -> Value {
    let entries: serde_json::Map<String, Value> = c
        .entries
        .iter()
        .map(|(k, v)| {
            (k.clone(), json!({"text": v.text, "variables": v.variables, "source_string_id": v.source_string_id}))
        })
        .collect();
    json!({"locale": c.locale, "kind": c.kind, "base_locale": c.base_locale, "layout_profile": c.layout_profile,
        "font": {"profile": c.font.profile, "path": c.font.path}, "entries": entries})
}

fn args_from(v: &Value) -> Vec<FormatArg> {
    v.as_array()
        .unwrap()
        .iter()
        .map(|a| match a["t"].as_str().unwrap() {
            "i" => FormatArg::Int(a["v"].as_i64().unwrap()),
            "s" => FormatArg::Str(a["v"].as_str().unwrap().to_string()),
            "f" => FormatArg::Float(a["v"].as_f64().unwrap()),
            t => panic!("未知參數型別 {t}"),
        })
        .collect()
}

/// Rust 的 `go_sprintf` 是否支援這個動詞與參數型別（見 [`KNOWN_DIFFERENCES`]）。
fn supported(verb: char, arg: &FormatArg) -> bool {
    matches!(
        (verb, arg),
        ('d' | 'v' | 'x' | 'X' | 'c', FormatArg::Int(_))
            | ('s' | 'v' | 'q' | 'x' | 'X', FormatArg::Str(_))
            | ('f' | 'F', FormatArg::Float(_))
    )
}

// ---- 變造操作（對應 Go 的 mutation） ----

fn seg_ref<'a>(v: &'a mut Value, seg: &Value) -> &'a mut Value {
    match seg {
        Value::String(k) => v.get_mut(k).unwrap_or_else(|| panic!("路徑段 {k} 不存在")),
        Value::Number(n) => &mut v[n.as_u64().unwrap() as usize],
        _ => panic!("路徑段 {seg}"),
    }
}

fn apply(tree: &mut Value, m: &Value) {
    let path = m["path"].as_array().unwrap();
    let (last, parents) = path.split_last().unwrap();
    let mut parent = tree;
    for seg in parents {
        parent = seg_ref(parent, seg);
    }
    match m["op"].as_str().unwrap() {
        "set" => match last {
            Value::String(k) => {
                parent.as_object_mut().unwrap().insert(k.clone(), m["value"].clone());
            }
            Value::Number(n) => parent[n.as_u64().unwrap() as usize] = m["value"].clone(),
            _ => panic!("set {last}"),
        },
        "delete" => {
            parent.as_object_mut().unwrap().remove(last.as_str().unwrap());
        }
        "rename" => {
            let obj = parent.as_object_mut().unwrap();
            let v = obj.remove(last.as_str().unwrap()).unwrap_or(Value::Null);
            obj.insert(m["to"].as_str().unwrap().to_string(), v);
        }
        op => panic!("未知操作 {op}"),
    }
}

fn read_tree(path: impl AsRef<Path>) -> Value {
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}

fn write_mutated(path: &Path, base: &Value, ops: &Value, suffix: &str) {
    let mut tree = base.clone();
    for m in ops.as_array().map(Vec::as_slice).unwrap_or_default() {
        apply(&mut tree, m);
    }
    let mut raw = serde_json::to_vec(&tree).unwrap();
    raw.extend_from_slice(suffix.as_bytes());
    std::fs::create_dir_all(path.parent().unwrap()).unwrap();
    std::fs::write(path, raw).unwrap();
}

const VALID_PACK: &str = r#"{"schema_version":1,"pack_id":"test-en","locale":"en","kind":"official","layout_profile":"dos-8x8-latin","font":{"profile":"fd2-latin-8x8","path":"fonts/atlas.png"},"entries":{"battle.attack.hit":{"text":"%s deals %d damage","variables":["%s","%d"],"source_string_id":"native-1"}}}"#;

fn synthetic_content(ctx: &Ctx, n: usize) -> Value {
    let mut tree = read_tree(format!("{}/locales/en/content.json", ctx.assets));
    let entries: Vec<Value> = tree["entries"].as_array().unwrap()[..n].to_vec();
    tree["entries"] = Value::Array(entries);
    tree["entry_count"] = json!(n);
    tree
}

// ---- 各 op ----

fn run_load_official(ctx: &Ctx, input: &Value, want: &Value) {
    let locale = input["locale"].as_str().unwrap();
    let got = localization::load_official(&ctx.locales(), locale);
    let label = format!("load_official {locale}");
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    if let Ok(c) = got {
        check_json(&label, &want["catalog"], &catalog_json(&c));
    }
}

fn run_format(
    ctx: &Ctx,
    catalogs: &mut std::collections::BTreeMap<String, localization::Catalog>,
    input: &Value,
    want: &Value,
) {
    let locale = input["locale"].as_str().unwrap();
    let catalog = catalogs
        .entry(locale.to_string())
        .or_insert_with(|| localization::load_official(&ctx.locales(), locale).unwrap());
    let key = input["key"].as_str().unwrap();
    let args = args_from(&input["args"]);
    let got = catalog.format(key, &args);
    let label = format!("format {locale} {key} {}", input["args"]);
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    assert_eq!(got.unwrap_or_default(), want["text"].as_str().unwrap(), "{label}");
}

fn run_content(ctx: &Ctx, input: &Value, want: &Value) {
    let locale = input["locale"].as_str().unwrap();
    let got = localization::load_official_content(&ctx.locales(), locale);
    let label = format!("content {locale}");
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    let Ok(c) = got else { return };
    assert_eq!(c.locale, want["locale"].as_str().unwrap(), "{label}");
    assert_eq!(c.inventory_sha256, want["inventory_sha256"].as_str().unwrap(), "{label}");
    assert_eq!(c.glossary_sha256, want["glossary_sha256"].as_str().unwrap(), "{label}");
    assert_eq!(c.translation_engine, want["translation_engine"].as_str().unwrap(), "{label}");
    for line in want["lines"].as_array().unwrap() {
        let id = line[0].as_str().unwrap();
        let got = c.story_text(id);
        let got_hash = got.as_ref().ok().map(|t| Value::String(short(t))).unwrap_or(Value::Null);
        assert_eq!(got_hash, line[1], "{label} {id}");
        check_error(&format!("{label} {id}"), &line[2], &ctx.errv(got.err().map(|e| e.to_string())));
    }
}

fn entity_lookups(ctx: &Ctx, c: &localization::EntityCatalog, want: &Value, label: &str) {
    type Lookup = fn(&localization::EntityCatalog, i64) -> localization::Result<String>;
    let tables: [(&str, Lookup); 5] = [
        ("item", localization::EntityCatalog::item_name),
        ("character", localization::EntityCatalog::character_name),
        ("battle", localization::EntityCatalog::battle_name),
        ("command", localization::EntityCatalog::command_name),
        ("class", localization::EntityCatalog::class_name),
    ];
    for (name, f) in tables {
        for row in want[name].as_array().unwrap() {
            let id = row[0].as_i64().unwrap();
            let got = f(c, id);
            assert_eq!(got.as_deref().unwrap_or(""), row[1].as_str().unwrap(), "{label} {name} {id}");
            check_error(&format!("{label} {name} {id}"), &row[2], &ctx.errv(got.err().map(|e| e.to_string())));
        }
    }
}

fn run_entities(ctx: &Ctx, input: &Value, want: &Value) {
    let locale = input["locale"].as_str().unwrap();
    let got = localization::load_official_entities(&ctx.locales(), locale);
    let label = format!("entities {locale}");
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    if let Ok(c) = got {
        assert_eq!(c.locale, want["locale"].as_str().unwrap(), "{label}");
        entity_lookups(ctx, &c, &want["lookups"], &label);
    }
}

fn run_pack(ctx: &Ctx, input: &Value, want: &Value) {
    let dir = ctx.dir(input["dir"].as_str().unwrap());
    let base: Value = serde_json::from_str(VALID_PACK).unwrap();
    let real;
    let base = if input["label"].as_str().unwrap().starts_with("real:") {
        real = read_tree(format!("{}/locales/en/pack.json", ctx.assets));
        &real
    } else {
        &base
    };
    let suffix = input["suffix"].as_str().unwrap();
    let got = if input["official"].as_bool().unwrap() {
        let locale = input["locale"].as_str().unwrap();
        write_mutated(&dir.join(locale).join("pack.json"), base, &input["ops"], suffix);
        localization::load_official(&dir.to_string_lossy(), locale)
    } else {
        write_mutated(&dir.join("pack.json"), base, &input["ops"], suffix);
        localization::load(&dir.join("pack.json").to_string_lossy())
    };
    let label = format!("pack {}", input["label"]);
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    if let Ok(c) = got {
        check_json(&label, &want["catalog"], &catalog_json(&c));
    }
}

fn run_placeholders(ctx: &Ctx, index: i64, input: &Value, want: &Value) {
    let text = input["text"].as_str().unwrap();
    let base: Value = serde_json::from_str(VALID_PACK).unwrap();
    let content_base = synthetic_content(ctx, 3);
    let k = "battle.attack.hit";
    // Go 端以真實 Load 驗證過 `format` 清單成立；Rust 端 Load 成立、而多一個變數就失敗，表示 Rust 掃出的
    // 清單與 Go 相同。內容檔同理。
    for (vars, extra) in [(&want["format"], false), (&want["format"], true)] {
        let mut v = vars.as_array().unwrap().clone();
        if extra {
            v.push(json!("%d"));
        }
        let dir = ctx.dir(&format!("ph{index}-{extra}"));
        let ops = json!([{"op": "set", "path": ["entries", k, "text"], "value": text},
            {"op": "set", "path": ["entries", k, "variables"], "value": v}]);
        write_mutated(&dir.join("pack.json"), &base, &ops, "");
        let got = localization::load(&dir.join("pack.json").to_string_lossy());
        assert_eq!(got.is_err(), extra, "placeholders {text:?} format extra={extra}: {got:?}");
    }
    for extra in [false, true] {
        let mut v = want["content"].as_array().unwrap().clone();
        if extra {
            v.push(json!("%d"));
        }
        let dir = ctx.dir(&format!("phc{index}-{extra}"));
        let ops = json!([{"op": "set", "path": ["entries", 0, "text"], "value": text},
            {"op": "set", "path": ["entries", 0, "variables"], "value": v}]);
        write_mutated(&dir.join("en").join("content.json"), &content_base, &ops, "");
        let got = localization::load_official_content(&dir.to_string_lossy(), "en");
        assert_eq!(got.is_err(), extra, "placeholders {text:?} content extra={extra}: {:?}", got.err());
    }
}

fn run_locale_pattern(ctx: &Ctx, input: &Value, want: &Value) {
    let dir = ctx.dir(input["dir"].as_str().unwrap());
    let root = dir.to_string_lossy();
    let locale = input["locale"].as_str().unwrap();
    let label = format!("locale_pattern {locale:?}");
    let e1 = localization::load_official(&root, locale).err().map(|e| e.to_string());
    let e2 = localization::load_official_content(&root, locale).err().map(|e| e.to_string());
    let e3 = localization::load_official_entities(&root, locale).err().map(|e| e.to_string());
    check_error(&label, &want["pack"], &ctx.errv(e1));
    check_error(&label, &want["content"], &ctx.errv(e2));
    check_error(&label, &want["entities"], &ctx.errv(e3));
}

fn run_content_mut(ctx: &Ctx, input: &Value, want: &Value) {
    let dir = ctx.dir(input["dir"].as_str().unwrap());
    let base = synthetic_content(ctx, 12);
    write_mutated(&dir.join("en").join("content.json"), &base, &input["ops"], input["suffix"].as_str().unwrap());
    let got = localization::load_official_content(&dir.to_string_lossy(), input["locale"].as_str().unwrap());
    let label = format!("content_mut {}", input["label"]);
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    let Ok(c) = got else { return };
    assert_eq!(c.inventory_sha256, want["inventory_sha256"].as_str().unwrap(), "{label}");
    assert_eq!(c.glossary_sha256, want["glossary_sha256"].as_str().unwrap(), "{label}");
    assert_eq!(c.translation_engine, want["translation_engine"].as_str().unwrap(), "{label}");
    for line in want["lines"].as_array().unwrap() {
        let id = line[0].as_str().unwrap();
        let got = c.story_text(id);
        assert_eq!(got.as_deref().unwrap_or(""), line[1].as_str().unwrap(), "{label} {id}");
        check_error(&format!("{label} {id}"), &line[2], &ctx.errv(got.err().map(|e| e.to_string())));
    }
}

fn run_entities_mut(ctx: &Ctx, base: &Value, input: &Value, want: &Value) {
    let dir = ctx.dir(input["dir"].as_str().unwrap());
    write_mutated(&dir.join("en").join("entities.json"), base, &input["ops"], input["suffix"].as_str().unwrap());
    let got = localization::load_official_entities(&dir.to_string_lossy(), "en");
    let label = format!("entities_mut {}", input["label"]);
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    if let Ok(c) = got {
        entity_lookups(ctx, &c, &want["lookups"], &label);
    }
}

fn music_result(ctx: &Ctx, got: musiccatalog::Result<musiccatalog::Catalog>, want: &Value, label: &str) {
    check_error(label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    let Ok(c) = got else { return };
    let w = &want["catalog"];
    assert_eq!(c.schema, w["schema"].as_str().unwrap(), "{label}");
    assert_eq!(c.schema_version, w["schema_version"].as_i64().unwrap(), "{label}");
    // Go 的 Source／Loop／Profile／Track／Render 都有 JSON 標籤，Rust serde 的欄位名照標籤；直接比整棵樹。
    check_json(label, &w["source"], &serde_json::to_value(&c.source).unwrap());
    check_json(label, &w["loop"], &serde_json::to_value(&c.loop_).unwrap());
    check_json(label, &w["profiles"], &serde_json::to_value(&c.profiles).unwrap());
    check_json(label, &w["tracks"], &serde_json::to_value(&c.tracks).unwrap());
    for row in want["resolves"].as_array().unwrap() {
        let (profile, track) = (row[0].as_str().unwrap(), row[1].as_str().unwrap());
        let got = c.resolve(profile, track);
        let path = got.as_deref().unwrap_or("").replace(&ctx.tmp_str, "<tmp>");
        assert_eq!(path, row[2].as_str().unwrap(), "{label} resolve {profile} {track}");
        check_error(
            &format!("{label} resolve {profile} {track}"),
            &row[3],
            &ctx.errv(got.err().map(|e| e.to_string())),
        );
    }
}

fn link_renders(ctx: &Ctx, dir: &Path) {
    for d in ["music_fm", "music_mt32"] {
        std::os::unix::fs::symlink(Path::new(&ctx.assets).join(d), dir.join(d)).unwrap();
    }
}

fn run_music_mut(ctx: &Ctx, base: &Value, input: &Value, want: &Value) {
    let dir = ctx.dir(input["dir"].as_str().unwrap());
    write_mutated(&dir.join("music_catalog.json"), base, &input["ops"], input["suffix"].as_str().unwrap());
    if input["renders"].as_bool().unwrap() {
        link_renders(ctx, &dir);
    }
    music_result(ctx, musiccatalog::load(&dir.to_string_lossy()), want, &format!("music_mut {}", input["label"]));
}

fn run_music_ogg(ctx: &Ctx, base: &Value, input: &Value, want: &Value) {
    let dir = ctx.dir(input["dir"].as_str().unwrap());
    let orig_rel = input["orig"].as_str().unwrap();
    let mut data = std::fs::read(Path::new(&ctx.assets).join(orig_rel)).unwrap();
    let truncate = input["truncate"].as_i64().unwrap();
    if truncate >= 0 {
        data.truncate(truncate as usize);
    }
    for p in input["patch"].as_array().map(Vec::as_slice).unwrap_or_default() {
        data[p[0].as_u64().unwrap() as usize] = p[1].as_u64().unwrap() as u8;
    }
    for v in input["append"].as_array().map(Vec::as_slice).unwrap_or_default() {
        data.push(v.as_u64().unwrap() as u8);
    }
    let label = format!("music_ogg {}", input["label"]);
    assert_eq!(digest(&data), input["data_sha256"].as_str().unwrap(), "{label}: 重建的 OGG 與 Go 不同");
    for d in ["music_fm", "music_mt32"] {
        std::fs::create_dir_all(dir.join(d)).unwrap();
        let mut names: Vec<_> =
            std::fs::read_dir(Path::new(&ctx.assets).join(d)).unwrap().map(|e| e.unwrap().file_name()).collect();
        names.sort();
        for name in names {
            std::os::unix::fs::symlink(Path::new(&ctx.assets).join(d).join(&name), dir.join(d).join(&name)).unwrap();
        }
    }
    let target = dir.join(orig_rel);
    std::fs::remove_file(&target).unwrap();
    std::fs::write(&target, &data).unwrap();
    write_mutated(&dir.join("music_catalog.json"), base, &input["ops"], "");
    music_result(ctx, musiccatalog::load(&dir.to_string_lossy()), want, &label);
}

fn run_format_spec(ctx: &Ctx, index: i64, input: &Value, want: &Value, skipped: &mut usize) -> bool {
    let text = input["text"].as_str().unwrap();
    let args = args_from(&input["args"]);
    let verb = input["verb"].as_str().unwrap().chars().next().unwrap();
    let go_ok = want["error"].is_null();
    if go_ok && !supported(verb, &args[0]) {
        *skipped += 1;
        return false;
    }
    let base: Value = serde_json::from_str(VALID_PACK).unwrap();
    let dir = ctx.dir(&format!("fs{index}"));
    let vars = if text.contains('%') { json!([&text[1..text.len() - 1]]) } else { json!([]) };
    let ops = json!([{"op": "set", "path": ["entries", "battle.attack.hit", "text"], "value": text},
        {"op": "set", "path": ["entries", "battle.attack.hit", "variables"], "value": vars}]);
    write_mutated(&dir.join("pack.json"), &base, &ops, "");
    let catalog = localization::load(&dir.join("pack.json").to_string_lossy()).unwrap();
    let got = catalog.format("battle.attack.hit", &args);
    let label = format!("format_spec {text} {}", input["args"]);
    check_error(&label, &want["error"], &ctx.errv(got.as_ref().err().map(|e| e.to_string())));
    assert_eq!(got.unwrap_or_default(), want["text"].as_str().unwrap(), "{label}");
    true
}

#[test]
fn golden_present_loc() {
    let _ = (KNOWN_DIFFERENCES, NORMALIZATIONS);
    let Some(path) = std::env::var_os("FD2_GOLDEN_PRESENT_LOC").map(PathBuf::from).filter(|p| p.exists()) else {
        assert!(!require_golden(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_PRESENT_LOC");
        eprintln!("略過：沒有 FD2_GOLDEN_PRESENT_LOC");
        return;
    };
    let Some(assets) = std::env::var("FD2_GOLDEN_ASSETS").ok().filter(|a| Path::new(a).is_dir()) else {
        assert!(!require_golden(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_ASSETS");
        eprintln!("略過：沒有 FD2_GOLDEN_ASSETS");
        return;
    };
    let golden: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    assert_eq!(golden["schema"], "fd2-golden-present-loc-v1");
    for f in golden["files"].as_array().unwrap() {
        let raw = std::fs::read(Path::new(&assets).join(f["path"].as_str().unwrap()))
            .unwrap_or_else(|e| panic!("讀不到真實輸入 {}: {e}", f["path"]));
        assert_eq!(raw.len() as u64, f["size"].as_u64().unwrap(), "{} 大小", f["path"]);
        assert_eq!(digest(&raw), f["sha256"].as_str().unwrap(), "{} SHA-256", f["path"]);
    }
    let tmp = std::env::temp_dir().join(format!("fd2-golden-present-loc-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&tmp);
    std::fs::create_dir_all(&tmp).unwrap();
    let ctx = Ctx { assets: assets.clone(), tmp_str: tmp.to_string_lossy().into_owned(), tmp };
    let entities_base = read_tree(format!("{assets}/locales/en/entities.json"));
    let music_base = read_tree(format!("{assets}/music_catalog.json"));
    let mut catalogs = std::collections::BTreeMap::new();
    let mut counts = std::collections::BTreeMap::<String, usize>::new();
    let mut skipped = 0usize;
    for case in golden["cases"].as_array().unwrap() {
        let (op, input, want) = (case["op"].as_str().unwrap(), &case["input"], &case["result"]);
        let index = case["index"].as_i64().unwrap();
        let mut compared = true;
        match op {
            "load_official" => run_load_official(&ctx, input, want),
            "format" => run_format(&ctx, &mut catalogs, input, want),
            "content" => run_content(&ctx, input, want),
            "entities" => run_entities(&ctx, input, want),
            "pack" => run_pack(&ctx, input, want),
            "placeholders" => run_placeholders(&ctx, index, input, want),
            "locale_pattern" => run_locale_pattern(&ctx, input, want),
            "content_mut" => run_content_mut(&ctx, input, want),
            "entities_mut" => run_entities_mut(&ctx, &entities_base, input, want),
            "music_load" => music_result(&ctx, musiccatalog::load(&assets), want, "music_load"),
            "music_mut" => run_music_mut(&ctx, &music_base, input, want),
            "music_ogg" => run_music_ogg(&ctx, &music_base, input, want),
            "format_spec" => compared = run_format_spec(&ctx, index, input, want, &mut skipped),
            other => panic!("未知 op {other}"),
        }
        if compared {
            *counts.entry(op.to_string()).or_default() += 1;
        }
    }
    let _ = std::fs::remove_dir_all(&ctx.tmp);
    let total: usize = counts.values().sum();
    eprintln!("present-loc 黃金：比對 {total} 案例 {counts:?}；Format 不支援的組合跳過 {skipped}");
    assert!(counts.get("format_spec").copied().unwrap_or(0) > 300, "format_spec 比對太少");
}
