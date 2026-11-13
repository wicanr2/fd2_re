//! 與凍結 Go 版 `remake/cmd/fd2-golden-rules-f` 的黃金差分：fd2-rules F 組（劇本事件、回合與快照）。
//!
//! 由 `tools/rust_golden_rules_f.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_RULES_F`；真實輸入（受版控的
//! `remake/assets`）掛在 `FD2_GOLDEN_REPO` 底下，先核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：黃金檔的每個腳本是一串 `op`（輸入 `args` 與 Go 結果 `result`）。Rust 端依序重做同一串
//! 操作，每一步都把結果組成與 Go 相同鍵集合的 JSON，逐欄比對。F 組與 A、B 組自己的錯誤文字逐字相同；
//! Go 標準函式庫產生的錯誤（JSON 解碼）只比接受或拒收。
//!
//! 快照雙向：每個 `snapshot` 步驟（1）把 Go 寫出的快照 JSON 讀成 `StateSnapshot` 再寫出，（2）把 Rust
//! 自己算出的戰場 `snapshot()` 寫出，兩者都要與 Go 原文相同；（3）讀入的快照 `restore_snapshot` 到新的
//! `State` 再 `snapshot()` 也要相同。兩份 Rust 輸出寫進 `FD2_GOLDEN_RULES_F_RUST_OUT`，由 Go 工具的
//! `-verify-rust` 讀回、RestoreSnapshot 後重新 Snapshot 與黃金比對。
//!
//! 比對時的表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在 [`KNOWN_DIFFERENCES`]。

use fd2_rules::battle::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為（PORTING.md 決定 2、`rust-port-no-go-stdlib`）。黃金案例都不落在這些輸入上。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文。
    "JSON 欄位名大小寫",
    // Go 讀 JSON null 到 int、bool 或結構欄位時保留零值不報錯；Rust 的純量欄位拒收 null。
    // 重製端資產都由工具寫出，不會在純量位置寫 null；合成變異不產生這種輸入。
    "純量欄位的 null",
    // LoadScenario 的 JSON 解碼錯誤屬標準函式庫，只比接受或拒收。
    "劇本 JSON 解碼錯誤文字",
    // Go 快照的 []*Unit 可以含 nil 元素；Rust 的 Vec<Unit> 沒有 nil，讀到 null 元素會拒收。
    "快照單位陣列的 nil 元素",
    // Go 以 strings.EqualFold 比 handler 位址字串（Unicode 大小寫摺疊）；Rust 用 ASCII 大小寫比較，
    // 位址字串只含 ASCII，結果相同。
    "handler 字串的非 ASCII 大小寫摺疊",
    // Go 可以把不在 State.Units 的 *Unit 當觸發者；Rust 以索引表示單位，黃金只用 State.Units 裡的單位。
    "不在 State.Units 的觸發者",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片／map 寫成 null、空的寫成 []／{}，nil []byte 寫 null、空的寫 ""；Rust 的 Vec／BTreeMap
    // 不分 nil，null、[]、{}、"" 視為相同。
    "null 與空陣列、空物件、空字串相同",
    // Go 把整數值的 float64 寫成 0，serde_json 寫成 0.0；只比數值。
    "數字只比數值",
    // Go 的 map 以字串鍵排序輸出，serde_json 的 BTreeMap 依整數鍵排序；物件只比鍵值集合。
    "物件鍵順序",
];

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

fn ints(v: &Value) -> Vec<i64> {
    v.as_array().map(|a| a.iter().map(|x| x.as_i64().unwrap()).collect()).unwrap_or_default()
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

/// 依 [`NORMALIZATIONS`] 比對兩個 JSON 值，回傳第一個不同的位置。物件的鍵集合也要相同。
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

/// Go `stateDigest`：每一步之後比對的戰場摘要。
fn state_digest(st: &State) -> Value {
    let b2i = |b: bool| i64::from(b);
    let units: Vec<Vec<i64>> = st
        .units
        .iter()
        .map(|u| {
            let mut row = vec![
                u.x,
                u.y,
                u.hp,
                u.max_hp,
                u.group,
                u.camp.0,
                b2i(u.on_field),
                b2i(u.acted),
                u.native_record_byte5 as i64,
                b2i(u.has_native_record_byte5),
                u.native_record_byte6 as i64,
                b2i(u.has_native_record_byte6),
                u.native_record_byte34 as i64,
                u.fig,
                u.map_selector_slot,
                b2i(u.has_map_selector_slot),
                u.inventory.len() as i64,
            ];
            row.extend(&u.inventory);
            row
        })
        .collect();
    let roster: Vec<i64> = st.roster.iter().map(|u| u.group).collect();
    let controls: Vec<[i64; 3]> =
        st.native_turn_event_controls.iter().map(|r| [r.turn as i64, r.event_id as i64, r.raw_camp as i64]).collect();
    let flags: Vec<&String> = st.flags.iter().filter(|(_, v)| **v).map(|(k, _)| k).collect();
    let pending: Vec<i64> = st.pending_groups.iter().filter(|(_, v)| **v).map(|(k, _)| *k).collect();
    let runtime: Vec<i64> =
        st.native_runtime_records.iter().flat_map(|r| [r.raw[5] as i64, r.raw[0x34] as i64]).collect();
    let selector_error = st.native_map_selector_error.as_ref().map(|e| e.to_string()).unwrap_or_default();
    json!({
        "units": units, "roster": roster, "turn": st.turn, "round": st.native_round_counter,
        "event_state": st.native_event_state.to_vec(), "controls": controls, "flags": flags,
        "pending": pending, "field_raw": hex(&st.native_field_control_raw),
        "runtime": runtime, "selector_error": selector_error,
    })
}

fn dialogues(lines: &[DialogLine]) -> Value {
    json!(lines.iter().map(|l| json!([l.speaker, l.text])).collect::<Vec<_>>())
}

/// Rust 寫出的快照，交給 Go 的 `-verify-rust` 讀回。
#[derive(Default)]
struct RustOut {
    snapshots: Vec<Value>,
}

struct Runner<'a> {
    script: usize,
    name: &'a str,
    st: State,
    sc: Option<Scenario>,
    pending: Vec<Action>,
    rows: &'a [u8],
}

impl Runner<'_> {
    fn sc(&mut self) -> &mut Scenario {
        self.sc.as_mut().expect("劇本腳本才有 Scenario")
    }

    fn exec_one(&mut self, index: usize) -> Value {
        let action = self.pending[index].clone();
        if let Some(when) = &action.native_when {
            match when.r#match(Some(&self.st)) {
                Err(e) => return json!({"when_err": e.to_string()}),
                Ok(false) => return json!({"skipped": true}),
                Ok(true) => {}
            }
        }
        if action.r#type == "native_death_op"
            && let Some(op) = &action.native_death_op
        {
            return json!({"death_err": err_of(&self.st.apply_native_death_op(op))});
        }
        let sc = self.sc.as_mut().expect("劇本腳本才有 Scenario");
        let r = sc.execute_action_checked(Some(&mut self.st), &action);
        let (dl, is_dialogue) = r.as_ref().cloned().unwrap_or_default();
        json!({"speaker": dl.speaker, "text": dl.text, "is_dialogue": is_dialogue, "err": err_of(&r)})
    }

    fn field_cells(&self) -> Value {
        let mut cells = Vec::new();
        for y in 0..self.st.h {
            for x in 0..self.st.w {
                for sel in 0..3u8 {
                    if let (id, true) = native_field_event_id_at(Some(&self.st), x, y, sel) {
                        cells.push([x, y, sel as i64 * 256 + id as i64]);
                    }
                }
            }
        }
        json!(cells)
    }

    fn field(&mut self, args: &Value) -> Value {
        let (x, y, sel, unit) = (i(args, "x"), i(args, "y"), i(args, "selector") as u8, i(args, "unit") as usize);
        let st = &mut self.st;
        match s(args, "kind") {
            "id" => {
                let (id, ok) = native_field_event_id_at(Some(st), x, y, sel);
                json!({"id": id, "ok": ok})
            }
            "mode" => {
                let (id, ok) = apply_native_field_mode_event(Some(st), Some(unit), x, y, sel);
                json!({"id": id, "ok": ok})
            }
            "activation" => {
                let r = apply_native_field_turn_activation_event(Some(st), x, y, sel);
                json!({"id": r.as_ref().copied().unwrap_or(0), "err": err_of(&r)})
            }
            "ch28" => {
                let r = apply_native_chapter28_field_turn_activation(Some(st), Some(unit), x, y, sel);
                json!({"id": r.as_ref().copied().unwrap_or(0), "err": err_of(&r)})
            }
            "e61" => {
                let plan = plan_native_field_event61(Some(st), Some(unit), x, y);
                let mut out = json!({"err": err_of(&plan)});
                if let Ok(plan) = plan {
                    out["plan"] = json!({
                        "event_id": plan.event_id, "missing_item": plan.missing_item,
                        "text_index": plan.text_index, "final_text": plan.final_text,
                        "presentation": serde_json::to_value(&plan.presentation).unwrap(),
                        "join_character": plan.join_character,
                    });
                    if !plan.missing_item {
                        let r = commit_native_field_event61(Some(st), &plan, i(args, "frames"));
                        out["joined"] = json!(r.as_ref().copied().unwrap_or(0));
                        out["commit_err"] = err_of(&r);
                    }
                }
                out
            }
            "e75" => {
                let plan = plan_native_field_event75(Some(st), Some(unit), x, y);
                let mut out = json!({"err": err_of(&plan)});
                if let Ok(plan) = plan {
                    out["plan"] = json!({"event_id": plan.event_id, "text_index": plan.text_index,
                        "activate": plan.activate, "noop": plan.noop});
                    if plan.activate {
                        out["commit_err"] = err_of(&commit_native_field_event75(Some(st), &plan));
                    }
                }
                out
            }
            kind => panic!("未知的 field kind {kind}"),
        }
    }

    fn result(&mut self) -> Value {
        let st = &self.st;
        let default = native_default_result_code(Some(st));
        let initial = native_initial_result_code(Some(st), self.sc.as_ref());
        let mut out = json!({
            "default": {"code": default.as_ref().copied().unwrap_or(0), "err": err_of(&default)},
            "initial": {"code": initial.as_ref().copied().unwrap_or(0), "err": err_of(&initial)},
            "rules": [],
        });
        if let Some(sc) = &self.sc {
            let rules: Vec<Value> = sc
                .native_result_rules
                .iter()
                .map(|rule| {
                    let r = rule.r#match(Some(st));
                    json!({"matched": r.as_ref().copied().unwrap_or(false), "err": err_of(&r)})
                })
                .collect();
            out["rules"] = json!(rules);
            out["code1"] = json!(sc.native_result_code1(Some(st)));
        }
        out
    }

    fn snapshot(&mut self, op_index: usize, golden: &Value, out: &mut RustOut) -> Value {
        let ctx = format!("{} op {op_index} snapshot", self.name);
        let own = self.st.snapshot();
        let Ok(own) = own else {
            return json!({"err": err_of(&own)});
        };
        let own_json = serde_json::to_value(&own).unwrap();
        let go_snap = &golden["snapshot"];
        if go_snap.is_object() {
            // （1）Go 寫的快照 Rust 讀入再寫出。
            let read: StateSnapshot =
                serde_json::from_value(go_snap.clone()).unwrap_or_else(|e| panic!("{ctx}：Rust 讀不進 Go 快照：{e}"));
            let reread = serde_json::to_value(&read).unwrap();
            check(&format!("{ctx}（讀入再寫出）"), go_snap, &reread);
            // （2）Rust 自己算出的快照。
            check(&format!("{ctx}（Rust 戰場）"), go_snap, &own_json);
            // （3）讀入的快照還原到新的 State 再快照。
            let mut fresh = State::default();
            fresh.restore_snapshot(&read).unwrap_or_else(|e| panic!("{ctx}：restore_snapshot：{e}"));
            let again = serde_json::to_value(fresh.snapshot().unwrap()).unwrap();
            check(&format!("{ctx}（還原後再快照）"), go_snap, &again);
            out.snapshots.push(json!({"script": self.script, "op": op_index, "reread": reread, "own": own_json}));
        }
        json!({"err": null, "snapshot": own_json})
    }

    fn run_op(&mut self, op_index: usize, op: &Value, script: &Value, out: &mut RustOut) -> Value {
        let args = &op["args"];
        let go = &op["result"];
        match s(op, "op") {
            "load" => {
                let map = PathBuf::from(script["map_abs"].as_str().unwrap());
                let st = load(&map.to_string_lossy());
                let Ok(st) = st else { return json!({"err": err_of(&st)}) };
                self.st = st;
                let bind = self.st.bind_native_future_item_rows(self.rows);
                let mut result = json!({"err": null, "bind_err": err_of(&bind)});
                if let Some(path) = script["scenario_abs"].as_str() {
                    let sc = load_scenario(path);
                    let Ok(sc) = sc else {
                        result["err"] = err_of(&sc);
                        return result;
                    };
                    result["scenario"] = serde_json::to_value(&sc).unwrap();
                    result["phase_selectors"] =
                        json!(sc.events.iter().map(|e| e.native_turn_phase_selector()).collect::<Vec<_>>());
                    self.sc = Some(sc);
                }
                result["digest"] = state_digest(&self.st);
                result
            }
            "setup" => {
                let sc = self.sc.as_mut().unwrap();
                let r = sc.setup_checked(Some(&mut self.st));
                let dl = r.as_ref().cloned().unwrap_or_default();
                json!({"dialogues": dialogues(&dl), "err": err_of(&r), "digest": state_digest(&self.st)})
            }
            "snapshot" => self.snapshot(op_index, go, out),
            "digest" => json!({"digest": state_digest(&self.st)}),
            "fired" => {
                let sc = self.sc.as_ref().unwrap();
                json!({"refs": serde_json::to_value(sc.fired_events()).unwrap(), "pending_joins": sc.pending_join_count()})
            }
            "field_cells" => json!({"cells": self.field_cells()}),
            "turn" => {
                (self.st.turn, self.st.native_round_counter) = (i(args, "turn"), i(args, "turn"));
                json!({})
            }
            "set_round" => {
                self.st.native_round_counter = i(args, "round");
                json!({})
            }
            "kill" => {
                let u = &mut self.st.units[i(args, "unit") as usize];
                u.hp = 0;
                json!({"name": u.name})
            }
            "damage" => {
                let u = &mut self.st.units[i(args, "unit") as usize];
                u.hp = i(args, "hp");
                if args["clear_acted"].as_bool().unwrap() {
                    u.acted = false;
                    u.native_record_byte5 &= !0x80;
                }
                json!({})
            }
            "mark_dead" => json!({"marked": self.st.mark_native_dead_records()}),
            "set_state" => {
                self.st.native_event_state[i(args, "index") as usize] = i(args, "value") as u8;
                json!({})
            }
            "set_control" => {
                self.st.native_turn_event_controls[i(args, "slot") as usize] = NativeTurnEventControl {
                    turn: i(args, "turn") as u8,
                    event_id: i(args, "event_id") as u8,
                    raw_camp: i(args, "raw_camp") as u8,
                };
                json!({})
            }
            "move" => {
                let u = &mut self.st.units[i(args, "unit") as usize];
                (u.x, u.y) = (i(args, "x"), i(args, "y"));
                json!({})
            }
            "set_inventory" => {
                let items = ints(&args["items"]);
                let u = &mut self.st.units[i(args, "unit") as usize];
                u.inventory = items.clone();
                u.equipped = vec![false; items.len()];
                u.inventory_slots = vec![0xff; 8];
                u.native_inventory_flags = vec![0x80; 8];
                for (k, &item) in items.iter().enumerate() {
                    u.inventory_slots[k] = item;
                    u.native_inventory_flags[k] = 0;
                }
                json!({})
            }
            "turn_events" => {
                let camp = i(args, "camp") as u8;
                let sc = self.sc.as_ref().unwrap();
                let r = sc.native_turn_events_at(Some(&self.st), camp);
                let ids: Vec<i64> = r.as_ref().map(|e| e.iter().map(|e| e.event_id).collect()).unwrap_or_default();
                let action_events = sc.native_turn_action_events_at(Some(&self.st), camp);
                self.pending = action_events.iter().flat_map(|e| e.actions.iter().cloned()).collect();
                json!({
                    "events": ids, "err": err_of(&r),
                    "action_events": action_events.iter().map(|e| e.event_id).collect::<Vec<_>>(),
                    "actions": serde_json::to_value(&self.pending).unwrap(),
                })
            }
            "append_group" => json!({"appended": self.st.append_group(i(args, "group"))}),
            "trigger" => {
                let st = &self.st;
                let actions = self.sc.as_mut().unwrap().trigger_actions(Some(st), s(args, "trigger"), s(args, "ctx"));
                self.pending = actions;
                json!({"actions": serde_json::to_value(&self.pending).unwrap()})
            }
            "exec" => self.exec_one(i(args, "index") as usize),
            "fire" => {
                let dl = self.sc.as_mut().unwrap().fire(Some(&mut self.st), s(args, "trigger"), s(args, "ctx"));
                json!({"dialogues": dialogues(&dl)})
            }
            "result" => self.result(),
            "recovery" => {
                let list = |plan: &[NativeEndTurnRecovery]| -> Value {
                    json!(
                        plan.iter()
                            .map(|r| [r.unit.map_or(-1, |u| u as i64), r.before, r.after, r.max_hp])
                            .collect::<Vec<_>>()
                    )
                };
                let plan = self.st.plan_native_end_turn_recovery();
                let validate = self.st.validate_native_end_turn_recovery(&plan);
                let applied = self.st.apply_native_end_turn_recovery();
                json!({"plan": list(&plan), "validate_err": err_of(&validate), "applied": list(&applied)})
            }
            "joins" => json!({"joins": self.sc().take_party_joins()}),
            "field" => self.field(args),
            "e76" => {
                let event = self.sc.as_ref().unwrap().native_turn_events[i(args, "event") as usize].clone();
                let plan = plan_native_turn_event76(Some(&self.st), &event);
                let mut out = json!({"final": plan.as_ref().is_ok_and(|p| p.r#final), "err": err_of(&plan)});
                if let Ok(plan) = plan
                    && !plan.r#final
                {
                    out["commit_err"] = err_of(&commit_native_turn_event76_repeat(Some(&mut self.st), &plan));
                }
                out
            }
            "e79" => {
                let event = self.sc.as_ref().unwrap().native_turn_events[i(args, "event") as usize].clone();
                let rng = i(args, "rng") as u16;
                let r = apply_native_turn_event79(Some(&mut self.st), &event, rng);
                let (next, targets) = r.as_ref().copied().unwrap_or((rng, [0, 0]));
                json!({"next": next, "targets": targets, "err": err_of(&r)})
            }
            "handler_prepare" => {
                self.st.roster = std::mem::take(&mut self.st.units);
                let party = self.sc.as_ref().unwrap().party_units(&self.st.own_deploy);
                self.st.append_native_map_selector_batch_or_legacy(party);
                json!({"digest": state_digest(&self.st)})
            }
            "adopt" => {
                let r = self.sc.as_mut().unwrap().adopt_handler_battle_state(Some(&mut self.st));
                json!({"err": err_of(&r), "digest": state_digest(&self.st)})
            }
            other => panic!("未知的 op {other}"),
        }
    }
}

struct Ctx {
    repo: PathBuf,
}

impl Ctx {
    /// 黃金檔的路徑相對於 `remake/`（Go 工具的工作目錄）。
    fn real(&self, rel: &str) -> String {
        self.repo.join("remake").join(rel).to_string_lossy().into_owned()
    }
}

fn golden() -> Option<(Value, Ctx)> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_RULES_F").filter(|p| Path::new(p).exists());
    let repo = std::env::var_os("FD2_GOLDEN_REPO").map(PathBuf::from).filter(|p| p.join("remake/assets").is_dir());
    let (Some(path), Some(repo)) = (path, repo) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_RULES_F 或 FD2_GOLDEN_REPO");
        eprintln!("略過：沒有 FD2_GOLDEN_RULES_F／FD2_GOLDEN_REPO");
        return None;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    Some((doc, Ctx { repo }))
}

fn temp_file(tag: &str, n: usize, body: &str) -> String {
    let dir = std::env::temp_dir().join(format!("fd2-golden-rules-f-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join(format!("{tag}-{n}.json"));
    std::fs::write(&path, body).unwrap();
    path.to_string_lossy().into_owned()
}

#[test]
fn golden_rules_f() {
    let Some((doc, ctx)) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-rules-f/1");
    eprintln!("刻意不照做：{KNOWN_DIFFERENCES:?}；比對正規化：{NORMALIZATIONS:?}");
    for file in doc["inputs"].as_array().unwrap() {
        let raw = std::fs::read(ctx.real(s(file, "path"))).unwrap_or_else(|e| panic!("讀不到 {}：{e}", file["path"]));
        assert_eq!(raw.len() as i64, i(file, "size"), "{} 大小", file["path"]);
        assert_eq!(digest(&raw), s(file, "sha256"), "{} SHA-256", file["path"]);
    }
    let rows = load_native_item_effect_row_prefix(&ctx.real("assets/data/native_item_effect_rows.json")).unwrap();
    let mut out = RustOut::default();
    let mut counts = std::collections::BTreeMap::<String, usize>::new();
    for (script_index, script) in doc["scripts"].as_array().unwrap().iter().enumerate() {
        let mut script = script.clone();
        script["map_abs"] = json!(ctx.real(s(&script, "map")));
        if let Some(sc) = script["scenario"].as_str() {
            script["scenario_abs"] = json!(ctx.real(sc));
        }
        let name = s(&script, "name").to_string();
        let mut runner = Runner {
            script: script_index,
            name: &name,
            st: State::default(),
            sc: None,
            pending: Vec::new(),
            rows: &rows,
        };
        for (op_index, op) in script["ops"].as_array().unwrap().iter().enumerate() {
            let got = runner.run_op(op_index, op, &script, &mut out);
            check(&format!("{name} op {op_index} {}", s(op, "op")), &op["result"], &got);
            *counts.entry(format!("op/{}", s(op, "op"))).or_default() += 1;
        }
    }
    for case in doc["cases"].as_array().unwrap() {
        let (n, input, go) = (i(case, "index") as usize, &case["input"], &case["result"]);
        let op = s(case, "op");
        let ctx_name = format!("case {n} {op}");
        let got = match op {
            "load_scenario" => {
                let path = temp_file("scenario", n, s(input, "json"));
                let r = load_scenario(&path);
                if go["decode_error"].as_bool().unwrap() {
                    // Go 標準函式庫的解碼錯誤：只比拒收。
                    assert!(r.is_err(), "{ctx_name}：Go 解碼失敗但 Rust 接受");
                    go.clone()
                } else {
                    let mut out = json!({"decode_error": false, "err": err_of(&r)});
                    if let Ok(sc) = r {
                        out["scenario"] = serde_json::to_value(&sc).unwrap();
                    }
                    out
                }
            }
            "restore" => {
                let snap: StateSnapshot = serde_json::from_str(s(input, "snapshot"))
                    .unwrap_or_else(|e| panic!("{ctx_name}：Rust 讀不進快照：{e}"));
                let mut dst = State::default();
                (dst.turn, dst.w) = (99, 5);
                let r = dst.restore_snapshot(&snap);
                let mut out = json!({"err": err_of(&r), "digest": state_digest(&dst)});
                if r.is_ok() {
                    out["snapshot"] = serde_json::to_value(dst.snapshot().unwrap()).unwrap();
                }
                out
            }
            "restore_fired" => {
                let mut sc = load_scenario(&ctx.real(s(input, "scenario"))).unwrap();
                let refs: Vec<FiredEventRef> = serde_json::from_value(input["refs"].clone()).unwrap();
                let r = sc.restore_fired_events(&refs);
                json!({"err": err_of(&r), "fired": serde_json::to_value(sc.fired_events()).unwrap()})
            }
            "march_commit" => {
                let mut st = State::default();
                (st.w, st.h) = (6, 6);
                for u in input["units"].as_array().unwrap() {
                    let flag = |k: &str| u[k].as_bool().unwrap();
                    // `Unit` 有 crate 私有欄位，整合測試不能用結構字面值，逐欄指定。
                    let mut unit = Unit::default();
                    (unit.x, unit.y) = (i(u, "x"), i(u, "y"));
                    unit.native_map_presentation = NativeMapPresentationState {
                        x: i(u, "px") as u8,
                        y: i(u, "py") as u8,
                        pose: i(u, "pose") as u8,
                        ..Default::default()
                    };
                    unit.has_native_map_presentation = flag("has_presentation");
                    (unit.native_record_byte5, unit.has_native_record_byte5) = (i(u, "byte5") as u8, flag("has5"));
                    (unit.native_record_byte6, unit.has_native_record_byte6) = (i(u, "byte6") as u8, flag("has6"));
                    st.units.push(unit);
                }
                let path = input["path"]
                    .as_array()
                    .unwrap()
                    .iter()
                    .map(|c| Cell { x: c[0].as_i64().unwrap(), y: c[1].as_i64().unwrap() });
                let step = NativeSystemGroupMarchStep {
                    unit_index: i(input, "index"),
                    path: path.collect(),
                    events: Vec::new(),
                };
                let r = st.commit_native_system_group_march_step(&step);
                let after: Vec<Value> = st
                    .units
                    .iter()
                    .map(|u| {
                        json!({"x": u.x, "y": u.y, "px": u.native_map_presentation.x, "py": u.native_map_presentation.y,
                            "pose": u.native_map_presentation.pose, "byte5": u.native_record_byte5, "acted": u.acted})
                    })
                    .collect();
                json!({"err": err_of(&r), "units": after})
            }
            other => panic!("未知的案例 {other}"),
        };
        check(&ctx_name, go, &got);
        *counts.entry(format!("case/{op}")).or_default() += 1;
    }
    if let Some(path) = std::env::var_os("FD2_GOLDEN_RULES_F_RUST_OUT") {
        let body = serde_json::to_vec(&json!({"snapshots": out.snapshots})).unwrap();
        std::fs::write(&path, body).unwrap();
        eprintln!("Rust 快照 {} 份寫到 {}", out.snapshots.len(), Path::new(&path).display());
    }
    eprintln!("通過：{counts:?}");
}
