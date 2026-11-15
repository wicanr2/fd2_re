//! Go `remake/cmd/fd2/rust_golden_r5h_layers_test.go` 的 Rust 端（R5 階段 2 r5h：HUD 與選單組的局部黃金，預設關閉）。
//!
//! 設 `FD2_GOLDEN_R5H_LAYERS_OUT=<目錄>` 時，章重播的 [`Driver::pump`] 每一幀開頭呼叫 [`R5hLayerCapture::observe`]
//! （Go `ch01FrameObserver` 同一位置）。掛點、60 Hz 虛擬時鐘與全域亂數同 R5 黃金畫格的 chapter 模式。
//!
//! 每次擷取依 `Draw` 的分支順序只畫本組的圖層（做法見 Go 檔開頭）：
//! - recovery：`Draw` 走到 `nativeEndTurnRecovery` 分支時呼叫整個 [`Game::draw`]（`Fill` 黑、END 回收、`defer`
//!   的 DAC 白／黑 `Fill`，沒有別組圖層）。
//! - tail：游標與左下 HUD（[`Game::draw_field_cursor_hud`]）、移動面板與回合橫幅（[`Game::draw_field_map_overlays`]）、
//!   設施／班級工作、END 對話、指令環、選單、系統資訊與角色狀態（[`Game::draw_field_command_ui`]）各畫到一張
//!   `Clear` 的 640×400 畫布；中間照 `Draw` 把原生整幀畫到另一張畫布（取 `nativeMapPresented` 與它對狀態的改動）。
//!
//! 比對鍵與 Go 逐字同格式，重播結束寫 `<目錄>/r5h-layers.json`，`tools/rust_r5h_layers_compare.py` 逐擷取點比對。

use super::*;
use sha2::Digest;
use std::path::PathBuf;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const R5H_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const R5H_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct R5hLayerRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) update_frame: i64,
    pub(crate) node: String,
    pub(crate) key: String,
}

/// Go `r5hLayerRecorder`。
pub(crate) struct R5hLayerCapture {
    out: PathBuf,
    screen: Picture,
    map_screen: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    seq: sha2::Sha256,
    rows: Vec<R5hLayerRow>,
    branches: BTreeMap<String, i64>,
    stats: BTreeMap<String, i64>,
}

/// Go `r5hLayerSHA`：畫布全為 0（全透明，本層沒有畫）時回 `0`，否則 RGBA 的 SHA-256。
fn r5h_layer_sha(pix: &[u8]) -> String {
    if pix.iter().all(|&b| b == 0) {
        return "0".to_string();
    }
    sha2::Sha256::digest(pix).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go `r5hLayerBranch`：`Draw`（`main.go:9399..9800`）的 early-return 順序，只讀欄位：`spawnIntroTransition` 之前的
/// 分支（early）、spawn、turn staging（indexed）與事件 61（early）、END 回收（recovery）、AI 閒置回收與攻擊（early）、
/// 尾段（tail）。除了 recovery 以外與 [`golden_map_layers_branch`] 相同。
pub(crate) fn r5h_layer_branch(g: &Game) -> &'static str {
    if r5h_pre_spawn_early(g) {
        return "early";
    }
    if g.spawn_intro_transition.is_some() {
        return "spawn";
    }
    if g.native_turn_staging.as_ref().is_some_and(|j| j.indexed) || g.native_field_event61.is_some() {
        return "early";
    }
    if g.native_end_turn_recovery.is_some() {
        return "recovery";
    }
    if g.native_ai_idle_recovery.is_some() || g.atk.is_some() {
        return "early";
    }
    "tail"
}

/// `Draw` 在 `spawnIntroTransition` 之前 return 的分支（`main.go:9399..9738`），與 Go `r5hPreSpawnEarly` 相同。
fn r5h_pre_spawn_early(g: &Game) -> bool {
    g.startup_blocked
        || g.native_defeat.is_some()
        || !g.title_phase.is_empty()
        || g.native_ending.is_some()
        || g.native_ch20_sky_key.is_some()
        || g.native_ch23_loop.is_some()
        || g.native2189_a.is_some()
        || g.native_unit_present.is_some()
        || g.native_ch28_post_present.is_some()
        || g.transition_reveal.is_some()
        || g.m.is_none()
        || g.indexed_transition.is_some()
        || g.native_heal_presentation.is_some()
        || g.native_modifier_presentation.is_some()
        || g.native_ai_command_modifier.is_some()
        || g.native_cmd0_presentation.is_some()
        || g.native_cmd6_presentation.is_some()
        || g.native_cmd1_presentation.is_some()
        || g.native_cmd2_presentation.is_some()
        || g.native_cmd3_presentation.is_some()
        || g.native_cmd5_presentation.is_some()
        || g.native_cmd7_presentation.is_some()
        || g.native_cmd8_presentation.is_some()
        || g.native_cmd9_player.is_some()
        || g.native_cmd9_ai_presentation.is_some()
        || g.native_cmd1012.is_some()
        || g.native_cmd24_presentation.is_some()
        || g.native_cmd29_presentation.is_some()
        || g.native_cmd32_presentation.is_some()
        || g.native_cmd33_presentation.is_some()
        || g.native_cmd34_presentation.is_some()
        || g.native_cmd35_presentation.is_some()
        || g.native_ai_item_presentation.is_some()
        || g.native_town_departure.is_some()
        || g.native_palette_ramp.is_some()
        || g.native_palette_pulse.is_some()
}

impl R5hLayerCapture {
    pub(crate) fn new(out: PathBuf) -> R5hLayerCapture {
        R5hLayerCapture {
            out,
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            map_screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            pinned: false,
            clock_mode: "",
            last_frame: 0,
            virtual_frames: 0,
            captures: 0,
            seq: sha2::Sha256::new(),
            rows: Vec::new(),
            branches: BTreeMap::new(),
            stats: BTreeMap::new(),
        }
    }

    /// 讀 `FD2_GOLDEN_R5H_LAYERS_OUT`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<R5hLayerCapture> {
        let var = |name: &str| std::env::var(name).unwrap_or_default();
        let out = var("FD2_GOLDEN_R5H_LAYERS_OUT");
        if out.is_empty() {
            return None;
        }
        assert!(
            var("FD2_GOLDEN_FRAMES_OUT").is_empty()
                && var("FD2_GOLDEN_MAP_LAYERS_OUT").is_empty()
                && var("FD2_GOLDEN_DIALOGUE_LAYERS_OUT").is_empty()
                && var("FD2_GOLDEN_R5P_LAYERS_OUT").is_empty()
                && var("FD2_GOLDEN_R5C_LAYERS_OUT").is_empty()
                && var("FD2_GOLDEN_TTF_LAYERS_OUT").is_empty(),
            "FD2_GOLDEN_R5H_LAYERS_OUT 不能與其他擷取同時設（Go 端只有一個 ch01FrameObserver）"
        );
        Some(R5hLayerCapture::new(out.into()))
    }

    /// Go `goldenFrameRecorder.pinClock`（同 `FrameCapture::pin_clock`）。
    fn pin_clock(&mut self, g: &mut Game) {
        if !self.pinned {
            self.pinned = true;
            self.last_frame = g.frame;
            if g.player_clock.wall.is_zero() {
                g.player_clock.game = GameInstant((1i64 << 30) * 1_000_000_000);
                self.clock_mode = "virtual-60hz-fixed-origin";
            } else {
                self.clock_mode = "virtual-60hz-from-first-pump";
            }
            g.player_clock.wall = R5H_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = R5H_FAR_FUTURE;
        let step = d * R5H_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str, n: i64) {
        *self.stats.entry(name.to_string()).or_default() += n;
    }

    /// 本層畫布清成全透明、畫、取雜湊；有畫的層另記一次統計。
    fn layer(&mut self, name: &str, g: &mut Game, draw: impl FnOnce(&mut Game, &mut Picture)) -> String {
        self.screen.clear();
        draw(g, &mut self.screen);
        let sha = r5h_layer_sha(self.screen.read_pixels());
        if sha != "0" {
            self.stat(&format!("{name}_drawn"), 1);
        }
        sha
    }

    /// Go `r5hLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        // Go：照 Draw 開頭清三個本幀旗標（同 rust_golden_map_layers_test.rs）。
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = r5h_layer_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        match branch {
            "recovery" => {
                let frame = g.native_end_turn_recovery.as_ref().unwrap().frame;
                self.screen.clear();
                g.draw(&mut self.screen);
                let rgba = r5h_layer_sha(self.screen.read_pixels());
                let (alive, drawn) = match g.native_end_turn_recovery.as_ref() {
                    Some(j) => (true, j.drawn),
                    None => (false, false),
                };
                self.stat("recovery_frames", 1);
                format!("recovery|frame={frame}|alive={alive}|drawn={drawn}|rgba={rgba}")
            }
            "tail" => {
                assert!(
                    g.modern_story_portraits.is_none(),
                    "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）"
                );
                let legacy_viewport = g.story_bg || g.battle_event.is_some();
                let campaign_battle_view =
                    g.camp.is_none() || g.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == "battle");
                let cursor = !g.ring && !g.spell_open && !legacy_viewport && campaign_battle_view;
                let hud_unit = g.st.as_ref().is_some_and(|st| st.unit_at(g.cur_x, g.cur_y).is_some())
                    && g.font.is_some()
                    && !legacy_viewport;
                let cursor_hud = self.layer("cursor_hud", g, |g, screen| {
                    g.draw_field_cursor_hud(screen, legacy_viewport, campaign_battle_view)
                });
                self.stat("tail_frames", 1);
                if cursor {
                    self.stat("cursor", 1);
                }
                if hud_unit {
                    self.stat("hud_unit", 1);
                }
                let mut key = format!(
                    "tail|legacy={legacy_viewport}|cursor={cursor}|hud_unit={hud_unit}|cursor_hud={cursor_hud}"
                );
                // Go：Draw 的下一步是原生整幀（r5m），畫到另一張畫布，取 nativeMapPresented 與它對狀態的改動。
                let admitted = g.native_map_frame_admission(g.story_bg, campaign_battle_view);
                let mut presented = false;
                if admitted {
                    self.map_screen.clear();
                    presented = g.draw_native_map_frame(&mut self.map_screen);
                }
                key += &format!("|map={admitted}:{presented}");
                let banner_fallback = g.banner_t > 0
                    && !g.banner.is_empty()
                    && g.font.is_some()
                    && !g.phase_banner_glyphs_available()
                    && !legacy_viewport
                    && campaign_battle_view;
                let overlays = self.layer("overlays", g, |g, screen| {
                    g.draw_field_map_overlays(screen, presented, legacy_viewport, campaign_battle_view)
                });
                let moved = g.movement_panel_draw.is_some();
                if moved {
                    self.stat("move_panel", 1);
                }
                key += &format!("|move={moved}|banner_fallback={banner_fallback}|overlays={overlays}");
                // Go：TTF 段在語系缺鍵時 return 出 Draw（main.go:10137..10146），之後各層不畫。
                let ttf_return = g.font.is_some() && !g.battle_text_draws().1;
                key += &format!("|ttf_return={ttf_return}");
                if ttf_return {
                    return key;
                }
                let transient = g.transient_ui;
                let endturn_ui = g.native_system_end_turn_ui.is_some();
                let inmap = g.native_action_overlay_in_map_frame(presented);
                let info = g.native_system_info_ui.is_some();
                let status = g.native_player_status.is_some();
                let mut completed = true;
                let command = self.layer("command", g, |g, screen| {
                    completed = g.draw_field_command_ui(screen, presented);
                });
                if endturn_ui {
                    self.stat("endturn_ui", 1);
                }
                if inmap {
                    self.stat("overlay_inmap", 1);
                }
                key + &format!(
                    "|transient={transient}|endturn_ui={endturn_ui}|inmap={inmap}|info={info}|status={status}|command_ui={completed}|overlay_drawn={}|command={command}",
                    g.action_overlay_drawn
                )
            }
            other => other.to_string(),
        }
    }

    /// Go `r5hLayerRecorder.observe`：在一個 `update` 之後呼叫。
    pub(crate) fn observe(&mut self, g: &mut Game) {
        self.pin_clock(g);
        let key = self.layers(g);
        self.seq.update(key.as_bytes());
        self.seq.update(b"\n");
        let index = self.captures;
        self.captures += 1;
        if let Some(last) = self.rows.last_mut()
            && last.key == key
        {
            last.count += 1;
            return;
        }
        self.rows.push(R5hLayerRow { capture: index, count: 1, update_frame: g.frame, node: node_id(g), key });
    }

    /// Go `r5hLayerRecorder.finish`：寫 `r5h-layers.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5h_layers/1",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_r5h_layers_test.rs（tools/rust_r5h_layers.sh）",
            "scope": "第一章受控重播的每個 pump 幀（同 TestRustGoldenFrames chapter 模式）",
            "clock": self.clock_mode,
            "global_rand": GLOBAL_RAND_MODEL,
            "virtual_frames": self.virtual_frames,
            "captures": self.captures,
            "changed_rows": self.rows.len(),
            "key_sequence": "每次擷取一行比對鍵（rows[].key），串接後的 SHA-256",
            "sequence_sha256": seq,
            "branches": self.branches,
            "stats": self.stats,
            "rows": self.rows,
        });
        std::fs::create_dir_all(&self.out).expect("建立輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("r5h-layers.json"), raw).expect("寫 r5h-layers.json");
    }
}
