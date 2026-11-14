//! 與凍結 Go 版 `remake/cmd/fd2-golden-campaign-c` 的黃金差分：fd2-campaign C 組（存檔、CONTINUE、JOIN
//! 與結局演出）。
//!
//! 由 `tools/rust_golden_campaign_c.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_CAMPAIGN_C`；真實輸入掛在
//! `FD2_GOLDEN_REPO`（受版控的 `remake/assets` 與 `docs/data`）、`FD2_GOLDEN_FLAME2`（原版 FLAME2）與
//! `FD2_GOLDEN_PACK`（分離素材包）底下，先逐檔核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依 `op` 在 Rust 端重做同一串呼叫，結果與 Go 逐欄比對；Go 結構以 Go 的 JSON（有標籤用
//! 標籤，沒有標籤用 Go 欄位名）比對，畫面比 320×200 VGA 與 768 位元組調色盤的 SHA-256。錯誤文字逐字相同。
//! 比對時的表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。

use fd2_campaign::campaign as c;
use fd2_campaign::ending as e;
use fd2_formats::{dato, fdother, fdtxt, save};
use fd2_rules::battle;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

/// 刻意不照做 Go 的行為（`rust-port-no-go-stdlib`、PORTING.md）。黃金案例不依賴這些差異。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文。
    "JSON 欄位名大小寫",
    // Go 讀 null 到整數、字串欄位時保留零值；Rust 只在切片與 map 接受 null。
    "純量欄位的 null",
    // Go `ValidateNativeContinueBattleHandoff` 另檢查 `Roster == nil || PendingGroups == nil`；Rust 的 Vec／BTreeMap
    // 不分 nil 與空值，兩者只由 MaterializeNativeContinuePendingGroups 一起設定，已由
    // `has_native_pending_group_binding` 涵蓋。
    "handoff 的 nil roster／pending groups",
    // Go `RunRecoveredPrefix`、`Player.Advance`、`ANIPlayer.Advance` 出錯時同時回傳停下的段索引或狀態；Rust 的
    // Result 只帶錯誤（狀態留在 Player 欄位）。
    "錯誤時的附帶回傳值",
    // Go 的 `ContinueRuntimeInput{}`（未驗證的零值）在 Rust 無法建構（validated 欄位私有、沒有 Default）。
    "未驗證的 ContinueRuntimeInput 零值",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片、nil map 寫成 null，空的寫成 []／{}；Rust 不分 nil，null、[]、{}、"" 視為相同。
    "null 與空陣列、空物件、空字串相同",
    // Go 把整數值的 float64（Unit.Exp）寫成 73，serde_json 寫成 73.0；只比數值。
    "數字只比數值",
    // 結果物件裡「錯誤時不輸出」的鍵：Go 端缺席，Rust 端寫 null；缺席與 null 視為相同。
    "缺席的鍵與 null 相同",
    // Go map 迭代順序決定先報哪個錯誤的案例（黃金工具重呼叫 64 次偵測），只比拒收，文字記為 map-order。
    "map 順序決定的錯誤只比拒收",
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

fn record_of(hex_text: &str) -> save::PersistentRecord {
    let raw = unhex(hex_text);
    let mut record = save::PersistentRecord::default();
    record.raw.copy_from_slice(&raw);
    record
}

fn unit_from(v: &Value) -> battle::Unit {
    serde_json::from_value(v.clone()).unwrap_or_else(|e| panic!("Unit JSON 讀不進來：{e}\n{v}"))
}

fn uval<T: serde::Serialize>(u: &T) -> Value {
    serde_json::to_value(u).unwrap()
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
                _ => match (a.as_u64(), b.as_u64()) {
                    (Some(x), Some(y)) => x == y,
                    _ => a.as_f64() == b.as_f64(),
                },
            };
            (!same).then(|| format!("{path}: go={a} rust={b}"))
        }
        (Value::Array(a), Value::Array(b)) if !a.is_empty() || !b.is_empty() => {
            if a.len() != b.len() {
                return Some(format!("{path}: 長度 go={} rust={}", a.len(), b.len()));
            }
            a.iter().zip(b).enumerate().find_map(|(k, (x, y))| diff(&format!("{path}[{k}]"), x, y))
        }
        (Value::Object(a), Value::Object(b)) if !a.is_empty() || !b.is_empty() => {
            for (k, x) in a {
                match b.get(k) {
                    Some(y) => {
                        if let Some(d) = diff(&format!("{path}.{k}"), x, y) {
                            return Some(d);
                        }
                    }
                    None if x.is_null() => {}
                    None => return Some(format!("{path}.{k}: Rust 沒有這個欄位（go={x}）")),
                }
            }
            b.iter()
                .find(|(k, y)| !a.contains_key(*k) && !y.is_null())
                .map(|(k, y)| format!("{path}.{k}: Go 沒有這個欄位（rust={y}）"))
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

fn err_of<T>(r: &Result<T, c::CampaignError>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

struct Ctx {
    repo: PathBuf,
    flame: PathBuf,
    pack: PathBuf,
    catalog: c::NativeCharacterCatalog,
    gate: c::NativeIntermissionGateTable,
    join: c::NativeJoinConstructorTable,
    item_rows: Vec<u8>,
    /// ending_player 跑完後的合成器（蒙太奇接在同一個合成器之後）。
    after_player: Option<e::IndexedCompositor>,
    montage_assets: Option<e::MontageCycleAssets>,
}

impl Ctx {
    fn file(&self, label: &str) -> PathBuf {
        if let Some(rest) = label.strip_prefix("flame2/") {
            self.flame.join(rest)
        } else if let Some(rest) = label.strip_prefix("pack/") {
            self.pack.join(rest)
        } else {
            self.repo.join(label)
        }
    }

    fn remake(&self, rel: &str) -> String {
        self.repo.join("remake").join(rel).to_string_lossy().into_owned()
    }

    fn pack(&self, rel: &str) -> String {
        self.pack.join(rel).to_string_lossy().into_owned()
    }

    fn items(&self, variant: &str) -> &[u8] {
        match variant {
            "full" => &self.item_rows,
            "short" => &self.item_rows[..battle::NATIVE_ITEM_EFFECT_ROW_SIZE as usize],
            "empty" => &[],
            other => panic!("未知的 items {other}"),
        }
    }
}

// ---- 存檔槽、持續隊伍與 JOIN ----

fn slot_from(input: &Value) -> save::ChapterSlotSnapshot {
    let records_raw = unhex(s(input, "records"));
    let mut records = [save::PersistentRecord::default(); save::ROSTER_UNITS];
    for (index, record) in records.iter_mut().enumerate() {
        record.raw.copy_from_slice(&records_raw[index * save::UNIT_SIZE..(index + 1) * save::UNIT_SIZE]);
    }
    let mut metadata = [0u8; save::METADATA_SIZE];
    metadata.copy_from_slice(&unhex(s(input, "metadata")));
    save::ChapterSlotSnapshot {
        slot: i(input, "slot") as usize,
        verified: save::VerifiedMetadata {
            chapter: i(input, "chapter") as u8,
            roster_count: i(input, "roster_count") as u8,
            currency: i(input, "currency") as u32,
            hud_gate_a: i(input, "hud_gate_a") as u8,
            raw_53af9: i(input, "raw_53af9") as u8,
            raw_51e61: i(input, "raw_51e61") as u8,
            raw_51e62: i(input, "raw_51e62") as u8,
        },
        metadata,
        records,
        source_save_sha256: s(input, "source_sha256").to_string(),
    }
}

fn plan_val(p: &c::NativeChapterSlotRestorePlan) -> Value {
    let roster: serde_json::Map<String, Value> = p.party_roster.iter().map(|(k, u)| (k.to_string(), uval(u))).collect();
    let members: serde_json::Map<String, Value> =
        p.party_members.iter().map(|(k, v)| (k.to_string(), Value::Bool(*v))).collect();
    json!({
        "Slot": p.slot, "NativeChapterIndex": p.native_chapter_index, "DisplayChapter": p.display_chapter,
        "EntryNode": p.entry_node, "Currency": p.currency, "HUDGateA": p.hud_gate_a, "Raw53AF9": p.raw53_af9,
        "Raw51E61": p.raw51_e61, "Raw51E62": p.raw51_e62, "PartyMembers": members,
        "PartyJoinOrder": p.party_join_order, "PartyRoster": roster,
    })
}

fn restore_plan(ctx: &mut Ctx, name: &str, input: &Value, result: &Value) {
    let snapshot = slot_from(input);
    let mut graph = c::Campaign::default();
    for node in input["nodes"].as_array().into_iter().flatten() {
        let (id, kind) = (node[0].as_str().unwrap(), node[1].as_str().unwrap());
        graph.nodes.insert(id.to_string(), c::Node { r#type: kind.to_string(), ..Default::default() });
    }
    let graph_arg = (!b(input, "nil_graph")).then_some(&graph);
    let r = c::build_native_chapter_slot_restore_plan(&snapshot, &mut ctx.catalog, &ctx.gate, graph_arg);
    let mut got = json!({"err": err_of(&r)});
    if let Ok(plan) = &r {
        got["plan"] = plan_val(plan);
    }
    check(name, result, &got);
}

fn slot_save(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let baseline = slot_from(input);
    let roster: BTreeMap<i64, battle::Unit> = input["roster"]
        .as_array()
        .into_iter()
        .flatten()
        .map(|pair| (pair[0].as_i64().unwrap(), unit_from(&pair[1])))
        .collect();
    let order = ints(&input["join_order"]);
    let r = c::build_native_chapter_slot(
        &baseline,
        &roster,
        &order,
        i(input, "chapter") as u8,
        i(input, "gold") as u32,
        &ctx.join,
        ctx.items(s(input, "items")),
    );
    let mut got = json!({"err": err_of(&r)});
    if result["err_order_dependent"] == json!(true) && r.is_err() {
        got["err"] = json!("map-order");
        got["err_order_dependent"] = json!(true);
    }
    if let Ok(slot) = &r {
        got["roster"] = json!(hex(&slot.roster));
        got["metadata"] = json!(hex(&slot.metadata));
    }
    check(name, result, &got);
}

fn encode_record(name: &str, input: &Value, result: &Value) {
    let baseline = record_of(s(input, "baseline"));
    let unit = (!input["unit"].is_null()).then(|| unit_from(&input["unit"]));
    let r = c::encode_native_persistent_record(baseline, unit.as_ref());
    let record = r.as_ref().ok().map(|rec| json!(hex(&rec.raw)));
    check(name, result, &json!({"err": err_of(&r), "record": record}));
}

fn join_base(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let table = c::load_native_join_base_table(&ctx.remake("assets/data/native_join_base_units.json")).unwrap();
    let r = table.load_base_unit(i(input, "id"));
    let mut got = json!({"err": err_of(&r)});
    if let Ok(unit) = &r {
        got["unit"] = uval(unit);
    }
    check(name, result, &got);
}

fn join_record(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let residual = record_of(s(input, "residual"));
    let r = ctx.join.materialize_persistent_record_on(residual, i(input, "id"), ctx.items(s(input, "items")));
    let record = r.as_ref().ok().map(|rec| json!(hex(&rec.raw)));
    check(name, result, &json!({"err": err_of(&r), "record": record}));
}

fn join_unit(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let residual = record_of(s(input, "residual"));
    let base = unit_from(&input["base"]);
    let r = ctx.join.materialize_persistent_unit_on(residual, i(input, "id"), base, ctx.items(s(input, "items")));
    let mut got = json!({"err": err_of(&r)});
    if let Ok(unit) = &r {
        got["unit"] = uval(unit);
    }
    check(name, result, &got);
}

fn party_record(ctx: &mut Ctx, name: &str, input: &Value, result: &Value) {
    let r = c::materialize_native_persistent_party_record(record_of(s(input, "record")), &mut ctx.catalog);
    let mut got = json!({"err": err_of(&r)});
    if let Ok(unit) = &r {
        got["unit"] = uval(unit);
    }
    check(name, result, &got);
}

// ---- CONTINUE ----

fn header_from(v: &Value) -> save::CurrentRuntimeHeader {
    save::CurrentRuntimeHeader {
        turn_counter: i(v, "TurnCounter") as u8,
        runtime_count: i(v, "RuntimeCount") as u8,
        chapter: i(v, "Chapter") as u8,
        camera_x: i(v, "CameraX") as u8,
        camera_y: i(v, "CameraY") as u8,
        cursor_x: i(v, "CursorX") as u8,
        cursor_y: i(v, "CursorY") as u8,
        visible_cursor_x: i(v, "VisibleCursorX") as u8,
        visible_cursor_y: i(v, "VisibleCursorY") as u8,
        persistent_count: i(v, "PersistentCount") as u8,
        currency: i(v, "Currency") as u32,
        raw_53af9: i(v, "Raw53AF9") as u8,
        hud_gate_a: i(v, "HUDGateA") as u8,
        raw_51e61: i(v, "Raw51E61") as u8,
        raw_51e62: i(v, "Raw51E62") as u8,
    }
}

fn snapshot_from(input: &Value) -> save::CurrentSnapshot {
    let mut snapshot = save::CurrentSnapshot { header: header_from(&input["header"]), ..Default::default() };
    snapshot.native_field_control.copy_from_slice(&unhex(s(input, "field_control")));
    snapshot.native_event_state.copy_from_slice(&unhex(s(input, "event_state")));
    let persistent = unhex(s(input, "persistent"));
    for (index, record) in snapshot.persistent_records.iter_mut().enumerate() {
        record.raw.copy_from_slice(&persistent[index * save::UNIT_SIZE..(index + 1) * save::UNIT_SIZE]);
    }
    snapshot.runtime_records =
        input["runtime"].as_array().into_iter().flatten().map(|r| record_of(r.as_str().unwrap())).collect();
    snapshot
}

fn state_from_asset(asset: &battle::State) -> battle::State {
    let mut state = battle::State::default();
    (state.w, state.h) = (asset.w, asset.h);
    state.native_composition_event_bytes = asset.native_composition_event_bytes.clone();
    state.native_field_event_slots = asset.native_field_event_slots.clone();
    state.native_field_events = asset.native_field_events.clone();
    state.native_field_event_rules = asset.native_field_event_rules.clone();
    state
}

/// Go `summary`：CONTINUE 五個轉接器讀寫的 State 欄位（Go 欄位名）。
fn summary(st: &battle::State) -> Value {
    let pending: Vec<i64> = st.pending_groups.iter().filter(|(_, ok)| **ok).map(|(g, _)| *g).collect();
    let view = &st.native_map_view_state;
    let hud = &st.native_map_hud_state;
    json!({
        "W": st.w, "H": st.h,
        "Units": st.units.iter().map(uval).collect::<Vec<_>>(),
        "Roster": st.roster.iter().map(uval).collect::<Vec<_>>(),
        "PendingGroups": pending,
        "NativeRuntimeRecords": st.native_runtime_records.iter()
            .map(|r| json!([hex(&r.raw), r.selector_key, r.selector_slot])).collect::<Vec<_>>(),
        "NativeFieldUnitControls": st.native_field_unit_controls.iter().map(|c| json!(hex(&c.raw))).collect::<Vec<_>>(),
        "NativeTurnEventControls": st.native_turn_event_controls.iter()
            .map(|t| json!([t.turn, t.event_id, t.raw_camp])).collect::<Vec<_>>(),
        "NativeFieldEvents": st.native_field_events.iter().map(|f| json!([f.event_id, f.selector])).collect::<Vec<_>>(),
        "NativeChestControls": st.native_chest_controls.iter().map(|c| json!([c.raw_type, c.value])).collect::<Vec<_>>(),
        "NativeFieldControlRaw": hex(&st.native_field_control_raw),
        "NativeEventState": hex(&st.native_event_state),
        "NativeRoundCounter": st.native_round_counter,
        "NativeMapViewState": {"CameraX": view.camera_x, "CameraY": view.camera_y, "CursorX": view.cursor_x,
            "CursorY": view.cursor_y, "VisibleCursorX": view.visible_cursor_x, "VisibleCursorY": view.visible_cursor_y},
        "NativeMapHUDState": {"DisplayGateA": hud.display_gate_a, "DisplayGateB": hud.display_gate_b, "AnchorX": hud.anchor_x},
        "NativeMapRangeMode": st.native_map_range_mode,
        "NativeMapCycleState": {"Idle": st.native_map_cycle_state.idle, "Moving": st.native_map_cycle_state.moving,
            "LastTimerTick": st.native_map_cycle_state.last_timer_tick},
        "NativeTerrainPhaseState": {"Phase": st.native_terrain_phase_state.phase,
            "LastTimerTick": st.native_terrain_phase_state.last_timer_tick},
        "NativeTerrainFlipState": {"Value": st.native_terrain_flip_state.value,
            "LastTimerTick": st.native_terrain_flip_state.last_timer_tick},
        "NativeUnitPixelShiftState": {"Value": st.native_unit_pixel_shift_state.value,
            "LastTimerTick": st.native_unit_pixel_shift_state.last_timer_tick},
        "HasSelectorCache": st.native_map_selector_cache.is_some(),
        "HasSelectorError": st.native_map_selector_error.is_some(),
        "flags": [st.has_native_field_control_state, st.has_native_turn_event_control_state,
            st.has_native_chest_control_state, st.has_native_runtime_unit_projection,
            st.has_native_pending_group_binding, st.has_native_map_cycle_state, st.has_native_terrain_phase_state,
            st.has_native_map_binary_timing_state, st.has_native_map_view_state, st.has_native_map_hud_state,
            st.has_native_map_range_mode_state],
    })
}

fn continue_case(ctx: &mut Ctx, name: &str, input: &Value, result: &Value) {
    let snapshot = snapshot_from(input);
    let cv = &input["context"];
    let context = save::ContinueRuntimeContext {
        chapter: i(cv, "Chapter"),
        field_width: i(cv, "FieldWidth"),
        field_height: i(cv, "FieldHeight"),
        selector_group_count: i(cv, "SelectorGroupCount"),
        title_timer_tick: i(cv, "TitleTimerTick"),
        has_title_timer_tick: b(cv, "HasTitleTimerTick"),
    };
    let preflight = save::build_continue_runtime_input(&snapshot, context);
    let mut got = json!({"preflight_err": preflight.as_ref().err().map(|e| e.to_string())});
    let Ok(cri) = preflight else {
        check(name, result, &got);
        return;
    };
    let asset = battle::load(&ctx.remake(s(input, "map"))).unwrap();
    let scenario = battle::load_scenario(&ctx.remake(s(input, "scenario"))).unwrap();
    let asset_chapter = i(input, "asset_chapter");
    let items = ctx.items(s(input, "items")).to_vec();
    let mut st = state_from_asset(&asset);
    let mut stages = Vec::new();
    let r = c::materialize_native_continue_field_boundary(Some(&mut st), &cri, asset_chapter);
    stages.push(json!({"stage": "field_boundary", "err": err_of(&r), "state": summary(&st)}));
    let r = c::materialize_native_continue_runtime_units(Some(&mut st), &cri, &mut ctx.catalog);
    stages.push(json!({"stage": "runtime_units", "err": err_of(&r), "state": summary(&st)}));
    let r = c::materialize_native_continue_map_timing(Some(&mut st), &cri);
    stages.push(json!({"stage": "map_timing", "err": err_of(&r), "state": summary(&st)}));
    let r = c::materialize_native_continue_pending_groups(
        Some(&mut st),
        &cri,
        asset_chapter,
        Some(&asset),
        Some(&scenario),
        &items,
    );
    let mut stage = json!({"stage": "pending_groups", "err": err_of(&r), "state": summary(&st)});
    let go_stage = &result["stages"][3];
    if go_stage["err_order_dependent"] == json!(true) && r.is_err() {
        stage["err"] = json!("map-order");
        stage["err_order_dependent"] = json!(true);
    }
    stages.push(stage);
    let r = c::materialize_native_continue_interactive_boundary(Some(&mut st), &cri);
    stages.push(json!({"stage": "interactive", "err": err_of(&r), "state": summary(&st)}));
    let r = c::validate_native_continue_battle_handoff(&cri, Some(&st), Some(&scenario));
    stages.push(json!({"stage": "handoff", "err": err_of(&r), "state": summary(&st)}));
    got["stages"] = Value::Array(stages);
    let r = c::build_native_current_runtime_records(Some(&st));
    got["runtime_records"] =
        json!(r.as_deref().unwrap_or_default().iter().map(|rec| hex(&rec.raw)).collect::<Vec<_>>());
    got["runtime_err"] = err_of(&r);
    if let Ok(joined) =
        ctx.join.materialize_persistent_unit(i(input, "join_id"), &battle::Unit::default(), &ctx.item_rows)
    {
        st.units.push(joined);
        st.native_runtime_records.push(battle::NativeRuntimeRecordState::default());
    }
    let r = c::build_native_current_persistent_records(Some(&st), &snapshot, &ctx.join, &items);
    got["persistent_err"] = err_of(&r);
    if let Ok((records, count)) = &r {
        let all: Vec<u8> = records.iter().flat_map(|rec| rec.raw).collect();
        got["persistent_records"] = json!(hex(&all));
        got["persistent_count"] = json!(count);
    }
    check(name, result, &got);
}

// ---- 結局 ----

fn shared(compositor: e::IndexedCompositor) -> e::SharedCompositor {
    Arc::new(Mutex::new(compositor))
}

fn sums(c: &e::IndexedCompositor) -> (String, String) {
    (digest(&c.vga), digest(&c.palette))
}

fn ending_loaders(ctx: &Ctx, name: &str, result: &Value) {
    let timeline = e::load_timeline(&ctx.remake("assets/endings/native_2bce5.json")).unwrap();
    let phase = e::load_finale_phase(&ctx.remake("assets/endings/native_2c405.json")).unwrap();
    let montage = e::load_montage(&ctx.remake("assets/endings/native_2c548.json")).unwrap();
    let tail = e::load_montage_tail(&ctx.remake("assets/endings/native_2c194_tail.json")).unwrap();
    let tail_plan: Vec<Value> = tail
        .plan()
        .unwrap()
        .iter()
        .map(|t| {
            json!({"Index": t.index, "Global540FF": t.global540_ff, "Record0Byte6": t.record0_byte6,
                "Record0Byte7": t.record0_byte7, "Record1Byte6": t.record1_byte6, "Record1Byte7": t.record1_byte7})
        })
        .collect();
    let tail_visual: Vec<Value> = tail
        .plan_visual_resources()
        .unwrap()
        .iter()
        .map(|v| {
            json!({"Index": v.index, "TAI": v.tai, "BG": v.bg, "Record1FIGANIBase": v.record1_figani_base,
                "Record1FIGANIAux": v.record1_figani_aux, "Record0FIGANIBase": v.record0_figani_base,
                "Record0FIGANIAux": v.record0_figani_aux})
        })
        .collect();
    let fade: Vec<Value> = montage
        .plan_figure_fade(2)
        .unwrap()
        .iter()
        .map(|p| json!({"Stage": p.stage, "SourceOffset": p.source_offset, "PaletteDelta": p.palette_delta}))
        .collect();
    let mirror: Vec<Value> = montage
        .plan_mirror_figure_fade(0, 0)
        .unwrap()
        .iter()
        .map(|p| {
            json!({"Stage": p.stage, "PrimarySourceOffset": p.primary_source_offset, "PaletteDelta": p.palette_delta,
                "DrawSecondary": p.draw_secondary, "DrawPlatform": p.draw_platform})
        })
        .collect();
    let grid: Vec<Value> = montage
        .plan_dialogue_frame_grid()
        .unwrap()
        .iter()
        .map(|p| json!({"ResourceIndex": p.resource_index, "DestinationByte": p.destination_byte}))
        .collect();
    let got = json!({"timeline": uval(&timeline), "phase": uval(&phase), "montage": uval(&montage), "tail": uval(&tail),
        "tail_plan": tail_plan, "tail_visual": tail_visual, "fade": fade, "mirror": mirror, "grid": grid});
    check(name, result, &got);
}

fn ending_player(ctx: &mut Ctx, name: &str, input: &Value, result: &Value) {
    let timeline = e::load_timeline(&ctx.remake("assets/endings/native_2bce5.json")).unwrap();
    let phase = e::load_finale_phase(&ctx.remake("assets/endings/native_2c405.json")).unwrap();
    let frames = fdother::decode_resource(ctx.flame.join("FDOTHER.DAT"), timeline.resource.index).unwrap();
    let clip = fd2_formats::afm::decode_resource(ctx.flame.join("ANI.DAT"), 2).unwrap();
    let strings = fdtxt::load_separated_resource(ctx.pack("text"), 31).unwrap();
    let font = fdtxt::load_separated_font(ctx.pack("fonts")).unwrap();
    let compositor = shared(e::new_indexed_compositor());
    let mut player = e::new_player(timeline, &frames, Some(&clip), Some(compositor.clone())).unwrap();
    player
        .enable_recovered_phase0(
            &phase,
            &e::Phase0Assets { strings: Some(strings), font: Some(font), ..Default::default() },
        )
        .unwrap();
    let mut steps = Vec::new();
    for ms in ints(&input["elapsed"]) {
        let r = player.advance(ms);
        let state = match &r {
            Ok(state) => *state,
            Err(_) => player.state,
        };
        let (vga, palette) = sums(&compositor.lock().unwrap());
        let mut step = json!({"state": state.0, "err": err_of(&r), "segment": player.segment, "wait": player.wait_ms,
            "vga": vga, "palette": palette});
        if let Some(blocked) = &player.blocked {
            step["blocked"] = json!([blocked.op, blocked.source]);
        }
        let mut resumed = false;
        if state == e::PLAYBACK_BLOCKED {
            let (then, ok26) = player.blocked_dialogue(26);
            let (other, ok29) = player.blocked_dialogue(29);
            step["dialogue"] = json!([then.len(), ok26, other.len(), ok29]);
            let (cue, ok) = player.audio_cue_at_blocked_boundary();
            step["cue"] = json!([cue.source, ok]);
            resumed = player.resume_blocked_dialogue();
            step["resumed"] = json!(resumed);
        }
        steps.push(step);
        if r.is_err() || (state != e::PLAYBACK_RUNNING && !resumed) {
            break;
        }
    }
    let final_compositor = compositor.lock().unwrap().clone();
    let got = json!({"steps": steps, "rgba": digest(&final_compositor.rgba().pix)});
    check(name, result, &got);
    ctx.after_player = Some(final_compositor);
}

fn montage_paths(ctx: &Ctx) -> e::MontageArchivePaths {
    e::MontageArchivePaths {
        surface_root: ctx.pack("surfaces"),
        item_panel_root: ctx.pack("ui"),
        animation_root: ctx.pack("animations"),
        portrait_root: ctx.pack("portraits"),
        text_root: ctx.pack("text"),
        font_root: ctx.pack("fonts"),
    }
}

fn montage_cycle(ctx: &mut Ctx, name: &str, input: &Value, result: &Value) {
    let montage = e::load_montage(&ctx.remake("assets/endings/native_2c548.json")).unwrap();
    let units: Vec<Vec<u8>> = input["units"].as_array().unwrap().iter().map(|u| unhex(u.as_str().unwrap())).collect();
    let groups = unhex(s(input, "groups"));
    if ctx.montage_assets.is_none() {
        ctx.montage_assets = Some(e::load_montage_cycle_assets(&montage, &montage_paths(ctx), &units).unwrap());
    }
    let assets = ctx.montage_assets.clone().unwrap();
    let before = ctx.after_player.as_ref().expect("montage_cycle 需要先跑 ending_player");
    let mut cc = e::new_indexed_compositor();
    cc.vga.copy_from_slice(&before.vga);
    cc.present_ani(&before.vga, &before.baseline).unwrap();
    cc.palette = before.palette;
    let cc = shared(cc);
    let mut cycle = e::new_montage_cycle(&montage, assets, &units, &groups, Some(cc.clone())).unwrap();
    let skip_at = i(input, "skip_at");
    let randoms = ints(&input["randoms"]);
    let mut steps = Vec::new();
    let mut portrait_steps = 0;
    for random in randoms {
        if cycle.ready() {
            break;
        }
        let r = cycle.step(random as u8);
        let mut observed = false;
        if cycle.phase == e::MONTAGE_PHASE_PORTRAIT {
            portrait_steps += 1;
            if portrait_steps == skip_at {
                observed = cycle.observe_input_change();
            }
        }
        let (vga, palette) = sums(&cc.lock().unwrap());
        steps.push(json!({"phase": cycle.phase.0, "plan": cycle.plan_index, "ticks": cycle.delay_ticks,
            "ms": cycle.delay_ms, "err": err_of(&r), "observed": observed, "portrait": cycle.portrait,
            "vga": vga, "palette": palette}));
        if r.is_err() {
            break;
        }
    }
    check(name, result, &json!({"steps": steps, "ready": cycle.ready()}));
}

fn montage_tail(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let tail = e::load_montage_tail(&ctx.remake("assets/endings/native_2c194_tail.json")).unwrap();
    let assets = match s(input, "source") {
        "archive" => {
            e::load_montage_tail_assets_archive(&tail, &ctx.flame.join("FDOTHER.DAT").to_string_lossy()).unwrap()
        }
        _ => e::load_montage_tail_assets(
            &tail,
            &e::MontageTailAssetPaths {
                surface_root: ctx.pack("surfaces"),
                palette_root: ctx.pack("palette"),
                animation_root: ctx.pack("animations"),
            },
        )
        .unwrap(),
    };
    let sets = e::load_montage_tail_visual_sets(
        &tail,
        &e::MontageTailVisualPaths { surface_root: ctx.pack("surfaces"), animation_root: ctx.pack("animations") },
    )
    .unwrap();
    let tc = shared(e::new_indexed_compositor());
    let mut tp = e::new_montage_tail_player(&tail, &assets, &sets, Some(tc.clone())).unwrap();
    let mut steps = Vec::new();
    for _ in 0..8000 {
        if tp.ready() {
            break;
        }
        let r = tp.step();
        let (vga, palette) = sums(&tc.lock().unwrap());
        steps.push(json!({"phase": tp.phase.0, "segment": tp.segment, "frame": tp.frame, "inner": tp.inner,
            "base": [tp.base_frame, tp.base_inner], "effect": tp.effect_step, "sound": tp.sound_marker,
            "ticks": tp.delay_ticks, "err": err_of(&r), "vga": vga, "palette": palette}));
        if r.is_err() {
            break;
        }
    }
    check(name, result, &json!({"steps": steps, "ready": tp.ready()}));
}

fn tail_loader(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let tail = e::load_montage_tail(&ctx.remake("assets/endings/native_2c194_tail.json")).unwrap();
    let records: Vec<save::PersistentRecord> =
        input["records"].as_array().unwrap().iter().map(|r| record_of(r.as_str().unwrap())).collect();
    let paths =
        e::MontageTailLoaderPaths { fdfield_root: ctx.pack("fields/fdfield"), fdicon_root: ctx.pack("sprites/fdicon") };
    let r = e::build_montage_tail_loader_baseline(&tail, &records, &paths, ctx.items(s(input, "items")));
    let baseline = r.as_ref().cloned().unwrap_or_default();
    let runtime: Vec<String> =
        (0..baseline.runtime_count()).map(|index| hex(&baseline.runtime_record(index).unwrap())).collect();
    check(name, result, &json!({"err": err_of(&r), "runtime": runtime}));
}

fn portrait_frame(ctx: &mut Ctx, name: &str, input: &Value, result: &Value) {
    let montage = e::load_montage(&ctx.remake("assets/endings/native_2c548.json")).unwrap();
    let assets = ctx.montage_assets.clone().expect("portrait_frame 需要先載入蒙太奇素材");
    let unit = unhex(s(input, "unit"));
    let empty = Vec::new();
    let portraits = assets.portraits.get(&i64::from(unit[7])).unwrap_or(&empty);
    let r = e::compose_montage_portrait_frame(
        &montage,
        &assets.grid,
        &unit,
        i(input, "edi"),
        i(input, "frame"),
        portraits,
        assets.current.as_ref(),
        assets.permanent.as_ref(),
        assets.font.as_ref(),
    );
    let sha = r.as_ref().ok().map(|f| json!(digest(f)));
    check(name, result, &json!({"err": err_of(&r), "sha": sha}));
}

fn dialogue(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let operand = i(input, "portrait");
    let Ok(portraits) = dato::load_separated_resource(ctx.pack("portraits"), operand) else {
        check(name, result, &json!({"err": "portrait unavailable"}));
        return;
    };
    let entries = fdother::load_separated_item_panel_entries(ctx.pack("ui")).unwrap();
    let max_cell = entries.raw.keys().copied().max().unwrap_or(0);
    let mut cells = vec![fdother::RawCell::default(); (max_cell + 1) as usize];
    for (index, cell) in &entries.raw {
        cells[*index as usize] = cell.clone();
    }
    assert_eq!(s(input, "background"), "k*7");
    let background: Vec<u8> = (0..e::BYTES as usize).map(|k| (k * 7) as u8).collect();
    let font = fdtxt::load_separated_font(ctx.pack("fonts")).unwrap();
    let glyphs: BTreeMap<String, i64> = input["glyphs"]
        .as_object()
        .map(|o| o.iter().map(|(k, v)| (k.clone(), v.as_i64().unwrap())).collect())
        .unwrap_or_default();
    let pages: Vec<Vec<String>> = serde_json::from_value(input["pages"].clone()).unwrap();
    let base = e::compose_native_dialogue_base(&background, &cells, &portraits[0]);
    let mut got = json!({"base_err": err_of(&base), "base": base.as_ref().ok().map(|b| digest(b))});
    let mut page_sums = Vec::new();
    if let Ok(base) = &base {
        for page in 0..pages.len() {
            let frames =
                e::compose_native_dialogue_progressive_frames(base, Some(&font), Some(&glyphs), &pages, page as i64);
            let sums: Vec<String> = frames.as_deref().unwrap_or_default().iter().map(|f| digest(f)).collect();
            page_sums.push(json!({"err": err_of(&frames), "frames": sums}));
        }
        let mouth = e::compose_native_dialogue_mouth_frame(base, &portraits[3]);
        got["mouth_err"] = err_of(&mouth);
        got["mouth"] = json!(mouth.as_ref().ok().map(|m| digest(m)));
    }
    got["pages"] = Value::Array(page_sums);
    check(name, result, &got);
}

fn phase0(ctx: &Ctx, name: &str, input: &Value, result: &Value) {
    let phase = e::load_finale_phase(&ctx.remake("assets/endings/native_2c405.json")).unwrap();
    let strings = fdtxt::load_separated_resource(ctx.pack("text"), 31).unwrap();
    let font = fdtxt::load_separated_font(ctx.pack("fonts")).unwrap();
    let mut baseline = [0u8; 768];
    baseline.copy_from_slice(&unhex(s(input, "baseline")));
    let pc = shared(e::new_indexed_compositor());
    let assets = e::Phase0Assets { strings: Some(strings), font: Some(font), baseline, ..Default::default() };
    let mut p0 = e::new_phase0_player_from_assets(&phase, &assets, Some(pc.clone())).unwrap();
    let mut steps = Vec::new();
    for ms in ints(&input["elapsed"]) {
        let r = p0.advance(ms);
        let done = *r.as_ref().unwrap_or(&false);
        let (vga, palette) = sums(&pc.lock().unwrap());
        steps.push(json!({"done": done, "err": err_of(&r), "vga": vga, "palette": palette}));
        if done || r.is_err() {
            break;
        }
    }
    check(name, result, &json!({"steps": steps}));
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_CAMPAIGN_C").filter(|p| Path::new(p).exists());
    let dir = |var: &str, probe: &str| std::env::var_os(var).map(PathBuf::from).filter(|p| p.join(probe).exists());
    let repo = dir("FD2_GOLDEN_REPO", "remake/assets");
    let flame = dir("FD2_GOLDEN_FLAME2", "FD2.SAV");
    let pack = dir("FD2_GOLDEN_PACK", "ui/fdother_005_item_panel");
    let (Some(path), Some(repo), Some(flame), Some(pack)) = (path, repo, flame, pack) else {
        assert!(
            !require,
            "FD2_REQUIRE_GOLDEN=1 但缺少 FD2_GOLDEN_CAMPAIGN_C／FD2_GOLDEN_REPO／FD2_GOLDEN_FLAME2／FD2_GOLDEN_PACK"
        );
        eprintln!("略過：沒有 FD2_GOLDEN_CAMPAIGN_C 或真實輸入");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let remake = |rel: &str| repo.join("remake").join(rel).to_string_lossy().into_owned();
    let ctx = Ctx {
        catalog: c::load_native_character_catalog(&remake("assets/data/native_character_catalog.json")).unwrap(),
        gate: c::load_native_intermission_gate_table(&remake("assets/data/native_intermission_gate.json")).unwrap(),
        join: c::load_native_join_constructor_table(&remake("assets/data/native_join_constructor.json")).unwrap(),
        item_rows: battle::load_native_item_effect_row_prefix(&remake("assets/data/native_item_effect_rows.json"))
            .unwrap(),
        repo,
        flame,
        pack,
        after_player: None,
        montage_assets: None,
    };
    Some((doc, ctx))
}

#[test]
fn golden_campaign_c() {
    let Some((doc, mut ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-campaign-c/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    let files = doc["files"].as_array().unwrap();
    for file in files {
        let path = ctx.file(s(file, "path"));
        let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀不到 {}：{e}", path.display()));
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
            "restore_plan" => restore_plan(&mut ctx, &name, input, result),
            "slot_save" => slot_save(&ctx, &name, input, result),
            "encode_record" => encode_record(&name, input, result),
            "join_base" => join_base(&ctx, &name, input, result),
            "join_record" => join_record(&ctx, &name, input, result),
            "join_unit" => join_unit(&ctx, &name, input, result),
            "party_record" => party_record(&mut ctx, &name, input, result),
            "catalog" => check(&name, result, &json!({"catalog": uval(&ctx.catalog)})),
            "gate" => check(&name, result, &json!({"gate": uval(&ctx.gate)})),
            "continue" => continue_case(&mut ctx, &name, input, result),
            "ending_loaders" => ending_loaders(&ctx, &name, result),
            "ending_player" => ending_player(&mut ctx, &name, input, result),
            "montage_cycle" => montage_cycle(&mut ctx, &name, input, result),
            "montage_tail" => montage_tail(&ctx, &name, input, result),
            "tail_loader" => tail_loader(&ctx, &name, input, result),
            "portrait_frame" => portrait_frame(&mut ctx, &name, input, result),
            "dialogue" => dialogue(&ctx, &name, input, result),
            "phase0" => phase0(&ctx, &name, input, result),
            _ => panic!("{name}：未知的 op"),
        }
        *counts.entry(op.to_string()).or_default() += 1;
    }
    let total: usize = counts.values().sum();
    eprintln!("golden_campaign_c：{} 個輸入檔、{total} 個案例全部相同：{counts:?}", files.len());
}
