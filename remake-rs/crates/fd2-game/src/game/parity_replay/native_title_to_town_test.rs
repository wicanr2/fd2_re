//! Go `remake/cmd/fd2/native_title_to_town_test.go` 的旅程紀錄與故事驅動：`journeyTrace`、`observe`、
//! `milestone`、`storyEnterReady`、`driveStory`、`resultInputReachable`、`writeJourney`。
//! `TestPlayTitleStartThroughChapterOneToTown` 與 `compareJourneyToEvidence` 是另一條驗收，R8 再移植。

use super::*;
use serde::Serialize;

/// Go `native_title_to_town_test.go:31`..`40`。
pub(crate) const JOURNEY_SEED: &str = "20260911";
pub(crate) const JOURNEY_TITLE_FRAMES: i64 = 20000;
pub(crate) const JOURNEY_STORY_FRAMES: i64 = 400000;
pub(crate) const JOURNEY_STALL_FRAMES: i64 = 6000;
pub(crate) const JOURNEY_BATTLE_NODE: &str = "battle_ch01";
pub(crate) const JOURNEY_POST_NODE: &str = "story_ch02";
pub(crate) const JOURNEY_OPENING_NODE: &str = "story_ch00_handler";
pub(crate) const JOURNEY_TOWN_SETTLE_MAX: i64 = 600;

fn is_zero_i64(v: &i64) -> bool {
    *v == 0
}

fn is_false(v: &bool) -> bool {
    !*v
}

/// Go `native_title_to_town_test.go:43`：`journeyLine`（欄位順序與 JSON 標籤照 Go）。
#[derive(Debug, Clone, Default, Serialize)]
pub(crate) struct JourneyLine {
    pub(crate) node: String,
    pub(crate) phase: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub(crate) source: String,
    pub(crate) string: i64,
    pub(crate) utterance: i64,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub(crate) control: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub(crate) text: String,
    pub(crate) pages: i64,
    #[serde(skip_serializing_if = "is_zero_i64")]
    pub(crate) turn: i64,
    #[serde(rename = "presses_before")]
    pub(crate) presses: i64,
    #[serde(skip_serializing_if = "is_false")]
    pub(crate) death: bool,
}

/// Go `native_title_to_town_test.go:57`：`journeyUnit`。
#[derive(Debug, Clone, Default, Serialize)]
pub(crate) struct JourneyUnit {
    pub(crate) camp: i64,
    pub(crate) group: i64,
    pub(crate) record: [i64; 2],
    pub(crate) at: [i64; 2],
    pub(crate) hp: i64,
    #[serde(rename = "identity")]
    pub(crate) id: i64,
    #[serde(rename = "has_identity")]
    pub(crate) has_id: bool,
}

/// Go `native_title_to_town_test.go:67`：`journeySpawn`。Go 以 `*battle.Unit` 指標記住單位，Rust 記 `st.units`
/// 的索引。
#[derive(Debug, Clone, Default, Serialize)]
pub(crate) struct JourneySpawn {
    pub(crate) turn: i64,
    pub(crate) camp: i64,
    pub(crate) group: i64,
    pub(crate) record: [i64; 2],
    #[serde(rename = "first_seen")]
    pub(crate) first: [i64; 2],
    pub(crate) settled: [i64; 2],
    #[serde(skip)]
    pub(crate) settled_done: bool,
    #[serde(skip)]
    pub(crate) unit: Option<UnitIdx>,
}

/// Go `native_title_to_town_test.go:78`：`journeyMilestone`。
#[derive(Debug, Clone, Default, Serialize)]
pub(crate) struct JourneyMilestone {
    pub(crate) presses: i64,
    pub(crate) node: String,
    pub(crate) turn: i64,
    #[serde(rename = "own_on_field")]
    pub(crate) own: i64,
    #[serde(rename = "enemy_on_field")]
    pub(crate) enemy: i64,
    #[serde(rename = "ally_on_field")]
    pub(crate) ally: i64,
    #[serde(rename = "enemy_pending")]
    pub(crate) pending: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) units: Option<Vec<JourneyUnit>>,
}

/// Go `native_title_to_town_test.go:102`：`journeyTrace`。第一章章重播只用到開場到第一戰取得操作權的部分，
/// `town` 一律是 `null`（Go 的 `Town` 只由 `runTitleToTownJourney` 填）。
#[derive(Debug, Default, Serialize)]
pub(crate) struct JourneyTrace {
    pub(crate) seed: String,
    pub(crate) nodes: Option<Vec<String>>,
    pub(crate) lines: Option<Vec<JourneyLine>>,
    pub(crate) presses: i64,
    #[serde(rename = "presses_by_phase")]
    pub(crate) presses_by_phase: BTreeMap<String, i64>,
    #[serde(rename = "first_battlefield_line")]
    pub(crate) first_map_line: Option<JourneyMilestone>,
    #[serde(rename = "player_control")]
    pub(crate) player_control: Option<JourneyMilestone>,
    pub(crate) spawns: Option<Vec<JourneySpawn>>,
    pub(crate) result: String,
    #[serde(rename = "result_turn")]
    pub(crate) result_turn: i64,
    pub(crate) town: Option<()>,
    pub(crate) frames: i64,

    #[serde(skip)]
    last_node: String,
    #[serde(skip)]
    last_line_sig: String,
    /// Go `known map[*battle.Unit]bool`：Rust 記 `st.units` 的索引；戰場狀態換掉（節點改變）時清空。
    #[serde(skip)]
    known: BTreeSet<usize>,
    #[serde(skip)]
    known_node: String,
}

/// Go `native_title_to_town_test.go:121`：`newJourneyTrace`。
pub(crate) fn new_journey_trace() -> JourneyTrace {
    JourneyTrace { seed: JOURNEY_SEED.to_string(), ..Default::default() }
}

/// `g.camp.NodeID()`；Go 的 `g.camp` 在這些路徑上不會是 nil。
pub(crate) fn node_id(g: &Game) -> String {
    g.camp.as_ref().map(|c| c.node_id()).unwrap_or_default()
}

/// `g.camp.Node()` 的複本（Rust 的 `handle_native_story_input` 收 `Option<&mut Node>`）。
pub(crate) fn node(g: &Game) -> Option<campaign::Node> {
    g.camp.as_ref().and_then(|c| c.node().cloned())
}

/// Go `native_title_to_town_test.go:128`：`journeyPhase`。
pub(crate) fn journey_phase(g: &Game) -> String {
    if !g.title_phase.is_empty() {
        return "title".to_string();
    }
    let Some(n) = node(g) else {
        return "none".to_string();
    };
    match node_id(g).as_str() {
        JOURNEY_OPENING_NODE => "opening".to_string(),
        JOURNEY_BATTLE_NODE => "battle".to_string(),
        JOURNEY_POST_NODE => "postbattle".to_string(),
        _ => n.r#type,
    }
}

/// Go `native_title_to_town_test.go:151`：`journeyTurn`。
pub(crate) fn journey_turn(g: &Game) -> i64 {
    g.st.as_ref().map_or(0, |st| st.turn)
}

/// Go `native_chapter_one_play_test.go` 的 `nativeCampCode` 之外，旅程用的是 `int(u.Camp)`（Go 的 iota 值）。
fn camp_int(camp: battle::Camp) -> i64 {
    camp.0
}

impl JourneyTrace {
    /// Go `native_title_to_town_test.go:146`：`press`。
    pub(crate) fn press(&mut self, phase: &str) {
        self.presses += 1;
        *self.presses_by_phase.entry(phase.to_string()).or_default() += 1;
    }

    /// Go `native_title_to_town_test.go:159`：`observe`，每一幀呼叫一次。
    pub(crate) fn observe(&mut self, g: &Game) {
        self.frames += 1;
        let mut node_name = String::new();
        if g.camp.is_some() {
            node_name = node_id(g);
            if let Some(n) = node(g)
                && node_name != self.last_node
            {
                self.nodes.get_or_insert_with(Vec::new).push(format!("{node_name}:{}", n.r#type));
            }
        }
        self.last_node = node_name.clone();

        if g.dialog.is_empty() {
            self.last_line_sig.clear();
        } else {
            let top = g.dialog.last().unwrap();
            let mut line = JourneyLine {
                node: node_name.clone(),
                phase: journey_phase(g),
                presses: self.presses,
                turn: journey_turn(g),
                death: g.death_program_running,
                ..Default::default()
            };
            if let Some(nd) = top.native_dialogue.as_ref() {
                line.source = nd.source_dat.clone();
                line.string = nd.string_index;
                line.utterance = nd.utterance;
                line.control = nd.control.clone();
                line.pages = nd.pages.len() as i64;
            } else {
                line.string = -1;
                line.utterance = -1;
                line.text = top.text.clone();
                line.pages = dlg_page_count(top);
            }
            let sig = format!(
                "{node_name}|{}|{}|{}|{}|{}|{}",
                g.beat_idx,
                g.dialog.len(),
                line.source,
                line.string,
                line.utterance,
                line.text
            );
            if sig != self.last_line_sig {
                let first_map = self.first_map_line.is_none() && line.source == "FDTXT_001";
                self.lines.get_or_insert_with(Vec::new).push(line);
                if first_map {
                    self.first_map_line = Some(self.milestone(g, false));
                }
            }
            self.last_line_sig = sig;
        }

        if let Some(st) = g.st.as_ref() {
            if self.known_node != node_name {
                self.known.clear();
                self.known_node = node_name.clone();
            }
            for (i, u) in st.units.iter().enumerate() {
                if self.known.contains(&i) || !u.on_field {
                    continue;
                }
                self.known.insert(i);
                if node_name != JOURNEY_BATTLE_NODE || self.player_control.is_none() {
                    continue; // 開局的單位記在 player_control，不算援軍
                }
                self.spawns.get_or_insert_with(Vec::new).push(JourneySpawn {
                    turn: st.turn,
                    camp: camp_int(u.camp),
                    group: u.group,
                    record: [i64::from(u.native_position_record.x_word), i64::from(u.native_position_record.y_word)],
                    first: [u.x, u.y],
                    unit: Some(i),
                    ..Default::default()
                });
            }
        }
    }

    /// Go `native_title_to_town_test.go:221`：`milestone`。
    pub(crate) fn milestone(&self, g: &Game, with_units: bool) -> JourneyMilestone {
        let mut m = JourneyMilestone { presses: self.presses, turn: journey_turn(g), ..Default::default() };
        if g.camp.is_some() {
            m.node = node_id(g);
        }
        let Some(st) = g.st.as_ref() else {
            return m;
        };
        for u in &st.units {
            if !u.on_field || !u.alive() {
                continue;
            }
            match u.camp {
                battle::OWN => m.own += 1,
                battle::ENEMY => m.enemy += 1,
                battle::ALLY => m.ally += 1,
                _ => {}
            }
            if with_units {
                m.units.get_or_insert_with(Vec::new).push(JourneyUnit {
                    camp: camp_int(u.camp),
                    group: u.group,
                    record: [i64::from(u.native_position_record.x_word), i64::from(u.native_position_record.y_word)],
                    at: [u.x, u.y],
                    hp: u.hp,
                    id: u.native_identity,
                    has_id: u.has_native_identity,
                });
            }
        }
        m.pending = st.pending_count(battle::ENEMY);
        m
    }
}

/// Go `native_title_to_town_test.go:254`：`storyEnterReady`，故事對白停在等鍵。
pub(crate) fn story_enter_ready(g: &Game) -> bool {
    if g.dialog.is_empty() || g.dlg_phase != 0 || g.dlg_scroll_t != 0 || g.native_dialogue_closing_live {
        return false;
    }
    let top = g.dialog.last().unwrap();
    if top.native_dialogue.is_none() {
        return g.story_walks.is_empty() && g.fade.is_none();
    }
    let page = g.dlg_page;
    page >= 0
        && (page as usize) < g.native_dialogue_progressive.len()
        && !g.native_dialogue_progressive[page as usize].is_empty()
        && g.native_dialogue_progress >= g.native_dialogue_progressive[page as usize].len() as i64 - 1
}

/// Go `native_title_to_town_test.go:267`：`storySignature`。
pub(crate) fn story_signature(g: &Game) -> String {
    format!(
        "{}|{}|{}|{}|{}|{}",
        node_id(g),
        g.beat_idx,
        g.dialog.len(),
        g.dlg_page,
        g.native_dialogue_progress,
        g.native_dialogue_closing_live
    )
}

/// Go `native_title_to_town_test.go:277`：`progressSignature`，只用來偵測「卡住」。
pub(crate) fn progress_signature(g: &Game) -> String {
    let mut s = story_signature(g);
    if let Some(st) = g.st.as_ref() {
        s += &format!("|t{}|ai{}|r{}", st.turn, g.ai_busy, g.result);
    }
    format!("{s}|{}|{}|{}|{}", g.title_phase, g.title_flash, g.battle_event.is_some(), g.native_turn_staging.is_some())
}

/// Go `native_title_to_town_test.go:335`：`resultInputReachable`。
pub(crate) fn result_input_reachable(g: &Game) -> bool {
    !g.result.is_empty()
        && !g.ai_busy
        && g.atk.is_none()
        && g.walk.is_none()
        && g.battle_event.is_none()
        && g.native_turn_staging.is_none()
        && g.dialog.is_empty()
        && !g.ring
        && g.native_class_ui_job.is_none()
        && g.spawn_intro_transition.is_none()
        && g.native_unit_present.is_none()
        && g.native_ai_idle_recovery.is_none()
        && g.native_heal_presentation.is_none()
        && g.native_modifier_presentation.is_none()
        && g.native_ai_command_modifier.is_none()
        && g.native_ai_item_presentation.is_none()
        && g.native_cmd0_presentation.is_none()
        && g.native_cmd1_presentation.is_none()
}

impl Driver {
    /// Go `native_title_to_town_test.go:287`：`driveStory`。推過故事／處理器過場節點，直到 `stop` 回報 true。
    /// Go 這裡直接呼叫 `g.Update()`（不經 `parityUpdateGame`）。
    pub(crate) fn drive_story(&mut self, j: &mut JourneyTrace, mut stop: impl FnMut(&Game) -> bool) {
        let (mut last_sig, mut still) = (String::new(), 0);
        for _ in 0..JOURNEY_STORY_FRAMES {
            j.observe(&self.g);
            if stop(&self.g) {
                return;
            }
            if let Some(mut n) = node(&self.g)
                && (n.r#type == "story" || n.r#type == "cutscene")
                && story_enter_ready(&self.g)
            {
                let before = story_signature(&self.g);
                self.g.handle_native_story_input(Some(&mut n), NativeStoryInput { enter: true, escape: false });
                if story_signature(&self.g) != before {
                    j.press(&journey_phase(&self.g));
                }
            }
            ack_presents(&mut self.g);
            self.g.advance_virtual_frame(&mut self.clock);
            if let Err(err) = self.g.update() {
                panic!("Update：{err}");
            }
            let g = &self.g;
            if !g.load_err.is_empty() {
                panic!("故事節點 {} 執行期錯誤：{}\n阻塞：{}", node_id(g), g.load_err, ch01_blockers(g));
            }
            let sig = progress_signature(g);
            if sig == last_sig {
                still += 1;
                if still > JOURNEY_STALL_FRAMES {
                    panic!(
                        "故事節點 {} 卡住 {still} 幀（beat={} dialog={} page={} progress={}）\n阻塞：{}",
                        node_id(g),
                        g.beat_idx,
                        g.dialog.len(),
                        g.dlg_page,
                        g.native_dialogue_progress,
                        ch01_blockers(g)
                    );
                }
            } else {
                (last_sig, still) = (sig, 0);
            }
        }
        panic!("故事節點 {} 在 {JOURNEY_STORY_FRAMES} 幀內沒有結束", node_id(&self.g));
    }
}

/// Go `native_title_to_town_test.go:506`：`writeJourney`，`json.MarshalIndent(j, "", " ")` 加換行。
pub(crate) fn write_journey(path: &std::path::Path, j: &JourneyTrace) {
    let mut out = Vec::new();
    let formatter = serde_json::ser::PrettyFormatter::with_indent(b" ");
    let mut ser = serde_json::Serializer::with_formatter(&mut out, formatter);
    j.serialize(&mut ser).expect("journey JSON");
    out.push(b'\n');
    std::fs::write(path, out).expect("寫 start-entry.json");
}
