//! Go `remake/cmd/fd2/chapter_parity_replay_test.go`：章重播（111 章工作單元的重製側）。從同一份建構槽由正常標題
//! START 進入第一章，照原版側驅動器留下的語意動作（`actions.jsonl`）走同一組決定，每一步寫出與 oracle 同形的
//! checkpoint（單位、回合、金幣、亂數字組、節點）。
//!
//! 環境變數與 Go 相同：`FD2_PARITY_CHAPTER`、`FD2_PARITY_SLOT`、`FD2_PARITY_ORACLE_RUN`、`FD2_PARITY_OUT`、
//! `FD2_PARITY_ENTRY`、`FD2_PARITY_CAPTURE_FRAMES`。`tools/rust_parity_ch01.sh` 在 Docker 內以
//! `cargo test -- --ignored` 執行本測試。
//!
//! 只移植第一章 START、`FD2_PARITY_CAPTURE_FRAMES=0` 與第一章 38 個動作（mark、select、move、wait、attack、
//! attack_result、end_turn、force_enemy_clear、town_enter、town_save）會走到的分支；其餘停在 `todo!("R8: …")`。
//! 畫面擷取（`frame`）是 R5。

use super::*;
use std::fs::File;
use std::io::Write as _;
use std::path::{Path, PathBuf};
use std::sync::Mutex;

/// Go `chapter_parity_replay_test.go:44`：`parityAction`（只解出第一章用到的欄位；其餘欄位存在時拒收的動作種類
/// 都是 R8）。
#[derive(Debug, Clone, Default, serde::Deserialize)]
#[serde(default)]
pub(crate) struct ParityAction {
    pub(crate) hp_lock: Option<serde_json::Value>,
    pub(crate) preserved_indices: Vec<i64>,
    pub(crate) kind: String,
    pub(crate) seq: i64,
    pub(crate) round: i64,
    pub(crate) rng_word: Option<i64>,
    pub(crate) gold: Option<i64>,
    pub(crate) at: Vec<i64>,
    #[serde(rename = "frm")]
    pub(crate) from: Vec<i64>,
    pub(crate) to: Vec<i64>,
    pub(crate) target: Vec<i64>,
    pub(crate) index: Option<i64>,
    pub(crate) ui: String,
    #[serde(rename = "move")]
    pub(crate) r#move: String,
    #[serde(rename = "probe_index")]
    pub(crate) probe: Option<i64>,
    pub(crate) cleared: Option<i64>,
    pub(crate) gold_before: Option<i64>,
    pub(crate) gold_after: Option<i64>,
    pub(crate) slot: Option<i64>,
    pub(crate) item_id: Option<i64>,
    pub(crate) key: String,
    pub(crate) label: String,
    /// end_turn 開系統選單那個 enter 的 seq；在它之前的方向鍵是把游標推到空格。
    pub(crate) before_seq: Option<i64>,
}

/// Go `chapter_parity_replay_test.go:71`：`parityUnit`。
#[derive(Debug, Clone, Default)]
pub(crate) struct ParityUnit {
    pub(crate) group: i64,
    pub(crate) record_byte8: Option<i64>,
    pub(crate) native_index: i64,
    pub(crate) camp: i64,
    pub(crate) x: i64,
    pub(crate) y: i64,
    pub(crate) hp: i64,
    pub(crate) identity: i64,
    pub(crate) acted: i64,
    pub(crate) px: i64,
    pub(crate) py: i64,
}

/// Go `chapter_parity_replay_test.go:87`：`parityCheckpoint`。第一章章重播不寫鎖血、畫面與調色盤欄位（全是
/// `omitempty`），這裡只保留會出現的欄位；序列化照 Go `encoding/json` 的欄位順序與 `omitempty`。
#[derive(Debug, Clone, Default)]
pub(crate) struct ParityCheckpoint {
    pub(crate) index: i64,
    pub(crate) kind: String,
    pub(crate) oracle_seq: i64,
    pub(crate) node: String,
    pub(crate) ui: String,
    pub(crate) round: i64,
    pub(crate) gold: i64,
    pub(crate) rng_word: i64,
    pub(crate) rng_synced: bool,
    /// Go `[]int`；nil 寫 `null`。
    pub(crate) cursor: Option<Vec<i64>>,
    /// `omitempty`。
    pub(crate) camera: Option<Vec<i64>>,
    /// Go `[]parityUnit`；nil 寫 `null`。
    pub(crate) units: Option<Vec<ParityUnit>>,
    /// `omitempty`。
    pub(crate) note: String,
}

/// Go `encoding/json` 的字串跳脫：`"`、`\`、控制字元，另外把 `<`、`>`、`&`、U+2028、U+2029 寫成 `\uXXXX`
/// （`json.Marshal` 預設 `SetEscapeHTML(true)`）。
pub(crate) fn go_json_string(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' | '>' | '&' | '\u{2028}' | '\u{2029}' => out.push_str(&format!("\\u{:04x}", c as u32)),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

fn go_json_ints(v: &[i64]) -> String {
    let parts: Vec<String> = v.iter().map(|x| x.to_string()).collect();
    format!("[{}]", parts.join(","))
}

impl ParityUnit {
    fn go_json(&self) -> String {
        let mut s = String::from("{");
        if self.group != 0 {
            s += &format!("\"group\":{},", self.group);
        }
        if let Some(b) = self.record_byte8 {
            s += &format!("\"record_byte8\":{b},");
        }
        s += &format!(
            "\"native_index\":{},\"camp\":{},\"x\":{},\"y\":{},\"hp\":{},\"identity\":{},\"acted\":{},\"px\":{},\"py\":{}}}",
            self.native_index, self.camp, self.x, self.y, self.hp, self.identity, self.acted, self.px, self.py
        );
        s
    }
}

impl ParityCheckpoint {
    /// Go `json.Marshal(cp)`（`chapter_parity_replay_test.go:2322`）。
    pub(crate) fn go_json(&self) -> String {
        let mut s = format!(
            "{{\"index\":{},\"kind\":{},\"oracle_seq\":{},\"node\":{},\"ui\":{},\"round\":{},\"gold\":{},\"rng_word\":{},\"rng_synced\":{},\"cursor\":{}",
            self.index,
            go_json_string(&self.kind),
            self.oracle_seq,
            go_json_string(&self.node),
            go_json_string(&self.ui),
            self.round,
            self.gold,
            self.rng_word,
            self.rng_synced,
            self.cursor.as_deref().map_or_else(|| "null".to_string(), go_json_ints),
        );
        if let Some(camera) = self.camera.as_deref() {
            s += &format!(",\"camera\":{}", go_json_ints(camera));
        }
        s += ",\"units\":";
        match self.units.as_deref() {
            None => s += "null",
            Some(units) => {
                let parts: Vec<String> = units.iter().map(ParityUnit::go_json).collect();
                s += &format!("[{}]", parts.join(","));
            }
        }
        if !self.note.is_empty() {
            s += &format!(",\"note\":{}", go_json_string(&self.note));
        }
        s.push('}');
        s
    }
}

/// `checkpoints.jsonl` 的寫入端與計數（Go `parityReplay.log`、`index`）。AI 觀察者也會寫 divergence
/// checkpoint，所以與觀察者共用。
pub(crate) struct ParityLog {
    pub(crate) file: File,
    pub(crate) index: i64,
}

/// Go `chapter_parity_replay_test.go:609`：`parityAIEntry`。
#[derive(Debug, Clone, Copy)]
pub(crate) struct ParityAIEntry {
    pub(crate) control_seq: i64,
    pub(crate) unit: i64,
    pub(crate) rng_word: i64,
}

/// AI 觀察者的狀態（Go `aiEntries`、`aiCursor`、`aiOrderDivergences`）。
#[derive(Debug, Default)]
pub(crate) struct ParityAIState {
    pub(crate) entries: Vec<ParityAIEntry>,
    pub(crate) cursor: usize,
    pub(crate) divergences: i64,
}

/// 升級成長觀察者的狀態（Go `growthEntries`、`growthCursor`）。
#[derive(Debug, Default)]
pub(crate) struct ParityGrowthState {
    pub(crate) entries: Vec<ParityAIEntry>,
    pub(crate) cursor: usize,
}

/// Go `chapter_parity_replay_test.go:113`：`parityReplay`（第一章用到的欄位）。
pub(crate) struct ParityReplay {
    pub(crate) d: Driver,
    pub(crate) out: PathBuf,
    /// 原版側輸出目錄（checkpoint-NNNN.json）
    pub(crate) run: PathBuf,
    pub(crate) log: Arc<Mutex<ParityLog>>,
    pub(crate) battle: String,
    /// LOAD 進的城鎮（本章戰前）
    pub(crate) town: String,
    /// 戰後城鎮
    pub(crate) town_after: String,
    pub(crate) preparation_after: String,
    pub(crate) postbattle: String,
    pub(crate) battle_done: bool,
    pub(crate) ai: Arc<Mutex<ParityAIState>>,
    pub(crate) growth: Arc<Mutex<ParityGrowthState>>,
    /// 原版側 control-history.jsonl 的逐序號按鍵
    pub(crate) keys: Option<BTreeMap<i64, String>>,
    pub(crate) prev_seq: i64,
    pub(crate) keys_consumed_through: i64,
    pub(crate) rejected_attack: bool,
    pub(crate) walk_started_early: bool,
    pub(crate) stopped_before_ai: bool,
    pub(crate) skip_frames: bool,
    pub(crate) start_entry_trace: Option<JourneyTrace>,
}

/// Go `chapter_parity_replay_test.go:616`：`readParityAIEntries`。讀不到檔回空（Go 回 nil）。
pub(crate) fn read_parity_ai_entries(path: &Path, eip: &str) -> Vec<ParityAIEntry> {
    let Ok(rows) = read_jsonl(path) else {
        return Vec::new();
    };
    let mut out = Vec::new();
    for row in rows {
        let stack = row["stack"].as_array().cloned().unwrap_or_default();
        if row["eip"] != eip || !row["rng_word"].is_i64() || stack.len() < 2 {
            continue;
        }
        let raw = stack[1].as_str().unwrap_or_default();
        let unit = i64::from_str_radix(raw.trim_start_matches("0x"), 16)
            .unwrap_or_else(|e| panic!("eip-trace.jsonl 堆疊引數：{e}：{row}"));
        out.push(ParityAIEntry {
            control_seq: row["control_seq"].as_i64().unwrap_or_default(),
            unit,
            rng_word: row["rng_word"].as_i64().unwrap(),
        });
    }
    out
}

/// Go `chapter_parity_replay_test.go:718`：`readParityControlKeys`（seq → key）。讀不到檔回 `None`。
pub(crate) fn read_parity_control_keys(path: &Path) -> Option<BTreeMap<i64, String>> {
    let rows = read_jsonl(path).ok()?;
    let mut out = BTreeMap::new();
    for row in rows {
        out.insert(row["seq"].as_i64().unwrap_or_default(), row["key"].as_str().unwrap_or_default().to_string());
    }
    Some(out)
}

/// Go `chapter_parity_replay_test.go:1542`：`readParityActions`。
pub(crate) fn read_parity_actions(path: &Path) -> Vec<ParityAction> {
    let raw = std::fs::read_to_string(path).unwrap_or_else(|e| panic!("{}：{e}", path.display()));
    raw.lines()
        .map(str::trim)
        .filter(|l| !l.is_empty())
        .map(|l| serde_json::from_str(l).unwrap_or_else(|e| panic!("actions.jsonl：{e}：{l}")))
        .collect()
}

/// Go `chapter_parity_replay_test.go:1674`：`nativeCampCode`，raw `+6`：0 敵方、1 友軍、2 我方。
pub(crate) fn native_camp_code(camp: battle::Camp) -> i64 {
    match camp {
        battle::OWN => 2,
        battle::ALLY => 1,
        _ => 0,
    }
}

/// Go `chapter_parity_replay_test.go:1599`：`ui`。
pub(crate) fn parity_ui(g: &Game) -> String {
    if !g.title_phase.is_empty() {
        return "title".to_string();
    }
    let Some(n) = node(g) else {
        return "unknown".to_string();
    };
    match n.r#type.as_str() {
        "town" => "town".to_string(),
        "shop" => "shop".to_string(),
        "church" => "church".to_string(),
        "hotel" => "hotel".to_string(),
        "preparation" => {
            if g.prep_record_slots {
                return "record_slots".to_string();
            }
            if g.native_preparation_prompt_active() && n.cancel.is_empty() {
                return "record_question".to_string();
            }
            "preparation".to_string()
        }
        "battle" => {
            if !g.dialog.is_empty() || g.native_death_reward_message_awaiting_key() {
                "dialogue".to_string()
            } else if g.ring {
                "ring".to_string()
            } else if g.sel.is_some() {
                "target".to_string()
            } else if g.ai_busy {
                "enemy".to_string()
            } else {
                "cursor".to_string()
            }
        }
        "story" | "cutscene" => "dialogue".to_string(),
        other => other.to_string(),
    }
}

/// Go `chapter_parity_replay_test.go:1644`：`units`。沒有鎖血政策（第一章）時不寫 `record_byte8`。
pub(crate) fn parity_units(g: &Game) -> Option<Vec<ParityUnit>> {
    let st = g.st.as_ref()?;
    let mut out = Vec::new();
    for (native_index, u) in st.units.iter().enumerate() {
        if !u.on_field || u.hp <= 0 {
            continue;
        }
        out.push(ParityUnit {
            group: u.group,
            record_byte8: None,
            native_index: native_index as i64,
            camp: native_camp_code(u.camp),
            x: u.x,
            y: u.y,
            hp: u.hp,
            identity: if u.has_native_identity { u.native_identity } else { -1 },
            acted: i64::from(u.acted),
            px: i64::from(u.native_map_presentation.x),
            py: i64::from(u.native_map_presentation.y),
        });
    }
    if out.is_empty() { None } else { Some(out) }
}

/// Go `chapter_parity_replay_test.go:2321`：`write`。
pub(crate) fn parity_write(log: &Mutex<ParityLog>, mut cp: ParityCheckpoint) {
    let mut log = log.lock().unwrap();
    cp.index = log.index;
    writeln!(log.file, "{}", cp.go_json()).expect("寫 checkpoints.jsonl");
    log.index += 1;
}

/// Go `chapter_parity_replay_test.go:2274`：`checkpoint`。`withFrame` 的畫面擷取是 R5；第一章重播以
/// `FD2_PARITY_CAPTURE_FRAMES=0` 執行（Go `frame` 在 `skipFrames` 時回空字串，調色盤等欄位也都是 nil）。
pub(crate) fn parity_checkpoint(
    g: &Game,
    log: &Mutex<ParityLog>,
    skip_frames: bool,
    kind: &str,
    seq: i64,
    ui: &str,
    with_frame: bool,
    extra: &[String],
) -> ParityCheckpoint {
    let mut cp = ParityCheckpoint {
        kind: kind.to_string(),
        oracle_seq: seq,
        node: node_id(g),
        ui: ui.to_string(),
        gold: g.gold,
        rng_word: i64::from(g.native_rng_state),
        cursor: Some(vec![g.cur_x, g.cur_y]),
        units: parity_units(g),
        ..Default::default()
    };
    if let Some(st) = g.st.as_ref().filter(|st| st.has_native_map_view_state) {
        cp.camera = Some(vec![st.native_map_view_state.camera_x, st.native_map_view_state.camera_y]);
    }
    for e in extra {
        if e == "rng_synced" {
            cp.rng_synced = true;
        } else {
            cp.note = e.clone();
        }
    }
    if let Some(st) = g.st.as_ref() {
        cp.round = st.turn;
    }
    if with_frame && !skip_frames {
        // R5 階段 3 改標 R8：FD2_PARITY_CAPTURE_FRAMES=1 的索引畫格變體是原版收據層（verify_chapter_parity.py）的
        // 輸入；R5 的黃金畫格都以 FD2_PARITY_CAPTURE_FRAMES=0 跑，開場與第一章的驗收沒有走到。
        todo!("R8: chapter_parity_replay_test.go:1696 frame（320×200 索引畫面擷取，原版收據層）")
    }
    parity_write(log, cp.clone());
    cp.index = log.lock().unwrap().index - 1;
    cp
}

/// Go `chapter_parity_replay_test.go:1566`：`playerHasControl`。
pub(crate) fn parity_player_has_control(g: &Game, battle_node: &str) -> bool {
    g.camp.is_some()
        && node_id(g) == battle_node
        && g.st.is_some()
        && g.result.is_empty()
        && !g.ai_busy
        && g.battle_event.is_none()
        && g.native_turn_staging.is_none()
        && g.banner_t == 0
        && g.dialog.is_empty()
        && g.walk.is_none()
        && g.atk.is_none()
        && !g.ring
        && g.native_class_ui_job.is_none()
        && g.spawn_intro_transition.is_none()
        && g.indexed_transition.is_none()
        && g.native_unit_present.is_none()
        && g.act_job.is_none()
        && g.cam_pan.is_none()
        && g.focus_job.is_none()
        && g.native_player_focus.is_none()
        && g.native_ch28_post_present.is_none()
        && !g.native_action_death_presentation_pending()
        && g.native_end_turn_recovery.is_none()
        && g.native_palette_pulse.is_none()
}

/// 原版 oracle 的 `checkpoint-NNNN.json`。
fn oracle_checkpoint(run: &Path, seq: i64) -> std::io::Result<Vec<u8>> {
    std::fs::read(run.join(format!("checkpoint-{seq:04}.json")))
}

/// Go `chapter_parity_replay_test.go:1477`：`verifyParityPreparationOwner`。
pub(crate) fn verify_parity_preparation_owner(run: &Path, seq: i64, owner: &str) -> std::result::Result<(), String> {
    let data = oracle_checkpoint(run, seq).map_err(|e| e.to_string())?;
    let cp: serde_json::Value = serde_json::from_slice(&data).map_err(|e| e.to_string())?;
    if cp["exe_sha256"] != PARITY_EXE_SHA256 {
        return Err("選人來源EXE未知".into());
    }
    if cp["input_chain"].as_array().is_some_and(|a| a.iter().any(|x| x == owner)) {
        return Ok(());
    }
    Err(format!("選人seq{seq}缺少已證實owner {owner}"))
}

/// Go `chapter_parity_replay_test.go:2969`：`parityTerminalENDOwner`。
pub(crate) fn parity_terminal_end_owner(data: &[u8]) -> std::result::Result<bool, String> {
    let cp: serde_json::Value = serde_json::from_slice(data).map_err(|e| e.to_string())?;
    let chain = cp["input_chain"].as_array().cloned().unwrap_or_default();
    if cp["exe_sha256"] != PARITY_EXE_SHA256 || chain.is_empty() {
        return Err("terminal END owner source unavailable".into());
    }
    for address in chain {
        let text = address.as_str().unwrap_or_default();
        // Go `strconv.ParseUint(address, 0, 32)`：oracle 的位址都是 0x 前綴的十六進位。
        let value = u64::from_str_radix(text.trim_start_matches("0x").trim_start_matches("0X"), 16)
            .map_err(|e| format!("strconv.ParseUint: parsing {text:?}: {e}"))?;
        if (0x1A30B..0x1A7BD).contains(&value) {
            return Ok(true);
        }
    }
    Ok(false)
}

/// Go `chapter_parity_replay_test.go:1079`：`parityClearBeforeAI`。
pub(crate) fn parity_clear_before_ai(end: &ParityAction, clear: &ParityAction) -> std::result::Result<bool, String> {
    if end.kind != "end_turn"
        || clear.kind != "force_enemy_clear"
        || clear.seq <= end.seq
        || end.round <= 0
        || clear.round < end.round
        || clear.round > end.round + 1
    {
        return Err(format!("清敵時序缺少可比的END／clear回合：{end:?}／{clear:?}"));
    }
    Ok(clear.round == end.round)
}

fn unit_at(g: &Game, i: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().expect("戰場狀態").units[i]
}

pub(super) fn sha256_hex_of(raw: &[u8]) -> String {
    use sha2::Digest as _;
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

impl ParityReplay {
    fn checkpoint(&mut self, kind: &str, seq: i64, ui: &str, with_frame: bool, extra: &[String]) -> ParityCheckpoint {
        parity_checkpoint(&self.d.g, &self.log, self.skip_frames, kind, seq, ui, with_frame, extra)
    }

    fn ui(&self) -> String {
        parity_ui(&self.d.g)
    }

    fn key(&self, seq: i64) -> &str {
        self.keys.as_ref().and_then(|k| k.get(&seq)).map_or("", String::as_str)
    }

    /// Go `chapter_parity_replay_test.go:2327`：`note`。
    fn note(&mut self, action: &ParityAction, text: &str) {
        let g = &self.d.g;
        let cp = ParityCheckpoint {
            kind: action.kind.clone(),
            oracle_seq: action.seq,
            node: node_id(g),
            ui: parity_ui(g),
            gold: g.gold,
            rng_word: i64::from(g.native_rng_state),
            note: text.to_string(),
            ..Default::default()
        };
        parity_write(&self.log, cp);
    }

    /// Go `chapter_parity_replay_test.go:2332`：`syncRNG`。
    fn sync_rng(&mut self, action: &ParityAction) -> bool {
        let Some(word) = action.rng_word else {
            return false;
        };
        self.d.g.native_rng_state = word as u16;
        true
    }

    fn player_has_control(&self) -> bool {
        parity_player_has_control(&self.d.g, &self.battle)
    }

    fn pump_player_control(&mut self, budget: i64) -> bool {
        let battle = self.battle.clone();
        self.d.pump(budget, |g| parity_player_has_control(g, &battle))
    }

    /// Go `chapter_parity_replay_test.go:1576`：`settleTown`。
    pub(crate) fn settle_town(&mut self) {
        for frame in 0..JOURNEY_TOWN_SETTLE_MAX {
            let g = &self.d.g;
            if g.native_class_ui_job.is_none()
                && g.fade.is_none()
                && g.transition_reveal.is_none()
                && g.indexed_transition.is_none()
                && frame > 10
            {
                return;
            }
            ack_presents(&mut self.d.g);
            if let Err(err) = parity_update_game(&mut self.d) {
                panic!("Update：{err}");
            }
            // R5 階段 2 r5c：戰後 settleTown 的 gap 擷取點（只在 gap_scope 期間）。
            self.d.observe_r5c_gap("settle", true);
        }
    }

    /// Go `chapter_parity_replay_test.go:1590`：`enterTownOption`。
    fn enter_town_option(&mut self, selection: i64) {
        let g = &mut self.d.g;
        let (source, ok) = g.compose_native_town_frame();
        if ok {
            g.prep_prompt_source = source;
        }
        g.camp.as_mut().expect("campaign").advance(&format!("opt{selection}"));
        g.enter_node();
    }

    /// Go `chapter_parity_replay_test.go:1107`：`mark`。
    fn mark(&mut self, action: &ParityAction) {
        match action.label.as_str() {
            "battle_start" => {
                if node(&self.d.g).is_some_and(|n| n.r#type == "preparation") {
                    assert!(
                        self.d.g.handle_native_preparation_input(NativePreparationInput {
                            enter: true,
                            ..Default::default()
                        })
                    );
                    assert!(
                        self.d.pump(600, |g| node(g).is_some_and(|n| n.r#type != "preparation")),
                        "確認出戰後未離開整備：{}",
                        ch01_blockers(&self.d.g)
                    );
                }
                let mut j = self.start_entry_trace.take().unwrap_or_else(new_journey_trace);
                let battle = self.battle.clone();
                self.d.drive_story(&mut j, |g| node_id(g) == battle);
                let battle = self.battle.clone();
                let jr = &mut j;
                if !self.d.pump(JOURNEY_STORY_FRAMES, |g| {
                    jr.observe(g);
                    if parity_player_has_control(g, &battle) {
                        return true;
                    }
                    if (g.battle_event.is_some() || g.native_turn_staging.is_some())
                        && !g.dialog.is_empty()
                        && story_enter_ready(g)
                    {
                        g.handle_battle_event_dialogue_input(true);
                    }
                    false
                }) {
                    panic!("戰場沒有交出操作權\n阻塞：{}", ch01_blockers(&self.d.g));
                }
                self.postbattle = node(&self.d.g).map(|n| n.on_win).unwrap_or_default();
                j.player_control = Some(j.milestone(&self.d.g, true));
                write_journey(&self.out.join("start-entry.json"), &j);
                self.checkpoint("battle_start", action.seq, "cursor", true, &[]);
            }
            "town_after_battle" => {
                self.ensure_town();
                self.checkpoint("town_after_battle", action.seq, "town", true, &[]);
            }
            "town_loaded" => {
                self.settle_town();
                self.checkpoint("town_loaded", action.seq, "town", true, &[]);
            }
            "departure_prompt" => {
                for _ in 0..5 {
                    if self.d.g.camp_sel == 2 {
                        break;
                    }
                    assert!(self.d.g.move_native_town_selection(1), "無法移到城鎮出口");
                }
                assert_eq!(self.d.g.camp_sel, 2);
                self.d.g.tick_input = TickInput { pressed: [Key::Enter].into(), ..Default::default() };
                assert!(self.d.g.camp_input());
                self.d.g.tick_input = TickInput::default();
                assert!(node(&self.d.g).is_some_and(|n| n.r#type == "preparation"));
                assert!(
                    self.d.pump(600, |g| g.native_preparation_prompt_active() && !g.native_class_ui_blocks_input())
                );
                self.checkpoint("departure_prompt", action.seq, "preparation", true, &[]);
            }
            "ending_entry"
            | "ending_terminal"
            | "captain_reinforcements_arrived"
            | "before_clear_viewport"
            | "record_question"
            | "record_slots"
            | "record_party_selection"
            | "party_selection"
            | "departure_confirmation"
            | "defeat_return_title" => {
                todo!("R8: chapter_parity_replay_test.go:1107 mark {:?}（第一章沒有）", action.label)
            }
            _ => {
                // 戰場裡的 mark 通常接在 await round>=N 之後：原版側已經推過敵方階段、回到下一回合的游標。
                let in_battle = self.d.g.camp.is_some()
                    && node_id(&self.d.g) == self.battle
                    && self.d.g.st.as_ref().is_some_and(|st| action.round > st.turn);
                if in_battle {
                    let (battle, round) = (self.battle.clone(), action.round);
                    if !self.d.pump(JOURNEY_STORY_FRAMES, |g| {
                        parity_player_has_control(g, &battle) && g.st.as_ref().unwrap().turn >= round
                    }) {
                        panic!(
                            "mark {}(seq {})：推不到第 {} 回合的操作權\n阻塞：{}",
                            action.label,
                            action.seq,
                            action.round,
                            ch01_blockers(&self.d.g)
                        );
                    }
                }
                let ui = self.ui();
                self.checkpoint(&format!("mark:{}", action.label), action.seq, &ui, true, &[]);
            }
        }
    }

    /// Go `chapter_parity_replay_test.go:2340`：`selectUnit`。
    fn select_unit(&mut self, action: &ParityAction) -> Option<UnitIdx> {
        if !self.pump_player_control(JOURNEY_STORY_FRAMES) {
            panic!("select(seq {})：玩家沒有操作權\n阻塞：{}", action.seq, ch01_blockers(&self.d.g));
        }
        // 先重走上一個動作到 select 之間的方向鍵：游標經過的格子會推 HUD anchor。
        self.replay_direction_keys(action);
        if !self.d.g.position_screenshot_cursor(action.at[0], action.at[1]) {
            panic!("select(seq {})：游標移不到 ({},{})", action.seq, action.at[0], action.at[1]);
        }
        self.d.g.confirm();
        let selected = self
            .d
            .g
            .sel
            .filter(|&s| unit_at(&self.d.g, s).x == action.at[0] && unit_at(&self.d.g, s).y == action.at[1]);
        if selected.is_none() {
            let (ui, note) = (
                self.ui(),
                format!(
                    "divergence: 原版在 ({},{}) 選到單位，重製端沒有（err={:?}）",
                    action.at[0], action.at[1], self.d.g.load_err
                ),
            );
            self.checkpoint("select", action.seq, &ui, true, &[note]);
            if self.d.g.sel.is_some() {
                self.release_selection();
            }
            if self.d.g.native_system_cursor_overlay || self.d.g.ring {
                // 空格上的 enter 開的是系統選單；等同原版側送 esc 退回地圖游標。
                self.d.pump(240, |g| !g.action_overlay_blocks_input());
                self.d.g.reset_action_overlay_lifecycle();
            }
            return None;
        }
        self.checkpoint("select", action.seq, "target", true, &[]);
        self.d.g.sel
    }

    /// Go `chapter_parity_replay_test.go:2371`：`releaseSelection`。
    fn release_selection(&mut self) {
        let g = &mut self.d.g;
        g.clear_native_player_movement();
        (g.sel, g.moved) = (None, false);
        g.reach.clear();
    }

    /// Go `chapter_parity_replay_test.go:2378`：`standDown`。
    fn stand_down(&mut self, action: &ParityAction, actor: Option<UnitIdx>) {
        let Some(actor) = actor.filter(|&a| self.d.g.sel == Some(a)) else {
            return;
        };
        if !self.d.g.ring {
            self.d.g.confirm();
            self.d.pump(240, |g| g.ring);
        }
        if !self.d.g.ring || !native_action_selectable(self.d.g.action_overlay_availability(), 3) {
            return;
        }
        self.d.g.ring_sel = 3;
        self.d.close_ring(Box::new(|g: &mut Game| g.finish_selected_wait()));
        self.d.pump(CH01_FRAME_BUDGET, |g| unit_at(g, actor).acted || unit_at(g, actor).hp <= 0);
        self.checkpoint("stand_down", action.seq, "cursor", false, &["divergence 之後就地待機".to_string()]);
    }

    /// Go `chapter_parity_replay_test.go:2396`：`moveUnit`。
    fn move_unit(&mut self, action: &ParityAction, actor: Option<UnitIdx>) -> bool {
        let Some(actor) = actor else {
            panic!("move(seq {})：沒有先 select", action.seq);
        };
        if !self.replay_cursor_keys(action) && !self.d.g.position_screenshot_cursor(action.to[0], action.to[1]) {
            panic!("move(seq {})：游標移不到 ({},{})", action.seq, action.to[0], action.to[1]);
        }
        if self.walk_started_early {
            self.walk_started_early = false;
        } else {
            if self.d.g.cur_x != action.to[0] || self.d.g.cur_y != action.to[1] {
                panic!(
                    "move(seq {})：照原版按鍵走完游標在 ({},{})，不是 ({},{})",
                    action.seq, self.d.g.cur_x, self.d.g.cur_y, action.to[0], action.to[1]
                );
            }
            self.d.g.confirm();
        }
        if self.d.g.walk.is_none() && !(self.d.g.ring && action.from == action.to) {
            let (ui, note) = (
                self.ui(),
                format!(
                    "divergence: ({},{})→({},{}) 原版接受、重製端拒絕（err={:?}）",
                    action.from[0], action.from[1], action.to[0], action.to[1], self.d.g.load_err
                ),
            );
            self.checkpoint("move", action.seq, &ui, true, &[note]);
            let (x, y) = (unit_at(&self.d.g, actor).x, unit_at(&self.d.g, actor).y);
            if !self.d.g.position_screenshot_cursor(x, y) {
                panic!("move(seq {})：分岔後游標移不回 ({x},{y})", action.seq);
            }
            self.stand_down(action, Some(actor));
            return false;
        }
        if !self.d.pump(CH01_FRAME_BUDGET, |g| g.walk.is_none()) {
            panic!("move(seq {})：移動動畫沒有結束", action.seq);
        }
        if !self.d.pump(240, |g| g.ring) {
            panic!("move(seq {})：走到 ({},{}) 之後沒有開指令環", action.seq, action.to[0], action.to[1]);
        }
        self.settle_action_overlay(action);
        self.checkpoint("move", action.seq, "ring", true, &[]);
        true
    }

    /// Go `chapter_parity_replay_test.go:2442`：`replayCursorKeys`。
    fn replay_cursor_keys(&mut self, action: &ParityAction) -> bool {
        if self.keys.is_none() {
            return false;
        }
        let mut walked = false;
        for seq in self.prev_seq + 1..action.seq {
            match self.key(seq).to_string().as_str() {
                "up" => self.d.g.move_map_cursor(0, -1),
                "down" => self.d.g.move_map_cursor(0, 1),
                "left" => self.d.g.move_map_cursor(-1, 0),
                "right" => self.d.g.move_map_cursor(1, 0),
                "enter" => {
                    let (at_x, at_y) = (self.d.g.cur_x, self.d.g.cur_y);
                    self.d.g.confirm();
                    if self.d.g.walk.is_some() {
                        // 驅動端有時在 enter 之後多等一格才確認進了指令環，move 就記在後一個 seq。
                        if at_x == action.to[0]
                            && at_y == action.to[1]
                            && self.only_idle_keys_until(seq + 1, action.seq)
                        {
                            self.walk_started_early = true;
                            return true;
                        }
                        panic!(
                            "move(seq {})：原版在 seq {seq} 的 enter 被拒絕，重製端卻接受並開始走行（游標 ({at_x},{at_y})）",
                            action.seq
                        );
                    }
                }
                _ => continue,
            }
            walked = true;
        }
        walked
    }

    /// Go `chapter_parity_replay_test.go:2490`：`replayDirectionKeys`。
    fn replay_direction_keys(&mut self, action: &ParityAction) {
        let previous = self.prev_seq.max(self.keys_consumed_through);
        let mut first_key = -1;
        for seq in previous + 1..action.seq {
            if matches!(self.key(seq), "up" | "down" | "left" | "right") {
                first_key = seq;
                break;
            }
        }
        if first_key < 0 {
            return;
        }
        let Ok(raw) = oracle_checkpoint(&self.run, first_key - 1) else {
            return;
        };
        let Ok(cp) = serde_json::from_slice::<serde_json::Value>(&raw) else {
            return;
        };
        if cp["view"]["cursor_x"].as_i64().unwrap_or_default() != self.d.g.cur_x
            || cp["view"]["cursor_y"].as_i64().unwrap_or_default() != self.d.g.cur_y
        {
            return;
        }
        for seq in previous + 1..action.seq {
            match self.key(seq) {
                "up" => self.d.g.move_map_cursor(0, -1),
                "down" => self.d.g.move_map_cursor(0, 1),
                "left" => self.d.g.move_map_cursor(-1, 0),
                "right" => self.d.g.move_map_cursor(1, 0),
                _ => {}
            }
        }
    }

    /// Go `chapter_parity_replay_test.go:2537`：`onlyIdleKeysUntil`。
    fn only_idle_keys_until(&self, from: i64, to: i64) -> bool {
        (from..to).all(|seq| self.key(seq).is_empty())
    }

    /// Go `chapter_parity_replay_test.go:2550`：`settleActionOverlay`。展開幀靠「這一幀被畫過」推進。
    fn settle_action_overlay(&mut self, action: &ParityAction) {
        let mut i = 0;
        while i < 240 && self.d.g.ring && self.d.g.action_overlay_blocks_input() {
            self.d.g.mark_action_overlay_drawn();
            if let Err(err) = parity_update_game(&mut self.d) {
                panic!("Update：{err}");
            }
            i += 1;
        }
        if self.d.g.ring && self.d.g.action_overlay_blocks_input() {
            panic!("{}(seq {})：指令環展開沒有完成", action.kind, action.seq);
        }
    }

    /// Go `chapter_parity_replay_test.go:2580`：`attack`。
    fn attack(&mut self, action: &ParityAction, actor: Option<UnitIdx>) -> Option<UnitIdx> {
        self.rejected_attack = false;
        let Some(actor) = actor else {
            panic!("attack(seq {})：沒有先 select", action.seq);
        };
        let available = self.d.g.action_overlay_availability();
        if !native_action_selectable(available, 0) {
            let u = unit_at(&self.d.g, actor);
            let (ui, note) = (
                self.ui(),
                format!(
                    "divergence: 原版可攻擊 ({},{})→({},{})，重製端指令環攻擊不可選 availability={}",
                    u.x,
                    u.y,
                    action.target[0],
                    action.target[1],
                    go_int_slice(&available)
                ),
            );
            self.checkpoint("attack_armed", action.seq, &ui, true, &[note]);
            self.wait_instead_of_attack(actor);
            return None;
        }
        self.d.g.ring_sel = 0;
        if !self.d.close_ring(Box::new(|g: &mut Game| g.begin_player_attack_targeting())) {
            panic!("attack(seq {})：指令環沒有收合", action.seq);
        }
        if !self.d.g.position_screenshot_cursor(action.target[0], action.target[1]) {
            panic!("attack(seq {})：游標移不到目標 ({},{})", action.seq, action.target[0], action.target[1]);
        }
        let mut target = None;
        for (i, u) in self.d.g.st.as_ref().unwrap().units.iter().enumerate() {
            if u.on_field && u.hp > 0 && u.x == action.target[0] && u.y == action.target[1] {
                target = Some(i);
            }
        }
        let Some(target) = target else {
            let (ui, note) = (
                self.ui(),
                format!("divergence: 原版攻擊目標格 ({},{}) 重製端沒有單位", action.target[0], action.target[1]),
            );
            self.checkpoint("attack_armed", action.seq, &ui, true, &[note]);
            let (x, y) = (unit_at(&self.d.g, actor).x, unit_at(&self.d.g, actor).y);
            if self.d.g.position_screenshot_cursor(x, y) {
                self.d.g.confirm();
                self.d.pump(240, |g| g.ring);
            }
            self.wait_instead_of_attack(actor);
            return None;
        };
        let mut extra = Vec::new();
        if self.sync_rng(action) {
            extra.push("rng_synced".to_string());
        }
        self.checkpoint("attack_armed", action.seq, "target", true, &extra);
        self.d.g.confirm();
        let g = &self.d.g;
        if g.st.as_ref().unwrap().has_native_map_view_state
            && target == actor
            && g.sel == Some(actor)
            && g.moved
            && !g.ring
            && !unit_at(g, actor).acted
            && g.atk.is_none()
            && g.load_err.is_empty()
        {
            // #115：原版自己的格不通過0x115B6，仍在target等鍵；不得強推acted。
            self.rejected_attack = true;
            return Some(target);
        }
        let hp = unit_at(g, target).hp;
        self.d.pump(CH01_FRAME_BUDGET, |g| {
            let t = unit_at(g, target);
            g.atk.is_none() && g.walk.is_none() && !g.ring && (t.hp != hp || t.hp <= 0 || unit_at(g, actor).acted)
        });
        // 擊倒有掉落時，原版在行動收尾前停在 0x1AA1D 的訊息等鍵處；先停在那裡讓檢查點拍到。
        if !self.d.pump(CH01_FRAME_BUDGET, |g| {
            let a = unit_at(g, actor);
            a.acted || a.hp <= 0 || g.native_death_reward_message_awaiting_key() || g.native_level_up_dialogue_at_wait()
        }) {
            panic!("attack(seq {})：攻擊之後沒有行動完畢\n阻塞：{}", action.seq, ch01_blockers(&self.d.g));
        }
        Some(target)
    }

    /// Go `chapter_parity_replay_test.go:2647`：`dismissNativeLevelUpDialogue`。
    fn dismiss_native_level_up_dialogue(&mut self) {
        if !self.d.pump(CH01_FRAME_BUDGET * 4, |g| {
            if g.native_level_up_dialogue.is_some() {
                g.step_native_level_up_dialogue(true);
            }
            g.native_level_up_dialogue.is_none()
        }) {
            panic!("升級對話推不完\n阻塞：{}", ch01_blockers(&self.d.g));
        }
    }

    /// Go `chapter_parity_replay_test.go:2660`：`waitInsteadOfAttack`。
    fn wait_instead_of_attack(&mut self, actor: UnitIdx) {
        if self.d.g.ring && native_action_selectable(self.d.g.action_overlay_availability(), 3) {
            self.d.g.ring_sel = 3;
            self.d.close_ring(Box::new(|g: &mut Game| g.finish_selected_wait()));
            self.d.pump(CH01_FRAME_BUDGET, |g| unit_at(g, actor).acted || unit_at(g, actor).hp <= 0);
        }
    }

    /// Go `chapter_parity_replay_test.go:2693`：`checkpointAttackResult`。
    fn checkpoint_attack_result(&mut self, action: &ParityAction, _actor: Option<UnitIdx>) {
        if self.rejected_attack {
            todo!("R8: chapter_parity_replay_test.go:2670 finishRejectedAttackFromKeys（#115，第一章沒有）")
        }
        let battle = self.battle.clone();
        let death_dialogue_open =
            |g: &Game| (g.battle_event.is_some() || g.native_turn_staging.is_some()) && !g.dialog.is_empty();
        let settled = |g: &Game| {
            parity_player_has_control(g, &battle)
                || g.native_death_reward_message_awaiting_key()
                || g.native_level_up_dialogue_at_wait()
                || death_dialogue_open(g)
                || !g.result.is_empty()
                || node_id(g) != battle
        };
        self.d.pump(CH01_FRAME_BUDGET, |g| settled(g));
        if death_dialogue_open(&self.d.g) {
            // 對白一出現就停下，不按鍵推到這一頁打完字。
            let mut frame = 0;
            while frame < CH01_FRAME_BUDGET && !story_enter_ready(&self.d.g) {
                ack_presents(&mut self.d.g);
                if let Err(err) = parity_update_game(&mut self.d) {
                    panic!("Update：{err}");
                }
                frame += 1;
            }
        }
        let ui = self.ui();
        self.checkpoint("attack_result", action.seq, &ui, true, &[]);
        if death_dialogue_open(&self.d.g) {
            self.d.pump(CH01_FRAME_BUDGET, |g| {
                parity_player_has_control(g, &battle)
                    || g.native_death_reward_message_awaiting_key()
                    || g.native_level_up_dialogue_at_wait()
                    || !g.result.is_empty()
                    || node_id(g) != battle
            });
        }
        // 原版側驅動端在 attack_result 之後對每個等鍵處送 enter，直到回到游標才做下一個動作。
        for _ in 0..8 {
            if self.d.g.native_level_up_dialogue.is_some() {
                self.dismiss_native_level_up_dialogue();
            } else if self.d.g.native_death_reward_message_awaiting_key() {
                self.d.g.acknowledge_native_death_reward_message();
                self.d.pump(CH01_FRAME_BUDGET, |g| g.native_system_end_turn_ui.is_none());
            } else {
                return;
            }
            self.d.pump(CH01_FRAME_BUDGET, |g| settled(g));
        }
        panic!("attack_result(seq {})：攻擊之後的等鍵訊息按不完\n阻塞：{}", action.seq, ch01_blockers(&self.d.g));
    }

    /// Go `chapter_parity_replay_test.go:2751`：`waitUnit`。
    fn wait_unit(&mut self, action: &ParityAction, actor: Option<UnitIdx>) {
        let Some(actor) = actor else {
            panic!("wait(seq {})：沒有先 select", action.seq);
        };
        let available = self.d.g.action_overlay_availability();
        if !native_action_selectable(available, 3) {
            panic!("wait(seq {})：待機不可選 availability={}", action.seq, go_int_slice(&available));
        }
        self.d.g.ring_sel = 3;
        if !self.d.close_ring(Box::new(|g: &mut Game| g.finish_selected_wait())) {
            panic!("wait(seq {})：指令環沒有收合", action.seq);
        }
        // 待機在寶物格會停在 0x190AC 的取得提示；原版側的 wait 檢查點是在提示問句等鍵時取的。
        let mut checkpointed = false;
        let (log, skip_frames, seq) = (self.log.clone(), self.skip_frames, action.seq);
        let ok = self.d.pump(CH01_FRAME_BUDGET, |g| {
            if !checkpointed
                && g.native_system_end_turn_confirm
                && g.native_class_ui_job.is_none()
                && g.native_system_end_turn_ui.as_ref().and_then(|s| s.treasure.as_ref()).is_some_and(|t| !t.await_ack)
            {
                parity_checkpoint(g, &log, skip_frames, "wait", seq, "treasure", true, &[]);
                checkpointed = true;
            }
            answer_native_treasure_prompt(g);
            let a = unit_at(g, actor);
            (a.acted && g.native_level_up_dialogue.is_none() && g.native_field_event_pending.is_none()) || a.hp <= 0
        });
        if !ok {
            panic!("wait(seq {})：待機之後沒有行動完畢\n阻塞：{}", action.seq, ch01_blockers(&self.d.g));
        }
        if !checkpointed {
            // selector0事件在Acted之後同步開對白。原版wait可停在第一段等鍵；不按鍵先等文字發布。
            if self.d.g.battle_event.is_some() {
                let mut frame = 0;
                while frame < CH01_FRAME_BUDGET && !story_enter_ready(&self.d.g) {
                    ack_presents(&mut self.d.g);
                    if let Err(err) = parity_update_game(&mut self.d) {
                        panic!("wait事件對白Update：{err}");
                    }
                    frame += 1;
                }
                if !story_enter_ready(&self.d.g) {
                    panic!("wait(seq {})：事件對白未到等鍵", action.seq);
                }
            }
            let ui = self.ui();
            self.checkpoint("wait", action.seq, &ui, true, &[]);
        }
    }

    /// Go `chapter_parity_replay_test.go:2888`：`endTurn`。
    fn end_turn(&mut self, action: &ParityAction, stop_before_ai: bool) {
        if node_id(&self.d.g) != self.battle || !self.d.g.result.is_empty() {
            self.note(action, "end_turn 時已不在戰場");
            return;
        }
        // 全員行動完原版會自己換手（0x13565）；先等到玩家重新拿到操作權，再照原版送 END。
        let battle = self.battle.clone();
        assert!(
            self.d.pump(CH01_FRAME_BUDGET * 6, |g| !g.result.is_empty() || parity_player_has_control(g, &battle)),
            "end_turn(seq {})：上一個動作未交回操作權\n阻塞：{}",
            action.seq,
            ch01_blockers(&self.d.g)
        );
        if !self.d.g.result.is_empty() {
            self.note(action, "end_turn 前戰鬥已分出勝負");
            return;
        }
        let mut extra = Vec::new();
        if self.sync_rng(action) {
            extra.push("rng_synced".to_string());
        }
        // 驅動端開系統選單前先把游標推到空格；before_seq 之後的方向鍵是選單內移動，不重走。
        if let Some(before_seq) = action.before_seq
            && action.at.len() == 2
            && let Some(st) = self.d.g.st.as_ref().filter(|st| st.has_native_map_view_state)
        {
            // `battle::State` 有私有欄位，不能用結構更新語法（A 組「fd2-rules／fd2-campaign 缺口」）。
            #[allow(clippy::field_reassign_with_default)]
            let mut probe = battle::State::default();
            (probe.w, probe.h) = (st.w, st.h);
            probe.native_map_view_state = st.native_map_view_state;
            probe.has_native_map_view_state = true;
            let mut steps = Vec::new();
            for seq in self.prev_seq + 1..before_seq {
                let step = match self.key(seq) {
                    "up" => Some((0, -1)),
                    "down" => Some((0, 1)),
                    "left" => Some((-1, 0)),
                    "right" => Some((1, 0)),
                    _ => None,
                };
                if let Some((dx, dy)) = step {
                    probe.move_native_map_cursor(dx, dy);
                    steps.push((dx, dy));
                }
            }
            if !steps.is_empty() {
                if probe.native_map_view_state.cursor_x == action.at[0]
                    && probe.native_map_view_state.cursor_y == action.at[1]
                {
                    for (dx, dy) in steps {
                        self.d.g.move_map_cursor(dx, dy);
                    }
                } else {
                    let view = &self.d.g.st.as_ref().unwrap().native_map_view_state;
                    let text = format!(
                        "END 前 {} 個方向鍵從 ({},{}) 走不到 at ({},{})，不重走",
                        steps.len(),
                        view.cursor_x,
                        view.cursor_y,
                        action.at[0],
                        action.at[1]
                    );
                    self.note(action, &text);
                }
            }
        }
        // 方向鍵可能使畫面快取失效。實際選單在下一幀才確認 END。
        ack_presents(&mut self.d.g);
        let before = self.d.g.st.as_ref().unwrap().turn;
        self.d.g.end_turn();
        if stop_before_ai {
            if !self.d.pump(CH01_FRAME_BUDGET * 6, |g| !g.result.is_empty() || (g.ai_busy && g.banner_t > 0)) {
                panic!("end_turn(seq {})：敵方回合沒有開始\n阻塞：{}", action.seq, ch01_blockers(&self.d.g));
            }
            let ui = self.ui();
            self.checkpoint("enemy_phase_start", action.seq, &ui, true, &extra);
            self.stopped_before_ai = true;
            return;
        }
        if !self.d.pump(CH01_FRAME_BUDGET * 6, |g| {
            !g.result.is_empty()
                || (!g.ai_busy
                    && g.native_turn_staging.is_none()
                    && g.st.as_ref().unwrap().turn > before
                    && parity_player_has_control(g, &battle))
        }) {
            let g = &self.d.g;
            panic!(
                "end_turn(seq {})：敵方回合沒有結束（turn {before}→{} aiBusy={}）\n阻塞：{}",
                action.seq,
                g.st.as_ref().unwrap().turn,
                g.ai_busy,
                ch01_blockers(g)
            );
        }
        if !self.d.g.result.is_empty() {
            let data = oracle_checkpoint(&self.run, action.seq).expect("讀 oracle checkpoint");
            let mid = parity_terminal_end_owner(&data).unwrap_or_else(|e| panic!("{e}"));
            // 沒有下一個玩家階段；只在 oracle 仍在 0x1A30B 時保留 END 自己的邊界。
            if mid {
                let ui = self.ui();
                self.checkpoint("end_turn", action.seq, &ui, false, &extra);
                return;
            }
        }
        let ui = self.ui();
        self.checkpoint("after_enemy_phase", action.seq, &ui, true, &extra);
    }

    /// Go `chapter_parity_replay_test.go:2992`：`forceEnemyClear`。
    fn force_enemy_clear(&mut self, action: &ParityAction, next: Option<&ParityAction>) {
        let mut defer_result = next.is_some_and(|n| n.kind == "end_turn");
        if let Some(next) = next.filter(|_| defer_result) {
            if next.round != action.round
                || next.before_seq.is_none_or(|b| b <= action.seq)
                || verify_parity_preparation_owner(&self.run, action.seq, "0x117F8").is_err()
                || verify_parity_preparation_owner(&self.run, next.before_seq.unwrap(), "0x118C6").is_err()
            {
                panic!("clear→END 缺少同回合正常玩家游標來源");
            }
            let data = oracle_checkpoint(&self.run, next.seq).expect("讀 oracle checkpoint");
            defer_result = parity_terminal_end_owner(&data).unwrap_or_else(|e| panic!("{e}"));
        }
        let battle = self.battle.clone();
        if !self.stopped_before_ai {
            self.d.pump(CH01_FRAME_BUDGET * 6, |g| !g.result.is_empty() || parity_player_has_control(g, &battle));
        }
        self.stopped_before_ai = false;
        // 與 oracle 的 force_enemy_clear 相同的注入：把場上 camp 0 的 HP 寫成 0（明示的修改路徑）。
        let mut cleared = 0;
        let preserved = action.preserved_indices.clone();
        for (index, u) in self.d.g.st.as_mut().unwrap().units.iter_mut().enumerate() {
            if u.on_field && u.camp == battle::ENEMY && u.hp > 0 {
                let _ = index;
                if parity_preserves_enemy(&preserved) {
                    continue;
                }
                u.hp = 0;
                cleared += 1;
            }
        }
        if !preserved.is_empty() && action.cleared != Some(cleared) {
            panic!("保護清敵筆數不符：重製{cleared}，原版{:?}", action.cleared);
        }
        let ui = self.ui();
        self.checkpoint("force_enemy_clear", action.seq, &ui, false, &[format!("cleared={cleared}（修改路徑）")]);
        if defer_result {
            // Oracle remains in the cursor owner. The following normal END must execute selector1 events.
            return;
        }
        self.d.g.st.as_mut().unwrap().mark_native_dead_records();
        self.d.g.check_result();
        if self.d.g.ai_busy && self.d.g.result.is_empty() {
            self.d.pump(CH01_FRAME_BUDGET * 6, |g| !g.result.is_empty() || parity_player_has_control(g, &battle));
            self.d.g.check_result();
        }
    }

    /// Go `chapter_parity_replay_test.go:3054`：`currentTown`。
    pub(crate) fn current_town(&self) -> String {
        if self.battle_done { self.town_after.clone() } else { self.town.clone() }
    }

    /// Go `chapter_parity_replay_test.go:3061`：`ensureTown`。
    fn ensure_town(&mut self) {
        let id = node_id(&self.d.g);
        if id == self.current_town() {
            return;
        }
        if id == self.town_after {
            self.battle_done = true;
            return;
        }
        let target = self.town_after.clone();
        self.ensure_postbattle_node(&target);
    }

    /// Go `chapter_parity_replay_test.go:3074`：`ensurePostbattleNode`。
    ///
    /// R5 階段 2 r5c：局部黃金打開時，整段（戰後迴圈、`drive_story`、`settle_town`）的每個 `update` 之後擷取，
    /// 對應 Go 逐字複本插入的擷取點（`rust_golden_r5c_layers_test.rs`）。
    fn ensure_postbattle_node(&mut self, target: &str) {
        if let Some(capture) = self.d.r5c_layer_capture.as_mut() {
            capture.gap_scope = true;
        }
        if let Some(capture) = self.d.r5s_frame_capture.as_mut() {
            capture.gap_scope = true;
        }
        let mut frame = 0;
        while frame < 3000 && node_id(&self.d.g) == self.battle {
            let g = &mut self.d.g;
            ack_presents(g);
            if result_input_reachable(g) && g.confirm_battle_result() {
                break;
            }
            if (g.battle_event.is_some() || g.native_turn_staging.is_some()) && !g.dialog.is_empty() {
                g.handle_battle_event_dialogue_input(true);
            }
            if let Err(err) = parity_update_game(&mut self.d) {
                panic!("Update：{err}");
            }
            self.d.observe_r5c_gap("postbattle", true);
            frame += 1;
        }
        if node_id(&self.d.g) == self.battle {
            panic!("戰鬥沒有結束：result={:?}\n阻塞：{}", self.d.g.result, ch01_blockers(&self.d.g));
        }
        let ui = self.ui();
        self.checkpoint("battle_result", 0, &ui, true, &[]);
        let mut j = new_journey_trace();
        let target = target.to_string();
        self.d.drive_story(&mut j, |g| node_id(g) == target);
        self.battle_done = true;
        self.settle_town();
        if let Some(capture) = self.d.r5c_layer_capture.as_mut() {
            capture.gap_scope = false;
        }
        if let Some(capture) = self.d.r5s_frame_capture.as_mut() {
            capture.gap_scope = false;
        }
    }

    /// Go `chapter_parity_replay_test.go:3115`：`townProbe`。
    fn town_probe(&mut self, action: &ParityAction) {
        self.ensure_town();
        let delta = if action.r#move == "right" { -1 } else { 1 };
        if !self.d.g.move_native_town_selection(delta) {
            panic!("town_enter(seq {})：城鎮選擇無法移動", action.seq);
        }
        self.d.observe_r5c_gap("town_move", false);
        let selection = self.d.g.camp_sel;
        self.enter_town_option(selection);
        self.d.pump(120, |g| g.native_class_ui_job.is_none() && g.fade.is_none());
        let ui = self.ui();
        self.checkpoint("town_enter", action.seq, &ui, true, &[]);
        // 退回城鎮，與原版側的 esc 相同。
        let town = self.current_town();
        match node(&self.d.g).map(|n| n.r#type) {
            None => {}
            Some(t) if t == "shop" => self.d.g.leave_shop(),
            Some(t) if t == "church" => self.d.g.leave_church(),
            Some(t) if t == "hotel" => self.d.g.leave_hotel(),
            Some(t) if t == "preparation" => {
                // 出口提示答 NO：走正式輸入 owner（esc），與原版側送的鍵相同。
                self.d.g.handle_native_preparation_input(NativePreparationInput { escape: true, ..Default::default() });
                self.d.pump(240, |g| node_id(g) == town);
            }
            Some(t) if t == "town" => {}
            Some(_) => {
                let g = &mut self.d.g;
                g.camp.as_mut().unwrap().advance("cancel");
                g.native_town_hub_return = true;
                g.enter_node();
            }
        }
        self.d.pump(120, |g| node_id(g) == town);
        if node_id(&self.d.g) != town {
            panic!("town_enter(seq {})：離開選項 {selection} 之後沒有回到城鎮：{:?}", action.seq, node_id(&self.d.g));
        }
    }

    /// Go `chapter_parity_replay_test.go:3222`：`townSave`。
    fn town_save(&mut self, action: &ParityAction) {
        self.ensure_town();
        let slot = action.slot.unwrap_or(0);
        // 城鎮 hub 停在酒店（0 號）按 enter → 圖示選單 right 一格到存檔 → enter 開四槽列表 → enter 確認。
        let mut guard = 0;
        while self.d.g.camp_sel != 0 && guard < 8 {
            if !self.d.g.move_native_town_selection(1) {
                panic!("town_save(seq {})：城鎮選擇移不到酒店", action.seq);
            }
            self.d.observe_r5c_gap("town_move", false);
            guard += 1;
        }
        self.enter_town_option(0);
        self.d.pump(120, |g| g.native_class_ui_job.is_none() && g.fade.is_none());
        if node(&self.d.g).is_none_or(|n| n.r#type != "hotel") || self.d.g.native_hotel_mode != "menu" {
            panic!(
                "town_save(seq {})：沒有進到原生酒店選單（node={:?} mode={:?}）",
                action.seq,
                node_id(&self.d.g),
                self.d.g.native_hotel_mode
            );
        }
        // R5 階段 2 r5c：每次酒店輸入之後的 gap 擷取點（Go 逐字複本的插入行）。
        for _ in 0..NATIVE_HOTEL_SERVICE_SAVE {
            self.d.g.handle_native_hotel_input(NativeHotelInput { delta: 1, ..Default::default() });
            self.d.observe_r5c_gap("hotel_input", false);
        }
        self.d.g.handle_native_hotel_input(NativeHotelInput { enter: true, ..Default::default() });
        self.d.observe_r5c_gap("hotel_input", false);
        if self.d.g.native_hotel_mode != "slots" {
            panic!("town_save(seq {})：存檔列表沒有開（mode={:?}）", action.seq, self.d.g.native_hotel_mode);
        }
        while self.d.g.native_hotel_slot_sel < slot {
            self.d.g.handle_native_hotel_input(NativeHotelInput { down: true, ..Default::default() });
            self.d.observe_r5c_gap("hotel_input", false);
        }
        self.d.g.handle_native_hotel_input(NativeHotelInput { enter: true, ..Default::default() });
        self.d.observe_r5c_gap("hotel_input", false);
        if self.d.g.native_hotel_mode != "saved" {
            panic!(
                "town_save(seq {})：存檔後沒有「記錄儲存完畢」（mode={:?}）",
                action.seq, self.d.g.native_hotel_mode
            );
        }
        // 收據記的是寫回後整份 FD2.SAV 的 sha256（與 oracle 的 save_sha256 同一個算法）。
        let mut note = if let Some(err) = self.d.g.native_chapter_slot_save_err.as_ref() {
            format!("原版槽寫回失敗：{err}")
        } else {
            match std::fs::read(native_current_save_path()) {
                Err(err) => format!("原版槽寫回後讀不到 FD2.SAV：{err}"),
                Ok(raw) => format!("save_sha256={}", sha256_hex_of(&raw)),
            }
        };
        if let Err(err) = std::fs::metadata(save_slot_path(slot)) {
            note += &format!("；重製端自有存檔未寫出：{err}");
        }
        self.verify_chapter_slot_save(slot);
        self.checkpoint("town_save", action.seq, "hotel", true, &[note]);
        // 原版側接著 esc 三次回到城鎮：訊息 → 選單 → 離開。
        self.d.g.handle_native_hotel_input(NativeHotelInput { esc: true, ..Default::default() });
        self.d.observe_r5c_gap("hotel_input", false);
        self.d.g.handle_native_hotel_input(NativeHotelInput { esc: true, ..Default::default() });
        self.d.observe_r5c_gap("hotel_input", false);
        let town = self.current_town();
        self.d.pump(120, |g| node_id(g) == town);
        if node_id(&self.d.g) != town {
            panic!("town_save(seq {})：離開酒店後沒有回到城鎮：{:?}", action.seq, node_id(&self.d.g));
        }
    }

    fn shop_input(&mut self, key: Option<Key>, confirm: bool, expected_mode: &str) {
        let input = TickInput { pressed: key.into_iter().collect(), ..Default::default() };
        self.d.g.handle_native_shop_input(&input, confirm);
        assert!(
            self.d.pump(700, |g| g.native_shop_ui_job.is_none() && g.fade.is_none()),
            "商店未完成演出，預期模式 {expected_mode}"
        );
        assert_eq!(self.d.g.native_shop_mode, expected_mode, "{}", self.d.g.msg);
    }

    fn shop_sell(&mut self, action: &ParityAction) {
        self.ensure_town();
        for _ in 0..6 {
            if self.d.g.camp_sel == 1 {
                break;
            }
            assert!(self.d.g.move_native_town_selection(1));
        }
        assert_eq!(self.d.g.camp_sel, 1, "武器店選項無法到達");
        self.enter_town_option(1);
        assert!(self.d.pump(700, |g| g.native_shop_ui_job.is_none() && g.fade.is_none()));
        assert_eq!(self.d.g.native_shop_mode, "menu");
        self.checkpoint("shop_menu", action.seq, "shop", false, &[]);
        let gold = self.d.g.gold;
        self.shop_input(Some(Key::ArrowRight), false, "menu");
        self.shop_input(None, true, "sell_roster");
        self.shop_input(None, true, "sell_items");
        self.shop_input(None, true, "sell_confirm");
        self.shop_input(None, true, "sell_roster");
        if let (Some(before), Some(after)) = (action.gold_before, action.gold_after) {
            assert_eq!(self.d.g.gold - gold, after - before, "販賣入帳金額");
        }
        self.d.g.leave_shop();
        self.settle_town();
        self.checkpoint("shop_sell", action.seq, "town", true, &[]);
    }

    fn secret_shop(&mut self, action: &ParityAction) {
        self.ensure_town();
        // 原版的快捷鍵由 Rust 已接線的工具列按鈕取代。先移入上緣展開，再點按入口。
        let hover = TickInput { cursor: (624, 2), ..Default::default() };
        let click = TickInput { cursor: (624, 10), mouse_left_just_pressed: true, ..Default::default() };
        assert!(self.d.g.update_toolbar(&hover, true));
        assert!(self.d.g.update_toolbar(&click, true));
        assert!(self.d.pump(700, |g| g.native_shop_ui_job.is_none() && g.fade.is_none()));
        assert_eq!(self.d.g.native_shop_mode, "menu", "工具列祕密商店無法開啟");
        self.checkpoint("secret_shop", action.seq, "shop", true, &[]);
        self.d.g.leave_shop();
        self.settle_town();
    }

    fn shop_buy(&mut self, action: &ParityAction) {
        self.ensure_town();
        for _ in 0..6 {
            if self.d.g.camp_sel == 3 {
                break;
            }
            assert!(self.d.g.move_native_town_selection(1));
        }
        assert_eq!(self.d.g.camp_sel, 3, "道具店選項無法到達");
        self.enter_town_option(3);
        assert!(self.d.pump(700, |g| g.native_shop_ui_job.is_none() && g.fade.is_none()));
        assert_eq!(self.d.g.native_shop_mode, "menu");
        let gold = self.d.g.gold;
        for mode in ["purchase", "confirm", "recipient_consumable", "purchase"] {
            self.shop_input(None, true, mode);
        }
        if let (Some(before), Some(after)) = (action.gold_before, action.gold_after) {
            assert_eq!(self.d.g.gold - gold, after - before, "購買扣款金額");
        } else {
            assert!(self.d.g.gold < gold, "購買未扣款");
        }
        self.d.g.leave_shop();
        self.settle_town();
        self.checkpoint("shop_buy", action.seq, "town", true, &[]);
    }

    /// Go `TestChapterParityReplay` 的動作迴圈（`chapter_parity_replay_test.go:972`..`1036`）。
    fn run_actions(&mut self, actions: &[ParityAction]) {
        let (mut actor, mut skip_unit): (Option<UnitIdx>, bool) = (None, false);
        for (i, action) in actions.iter().enumerate() {
            if skip_unit {
                if matches!(action.kind.as_str(), "move" | "stay" | "attack" | "attack_result" | "wait" | "cancel") {
                    self.note(action, "skipped after divergence");
                    continue;
                }
                skip_unit = false;
            }
            match action.kind.as_str() {
                "mark" => self.mark(action),
                "select" => {
                    actor = self.select_unit(action);
                    skip_unit = actor.is_none();
                }
                "move" => {
                    if !self.move_unit(action, actor) {
                        skip_unit = true;
                    }
                }
                "stay" => {
                    assert!(actor.is_some(), "stay(seq {}) 缺少角色", action.seq);
                    self.replay_cursor_keys(action);
                    self.d.g.confirm();
                    assert!(self.d.pump(240, |g| g.ring), "stay(seq {}) 未開啟指令環", action.seq);
                    self.settle_action_overlay(action);
                    self.checkpoint("stay", action.seq, "ring", true, &[]);
                }
                "attack" => {
                    if self.attack(action, actor).is_none() {
                        skip_unit = true;
                    }
                }
                "attack_result" => self.checkpoint_attack_result(action, actor),
                "wait" => self.wait_unit(action, actor),
                "end_turn" => {
                    let mut stop_before_ai = false;
                    if let Some(next) = actions.get(i + 1).filter(|n| n.kind == "force_enemy_clear") {
                        stop_before_ai = parity_clear_before_ai(action, next).unwrap_or_else(|e| panic!("{e}"));
                    }
                    self.end_turn(action, stop_before_ai);
                }
                "force_enemy_clear" => self.force_enemy_clear(action, actions.get(i + 1)),
                "town_enter" => self.town_probe(action),
                "town_save" => self.town_save(action),
                "shop_sell" => self.shop_sell(action),
                "secret_shop" => self.secret_shop(action),
                "shop_buy" => self.shop_buy(action),
                "hp_lock" | "item_use" | "cancel" | "preparation_save" | "ending_terminal" => {
                    todo!("R8: chapter_parity_replay_test.go:1003 動作 {:?}（第一章沒有）", action.kind)
                }
                _ => self.note(action, "unhandled action kind"),
            }
            if !self.d.g.load_err.is_empty() {
                panic!(
                    "動作 {}(seq {}) 之後執行期錯誤：{}\n阻塞：{}",
                    action.kind,
                    action.seq,
                    self.d.g.load_err,
                    ch01_blockers(&self.d.g)
                );
            }
            self.prev_seq = action.seq;
        }
    }
}

/// Go `chapter_parity_replay_test.go:743`：`observeGrowthRoll`。每一次升級成長擲骰前把原版側下一筆 0x1E54A 的
/// rng_word 抄過來；原版側沒有更多收據就照重製端目前的狀態擲。
fn observe_growth_roll(growth: &Mutex<ParityGrowthState>, state: u16) -> u16 {
    let mut growth = growth.lock().unwrap();
    if growth.cursor >= growth.entries.len() {
        return state;
    }
    let entry = growth.entries[growth.cursor];
    growth.cursor += 1;
    entry.rng_word as u16
}

/// Go `chapter_parity_replay_test.go:755`：`observeAIPlan`。每個 AI 計畫產生後對齊 RNG。
fn observe_ai_plan(
    g: &mut Game,
    plan: &battle::AIPlan,
    ai: &Mutex<ParityAIState>,
    log: &Mutex<ParityLog>,
    skip_frames: bool,
) {
    let mut ai = ai.lock().unwrap();
    let Some(u) = plan.u else {
        return;
    };
    if ai.cursor >= ai.entries.len() {
        return;
    }
    let index = u as i64;
    let entry = ai.entries[ai.cursor];
    if std::env::var("FD2_PARITY_AI_DIAGNOSTICS").unwrap_or_default() == "1" {
        eprintln!(
            "AI plan diagnostic: turn={} entry={} seq={} actor={index} rng_before={} rng_entry={}",
            g.st.as_ref().map_or(0, |st| st.turn),
            ai.cursor,
            entry.control_seq,
            g.native_rng_state,
            entry.rng_word
        );
    }
    if entry.unit == index {
        ai.cursor += 1;
        g.native_rng_state = entry.rng_word as u16;
        return;
    }
    ai.divergences += 1;
    let note = format!("divergence: 原版第 {} 個 AI 行動是 record {}，重製端是 record {index}", ai.cursor, entry.unit);
    parity_checkpoint(g, log, skip_frames, "ai_order", entry.control_seq, &parity_ui(g), false, &[note]);
    let start = ai.cursor + 1;
    for j in start..ai.entries.len().min(ai.cursor + 8) {
        if ai.entries[j].unit == index {
            ai.cursor = j + 1;
            g.native_rng_state = ai.entries[j].rng_word as u16;
            return;
        }
    }
}

fn env(name: &str) -> String {
    std::env::var(name).unwrap_or_default()
}

/// Go `chapter_parity_replay_test.go:802`：`TestChapterParityReplay`。
///
/// 父行程檢查 `FD2_PARITY_*`、把建構槽複製到輸出目錄當可寫覆蓋層，再以 Go `t.Setenv` 的那組環境變數
/// 在子行程執行重播（F 組 `f_child_env`）。缺環境變數時與 Go `t.Skip` 相同，印「略過」後返回。
#[test]
#[ignore = "章重播：由 tools/rust_parity_ch01.sh 以 --ignored 執行"]
fn test_chapter_parity_replay() {
    let (chapter_text, slot, run, out) =
        (env("FD2_PARITY_CHAPTER"), env("FD2_PARITY_SLOT"), env("FD2_PARITY_ORACLE_RUN"), env("FD2_PARITY_OUT"));
    if chapter_text.is_empty() || slot.is_empty() || run.is_empty() || out.is_empty() {
        eprintln!("略過：需要 FD2_PARITY_CHAPTER／FD2_PARITY_SLOT／FD2_PARITY_ORACLE_RUN／FD2_PARITY_OUT");
        return;
    }
    let chapter: i64 = chapter_text.parse().unwrap_or(0);
    if !(1..=30).contains(&chapter) {
        panic!("FD2_PARITY_CHAPTER={chapter_text:?}");
    }
    let mut entry_mode = env("FD2_PARITY_ENTRY");
    if entry_mode.is_empty() {
        entry_mode = "load".to_string();
    }
    if entry_mode != "load" && (entry_mode != "start" || chapter != 1) {
        panic!("FD2_PARITY_ENTRY={entry_mode:?}；START只適用首章");
    }
    let (run, out) = (PathBuf::from(run), PathBuf::from(out));
    let slot_copy = out.join("FD2.SAV");
    let child = std::env::var_os("FD2_GAME_F_TEST_DIR").is_some();
    if !child {
        std::fs::create_dir_all(&out).expect("建立輸出目錄");
        // 建構槽是唯讀輸入；酒店存檔要寫回同一份 FD2.SAV，所以在輸出目錄放一份複本當可寫覆蓋層。
        let raw = std::fs::read(&slot).unwrap_or_else(|e| panic!("{slot}：{e}"));
        std::fs::write(&slot_copy, raw).expect("寫 FD2.SAV 複本");
    }
    let slot_copy_env = slot_copy.clone();
    let Some(_data_home) = f_child_env(module_path!(), "test_chapter_parity_replay", move |dir| {
        vec![
            ("FD2_TITLE", "1".into()),
            ("FD2_SEED", "4".into()),
            ("FD2_MUTE", "1".into()),
            ("FD2_CAMPAIGN", CANONICAL_CAMPAIGN_REFERENCE.into()),
            ("FD2_SHOT_AI", "1".into()),
            ("FD2_NATIVE_SAVE", slot_copy_env.clone().into_os_string()),
            ("FD2_CUTSCENE_LOG", "1".into()),
            ("XDG_DATA_HOME", dir.as_os_str().to_os_string()),
        ]
    }) else {
        return;
    };
    run_chapter_parity_replay(chapter, &entry_mode, &run, &out);
}

/// `TestChapterParityReplay` 在 `t.Setenv` 之後的本體（子行程）。
fn run_chapter_parity_replay(chapter: i64, entry_mode: &str, run: &Path, out: &Path) {
    let mut actions = read_parity_actions(&run.join("actions.jsonl"));
    if actions.is_empty() {
        panic!("actions.jsonl 是空的");
    }
    let keys = read_parity_control_keys(&run.join("control-history.jsonl"));
    let legacy_end_sources =
        recover_parity_legacy_end_sources(run, &mut actions, keys.as_ref()).expect("核對舊版 END 的原版來源");
    let frame_setting = env("FD2_PARITY_CAPTURE_FRAMES");
    if !frame_setting.is_empty() && frame_setting != "0" && frame_setting != "1" {
        panic!("FD2_PARITY_CAPTURE_FRAMES={frame_setting:?}；只允許0或1");
    }
    if frame_setting != "0" {
        todo!(
            "R8: chapter_parity_replay_test.go:1696 frame（320×200 索引畫格變體，原版收據層；R5 黃金畫格與第一章 Rust 重播以 FD2_PARITY_CAPTURE_FRAMES=0 執行）"
        )
    }
    let g = load_game().expect("loadGame");
    // R5 r5f：Go 黃金載入了系統的 NotoSansCJK-Regular.ttc；Rust 端要求同一檔（缺檔照 FD2_REQUIRE_ORIGINAL）。
    require_golden_font(&g);
    if !g.load_err.is_empty() {
        panic!("{}", g.load_err);
    }
    if actions.iter().any(|a| a.kind == "hp_lock") {
        todo!("R8: chapter_parity_hp_lock_test.go 鎖血政策")
    }
    let settings: BTreeMap<&str, serde_json::Value> = BTreeMap::from([
        ("go_seed", 4.into()),
        ("native_rng_initial", g.native_rng_state.into()),
        ("method", "AI／攻擊確認／END／升級比較點承接原版受控 RNG word".into()),
        ("legacy_end_sources", serde_json::to_value(&legacy_end_sources).unwrap()),
        ("capture_frames", (frame_setting != "0").into()),
        ("modified_path_hp_lock", false.into()),
        ("entry_mode", entry_mode.into()),
    ]);
    let mut raw = serde_json::to_vec_pretty(&settings).unwrap();
    raw.push(b'\n');
    std::fs::write(out.join("replay-settings.json"), raw).expect("寫 replay-settings.json");

    let file = File::create(out.join("checkpoints.jsonl")).expect("建立 checkpoints.jsonl");
    let log = Arc::new(Mutex::new(ParityLog { file, index: 0 }));
    let eip = run.join("eip-trace.jsonl");
    let ai =
        Arc::new(Mutex::new(ParityAIState { entries: read_parity_ai_entries(&eip, "0x13A9F"), ..Default::default() }));
    let growth = Arc::new(Mutex::new(ParityGrowthState {
        entries: read_parity_ai_entries(&eip, "0x1E54A"),
        ..Default::default()
    }));
    let skip_frames = frame_setting == "0";
    let mut r = ParityReplay {
        d: Driver {
            g,
            clock: VirtualFrameClock::default(),
            frame_capture: FrameCapture::from_env(),
            map_layer_capture: MapLayerCapture::from_env(),
            dialogue_layer_capture: DialogueLayerCapture::from_env(),
            r5p_layer_capture: R5pLayerCapture::from_env(),
            r5c_layer_capture: R5cLayerCapture::from_env(),
            ttf_layer_capture: TtfLayerCapture::from_env(),
            r5b_layer_capture: R5bLayerCapture::from_env(),
            r5h_layer_capture: R5hLayerCapture::from_env(),
            r5s_frame_capture: R5sFrameCapture::from_env(),
        },
        out: out.to_path_buf(),
        run: run.to_path_buf(),
        log: log.clone(),
        battle: format!("battle_ch{chapter:02}"),
        town: format!("town_ch{chapter:02}"),
        town_after: format!("town_ch{:02}", chapter + 1),
        preparation_after: format!("preparation_ch{:02}", chapter + 1),
        postbattle: String::new(),
        battle_done: false,
        ai: ai.clone(),
        growth: growth.clone(),
        keys,
        prev_seq: 0,
        keys_consumed_through: 0,
        rejected_attack: false,
        walk_started_early: false,
        stopped_before_ai: false,
        skip_frames,
        start_entry_trace: None,
    };
    {
        let (ai, log) = (ai.clone(), log.clone());
        r.d.g.ai_plan_observer =
            Some(Box::new(move |g: &mut Game, plan: &battle::AIPlan| observe_ai_plan(g, plan, &ai, &log, skip_frames)));
        let growth = growth.clone();
        r.d.g.growth_roll_observer = Some(battle::Callback::new(Arc::new(move |_: &battle::Unit, state: u16| {
            observe_growth_roll(&growth, state)
        })));
    }
    if chapter == 29 {
        todo!("R8: chapter_parity_replay_test.go:931 event79 turn pair RNG")
    }
    // R5 階段 2 r5d：畫格擷取打開時照 Go 黃金的 rand.Seed(4) 接上全域亂數；關閉時維持 R4 的固定種子。
    if r.d.frame_capture.is_some()
        || r.d.map_layer_capture.is_some()
        || r.d.dialogue_layer_capture.is_some()
        || r.d.r5p_layer_capture.is_some()
        || r.d.r5c_layer_capture.is_some()
        || r.d.ttf_layer_capture.is_some()
        || r.d.r5b_layer_capture.is_some()
        || r.d.r5h_layer_capture.is_some()
        || r.d.r5s_frame_capture.is_some()
    {
        pin_global_rand(&mut r.d.g);
    }

    // 執行期失敗即關閉（loadErr）會讓 pump 直接失敗；離開前把它記成最後一筆 checkpoint（Go 的 defer）。
    let outcome = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        if entry_mode == "start" {
            r.start_from_normal_title();
        } else {
            let g = &mut r.d.g;
            g.title_phase = "menu".into();
            g.title_sel = 0;
            assert!(g.apply_title_menu_event(TITLE_MENU_DOWN));
            assert!(g.apply_title_menu_event(TITLE_MENU_CONFIRM));
            for _ in 0..24 {
                g.apply_title_menu_event(TITLE_MENU_TICK);
            }
            assert!(g.apply_title_slot_event(TITLE_SLOT_CONFIRM), "讀取第一槽失敗：{}", g.msg);
            let standalone = node(g).is_some_and(|n| n.r#type == "preparation" && n.cancel.is_empty())
                && node_id(g) == format!("preparation_ch{chapter:02}");
            assert!(
                (node_id(g) == r.town || standalone) && g.load_err.is_empty(),
                "讀檔後節點為 {}，錯誤訊息為 {}",
                node_id(g),
                g.load_err
            );
        }
        r.settle_town();
        r.run_actions(&actions);
        let ai_state = ai.lock().unwrap();
        eprintln!(
            "AI 入口比較：原版 {} 筆、已消費 {} 筆、順序分岔 {} 筆",
            ai_state.entries.len(),
            ai_state.cursor,
            ai_state.divergences
        );
        let (total, cursor) = (ai_state.entries.len(), ai_state.cursor);
        drop(ai_state);
        if total > 0 && cursor != total {
            let ui = r.ui();
            r.checkpoint(
                "ai_order",
                0,
                &ui,
                false,
                &[format!("divergence: 原版 AI 入口 {total} 筆，重製只消費 {cursor} 筆")],
            );
        }
        if env("FD2_PARITY_DEFEAT_TAIL") == "1" {
            todo!("R8: chapter_parity_replay_test.go verifyObservedDefeatTail")
        }
        let (ui, with_frame) = (r.ui(), r.d.g.title_phase.is_empty());
        r.checkpoint("end", 0, &ui, with_frame, &[]);
        if let Some(capture) = r.d.frame_capture.take() {
            capture.finish(serde_json::Map::new());
        }
        if let Some(capture) = r.d.map_layer_capture.take() {
            capture.finish();
        }
        if let Some(capture) = r.d.dialogue_layer_capture.take() {
            capture.finish();
        }
        if let Some(capture) = r.d.ttf_layer_capture.take() {
            capture.finish();
        }
        if let Some(capture) = r.d.r5p_layer_capture.take() {
            let mut extra = serde_json::Map::new();
            extra.insert("scope".into(), "第一章受控重播的每個 pump 幀（同 TestRustGoldenFrames chapter 模式）".into());
            capture.finish("chapter", extra);
        }
        if let Some(capture) = r.d.r5c_layer_capture.take() {
            capture.finish();
        }
        if let Some(capture) = r.d.r5b_layer_capture.take() {
            capture.finish();
        }
        if let Some(capture) = r.d.r5h_layer_capture.take() {
            capture.finish();
        }
        if let Some(capture) = r.d.r5s_frame_capture.take() {
            capture.finish();
        }
    }));
    if let Err(panic) = outcome {
        let g = &r.d.g;
        if !g.load_err.is_empty() {
            let cp = ParityCheckpoint {
                kind: "runtime_error".to_string(),
                node: node_id(g),
                ui: parity_ui(g),
                gold: g.gold,
                rng_word: i64::from(g.native_rng_state),
                units: parity_units(g),
                note: format!("loadErr: {}", g.load_err),
                ..Default::default()
            };
            parity_write(&log, cp);
        }
        std::panic::resume_unwind(panic);
    }
}
