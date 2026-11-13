//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-int` 的黃金差分：fd2-rules 跨組整合（R2 完成條件，
//! docs/goal/119-goal-rust-bevy-rewrite-20261113.md：以相同種子跑完整的 AI 戰鬥，Rust 與 Go 逐回合
//! 狀態相同）。
//!
//! 由 `tools/rust_golden_rules_int.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_INT`；真實輸入（受版控的
//! `remake/assets`）掛在 `FD2_GOLDEN_REPO` 底下，先核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 戰鬥重播：每場戰鬥以同一份規則重做 Go 工具的無頭驅動（驅動的組裝方式與刻意省略的部分寫在
//! Go 工具的 `driver` 型別註解，本檔的 [`Driver`] 逐項對應）。每一步把 Rust 的結果組成與 Go 相同鍵
//! 集合的 JSON 逐欄比對：計畫投影、執行結果、下一個 Int63（`GoMathRand` 複製一份再取，不消耗）、
//! 原版 RNG 狀態、勝負與停止原因；再把 Go 記下的單位欄位變化套到期望的單位物件上，與 Rust
//! `serde_json::to_value(Unit)` 比對全部單位，戰場摘要亦同。戰鬥結束比完整 `StateSnapshot`。
//! 每場戰鬥在第一個差異處停下並記錄，全部跑完才判定。
//!
//! 比對時的表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做或無法比對的情況列在 [`KNOWN_DIFFERENCES`]。

use fd2_rules::battle::*;
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use std::sync::Arc;

/// 刻意不照做 Go 的行為，或 Go 端本身不唯一的結果。
const KNOWN_DIFFERENCES: &[&str] = &[
    // combat.go 202、235、621、921 行以嚴格 < 在 reach（Go map）裡挑最近格，距離平手時 Go 取先迭代到的
    // （每次執行可能不同），Rust 依 Cell 升冪取第一個。Go 工具在 normalized 規劃器走到這幾個迴圈時把
    // 平手的候選數寫進該步的 `tie`；第一個差異落在帶 `tie` 的步驟時歸入這一項並計數，不算失敗。
    "reach 平手的 Go map 迭代順序",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片／map 寫成 null、空的寫成 []／{}；Rust 不分 nil，null、[]、{}、"" 視為相同。
    "null 與空陣列、空物件、空字串相同",
    // Go 把整數值的 float64 寫成 0，serde_json 寫成 0.0；只比數值。
    "數字只比數值",
    // 物件只比鍵值集合，不比鍵順序。
    "物件鍵順序",
];

// ---- 共用工具 ----

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn digest(b: &[u8]) -> String {
    hex(&Sha256::digest(b))
}

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn s<'a>(v: &'a Value, key: &str) -> &'a str {
    v[key].as_str().unwrap_or_else(|| panic!("欄位 {key} 不是字串：{}", v[key]))
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
        (Value::Array(a), Value::Array(b)) => {
            if a.len() != b.len() {
                return Some(format!("{path}: 長度 go={} rust={}", a.len(), b.len()));
            }
            a.iter().zip(b).enumerate().find_map(|(k, (x, y))| diff(&format!("{path}[{k}]"), x, y))
        }
        (Value::Object(a), Value::Object(b)) if !a.is_empty() && !b.is_empty() => {
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

fn err_of<T>(r: &Result<T, BattleError>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

fn idx(u: Option<UnitIdx>) -> i64 {
    u.map_or(-1, |i| i as i64)
}

fn cell(c: Cell) -> Value {
    json!([c.x, c.y])
}

fn cells(cs: &[Cell]) -> Value {
    Value::Array(cs.iter().map(|c| cell(*c)).collect())
}

fn manhattan(ax: i64, ay: i64, bx: i64, by: i64) -> i64 {
    (ax - bx).abs() + (ay - by).abs()
}

/// Go `dirToward`（cmd/fd2 main.go）。
fn dir_toward(ax: i64, ay: i64, tx: i64, ty: i64) -> i64 {
    let (dx, dy) = (tx - ax, ty - ay);
    if dx.abs() > dy.abs() {
        return if dx > 0 { 3 } else { 1 };
    }
    if dy > 0 { 0 } else { 2 }
}

// ---- 資產與戰場建置（Go `loadAssets`、`setup`、`rawify`）----

struct Ctx {
    repo: PathBuf,
}

impl Ctx {
    /// 黃金檔的路徑相對於 `remake/`（Go 工具的工作目錄）。
    fn real(&self, rel: &str) -> String {
        self.repo.join("remake").join(rel).to_string_lossy().into_owned()
    }
}

struct Assets {
    book: Vec<NativeCommandRecord>,
    spells: Vec<Spell>,
    cost_rows: Vec<Vec<u8>>,
    item_rows: Vec<u8>,
    resist: BTreeMap<i64, i64>,
    learn: BTreeMap<i64, Vec<CommandLearnEntry>>,
    learn_sel: BTreeMap<i64, i64>,
    growth: BTreeMap<i64, GrowthRow>,
    ai_cmd: BTreeMap<i64, i64>,
}

fn load_assets(ctx: &Ctx, doc: &Value) -> Assets {
    let spells = ctx.real("assets/spells.json");
    Assets {
        book: load_native_command_records(&spells).unwrap(),
        spells: load_spells(&spells).unwrap(),
        cost_rows: load_native_movement_cost_rows(&ctx.real("assets/data/native_movement_cost_rows.json")).unwrap(),
        item_rows: load_native_item_effect_row_prefix(&ctx.real("assets/data/native_item_effect_rows.json")).unwrap(),
        resist: load_native_command_resistances(&ctx.real("assets/data/native_command_resistances.json")).unwrap(),
        learn: load_command_learn(&ctx.real("assets/data/command_learn.json")).unwrap(),
        learn_sel: load_command_learn_selectors(&ctx.real("assets/data/class_change_growth.json")).unwrap(),
        growth: load_native_growth_slots(&ctx.real("assets/data/native_growth_slots.json")).unwrap(),
        // campaign.LoadAICommandSpellMap 不在 fd2-rules；Go 工具把結果寫進黃金檔。
        ai_cmd: doc["ai_command_spell"]
            .as_array()
            .unwrap()
            .iter()
            .map(|p| (p[0].as_i64().unwrap(), p[1].as_i64().unwrap()))
            .collect(),
    }
}

/// Go `rawify`：補齊劇本隊伍沒有的原版 provenance，規則固定。
fn rawify(units: &mut [Unit]) {
    for u in units {
        if !u.has_native_map_presentation {
            u.native_map_presentation = NativeMapPresentationState { x: u.x as u8, y: u.y as u8, ..Default::default() };
            u.has_native_map_presentation = true;
        }
        if !u.has_native_record_byte5 {
            u.native_record_byte5 = u8::from(!u.on_field || u.hp <= 0);
            u.has_native_record_byte5 = true;
        }
        if !u.has_native_record_byte6 {
            u.native_record_byte6 = [2, 1, 0][u.camp.0.rem_euclid(3) as usize];
            u.has_native_record_byte6 = true;
        }
        if !u.has_battle_fig {
            (u.battle_fig, u.has_battle_fig) = (4, true);
        }
        if !u.has_native_record_byte34 {
            (u.native_record_byte34, u.has_native_record_byte34) = (0, true);
        }
        if !u.has_native_record_byte35 {
            (u.native_record_byte35, u.has_native_record_byte35) = (0, true);
        }
        if !u.has_native_record_byte36 {
            (u.native_record_byte36, u.has_native_record_byte36) = (0, true);
        }
        if !u.has_native_record_word42 {
            (u.native_record_word42, u.has_native_record_word42) = (u.max_hp as u16, true);
        }
        if !u.has_native_record_word46 {
            (u.native_record_word46, u.has_native_record_word46) = (u.max_mp as u16, true);
        }
    }
}

fn setup(ctx: &Ctx, a: &Assets, map: &str, scenario: &str) -> (State, Option<Scenario>, Value) {
    let mut st = load(&ctx.real(map)).unwrap_or_else(|e| panic!("{map}: {e}"));
    let mut info = Map::new();
    info.insert("bind_items".into(), err_of(&st.bind_native_future_item_rows(&a.item_rows)));
    info.insert("bind_cost".into(), err_of(&st.bind_native_movement_cost_rows(&a.cost_rows)));
    st.native_command_book = a.book.clone();
    st.native_command_resistances = a.resist.clone();
    (st.command_learn, st.command_learn_selectors) = (Some(a.learn.clone()), Some(a.learn_sel.clone()));
    st.native_growth_rows = Some(a.growth.clone());
    st.spell_book = a.spells.clone();
    st.ai_command_spell = a.ai_cmd.clone();
    let mut sc = None;
    if !scenario.is_empty() {
        let mut loaded = load_scenario(&ctx.real(scenario)).unwrap_or_else(|e| panic!("{scenario}: {e}"));
        if !loaded.native_death_programs.is_empty() {
            let programs = loaded.clone();
            st.native_death_exp_cancel =
                Some(Callback(Arc::new(move |u: &Unit| programs.native_death_cancels_exp(Some(u)))));
        }
        let r = loaded.setup_checked(Some(&mut st));
        info.insert("setup_err".into(), err_of(&r));
        info.insert("setup_dialogues".into(), json!(r.as_ref().map_or(0, |l| l.len())));
        r.unwrap_or_else(|e| panic!("{scenario}: {e}"));
        sc = Some(loaded);
    }
    rawify(&mut st.units);
    (st, sc, Value::Object(info))
}

// ---- 投影（Go `planProj` 等）----

fn recovery_decision(d: &Option<NativeAIIdleRecoveryDecision>) -> Value {
    match d {
        None => Value::Null,
        Some(d) => json!([d.unit, d.accepted, d.current_hp, d.maximum_hp, d.next_hp, d.raw_gate25, d.raw_gate26]),
    }
}

fn plan_proj(p: &AIPlan) -> Value {
    let mut o = json!({
        "u": idx(p.u), "path": cells(&p.path), "target": idx(p.target), "spell": p.spell_id,
        "kind": p.native_action_kind.0, "cmd": p.native_command_id, "slot": p.native_item_slot, "item": p.native_item_id,
        "item_targets": p.native_item_target_indices, "dest": cell(p.native_action_destination),
        "intended": cell(p.native_mode_intended), "cands": cells(&p.native_mode_candidates),
        "blocked": cell(p.native_mode_blocked_cell), "blocked_found": p.native_mode_blocked_found,
        "opposite": cells(&p.native_mode_opposite), "route": p.native_ai14_ef0_route.0, "score": p.native_action_score,
        "fb_active": p.native_mode_fallback_active, "fb": p.native_mode_fallback, "wb5": p.native_mode_write_byte5,
        "rz": p.native_mode_write_range_zero, "focus": p.native_mode_focus_actor, "ev": p.native_mode_event_active,
        "ev_id": p.native_mode_event_id, "ev_dest": cell(p.native_mode_event_destination), "m2p": p.native_mode2_physical,
        "idle": recovery_decision(&p.native_idle_recovery), "fb_idle": recovery_decision(&p.native_fallback_idle_recovery),
        "err": p.native_error.as_ref().map(|e| e.to_string()), "scored": p.native_scored_commands,
    });
    if !p.native_mode11_stages.is_empty() {
        o["m11"] = Value::Array(
            p.native_mode11_stages
                .iter()
                .map(|s| {
                    json!({"ord": s.stage.ordinal, "route": s.stage.route.0,
                        "action": s.action.as_ref().map_or(Value::Null, plan_proj),
                        "rec": recovery_decision(&s.recovery)})
                })
                .collect(),
        );
    }
    o
}

fn level_ups(ev: &[LevelUpEvent]) -> Value {
    Value::Array(
        ev.iter()
            .map(|e| {
                let mut row = vec![e.new_lv, e.ap_gain, e.dp_gain, e.dx_gain, e.hp_gain, e.mp_gain];
                row.extend(&e.learned_command_ids);
                json!(row)
            })
            .collect(),
    )
}

fn attack_proj(a: Option<&AttackResult>) -> Value {
    a.map_or(Value::Null, |a| json!([a.amount, a.missed, a.crit, a.exp_gained, level_ups(&a.level_ups)]))
}

/// Go `extras`。
fn extras(st: &State) -> Value {
    let flags: Vec<&String> = st.flags.iter().filter(|(_, v)| **v).map(|(k, _)| k).collect();
    let pending: Vec<i64> = st.pending_groups.iter().filter(|(_, v)| **v).map(|(k, _)| *k).collect();
    let opened: Vec<i64> = st.opened_treasure.iter().flatten().filter(|(_, v)| **v).map(|(k, _)| *k).collect();
    let runtime: Vec<String> = st.native_runtime_records.iter().map(|r| hex(&r.raw)).collect();
    let controls: Vec<[u8; 3]> =
        st.native_turn_event_controls.iter().map(|r| [r.turn, r.event_id, r.raw_camp]).collect();
    let chests: Vec<[i64; 2]> =
        st.native_chest_controls.iter().map(|c| [i64::from(c.raw_type), i64::from(c.value)]).collect();
    json!({
        "turn": st.turn, "round": st.native_round_counter, "range": st.native_map_range_mode,
        "has_range": st.has_native_map_range_mode_state, "flags": flags, "event_state": st.native_event_state,
        "field_raw": hex(&st.native_field_control_raw), "pending": pending, "units": st.units.len(),
        "roster": st.roster.len(), "opened": opened, "exp_acc": st.native_experience_accumulator,
        "runtime": runtime, "controls": controls, "chests": chests,
        "treasures": st.treasures.as_ref().map_or(0, |t| t.len()),
        "view": st.has_native_map_view_state, "proj": st.has_native_runtime_unit_projection,
    })
}

// ---- 無頭驅動（Go `driver`）----

/// 逐步比對的位置：Go 的步驟清單與期望的單位物件、戰場摘要。
struct Expect<'a> {
    steps: &'a [Value],
    next: usize,
    units: Vec<Value>,
    ext: Value,
    rust_prev: Vec<String>,
}

struct Driver<'a> {
    st: State,
    sc: Option<Scenario>,
    rng: GoMathRand,
    nrng: u16,
    result: String,
    stopped: String,
    actions: i64,
    exp: Expect<'a>,
}

/// 第一個差異：步驟編號、該步的 Go 內容摘要與差異說明。
struct Mismatch {
    step: usize,
    kind: String,
    tie: bool,
    detail: String,
}

impl Driver<'_> {
    fn unit_values(&self) -> Vec<Value> {
        self.st.units.iter().map(|u| serde_json::to_value(u).unwrap()).collect()
    }

    fn mismatch(&self, kind: &str, tie: bool, detail: String) -> ! {
        std::panic::panic_any(Mismatch { step: self.exp.next, kind: kind.to_string(), tie, detail })
    }

    /// Go `emit`：組出這一步的 Rust 結果並與 Go 的第 `next` 步比對。
    fn emit(&mut self, kind: &str, body: Value) {
        let mut body = if body.is_null() { json!({}) } else { body };
        body["k"] = json!(kind);
        body["i63"] = json!(self.rng.clone().int63());
        body["nr"] = json!(self.nrng);
        if !self.result.is_empty() {
            body["res"] = json!(self.result);
        }
        if !self.stopped.is_empty() {
            body["stop"] = json!(self.stopped);
        }
        let Some(go) = self.exp.steps.get(self.exp.next) else {
            self.mismatch(kind, false, format!("Rust 多出一步：{body}"));
        };
        let tie = go.get("tie").is_some();
        let mut go_body = go.clone();
        let chg = go_body.as_object_mut().unwrap().remove("chg");
        let ext = go_body.as_object_mut().unwrap().remove("ext");
        go_body.as_object_mut().unwrap().remove("tie");
        if let Some(d) = diff("", &go_body, &body) {
            self.mismatch(kind, tie, format!("步驟內容 {d}；go={go_body}"));
        }
        // 單位：先把 Go 的變化套到期望物件，再比全部單位（Rust 未變且 Go 未變的直接跳過）。
        let mut go_changed = std::collections::BTreeSet::new();
        if let Some(Value::Object(chg)) = chg {
            for (k, entry) in chg {
                let n: usize = k.parse().unwrap();
                go_changed.insert(n);
                if let Some(new) = entry.get("new") {
                    assert_eq!(n, self.exp.units.len(), "新單位不在尾端");
                    self.exp.units.push(new.clone());
                    continue;
                }
                let obj = self.exp.units[n].as_object_mut().unwrap();
                if let Some(Value::Object(set)) = entry.get("set") {
                    for (key, v) in set {
                        obj.insert(key.clone(), v.clone());
                    }
                }
                if let Some(Value::Array(del)) = entry.get("del") {
                    for key in del {
                        obj.remove(key.as_str().unwrap());
                    }
                }
            }
        }
        if self.st.units.len() != self.exp.units.len() {
            self.mismatch(kind, tie, format!("單位數 go={} rust={}", self.exp.units.len(), self.st.units.len()));
        }
        for n in 0..self.st.units.len() {
            let text = serde_json::to_string(&self.st.units[n]).unwrap();
            if n < self.exp.rust_prev.len() && self.exp.rust_prev[n] == text && !go_changed.contains(&n) {
                continue;
            }
            let v: Value = serde_json::from_str(&text).unwrap();
            if let Some(d) = diff(&format!("units[{n}]"), &self.exp.units[n], &v) {
                self.mismatch(kind, tie, d);
            }
            if n < self.exp.rust_prev.len() {
                self.exp.rust_prev[n] = text;
            } else {
                self.exp.rust_prev.push(text);
            }
        }
        if let Some(ext) = ext {
            self.exp.ext = ext;
        }
        if let Some(d) = diff("ext", &self.exp.ext, &extras(&self.st)) {
            self.mismatch(kind, tie, d);
        }
        self.exp.next += 1;
    }

    fn stop(&mut self, reason: String) {
        if self.stopped.is_empty() {
            self.stopped = reason;
        }
    }

    fn check_result(&mut self) {
        if !self.result.is_empty() {
            return;
        }
        let Some(sc) = &self.sc else { return };
        if !sc.native_result_rules.is_empty() {
            let mut code = match native_initial_result_code(Some(&self.st), Some(sc)) {
                Ok(c) => c,
                Err(e) => {
                    let reason = format!("result: {e}");
                    self.stop(reason);
                    return;
                }
            };
            for rule in &sc.native_result_rules {
                match rule.r#match(Some(&self.st)) {
                    Err(e) => {
                        let reason = format!("result {}: {e}", rule.id);
                        self.stop(reason);
                        return;
                    }
                    Ok(true) => code = rule.code,
                    Ok(false) => {}
                }
            }
            self.result = match code {
                1 => "lose",
                2 => "win",
                _ => "",
            }
            .to_string();
            return;
        }
        let alive = self.st.units.iter().any(|u| {
            u.alive() && (u.name == "索爾" || (u.camp == OWN && u.has_native_identity && u.native_identity == 0))
        });
        if !alive {
            self.result = "lose".into();
            return;
        }
        if sc.native_result_code1(Some(&self.st)) {
            self.result = "lose".into();
            return;
        }
        self.result = self.st.result("");
    }

    fn walk(&mut self, u: UnitIdx, path: &[Cell]) {
        let unit = &mut self.st.units[u];
        for seg in path.windows(2) {
            let (a, b) = (seg[0], seg[1]);
            let pose = dir_toward(a.x, a.y, b.x, b.y);
            for tick in 1..=6 {
                (unit.x, unit.y) = (a.x, a.y);
                unit.set_map_pose(pose);
                unit.set_native_map_grid_motion(pose, tick);
            }
            if !unit.finish_native_map_grid_step(pose, b.x, b.y) {
                unit.set_map_placement(b.x, b.y, pose);
            }
        }
        let last = path[path.len() - 1];
        unit.set_map_placement(last.x, last.y, 0);
    }

    fn finish_action(&mut self, u: UnitIdx) {
        self.st.mark_native_dead_records();
        for x in &mut self.st.units {
            if x.has_native_map_presentation {
                x.set_map_pose(0);
            }
        }
        self.st.units[u].acted = true;
    }

    fn idle_recovery(&mut self, u: UnitIdx, decision: NativeAIIdleRecoveryDecision) -> Value {
        let records = match native_ai_scoring_records(&self.st.units) {
            Ok(r) => r,
            Err(e) => {
                self.stop(format!("idle recovery records: {e}"));
                return json!({"err": e.to_string()});
            }
        };
        let current = plan_native_ai_idle_recovery(&records, self.st.units.len() as i64, u as i64);
        match &current {
            Ok(c) if *c == decision => {}
            _ => {
                self.stop("idle recovery preflight".into());
                let changed = current.as_ref().map_or(true, |c| *c != decision);
                return json!({"err": err_of(&current), "changed": changed});
            }
        }
        let presentation = match build_native_ai_idle_recovery_presentation(decision) {
            Ok(p) => p,
            Err(e) => {
                self.stop(format!("idle recovery presentation: {e}"));
                return json!({"err": e.to_string()});
            }
        };
        self.st.units[u].hp = i64::from(presentation.after_hp);
        (self.st.native_map_range_mode, self.st.has_native_map_range_mode_state) =
            (i64::from(presentation.after_global_write.value), true);
        json!({"before": presentation.before_hp, "after": presentation.after_hp})
    }

    fn attack(&mut self, u: UnitIdx, tgt: UnitIdx) -> Value {
        let (ux, uy, tx, ty) = (self.st.units[u].x, self.st.units[u].y, self.st.units[tgt].x, self.st.units[tgt].y);
        self.st.units[u].set_map_pose(dir_toward(ux, uy, tx, ty));
        let rng: &mut dyn GoRand = &mut self.rng;
        let r = self.st.attack_native_physical_with_experience(Some(u), Some(tgt), self.nrng, Some(rng));
        match r {
            Err(e) => {
                self.stop(format!("attack: {e}"));
                json!({"err": e.to_string()})
            }
            Ok(result) => {
                self.nrng = result.rng_state;
                json!({"atk": attack_proj(Some(&result.attack)), "strikes": result.attack_strikes.len(),
                    "ctr": attack_proj(result.counter.as_ref()), "ctr_strikes": result.counter_strikes.len(),
                    "ctr_exp": result.counter_exp_gained, "ctr_lv": level_ups(&result.counter_level_ups)})
            }
        }
    }

    fn spell(&mut self, plan: &AIPlan) -> Value {
        let (Some(actor), Some(target)) = (plan.u, plan.target) else {
            self.stop("spell: context".into());
            return json!({"err": "context"});
        };
        let (a, t) = (&self.st.units[actor], &self.st.units[target]);
        let invalid = !a.on_field || !a.alive() || a.acted || a.sealed || !t.on_field || !t.alive();
        let (ax, ay, tx, ty, mp) = (a.x, a.y, t.x, t.y, a.mp);
        if invalid {
            self.stop("spell: actor or target invalid".into());
            return json!({"err": "invalid"});
        }
        let sp = self.st.spell_book.iter().find(|sp| sp.id == plan.spell_id).cloned();
        let Some(sp) = sp.filter(|sp| mp >= sp.mp && sp.mp >= 0) else {
            self.stop("spell: row or MP".into());
            return json!({"err": "row"});
        };
        let valid = self.st.ai_spell_candidates(Some(actor), &sp).contains(&target);
        if !valid || manhattan(ax, ay, tx, ty) > sp.dist {
            self.stop("spell: target".into());
            return json!({"err": "target"});
        }
        let results = self.st.cast_area(Some(actor), tx, ty, &sp, &mut self.rng);
        if results.is_empty() {
            self.st.units[actor].mp += sp.mp;
            self.stop("spell: no result".into());
            return json!({"err": "no result"});
        }
        let out: Vec<Value> = results
            .iter()
            .map(|r| json!([idx(r.target), r.amount, r.missed, r.exp_gained, level_ups(&r.level_ups)]))
            .collect();
        let (ax, ay) = (self.st.units[actor].x, self.st.units[actor].y);
        self.st.units[actor].set_map_pose(dir_toward(ax, ay, tx, ty));
        self.finish_action(actor);
        self.check_result();
        json!({"cast": out})
    }

    fn native_action(&mut self, plan: &AIPlan) -> Value {
        let actor = plan.u.unwrap();
        let Some(target) = plan.target else {
            self.stop("native action: no target".into());
            return json!({"err": "no target"});
        };
        let mut out = json!({});
        if plan.native_action_kind == NATIVE_AI_ACTION_COMMAND && (13..=16).contains(&plan.native_command_id) {
            let origin = plan.native_action_destination;
            if let Err(e) = self.st.native_ai_command_heal_targets(Some(actor), origin, plan.native_command_id) {
                self.stop(format!("heal targets: {e}"));
                return json!({"err": e.to_string()});
            }
            match self.st.execute_native_ai_command_heal_native(Some(actor), origin, plan.native_command_id, self.nrng)
            {
                Err(e) => {
                    self.stop(format!("heal: {e}"));
                    return json!({"err": e.to_string()});
                }
                Ok((results, next)) => {
                    self.nrng = next;
                    let heals: Vec<[i64; 2]> = results.iter().map(|r| [idx(r.target), r.restore.actual]).collect();
                    out["heal"] = json!(heals);
                }
            }
        } else {
            out["skipped"] = json!(true);
        }
        self.st.native_experience_accumulator = 0;
        let (ax, ay, tx, ty) =
            (self.st.units[actor].x, self.st.units[actor].y, self.st.units[target].x, self.st.units[target].y);
        self.st.units[actor].set_map_pose(dir_toward(ax, ay, tx, ty));
        self.finish_action(actor);
        self.check_result();
        out
    }

    fn set_range_zero(&mut self) {
        (self.st.native_map_range_mode, self.st.has_native_map_range_mode_state) = (0, true);
    }

    fn act(&mut self, plan: &AIPlan) -> Value {
        if plan.native_mode_fallback_active && plan.native_mode_fallback == 8 && plan.native_error.is_none() {
            return json!({"o": "skip8"});
        }
        if let Some(e) = &plan.native_error {
            self.stop(format!("native error: {e}"));
            return json!({"o": "native_error"});
        }
        self.actions += 1;
        let u = plan.u.unwrap();
        if !plan.native_mode11_stages.is_empty() {
            let mut stages = Vec::new();
            for stage in &plan.native_mode11_stages {
                if !self.stopped.is_empty() || !self.result.is_empty() {
                    break;
                }
                if let Some(rec) = stage.recovery {
                    stages.push(self.idle_recovery(u, rec));
                    continue;
                }
                let Some(action) = &stage.action else {
                    stages.push(json!("none"));
                    continue;
                };
                if action.native_mode_write_range_zero {
                    self.set_range_zero();
                }
                let au = action.u.unwrap();
                if action.path.len() >= 2 {
                    self.walk(au, &action.path);
                }
                match action.native_action_kind {
                    NATIVE_AI_ACTION_COMMAND | NATIVE_AI_ACTION_ITEM => stages.push(self.native_action(action)),
                    NATIVE_AI_ACTION_PHYSICAL => {
                        let Some(t) = action.target.filter(|t| self.st.units[*t].alive()) else {
                            self.stop("mode 11 physical: target".into());
                            stages.push(json!("bad target"));
                            continue;
                        };
                        let r = self.attack(au, t);
                        self.check_result();
                        if self.stopped.is_empty() {
                            self.finish_action(au);
                        }
                        stages.push(r);
                    }
                    _ => {
                        self.finish_action(au);
                        stages.push(json!("finish"));
                    }
                }
            }
            if self.stopped.is_empty() {
                self.finish_action(u);
            }
            return json!({"o": "mode11", "stages": stages});
        }
        if let Some(rec) = plan.native_idle_recovery {
            let r = self.idle_recovery(u, rec);
            if self.stopped.is_empty() {
                self.finish_action(u);
            }
            return json!({"o": "idle", "r": r});
        }
        let mut out = json!({"o": "act"});
        if plan.path.len() >= 2 {
            self.walk(u, &plan.path);
        } else if plan.native_mode_focus_actor
            && plan.target.is_none()
            && plan.native_mode_write_range_zero
            && plan.native_mode_fallback != 3
        {
            self.set_range_zero();
        }
        if plan.native_mode_write_range_zero {
            self.set_range_zero();
        }
        if plan.spell_id >= 0 {
            out["spell"] = self.spell(plan);
            return out;
        }
        if plan.native_action_kind == NATIVE_AI_ACTION_COMMAND || plan.native_action_kind == NATIVE_AI_ACTION_ITEM {
            out["native"] = self.native_action(plan);
            return out;
        }
        if let Some(t) = plan.target.filter(|t| self.st.units[*t].alive()) {
            out["attack"] = self.attack(u, t);
            if !self.stopped.is_empty() {
                return out;
            }
            self.check_result();
        }
        // finish
        if plan.native_mode_event_active {
            let mut cues: Vec<[i64; 4]> = Vec::new();
            let mut push = |c: NativeAIMode5AudioCue| {
                cues.push([i64::from(c.handle_linear_address), c.resource_id, c.index, c.loop_count])
            };
            let emit: &mut dyn FnMut(NativeAIMode5AudioCue) = &mut push;
            let r = self.st.apply_native_ai_mode5_event_with_audio_cue(
                Some(u),
                plan.native_mode_event_id,
                plan.native_mode_event_destination,
                Some(emit),
            );
            out["mode5"] = json!({"err": err_of(&r), "cues": cues});
            if let Err(e) = r {
                self.stop(format!("mode5: {e}"));
                return out;
            }
        }
        if let Some(rec) = plan.native_fallback_idle_recovery {
            out["fb_idle"] = self.idle_recovery(u, rec);
            if !self.stopped.is_empty() {
                return out;
            }
        }
        if plan.native_mode_write_byte5 {
            (self.st.units[u].native_record_byte5, self.st.units[u].has_native_record_byte5) = (1, true);
        }
        self.finish_action(u);
        out
    }

    fn events(&mut self, selector: i64) {
        if !self.stopped.is_empty() {
            return;
        }
        let Some(sc) = self.sc.as_mut() else { return };
        let mut pick = |e: &Event| e.native_turn_phase_selector() == selector;
        let accept: &mut dyn FnMut(&Event) -> bool = &mut pick;
        let mut actions = sc.trigger_actions_where(Some(&self.st), "on_turn_end", "", Some(accept));
        if selector >= 0 {
            for event in sc.native_turn_action_events_at(Some(&self.st), selector as u8) {
                actions.extend(event.actions);
            }
        }
        if actions.is_empty() {
            return;
        }
        let mut outcomes = Vec::new();
        for action in &actions {
            let outcome = self.run_action(action);
            outcomes.push(json!([action.r#type, outcome]));
            if !self.stopped.is_empty() {
                break;
            }
        }
        rawify(&mut self.st.units);
        self.emit("events", json!({"sel": selector, "acts": outcomes}));
    }

    fn run_action(&mut self, action: &Action) -> String {
        if let Some(when) = &action.native_when {
            match when.r#match(Some(&self.st)) {
                Err(e) => {
                    self.stop(format!("when: {e}"));
                    return "when_err".into();
                }
                Ok(false) => return "when_false".into(),
                Ok(true) => {}
            }
        }
        if action.r#type == "dialogue" && action.native_dialogue_ref.is_some() {
            return "dialogue".into();
        }
        match action.r#type.as_str() {
            "native_death_op" => {
                let Some(op) = action.native_death_op.as_ref().filter(|_| !action.native_source.is_empty()) else {
                    self.stop("death op source".into());
                    return "death_op_err".into();
                };
                if op.op == "staging" || op.op == "clear_hp_from" || op.op == "reward" {
                    return "death_op_presented".into();
                }
                if let Err(e) = self.st.apply_native_death_op(op) {
                    self.stop(format!("death op: {e}"));
                    return "death_op_err".into();
                }
                return "death_op".into();
            }
            "reset_pose" => {
                for u in &mut self.st.units {
                    u.set_map_pose(0);
                }
                return "reset_pose".into();
            }
            "redraw" | "pan" | "delay" | "native_acting" | "native_staging_present" => return "presented".into(),
            "native_range_zero" | "native_range_one" => {
                if action.native_source.is_empty() {
                    self.stop("range source".into());
                    return "range_err".into();
                }
                let value = i64::from(action.r#type == "native_range_one");
                (self.st.native_map_range_mode, self.st.has_native_map_range_mode_state) = (value, true);
                return "range".into();
            }
            _ => {}
        }
        if action.native_spawns.iter().any(|c| c.via == "spawn_group_with_intro") {
            return "intro".into();
        }
        let sc = self.sc.as_mut().unwrap();
        match sc.execute_action_checked(Some(&mut self.st), action) {
            Err(e) => {
                self.stop(format!("action: {e}"));
                "err".into()
            }
            Ok((dl, is_dialogue)) => {
                sc.take_party_joins();
                if is_dialogue { format!("dialogue_line:{}", dl.speaker) } else { "exec".into() }
            }
        }
    }

    fn transient(&mut self, selector: u8) {
        if !self.stopped.is_empty() || self.st.units.is_empty() {
            return;
        }
        if !self.st.has_native_runtime_unit_projection
            && self.st.units.iter().any(|u| !u.has_native_record_byte5 || !u.has_native_record_byte6)
        {
            return;
        }
        let r = self.st.advance_native_transient_phase_typed(selector);
        let mut body = json!({"sel": selector, "err": err_of(&r)});
        match r {
            Err(e) => self.stop(format!("transient: {e}")),
            Ok(result) => {
                let dmg: Vec<[i64; 5]> =
                    result.damage.iter().map(|x| [idx(x.unit), x.slot, x.amount, x.before, x.after]).collect();
                let exp: Vec<[i64; 2]> = result.expired.iter().map(|x| [idx(x.unit), x.offset]).collect();
                body["dmg"] = json!(dmg);
                body["expired"] = json!(exp);
                body["inactive"] = json!(result.inactive_slots);
            }
        }
        self.emit("transient", body);
    }

    fn phase(&mut self, ally: bool, max_plans: usize) {
        for n in 0.. {
            if !self.stopped.is_empty() || !self.result.is_empty() {
                return;
            }
            if n >= max_plans {
                self.stop("phase plan limit".into());
                self.emit("limit", Value::Null);
                return;
            }
            let plan = if ally {
                let (plan, handled) = self.st.next_ally_ai_plan();
                match plan.filter(|_| handled) {
                    Some(p) => p,
                    None => {
                        self.emit("ally_done", json!({"h": handled}));
                        return;
                    }
                }
            } else {
                match self.st.next_ai_plan() {
                    Some(p) => p,
                    None => {
                        self.emit("enemy_done", Value::Null);
                        return;
                    }
                }
            };
            let mut body = json!({"ally": ally, "p": plan_proj(&plan)});
            body["a"] = self.act(&plan);
            self.emit("ai", body);
        }
    }

    fn run(&mut self, turns: i64, max_plans: usize) {
        self.check_result();
        self.emit("init_result", Value::Null);
        for turn in 1..=turns {
            if !self.result.is_empty() || !self.stopped.is_empty() {
                return;
            }
            let recs: Vec<[i64; 4]> = self
                .st
                .apply_native_end_turn_recovery()
                .iter()
                .map(|r| [idx(r.unit), r.before, r.after, r.max_hp])
                .collect();
            self.emit("end", json!({"turn": turn, "rec": recs}));
            self.events(1);
            self.transient(1);
            if !self.stopped.is_empty() {
                return;
            }
            self.st.reset_native_ai_scan();
            self.phase(true, max_plans);
            if !self.stopped.is_empty() || !self.result.is_empty() {
                return;
            }
            for u in &mut self.st.units {
                if u.has_native_record_byte5 {
                    u.native_record_byte5 &= 0x7f;
                }
                u.acted = false;
            }
            self.emit("clear_bit7", Value::Null);
            self.events(0);
            self.transient(0);
            self.phase(false, max_plans);
            if !self.stopped.is_empty() || !self.result.is_empty() {
                return;
            }
            self.events(-1);
            self.st.turn += 1;
            if self.st.native_round_counter > 0 {
                self.st.native_round_counter += 1;
            }
            let view = self.st.has_native_map_view_state;
            for u in &mut self.st.units {
                u.acted = false;
                if view && u.has_native_record_byte5 {
                    u.native_record_byte5 &= 0x7f;
                }
                u.tick_status();
            }
            self.emit("complete_turn", Value::Null);
            self.transient(2);
            self.events(2);
            self.check_result();
            self.emit("player_phase", Value::Null);
        }
    }
}

// ---- 進入點 ----

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_INT").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let (Some(path), Some(repo)) = (path, repo) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_RULES_INT 或 FD2_GOLDEN_REPO");
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_INT／FD2_GOLDEN_REPO");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    Some((doc, Ctx { repo }))
}

/// 一場戰鬥：建置、比對初始狀態、逐步重播，最後比對結局與快照。
fn replay(ctx: &Ctx, a: &Assets, doc: &Value, battle: &Value) -> i64 {
    let (st, sc, info) = setup(ctx, a, s(battle, "map"), s(battle, "scenario"));
    let init = &battle["init"];
    if let Some(d) = diff("init.info", &init["info"], &info) {
        std::panic::panic_any(Mismatch { step: 0, kind: "init".into(), tie: false, detail: d });
    }
    let rng = GoMathRand::new(i(battle, "seed"));
    let mut driver = Driver {
        st,
        sc,
        rng,
        nrng: i(battle, "native_rng") as u16,
        result: String::new(),
        stopped: String::new(),
        actions: 0,
        exp: Expect {
            steps: battle["steps"].as_array().unwrap(),
            next: 0,
            units: init["units"].as_array().unwrap().clone(),
            ext: init["ext"].clone(),
            rust_prev: Vec::new(),
        },
    };
    let check_init = [
        ("init.units", json!(driver.unit_values())),
        ("init.ext", extras(&driver.st)),
        ("init.i63", json!(driver.rng.clone().int63())),
    ];
    for (label, rust) in check_init {
        let go = match label {
            "init.units" => &init["units"],
            "init.ext" => &init["ext"],
            _ => &init["i63"],
        };
        if let Some(d) = diff(label, go, &rust) {
            std::panic::panic_any(Mismatch { step: 0, kind: "init".into(), tie: false, detail: d });
        }
    }
    driver.exp.rust_prev = driver.st.units.iter().map(|u| serde_json::to_string(u).unwrap()).collect();
    driver.run(i(doc, "max_turns"), i(doc, "max_phase_plans") as usize);
    if driver.exp.next != driver.exp.steps.len() {
        let next = driver.exp.next;
        std::panic::panic_any(Mismatch {
            step: next,
            kind: "end".into(),
            tie: false,
            detail: format!("Rust 少了步驟：Rust {next} 步，Go {} 步", driver.exp.steps.len()),
        });
    }
    let end = &battle["end"];
    let mut rust_end = json!({"result": driver.result, "stopped": driver.stopped, "turn": driver.st.turn,
        "actions": driver.actions});
    match driver.st.snapshot() {
        Ok(snap) => rust_end["snapshot"] = serde_json::to_value(snap).unwrap(),
        Err(e) => rust_end["snapshot_err"] = json!(e.to_string()),
    }
    if let Some(d) = diff("end", end, &rust_end) {
        std::panic::panic_any(Mismatch { step: driver.exp.next, kind: "end".into(), tie: false, detail: d });
    }
    driver.exp.steps.len() as i64
}

fn march_step(step: &NativeSystemGroupMarchStep) -> Value {
    let events: Vec<Value> =
        step.events.iter().map(|e| json!([e.path_index, e.event_id, e.text_index, e.presentation])).collect();
    json!({"u": step.unit_index, "path": cells(&step.path), "events": events})
}

#[test]
fn golden_rules_int() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-int/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    for file in doc["inputs"].as_array().unwrap() {
        let raw = std::fs::read(ctx.real(s(file, "path"))).unwrap_or_else(|e| panic!("讀不到 {}：{e}", file["path"]));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    let assets = load_assets(&ctx, &doc);

    // 系統群行軍：同一戰場的目的格依序在同一個 State 上呼叫（與 Go 相同）。
    let mut march_count = 0;
    let mut current: Option<(String, State, String)> = None;
    for case in doc["march"].as_array().unwrap() {
        let name = s(case, "field").to_string();
        if current.as_ref().is_none_or(|(n, _, _)| *n != name) {
            let (st, _, _) = setup(&ctx, &assets, s(case, "map"), s(case, "scenario"));
            let before = serde_json::to_string(&st.units).unwrap();
            current = Some((name.clone(), st, before));
        }
        let (_, st, before) = current.as_mut().unwrap();
        let d = &case["dest"];
        let dest = Cell { x: d[0].as_i64().unwrap(), y: d[1].as_i64().unwrap() };
        let r = st.plan_native_system_group_march(dest);
        let plan = r.as_ref().cloned().unwrap_or(NativeSystemGroupMarchPlan { destination: dest, steps: Vec::new() });
        let after = serde_json::to_string(&st.units).unwrap();
        let got = json!({"field": name, "map": case["map"], "scenario": case["scenario"], "dest": cell(dest),
            "err": err_of(&r), "plan_dest": cell(plan.destination),
            "steps": plan.steps.iter().map(march_step).collect::<Vec<_>>(), "units_changed": after != *before});
        if let Some(diff) = diff("", case, &got) {
            panic!("march {name} {dest:?}：{diff}");
        }
        march_count += 1;
    }

    // 戰鬥重播。
    let battles = doc["battles"].as_array().unwrap();
    let (mut steps, mut failures, mut ties) = (0i64, Vec::new(), Vec::new());
    let mut results = BTreeMap::<String, usize>::new();
    let quiet = std::panic::take_hook();
    std::panic::set_hook(Box::new(|info| {
        if info.payload().downcast_ref::<Mismatch>().is_none() {
            eprintln!("{info}");
        }
    }));
    for battle in battles {
        let name = format!("{} seed={}", s(battle, "name"), battle["seed"]);
        match std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| replay(&ctx, &assets, &doc, battle))) {
            Ok(n) => {
                steps += n;
                *results.entry(s(&battle["end"], "result").to_string()).or_default() += 1;
            }
            Err(payload) => {
                let Some(m) = payload.downcast_ref::<Mismatch>() else {
                    failures.push(format!("{name}：非比對的 panic（見上方訊息）"));
                    continue;
                };
                let go_step = &battle["steps"][m.step];
                let line = format!(
                    "{name} 第 {} 步（{}）：{}；Go 計畫 {}",
                    m.step,
                    m.kind,
                    m.detail,
                    go_step.get("p").map_or("無".to_string(), |p| p.to_string())
                );
                if m.tie { ties.push(line) } else { failures.push(line) }
            }
        }
    }
    std::panic::set_hook(quiet);
    eprintln!(
        "戰鬥 {} 場、步驟 {steps}、勝負 {results:?}；系統群行軍 {march_count} 案；reach 平手造成的分歧 {} 場",
        battles.len(),
        ties.len()
    );
    for line in &ties {
        eprintln!("平手（KNOWN_DIFFERENCES）：{line}");
    }
    for line in &failures {
        eprintln!("不一致：{line}");
    }
    assert!(failures.is_empty(), "{} 場戰鬥與 Go 不一致", failures.len());
}
