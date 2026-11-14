//! 與凍結 Go 版 `remake/cmd/fd2-golden-campaign-b` 的黃金差分（campaign B 組：設施畫面與城鎮／
//! 商店／教會規則；見 `PORTING.md`）。
//!
//! 由 `tools/rust_golden_campaign_b.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_CAMPAIGN_B`。真實輸入：
//! `FD2_FLAME2_DIR`（原版 FLAME2）、`FD2_ASSET_PACK`（分離素材包）、`FD2_GOLDEN_REPO`（底下的
//! `remake/assets` 與 `docs`）。先核對黃金檔記錄的檔案大小與 SHA-256、素材包目錄的清單雜湊，再逐案
//! 以同一組參數重做並比對：畫面案例比每格畫面的 SHA-256，規則案例比結果與單位（Go 欄位名的 JSON）。
//! 設定 `FD2_REQUIRE_GOLDEN=1` 時缺檔即失敗。錯誤文字逐字比對。
//!
//! 比對時的表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。

use fd2_campaign::campaign::*;
use fd2_formats::{dato, fdicon, fdother, fdtxt};
use fd2_rules::battle::{self, GoMathRand, Unit};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為。黃金案例除了第一項以外都不落在這些輸入上。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go `church.go:257` 把 roster 裡不在 order 的 id 依 map 迭代順序附加，順序不固定；Rust 依遞增
    // 順序。order 不齊全的案例（黃金的 full=false）改比集合，齊全的照順序比。
    "ClassChangeCandidates 不在 order 的 id 的附加順序",
    // Go 讀 JSON 時欄位名不分大小寫；Rust 只認標籤原文（成長表另認 Go 欄位名原文 AP 等）。
    "JSON 欄位名大小寫",
    // 損壞 JSON／PNG 與作業系統錯誤的文字屬 Go 標準函式庫，只比接受或拒收（rust-port-no-go-stdlib）。
    "標準函式庫錯誤文字",
    // Go 的 nil 切片元素與 nil 接收者（`[]*battle.Unit` 的 nil、nil `*Strings`）：Rust 切片元素不會是
    // nil；需要 nil 的參數包成 Option。黃金只對 Option 參數傳 nil。
    "nil 切片元素",
];

/// 比對前的表示法正規化（不影響行為）。
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil。
    "null 與空陣列、空字串相同",
    // Go 把整數值的 float64 寫成 0，serde_json 寫成 0.0；只比數值。
    "數字只比數值",
    // Go 失敗時另回傳的值（gold、slot、cost、零值結構）由 Rust 端依 Go 的回傳規則補上再比。
    "失敗時的附帶回傳值依 Go 規則重建",
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

fn pair(v: &Value) -> [i64; 2] {
    let a = ints(v);
    [a[0], a[1]]
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

fn err_of<T>(r: &Result<T>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => json!(e.message()),
    }
}

/// Go `frameResult`：錯誤時 `frames` 為空。
fn frames_val(r: Result<Vec<Vec<u8>>>) -> Value {
    match r {
        Ok(frames) => json!({"err": null, "frames": frames.iter().map(|f| digest(f)).collect::<Vec<_>>()}),
        Err(e) => json!({"err": e.message(), "frames": []}),
    }
}

fn one(r: Result<Vec<u8>>) -> Value {
    frames_val(r.map(|f| vec![f]))
}

/// 和 Go 同一組真實輸入。
struct Ctx {
    flame: PathBuf,
    pack: PathBuf,
    repo: PathBuf,
    fdother_path: PathBuf,
    bg14: Vec<u8>,
    entries14: Vec<fdother::LMI1Entry>,
    dialogue: Vec<fdother::RawCell>,
    digits: Vec<fdother::Frame>,
    strings: fdtxt::Strings,
    font: fdtxt::Font,
    bank: fdicon::Bank,
    cells2: Vec<fdother::RawCell>,
    load_box: fdother::LMI1Entry,
    revive: Vec<fdother::Frame>,
    item_arch: battle::NativeItemPanelDataAssets,
    item_pack: battle::NativeItemPanelDataAssets,
    effect_rows: Vec<u8>,
    sys_panels: [fdother::Frame; 4],
    shops: BTreeMap<String, NativeShopAssets>,
    towns: BTreeMap<String, NativeTownAssets>,
    dato: std::cell::RefCell<BTreeMap<i64, dato::Frame>>,
    stats: BTreeMap<i64, ItemStats>,
    class_table: ClassChangeTable,
}

impl Ctx {
    fn repo_path(&self, rel: &str) -> String {
        self.repo.join(rel).to_string_lossy().into_owned()
    }

    fn portrait(&self, id: i64) -> dato::Frame {
        self.dato
            .borrow_mut()
            .entry(id)
            .or_insert_with(|| dato::decode_resource(self.flame.join("DATO.DAT"), id).unwrap()[0].clone())
            .clone()
    }

    fn sprite(&self, group: i64, pose: i64, cycle: i64) -> fdicon::Sprite {
        self.bank.sprite_for(group, pose, cycle).unwrap().clone()
    }

    fn shop(&self, key: &str) -> &NativeShopAssets {
        &self.shops[key]
    }

    fn load(flame: PathBuf, pack: PathBuf, repo: PathBuf) -> Ctx {
        let fdother_path = flame.join("FDOTHER.DAT");
        let fdtxt_path = flame.join("FDTXT.DAT");
        let res14 = fdother::read_resource(&fdother_path, 14).unwrap();
        let mut entries14 = vec![fdother::LMI1Entry::default(); 17];
        for index in [1usize, 16] {
            entries14[index] = fdother::parse_lmi1_opaque_entry(&res14, index as i64).unwrap();
        }
        for (index, entry) in entries14.iter_mut().enumerate().take(11).skip(3) {
            let cell = fdother::parse_lmi1_raw_entry(&res14, index as i64).unwrap();
            *entry = fdother::LMI1Entry { width: cell.width, height: cell.height, pixels: cell.pixels };
        }
        let mut bg14 = vec![0u8; 320 * 200];
        fdother::parse_lmi1_frame_entry(&res14, 0).unwrap().blit_at(&mut bg14, 320, 0, -1).unwrap();
        let revive = (0..9).map(|k| fdother::parse_lmi1_frame_entry(&res14, 23 + k).unwrap()).collect();
        let res5 = fdother::read_resource(&fdother_path, 5).unwrap();
        let mut dialogue = vec![fdother::RawCell::default(); 18];
        for (index, cell) in dialogue.iter_mut().enumerate().skip(1) {
            *cell = fdother::parse_lmi1_raw_entry(&res5, index as i64).unwrap();
        }
        let digits = (0..10).map(|d| fdother::parse_lmi1_frame_entry(&res5, 31 + d).unwrap()).collect();
        let strings = fdtxt::parse(&fdother::read_resource(&fdtxt_path, 0).unwrap()).unwrap();
        let font = fdtxt::parse_font(&fdother::read_resource(&fdother_path, 4).unwrap()).unwrap();
        let bank = fdicon::decode_file(flame.join("FDICON.B24")).unwrap();
        let cells2 = fdother::decode_raw_cell_resource(&fdother_path, 2).unwrap();
        let entries13 = fdother::parse_lmi1(&fdother::read_resource(&fdother_path, 13).unwrap()).unwrap();
        let (fo, ft) = (fdother_path.to_string_lossy().into_owned(), fdtxt_path.to_string_lossy().into_owned());
        let pack_s = pack.to_string_lossy().into_owned();
        let item_arch = battle::load_native_item_panel_data_assets_archive(&fo, &ft).unwrap();
        let item_pack = battle::load_native_item_panel_data_assets(&pack_s).unwrap();
        let effect_rows = battle::load_native_item_effect_row_prefix(
            &repo.join("remake/assets/data/native_item_effect_rows.json").to_string_lossy(),
        )
        .unwrap();
        let sys_panels = fdother::load_separated_system_info_panels(pack.join("ui")).unwrap();
        let mut shops = BTreeMap::new();
        for r in [12, 13, 29, 63] {
            shops.insert(format!("pack{r}"), load_separated_native_shop_assets(&pack_s, r).unwrap());
            shops.insert(format!("archive{r}"), decode_native_shop_assets(&fo, r).unwrap());
        }
        let towns = BTreeMap::from([
            ("pack".to_string(), load_native_town_assets(&pack_s).unwrap()),
            (
                "archive".to_string(),
                decode_native_town_assets_archive(&fo, &flame.join("FDICON.B24").to_string_lossy()).unwrap(),
            ),
        ]);
        let stats = load_item_stats(&repo.join("remake/assets/data/item.json").to_string_lossy()).unwrap();
        let class_table =
            load_class_change_table(&repo.join("docs/data/exe_tables/class_change_targets.json").to_string_lossy())
                .unwrap();
        Ctx {
            flame,
            pack,
            repo,
            fdother_path,
            bg14,
            entries14,
            dialogue,
            digits,
            strings,
            font,
            bank,
            cells2,
            load_box: entries13[16].clone(),
            revive,
            item_arch,
            item_pack,
            effect_rows,
            sys_panels,
            shops,
            towns,
            dato: Default::default(),
            stats,
            class_table,
        }
    }
}

/// Go `dirRow`：排序後每列「相對路徑\t大小\tSHA-256\n」的雜湊。
fn dir_manifest(root: &Path) -> (usize, String) {
    fn walk(base: &Path, dir: &Path, out: &mut Vec<String>) {
        for entry in std::fs::read_dir(dir).unwrap() {
            let path = entry.unwrap().path();
            if path.is_dir() {
                walk(base, &path, out);
            } else {
                let raw = std::fs::read(&path).unwrap();
                let rel = path.strip_prefix(base).unwrap().to_string_lossy().replace('\\', "/");
                out.push(format!("{rel}\t{}\t{}\n", raw.len(), digest(&raw)));
            }
        }
    }
    let mut lines = Vec::new();
    walk(root, root, &mut lines);
    lines.sort();
    (lines.len(), digest(lines.concat().as_bytes()))
}

fn check_real(ctx: &Ctx, real: &Value) {
    for file in real["files"].as_array().unwrap() {
        let logical = s(file, "path");
        let path = match logical.strip_prefix("flame2/") {
            Some(name) => ctx.flame.join(name),
            None => ctx.repo.join(logical),
        };
        let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀不到 {logical}：{e}"));
        assert_eq!(raw.len() as i64, i(file, "size"), "{logical} 大小");
        assert_eq!(digest(&raw), s(file, "sha256"), "{logical} SHA-256");
    }
    for dir in real["pack_dirs"].as_array().unwrap() {
        let rel = s(dir, "dir");
        let (count, sha) = dir_manifest(&ctx.pack.join(rel));
        assert_eq!(count as i64, i(dir, "files"), "素材包 {rel} 檔案數");
        assert_eq!(sha, s(dir, "manifest_sha256"), "素材包 {rel} 清單雜湊");
    }
}

// ---- 素材載入 ----

fn cell_tag(w: i64, h: i64, pixels: &[u8]) -> String {
    format!("{w}x{h}:{}", digest(pixels))
}

fn shop_digest(a: &NativeShopAssets) -> Value {
    let service: Vec<String> =
        a.service_cells.iter().flatten().map(|c| cell_tag(c.width, c.height, &c.pixels)).collect();
    let compare: Vec<String> = a.compare_cells.iter().map(|c| cell_tag(c.width, c.height, &c.pixels)).collect();
    let success: Vec<String> = a
        .success_frames
        .iter()
        .map(|f| {
            let (indexed, mask) = f.indexed_layers().unwrap();
            format!("{}x{}:{}:{}", f.width, f.height, digest(&indexed), digest(&mask))
        })
        .collect();
    json!({
        "resource": a.resource_id,
        "background": digest(&a.background),
        "decoration": cell_tag(a.decoration.width, a.decoration.height, &a.decoration.pixels),
        "gold": cell_tag(a.gold_roll_strip.width, a.gold_roll_strip.height, &a.gold_roll_strip.pixels),
        "service": service,
        "price": cell_tag(a.price_cell.width, a.price_cell.height, &a.price_cell.pixels),
        "panel": cell_tag(a.panel.width, a.panel.height, &a.panel.pixels),
        "compare": compare,
        "success": success,
        "raw": a.raw_entries.iter().map(|r| digest(r)).collect::<Vec<_>>(),
    })
}

fn town_digest(a: &NativeTownAssets) -> Value {
    json!({
        "backgrounds": a.backgrounds.iter().map(|b| digest(b)).collect::<Vec<_>>(),
        "label": cell_tag(a.label.width, a.label.height, &a.label.pixels),
        "pulse": a.pulse.iter().map(|p| format!("{}:{}:{}", digest(&p.pixels), digest(&p.mask), digest(&p.remap_mask))).collect::<Vec<_>>(),
    })
}

// ---- 規則結果的 Go JSON 形狀 ----

fn branch_val(b: &ClassChangeBranch) -> Value {
    json!({
        "Branch": b.branch, "Portrait": b.portrait, "ClassID": b.class_id,
        "MobilityIncrement": b.mobility_increment, "RequiredItemID": b.required_item_id,
        "InventoryIndex": b.inventory_index,
    })
}

fn growth_val(g: &ClassChangeGrowth) -> Value {
    json!({"AP": g.ap, "DP": g.dp, "DX": g.dx, "HP": g.hp, "MP": g.mp})
}

fn growth_from(v: &Value) -> ClassChangeGrowth {
    ClassChangeGrowth {
        ap: pair(&v["AP"]),
        dp: pair(&v["DP"]),
        dx: pair(&v["DX"]),
        hp: pair(&v["HP"]),
        mp: pair(&v["MP"]),
    }
}

/// Go `ItemStats` 的 JSON：`mv`、`range_min`、`range_max` 是 omitempty。
fn item_stats_val(st: &ItemStats) -> Value {
    let mut m = serde_json::Map::new();
    m.insert("type".into(), json!(st.r#type));
    m.insert("ap".into(), json!(st.ap));
    m.insert("hit".into(), json!(st.hit));
    m.insert("dp".into(), json!(st.dp));
    m.insert("ev".into(), json!(st.ev));
    for (k, v) in [("mv", st.mv), ("range_min", st.min), ("range_max", st.max)] {
        if v != 0 {
            m.insert(k.into(), json!(v));
        }
    }
    Value::Object(m)
}

fn plan_val(p: &NativeShopPurchaseSuccessPlan) -> Value {
    json!({
        "HubVariant": p.hub_variant, "ResourceID": p.resource_id, "FrameCount": p.frame_count,
        "EffectOffset": p.effect_offset, "PreDelayBIOSTicks": p.pre_delay_bios_ticks,
        "PerFrameDelayBIOSTicks": p.per_frame_delay_bios_ticks, "PostDelayBIOSTicks": p.post_delay_bios_ticks,
        "RestorePortraitMode0": p.restore_portrait_mode0, "OptionalEquipBefore": p.optional_equip_before,
        "DebitAfterPresentation": p.debit_after_presentation, "ReturnToProductLoop": p.return_to_product_loop,
    })
}

fn int_map<V>(m: &BTreeMap<i64, V>, f: impl Fn(&V) -> Value) -> Value {
    Value::Array(m.iter().map(|(k, v)| json!([k, f(v)])).collect())
}

/// Go 的 `(gold, err)`：失敗時 gold 是傳入值。
fn gold_val(input: i64, r: Result<i64>) -> Value {
    json!({"gold": *r.as_ref().unwrap_or(&input), "err": err_of(&r)})
}

// ---- 案例 ----

fn run_case(ctx: &Ctx, op: &str, inp: &Value, want: &Value, sets: &mut usize) -> Value {
    let (st, fo) = (Some(&ctx.strings), Some(&ctx.font));
    match op {
        "shop_assets" => shop_digest(ctx.shop(s(inp, "key"))),
        "town_assets" => town_digest(&ctx.towns[s(inp, "key")]),
        "shop_assets_reject" => {
            json!({"err": err_of(&load_separated_native_shop_assets(&ctx.pack.to_string_lossy(), i(inp, "resource")))})
        }
        "shop_archive_reject" => {
            json!({"err": err_of(&decode_native_shop_assets(&ctx.fdother_path.to_string_lossy(), i(inp, "resource")))})
        }
        "church_scene" => one(compose_native_church_scene(
            &ctx.bg14,
            &ctx.entries14[1],
            &ctx.dialogue,
            &ctx.digits,
            &ctx.portrait(131),
            &ctx.strings,
            &ctx.font,
            i(inp, "gold"),
            i(inp, "text"),
        )),
        "church_scene_base" => one(compose_native_church_scene_base(
            &ctx.bg14,
            &ctx.entries14[1],
            &ctx.dialogue,
            &ctx.digits,
            &ctx.portrait(131),
            i(inp, "gold"),
        )),
        "church_menu_transition" => {
            frames_val(native_church_menu_transition_frames(&ctx.bg14, &ctx.entries14, b(inp, "opening")))
        }
        "church_menu_frame" => {
            one(compose_native_church_menu_frame(&ctx.bg14, &ctx.entries14, i(inp, "selected"), i(inp, "pulse")))
        }
        "church_revive_success" => {
            frames_val(compose_native_church_revive_success_frames(&ctx.bg14, &ctx.revive, &ctx.portrait(131)).map(
                |(mut a, last)| {
                    a.push(last);
                    a
                },
            ))
        }
        "church_text_name" => {
            let mut frame = ctx.bg14.clone();
            let r = compose_native_church_text_with_name_at(
                &mut frame,
                &ctx.strings,
                &ctx.font,
                i(inp, "text"),
                i(inp, "name"),
                i(inp, "offset"),
            );
            json!({"result": one(r), "frame": digest(&frame)})
        }
        "revive_confirm" => {
            one(compose_native_revive_confirmation_question(&ctx.bg14, st, fo, i(inp, "name"), i(inp, "fee")))
        }
        "transient_expiry" => one(compose_native_transient_expiry_frame(
            &ctx.bg14,
            &ctx.dialogue,
            &ctx.portrait(131),
            st,
            fo,
            i(inp, "counter"),
        )),
        "death_reward_msg" => {
            let dialogue =
                compose_native_transient_expiry_dialogue(&ctx.bg14, &ctx.dialogue, &ctx.portrait(131)).unwrap();
            one(compose_native_death_reward_message_frame(&dialogue, st, fo, i(inp, "kind"), i(inp, "value")))
        }
        "class_list" => {
            let rows: Vec<NativeClassListRow> = inp["rows"]
                .as_array()
                .unwrap()
                .iter()
                .map(|r| {
                    let r = ints(r);
                    NativeClassListRow {
                        sprite: ctx.sprite(r[0], r[1], r[2]),
                        name_text_index: r[3],
                        current_class_text_id: r[4],
                        target_class_text_id: r[5],
                    }
                })
                .collect();
            let frame =
                compose_native_class_list_frame(&ctx.bg14, &ctx.entries14[16], &rows, i(inp, "selected"), st, fo);
            match frame {
                Ok(frame) => json!({
                    "frame": one(Ok(frame.clone())),
                    "opening": frames_val(native_class_list_opening_frames(&ctx.bg14, &frame)),
                    "closing": frames_val(native_class_list_closing_frames(&ctx.bg14, &frame)),
                }),
                Err(e) => json!({"frame": one(Err(e))}),
            }
        }
        "roster" => {
            let rows: Vec<NativeRosterRow> = inp["rows"]
                .as_array()
                .unwrap()
                .iter()
                .map(|r| {
                    let r = ints(r);
                    NativeRosterRow { sprite: ctx.sprite(r[0], r[1], r[2]), name_text_index: r[3] }
                })
                .collect();
            let selected = i(inp, "selected");
            if b(inp, "names") {
                one(compose_native_roster_frame(&ctx.bg14, &ctx.entries14[16], &rows, selected, st, fo))
            } else {
                one(compose_native_roster_frame_without_names(&ctx.bg14, &ctx.entries14[16], &rows, selected))
            }
        }
        "class_confirm" => one(compose_native_class_confirmation_frame(
            &ctx.bg14,
            &ctx.cells2,
            st,
            fo,
            i(inp, "name"),
            i(inp, "selected"),
            i(inp, "pulse"),
        )),
        "class_confirm_question" => match compose_native_class_confirmation_question(&ctx.bg14, st, fo, i(inp, "name"))
        {
            Ok(q) => json!({
                "question": one(Ok(q.clone())),
                "opening": frames_val(native_class_confirmation_opening_frames(&q, &ctx.cells2)),
                "closing": frames_val(native_class_confirmation_closing_frames(&q, &ctx.cells2)),
            }),
            Err(e) => json!({"question": one(Err(e))}),
        },
        "preparation" => preparation(ctx, inp),
        "preparation_layout" => {
            let r = native_preparation_portrait_layout(&ctx.portrait(0x4b));
            let (rect, mirror) = r.as_ref().map(|(r, m)| (*r, *m)).unwrap_or_default();
            json!({"err": err_of(&r), "rect": [rect.min.x, rect.min.y, rect.max.x, rect.max.y], "mirror": mirror})
        }
        "battle_question"
        | "battle_response"
        | "battle_end_response"
        | "treasure_item"
        | "death_abandon"
        | "death_cancel"
        | "treasure_gold" => battle_system(ctx, op, inp),
        "load_slots" => {
            let mut slots = [NativeLoadSlot::default(); 4];
            for (k, sl) in inp["slots"].as_array().unwrap().iter().enumerate() {
                let p = pair(sl);
                slots[k] = NativeLoadSlot { empty: p[0] == 1, chapter: p[1] as u8 };
            }
            one(compose_native_load_slots_frame(&ctx.bg14, &ctx.load_box, st, fo, slots, i(inp, "selection")))
        }
        "system_info" => {
            let assets = NativeSystemInfoAssets {
                panels: ctx.sys_panels.clone(),
                numbers: ctx.item_pack.clone(),
                font: ctx.item_pack.font.clone(),
            };
            let lines = ints(&inp["lines"]);
            let camps = ints(&inp["camps"]);
            let input = NativeSystemInfoInput {
                raw_chapter: i(inp, "chapter") as u8,
                raw_round: i(inp, "round") as u8,
                currency: i(inp, "currency") as u32,
                raw_camp_counts: [camps[0], camps[1], camps[2]],
                lines: [ctx.strings.words(lines[0]).unwrap(), ctx.strings.words(lines[1]).unwrap()],
            };
            one(compose_native_system_info_surface(&assets, &input))
        }
        op if op.starts_with("shop_") || op == "facility_portrait_offset" => shop_case(ctx, op, inp),
        "gold_roll" => {
            let shop = ctx.shop("pack12");
            let base = compose_native_shop_bare_scene(Some(shop), &ctx.digits, 0).unwrap();
            let (old, amount) = (i(inp, "old"), i(inp, "amount"));
            let r = if b(inp, "credit") {
                compose_native_gold_credit_frames(&base, &shop.gold_roll_strip, old, amount)
            } else {
                compose_native_gold_debit_frames(&base, &shop.gold_roll_strip, old, amount)
            };
            // Go 失敗時 next 是 oldGold。
            let next = r.as_ref().map(|(_, n)| *n).unwrap_or(old);
            let mut v = frames_val(r.map(|(f, _)| f));
            v["next"] = json!(next);
            v
        }
        "gold_plan" => {
            let r = plan_native_gold_roll(i(inp, "old"), i(inp, "next"), b(inp, "credit"));
            let rows: Vec<Value> =
                r.as_ref().map(|w| w.iter().map(|x| json!([x.rows, x.active])).collect()).unwrap_or_default();
            json!({"err": err_of(&r), "windows": rows})
        }
        "town_frame" => {
            let assets = &ctx.towns[s(inp, "town")];
            let (v, sel, p) = (i(inp, "variant"), i(inp, "selection"), i(inp, "pulse"));
            if b(inp, "cursor") {
                one(compose_native_town_frame(Some(assets), st, fo, v, sel, p))
            } else {
                one(compose_native_town_frame_without_cursor(Some(assets), st, fo, v, sel, p))
            }
        }
        "town_cursor" => {
            let r = native_town_cursor_geometry_for(i(inp, "variant"), i(inp, "selection"), i(inp, "pulse"));
            let g = r.clone().unwrap_or_default();
            json!({
                "err": err_of(&r), "position": [g.position.x, g.position.y], "frame": g.frame,
                "viewport": [g.viewport.min.x, g.viewport.min.y, g.viewport.max.x, g.viewport.max.y],
            })
        }
        "town_departure_geometry" => {
            let r = native_town_departure_geometry_for(i(inp, "variant"), i(inp, "selection"), i(inp, "step"));
            let g = r.clone().unwrap_or_default();
            json!({"err": err_of(&r), "center": [g.center.x, g.center.y], "spacing": g.spacing})
        }
        "town_departure" => {
            let town = compose_native_town_frame(Some(&ctx.towns["pack"]), st, fo, 1, 2, 0).unwrap();
            one(compose_native_town_departure_frame(&town, i(inp, "variant"), i(inp, "selection"), i(inp, "step")))
        }
        "selectors" => selectors(),
        op if op.starts_with("table_") || op == "can_equip" => tables(ctx, op, inp),
        "revive" => {
            let mut u = (!inp["unit"].is_null()).then(|| unit_from(&inp["unit"]));
            let (can_revive_v, can_change) = (can_revive(u.as_ref()), can_change_class(u.as_ref()));
            let (gold, cost, err) = revive_unit(i(inp, "gold"), u.as_mut(), i(inp, "fee"));
            json!({
                "gold": gold, "cost": cost, "err": err_of(&err), "unit": u.as_ref().map(unit_val),
                "can_revive": can_revive_v, "can_change_class": can_change,
            })
        }
        "class_change" => {
            let u0 = unit_from(&inp["unit"]);
            let (branch, ok) = native_class_change_target(Some(&u0), &ctx.class_table);
            let mut u = u0.clone();
            let mut rng = inp["seed"].as_i64().map(GoMathRand::new);
            let err = apply_class_change(
                Some(&mut u),
                i(inp, "target"),
                i(inp, "class"),
                i(inp, "group"),
                growth_from(&inp["row"]),
                rng.as_mut().map(|r| r as &mut dyn battle::GoRand),
                i(inp, "remove"),
            );
            let mut u2 = u.clone();
            recompute_after_class_change(Some(&mut u2), &ctx.stats);
            json!({"branch": branch_val(&branch), "ok": ok, "err": err_of(&err), "unit": unit_val(&u), "recomputed": unit_val(&u2)})
        }
        "candidates" => {
            let roster: BTreeMap<i64, Unit> =
                inp["roster"].as_array().unwrap().iter().map(|e| (i(e, "id"), unit_from(&e["unit"]))).collect();
            let got = class_change_candidates(&roster, &ints(&inp["order"]));
            if b(inp, "full") {
                json!(got)
            } else {
                // KNOWN_DIFFERENCES：order 不齊全時 Go 的附加順序不固定，改比「order 部分的前綴＋集合」。
                *sets += 1;
                let go = ints(want);
                let order: BTreeSet<i64> = ints(&inp["order"]).into_iter().collect();
                let prefix = |v: &[i64]| v.iter().take_while(|id| order.contains(id)).copied().collect::<Vec<_>>();
                assert_eq!(prefix(&go), prefix(&got), "candidates order 部分");
                let (gs, rs): (BTreeSet<i64>, BTreeSet<i64>) =
                    (go.iter().copied().collect(), got.iter().copied().collect());
                assert_eq!(gs, rs, "candidates 集合");
                want.clone()
            }
        }
        op if op.starts_with("unit_") => unit_op(ctx, &op[5..], inp),
        "camp_counts" => {
            let units: Vec<Unit> =
                inp["units"].as_array().map(|a| a.iter().map(unit_from).collect()).unwrap_or_default();
            let r = native_system_info_camp_counts(&units);
            json!({"counts": *r.as_ref().unwrap_or(&[0; 3]), "err": err_of(&r)})
        }
        "candidate_stats" => {
            let record = unhex(s(inp, "record"));
            let cand = native_shop_equipment_candidate_stats(&record, i(inp, "item"), &ctx.effect_rows);
            let cur = native_shop_equipment_current_stats(&record);
            json!({
                "candidate": *cand.as_ref().unwrap_or(&[0; 4]), "err": err_of(&cand),
                "current": *cur.as_ref().unwrap_or(&[0; 4]), "current_err": err_of(&cur),
            })
        }
        other => panic!("未知 op {other}"),
    }
}

fn preparation(ctx: &Ctx, inp: &Value) -> Value {
    let p = ctx.portrait(0x4b);
    let (st, fo) = (Some(&ctx.strings), Some(&ctx.font));
    match s(inp, "kind") {
        "question" => one(compose_native_preparation_confirmation_question(&ctx.bg14, &ctx.dialogue, &p, st, fo)),
        "departure" => one(compose_native_preparation_departure_question(&ctx.bg14, &ctx.dialogue, &p, st, fo)),
        "record" => one(compose_native_preparation_record_question(&ctx.bg14, &ctx.dialogue, &p, st, fo)),
        "required" => {
            one(compose_native_preparation_required_message(&ctx.bg14, &ctx.dialogue, &p, st, fo, i(inp, "identity")))
        }
        "frame" => one(compose_native_preparation_confirmation_frame(
            &ctx.bg14,
            &ctx.cells2,
            &ctx.dialogue,
            &p,
            st,
            fo,
            i(inp, "selected"),
            i(inp, "pulse"),
        )),
        kind => {
            let dialogue = compose_native_preparation_confirmation_dialogue(&ctx.bg14, &ctx.dialogue, &p).unwrap();
            let question =
                compose_native_preparation_confirmation_question(&ctx.bg14, &ctx.dialogue, &p, st, fo).unwrap();
            match kind {
                "opening" => frames_val(native_preparation_confirmation_opening_frames(
                    &ctx.bg14,
                    &dialogue,
                    &question,
                    &ctx.cells2,
                )),
                "closing" => frames_val(native_preparation_confirmation_closing_frames(
                    &ctx.bg14,
                    &dialogue,
                    &question,
                    &ctx.cells2,
                )),
                "portrait" => one(compose_native_confirmation_portrait(&question, &p)),
                other => panic!("未知 preparation {other}"),
            }
        }
    }
}

fn battle_system(ctx: &Ctx, op: &str, inp: &Value) -> Value {
    let p = ctx.portrait(0x4b);
    let (strings, font) = (&ctx.strings, &ctx.font);
    let (st, fo) = (Some(strings), Some(font));
    let base =
        compose_native_church_dialogue_overlay_at(&ctx.bg14, &ctx.dialogue, &p, native_facility_portrait_offset(0x4b))
            .unwrap();
    let question_of = |kind: &str| match kind {
        "end" => compose_native_battle_end_turn_question(&base, &p, strings, font),
        "exit" => compose_native_battle_exit_question(&base, &p, strings, font),
        "save" => compose_native_battle_current_save_question(&base, &p, strings, font),
        "load" => compose_native_battle_current_load_question(&base, &p, strings, font),
        "march" => compose_native_battle_group_march_question(&base, &p, strings, font),
        "discard" => compose_native_death_reward_discard_question(&base, &p, strings, font),
        "treasure" => compose_native_treasure_item_question(&base, &p, strings, font, false),
        "treasure_hidden" => compose_native_treasure_item_question(&base, &p, strings, font, true),
        other => panic!("未知問句 {other}"),
    };
    if op == "battle_question" {
        return one(question_of(s(inp, "kind")));
    }
    let question = question_of("end").unwrap();
    match op {
        "battle_response" => {
            let a = b(inp, "accepted");
            frames_val(match s(inp, "kind") {
                "end" => native_battle_end_turn_response_frames(&question, st, fo, a),
                "exit" => native_battle_exit_response_frames(&question, st, fo, a),
                "save" => native_battle_current_save_response_frames(&question, st, fo, a),
                "load" => native_battle_current_load_response_frames(&question, st, fo, a),
                "march" => native_battle_group_march_response_frames(&question, st, fo, a),
                other => panic!("未知回覆 {other}"),
            })
        }
        "battle_end_response" => one(compose_native_battle_end_turn_response(&question, st, fo, b(inp, "accepted"))),
        "treasure_item" => {
            frames_val(native_treasure_item_response_frames(&question, st, fo, i(inp, "item"), b(inp, "hidden")))
        }
        "death_abandon" => frames_val(native_death_reward_abandon_response_frames(&question, st, font, i(inp, "item"))),
        "death_cancel" => one(compose_native_death_reward_selector_cancel(&base, &p, st, font, i(inp, "item"))),
        "treasure_gold" => {
            frames_val(native_treasure_gold_response_frames(&question, st, fo, i(inp, "value"), b(inp, "hidden")))
        }
        other => panic!("未知 op {other}"),
    }
}

fn shop_case(ctx: &Ctx, op: &str, inp: &Value) -> Value {
    let (st, fo) = (Some(&ctx.strings), Some(&ctx.font));
    let resource_of = |variant: i64| match variant {
        1 => 12,
        3 => 29,
        _ => 63,
    };
    let portrait_of = |variant: i64| match variant {
        1 => 0x80,
        3 => 0x82,
        _ => 0x84,
    };
    // 依 variant 的商店、店員與穩定畫面（Go `genShop` 第二段的 source）。
    let variant_source = |variant: i64| {
        let shop = ctx.shop(&format!("pack{}", resource_of(variant)));
        let pid = portrait_of(variant);
        let p = ctx.portrait(pid);
        let source =
            compose_native_shop_scene(Some(shop), &ctx.dialogue, &ctx.digits, &p, pid, st, fo, 12345678, 0x1f5)
                .unwrap();
        (shop, pid, p, source)
    };
    match op {
        "facility_portrait_offset" => json!(native_facility_portrait_offset(i(inp, "id"))),
        "shop_scene" => {
            let shop = ctx.shop(&format!("{}{}", s(inp, "shop"), i(inp, "resource")));
            let pid = i(inp, "portrait");
            let p = ctx.portrait(pid);
            if b(inp, "text_layer") {
                one(compose_native_shop_scene(
                    Some(shop),
                    &ctx.dialogue,
                    &ctx.digits,
                    &p,
                    pid,
                    st,
                    fo,
                    i(inp, "gold"),
                    i(inp, "text"),
                ))
            } else {
                one(compose_native_shop_scene_without_text(
                    Some(shop),
                    &ctx.dialogue,
                    &ctx.digits,
                    &p,
                    pid,
                    i(inp, "gold"),
                ))
            }
        }
        "shop_bare" => {
            let shop = ctx.shop(&format!("{}{}", s(inp, "shop"), i(inp, "resource")));
            one(compose_native_shop_bare_scene(Some(shop), &ctx.digits, i(inp, "gold")))
        }
        "shop_service" => {
            let shop = ctx.shop(&format!("pack{}", i(inp, "resource")));
            let stable = compose_native_shop_bare_scene(Some(shop), &ctx.digits, 4321).unwrap();
            one(match s(inp, "kind") {
                "opening" => compose_native_shop_service_opening_frame(&stable, Some(shop), i(inp, "step")),
                "closing" => compose_native_shop_service_closing_frame(&stable, Some(shop), i(inp, "step")),
                _ => compose_native_shop_service_steady_frame(&stable, Some(shop), i(inp, "selected"), i(inp, "phase")),
            })
        }
        "shop_item_list" => {
            let shop = ctx.shop(&format!("pack{}", i(inp, "resource")));
            let stable = compose_native_shop_bare_scene(Some(shop), &ctx.digits, 4321).unwrap();
            let items = ints(&inp["items"]);
            let mode = battle::NativeFacilityPriceMode(i(inp, "mode") as u8);
            let f = if b(inp, "names") {
                compose_native_shop_item_list_frame
            } else {
                compose_native_shop_item_list_frame_without_names
            };
            one(f(
                &stable,
                Some(shop),
                &ctx.item_arch,
                &items,
                i(inp, "start"),
                i(inp, "selected"),
                &ctx.effect_rows,
                mode,
            ))
        }
        "shop_success_plan" => {
            let r = plan_native_shop_purchase_success(i(inp, "hub"));
            json!({"err": err_of(&r), "plan": plan_val(&r.clone().unwrap_or_default())})
        }
        "shop_text_index" => {
            let (index, ok) =
                native_shop_purchase_text_index(NativeShopPurchaseMessage(i(inp, "message") as u8), i(inp, "hub"));
            json!({"index": index, "ok": ok})
        }
        _ => {
            let variant = i(inp, "variant");
            let (shop, pid, p, source) = variant_source(variant);
            match op {
                "shop_purchase_msg" => one(compose_native_shop_purchase_message(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    st,
                    fo,
                    NativeShopPurchaseMessage(i(inp, "message") as u8),
                    i(inp, "hub"),
                    i(inp, "item"),
                    i(inp, "price"),
                )),
                "shop_purchase_confirm" => one(compose_native_shop_purchase_confirmation(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    st,
                    fo,
                    &ctx.cells2,
                    NativeShopPurchaseMessage(i(inp, "message") as u8),
                    variant,
                    3,
                    120,
                    i(inp, "selected"),
                    i(inp, "pulse"),
                )),
                "shop_insufficient" => {
                    let question = compose_native_shop_purchase_message(
                        &source,
                        &ctx.dialogue,
                        &p,
                        pid,
                        st,
                        fo,
                        NATIVE_SHOP_PURCHASE_QUESTION,
                        variant,
                        0,
                        50,
                    )
                    .unwrap();
                    let closing = native_class_confirmation_closing_frames(&question, &ctx.cells2).unwrap();
                    let mut post = closing[closing.len() - 1].clone();
                    let r = compose_native_shop_purchase_insufficient_gold(&mut post, st, fo, i(inp, "hub"));
                    json!({"result": one(r), "post": digest(&post)})
                }
                "shop_recipient_full" => one(compose_native_shop_purchase_recipient_full(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    st,
                    fo,
                    i(inp, "hub"),
                    i(inp, "name"),
                )),
                "shop_sell_question" => one(compose_native_shop_sell_question(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    st,
                    fo,
                    &ctx.cells2,
                    i(inp, "hub"),
                    0x81,
                    75,
                    1,
                    0,
                )),
                "shop_sell_base" => one(compose_native_shop_sell_question_base(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    st,
                    fo,
                    i(inp, "hub"),
                    0x81,
                    75,
                )),
                "shop_sell_empty" => one(compose_native_shop_sell_empty(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    &ctx.strings,
                    &ctx.font,
                    i(inp, "hub"),
                    4,
                )),
                "shop_transfer" => one(compose_native_shop_transfer_message(
                    &source,
                    &ctx.dialogue,
                    &p,
                    pid,
                    st,
                    fo,
                    NativeShopTransferMessage(i(inp, "message") as u8),
                    i(inp, "name"),
                )),
                "shop_success" => {
                    let bare = compose_native_shop_bare_scene(Some(shop), &ctx.digits, 1000).unwrap();
                    frames_val(
                        compose_native_shop_purchase_success_frames(&bare, Some(shop), &p, pid, i(inp, "hub")).map(
                            |(mut a, last)| {
                                a.push(last);
                                a
                            },
                        ),
                    )
                }
                "shop_consumable_recipient" => {
                    let rows: Vec<NativeRosterRow> = inp["rows"]
                        .as_array()
                        .unwrap()
                        .iter()
                        .map(|r| {
                            let r = ints(r);
                            NativeRosterRow { sprite: ctx.sprite(r[0], r[1], r[2]), name_text_index: r[3] }
                        })
                        .collect();
                    one(compose_native_shop_consumable_recipient_frame(
                        &source,
                        Some(shop),
                        &rows,
                        0,
                        i(inp, "item_type"),
                        st,
                        fo,
                    ))
                }
                "shop_equipment_recipient" => {
                    let rows: Vec<NativeShopEquipmentRecipientRow> = inp["rows"]
                        .as_array()
                        .unwrap()
                        .iter()
                        .map(|r| {
                            let (c, d) = (ints(&r["current"]), ints(&r["candidate"]));
                            NativeShopEquipmentRecipientRow {
                                sprite: ctx.sprite(i(r, "group"), 0, 0),
                                name_text_index: i(r, "name"),
                                current: [c[0], c[1], c[2], c[3]],
                                candidate: [d[0], d[1], d[2], d[3]],
                            }
                        })
                        .collect();
                    let f = if b(inp, "names") {
                        compose_native_shop_equipment_recipient_frame
                    } else {
                        compose_native_shop_equipment_recipient_frame_without_names
                    };
                    one(f(&source, Some(shop), &ctx.item_arch, &rows, i(inp, "selected")))
                }
                other => panic!("未知 op {other}"),
            }
        }
    }
}

fn selectors() -> Value {
    let (mut church, mut confirm, mut candidate, mut three, mut two_sel, mut two_win) =
        (vec![], vec![], vec![], vec![], vec![], vec![]);
    for current in -2..=5 {
        for delta in -2..=2 {
            church.push(json!([current, delta, advance_native_church_service_selection(current, delta)]));
            confirm.push(json!([current, delta, advance_native_class_confirmation(current, delta)]));
        }
    }
    for count in -1..=8 {
        for selected in -1..=9 {
            let (s0, v0) = native_class_candidate_window(count, selected);
            candidate.push(json!([count, selected, s0, v0]));
            for start in -1..=9 {
                let (s1, v1) = native_three_row_window(count, selected, start);
                three.push(json!([count, selected, start, s1, v1]));
            }
            for delta in -3..=3 {
                two_sel.push(json!([
                    selected,
                    count,
                    delta,
                    advance_native_two_column_selection(selected, count, delta)
                ]));
            }
        }
    }
    for count in -1..=13 {
        for selected in -1..=13 {
            for start in -1..=10 {
                let (s2, v2) = native_two_column_window(count, selected, start);
                two_win.push(json!([count, selected, start, s2, v2]));
            }
        }
    }
    let names: Vec<String> = (-2..=31).map(class_name).collect();
    json!({
        "church": church, "confirm": confirm, "candidate": candidate, "three": three,
        "two_selection": two_sel, "two_window": two_win, "class_names": names,
    })
}

fn tables(ctx: &Ctx, op: &str, inp: &Value) -> Value {
    let item = ctx.repo_path("remake/assets/data/item.json");
    let equip_path = ctx.repo_path("remake/assets/data/class_equip_types.json");
    match op {
        "table_revive_fee" => {
            let r = load_revive_fee_rates(&ctx.repo_path(s(inp, "path")));
            json!({"err": err_of(&r), "rates": r.ok()})
        }
        "table_class_change" => {
            let r = load_class_change_table(&ctx.repo_path(s(inp, "path")));
            let t = r.clone().unwrap_or_default();
            json!({
                "err": err_of(&r),
                "current": int_map(&t.current, |v| serde_json::to_value(v).unwrap()),
                "targets": int_map(&t.targets, |v| serde_json::to_value(v).unwrap()),
            })
        }
        "table_growth" => {
            let r = load_class_change_growth(&ctx.repo_path(s(inp, "path")));
            json!({"err": err_of(&r), "rows": int_map(&r.clone().unwrap_or_default(), growth_val)})
        }
        "table_item_stats" => {
            let r = load_item_stats(&item);
            json!({"err": err_of(&r), "rows": int_map(&r.clone().unwrap_or_default(), item_stats_val)})
        }
        "table_item_prices" => {
            let r = load_item_prices(&item);
            json!({"err": err_of(&r), "rows": int_map(&r.clone().unwrap_or_default(), |v| json!(v))})
        }
        "table_shop_eligibility" => {
            let r = load_shop_eligibility(&item, &equip_path);
            let (types, equip) = r.clone().unwrap_or_default();
            json!({"err": err_of(&r), "types": int_map(&types, |v| json!(v)), "equip": int_map(&equip, |v| json!(v))})
        }
        "can_equip" => {
            let (_, equip) = load_shop_eligibility(&item, &equip_path).unwrap();
            let mut out = vec![];
            for class in -1..=30 {
                for item_type in 0..=33 {
                    out.push(json!([class, item_type, can_equip(class, item_type, &equip)]));
                }
            }
            json!(out)
        }
        other => panic!("未知 op {other}"),
    }
}

fn unit_op(ctx: &Ctx, name: &str, inp: &Value) -> Value {
    let mut u = unit_from(&inp["unit"]);
    let mut out = match name {
        "buy" => {
            let good = Good { id: i(&inp["good"], "id"), name: "x".into(), price: i(&inp["good"], "price") };
            let gold = i(inp, "gold");
            gold_val(gold, buy_good(gold, Some(&mut u), &good))
        }
        "reserve" => {
            let good = Good { id: i(&inp["good"], "id"), name: "x".into(), price: i(&inp["good"], "price") };
            let gold = i(inp, "gold");
            let r = reserve_good(gold, Some(&mut u), &good);
            json!({"slot": *r.as_ref().unwrap_or(&-1), "err": err_of(&r), "finalized": finalize_good(gold, &good)})
        }
        "sell_good" => {
            let gold = i(inp, "gold");
            gold_val(gold, sell_good(gold, Some(&mut u), i(inp, "item"), i(inp, "price")))
        }
        "sell_slot" => {
            let gold = i(inp, "gold");
            gold_val(gold, sell_slot(gold, Some(&mut u), i(inp, "slot"), i(inp, "price")))
        }
        "sell_native" => {
            let gold = i(inp, "gold");
            gold_val(gold, sell_native_slot(gold, Some(&mut u), i(inp, "slot"), i(inp, "price"), &ctx.stats))
        }
        "equip" => json!({"err": err_of(&equip_item(Some(&mut u), i(inp, "slot"), &ctx.stats))}),
        "equip_native" => json!({"err": err_of(&equip_native_compact_slot(Some(&mut u), i(inp, "slot"), &ctx.stats))}),
        "recompute" => {
            recompute_equipment(Some(&mut u), &ctx.stats);
            json!({})
        }
        "attack_range" => {
            apply_equipped_attack_range(Some(&mut u), &ctx.stats);
            json!({})
        }
        "after_class_change" => {
            recompute_after_class_change(Some(&mut u), &ctx.stats);
            json!({})
        }
        "init_base" => {
            initialize_equipment_base(Some(&mut u), &ctx.stats);
            json!({})
        }
        "equipment_record" => {
            let r = native_shop_equipment_record_for_unit(Some(&u));
            let mut v = json!({"err": err_of(&r), "record": r.as_ref().map(|x| hex(x)).unwrap_or_default()});
            if let Ok(record) = &r {
                let cand = native_shop_equipment_candidate_stats(record, i(inp, "item"), &ctx.effect_rows);
                let cur = native_shop_equipment_current_stats(record);
                v["candidate"] = json!(*cand.as_ref().unwrap_or(&[0; 4]));
                v["candidate_err"] = err_of(&cand);
                v["current"] = json!(*cur.as_ref().unwrap_or(&[0; 4]));
                v["current_err"] = err_of(&cur);
            }
            v
        }
        other => panic!("未知單位 op {other}"),
    };
    out["unit"] = unit_val(&u);
    out
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_CAMPAIGN_B").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let flame = std::env::var_os("FD2_FLAME2_DIR").map(PathBuf::from).filter(|p| p.join("FDOTHER.DAT").exists());
    let pack = std::env::var_os("FD2_ASSET_PACK").map(PathBuf::from).filter(|p| p.join("shop").is_dir());
    let (Some(path), Some(repo), Some(flame), Some(pack)) = (path, repo, flame, pack) else {
        assert!(
            !require,
            "FD2_REQUIRE_GOLDEN=1 但缺 FD2_GOLDEN_CAMPAIGN_B／FD2_GOLDEN_REPO／FD2_FLAME2_DIR／FD2_ASSET_PACK"
        );
        eprintln!("略過：沒有 FD2_GOLDEN_CAMPAIGN_B／FD2_GOLDEN_REPO／FD2_FLAME2_DIR／FD2_ASSET_PACK");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    Some((doc, Ctx::load(flame, pack, repo)))
}

#[test]
fn golden_campaign_b() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-campaign-b/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    check_real(&ctx, &doc["real"]);
    let mut counts = BTreeMap::<String, usize>::new();
    let (mut frames, mut errs, mut sets) = (0usize, 0usize, 0usize);
    for case in doc["cases"].as_array().unwrap() {
        let op = s(case, "op");
        let (inp, want) = (&case["input"], &case["result"]);
        let got = run_case(&ctx, op, inp, want, &mut sets);
        if let Some(d) = diff("", want, &got) {
            panic!("case {} {op} {inp}：{d}", case["index"]);
        }
        *counts.entry(op.to_string()).or_default() += 1;
        let text = want.to_string();
        frames += text.matches("\"frames\":[\"").count();
        errs += usize::from(want.get("err").is_some_and(|e| !e.is_null()));
    }
    let total: usize = counts.values().sum();
    eprintln!(
        "campaign-b 黃金：{total} 案例全部相同（含畫面的案例 {frames}、Go 回錯誤的案例 {errs}、改比集合 {sets}）"
    );
    eprintln!("各 op 案例數：{counts:?}");
    assert_eq!(total, doc["cases"].as_array().unwrap().len());
}
