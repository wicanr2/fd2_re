//! Go `remake/cmd/fd2/rust_golden_r5b_layers_test.go` 的 Rust 端（R5 階段 2 r5b 組「戰鬥演出」的局部黃金，預設關閉）。
//!
//! 設 `FD2_GOLDEN_R5B_LAYERS_OUT=<目錄>` 時，章重播的 [`Driver::pump`] 每一幀開頭呼叫 [`R5bLayerCapture::observe`]
//! （Go `ch01FrameObserver` 同一位置）。掛點與 60 Hz 虛擬時鐘同 R5 黃金畫格 chapter 模式，全域亂數由章重播的
//! `pin_global_rand` 接上（Go `rand.Seed(4)`）。
//!
//! 每次擷取依 `Draw` 的 early-return 順序分支（做法見 Go 檔開頭）：`ch28`、`idle`、`atk` 三個本組分支與只有自己一層的
//! `ramp`、`spawn` 呼叫整個 [`Game::draw`]，記整張 640×400 雜湊（本組分支另記 Draw 前後的工作狀態）；尾段只照 `Draw`
//! 把原生整幀畫到另一張畫布（留下 `composeNativeMapFrame` 對 `native_map_vga`／`native_map_dac` 的重建，鍵只記成功
//! 與否）；其餘 early 分支不畫。比對鍵與 Go 逐字同格式，重播結束寫 `<目錄>/r5b-layers.json`，
//! `tools/rust_r5b_layers_compare.py` 逐擷取點比對，並把本組分支的雜湊與階段 0 整張黃金同一擷取點對照。

use super::*;
use sha2::Digest;
use std::path::PathBuf;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const R5B_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const R5B_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct R5bLayerRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) update_frame: i64,
    pub(crate) node: String,
    pub(crate) key: String,
}

/// Go `r5bLayerRecorder`。
pub(crate) struct R5bLayerCapture {
    out: PathBuf,
    screen: Picture,
    map_screen: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    seq: sha2::Sha256,
    rows: Vec<R5bLayerRow>,
    branches: BTreeMap<String, i64>,
    stats: BTreeMap<String, i64>,
}

fn sha_hex(raw: &[u8]) -> String {
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go `r5bLayerBranch`：`Draw`（`main.go:9398..9805`）的 early-return 順序，只讀欄位。
pub(crate) fn r5b_layer_branch(g: &Game) -> &'static str {
    if g.startup_blocked
        || g.native_defeat.is_some()
        || !g.title_phase.is_empty()
        || g.native_ending.is_some()
        || g.native_ch20_sky_key.is_some()
        || g.native_ch23_loop.is_some()
        || g.native2189_a.is_some()
        || g.native_unit_present.is_some()
    {
        return "early";
    }
    if g.native_ch28_post_present.is_some() {
        return "ch28";
    }
    if g.transition_reveal.is_some()
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
    {
        return "early";
    }
    if g.native_palette_ramp.is_some() {
        return "ramp";
    }
    if g.native_palette_pulse.is_some() {
        return "early";
    }
    if g.spawn_intro_transition.is_some() {
        return "spawn";
    }
    if g.native_turn_staging.as_ref().is_some_and(|j| j.indexed)
        || g.native_field_event61.is_some()
        || g.native_end_turn_recovery.is_some()
    {
        return "early";
    }
    if g.native_ai_idle_recovery.is_some() {
        return "idle";
    }
    if g.atk.is_some() {
        return "atk";
    }
    "tail"
}

/// Go `r5bAtkState`：Draw 前後的物理演出狀態。
fn atk_state(g: &Game) -> String {
    let Some(a) = g.atk.as_ref() else {
        return "atk=nil".into();
    };
    let Some(s) = a.native_scene.as_ref() else {
        return "scene=nil".into();
    };
    let body = match s.body.as_ref() {
        None => "none".to_string(),
        Some(p) => format!(
            "{}/{},drawn={},image={},last={},wait={},ramp={}",
            p.index,
            p.jobs.len(),
            p.drawn,
            p.image.is_some(),
            p.last_present,
            p.return_wait_millis > 0.0,
            p.return_ramp_started
        ),
    };
    format!(
        "prelude={}/{}|prelude_drawn={}|lead={}/{}|lead_drawn={}|counter={}|body={body}",
        s.prelude_frame,
        s.prelude_images.len(),
        s.prelude_drawn,
        s.lead_frame,
        s.lead_images.len(),
        s.lead_drawn,
        s.counter_active
    )
}

impl R5bLayerCapture {
    /// 讀 `FD2_GOLDEN_R5B_LAYERS_OUT`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<R5bLayerCapture> {
        let out = std::env::var("FD2_GOLDEN_R5B_LAYERS_OUT").unwrap_or_default();
        if out.is_empty() {
            return None;
        }
        for other in [
            "FD2_GOLDEN_FRAMES_OUT",
            "FD2_GOLDEN_MAP_LAYERS_OUT",
            "FD2_GOLDEN_DIALOGUE_LAYERS_OUT",
            "FD2_GOLDEN_R5P_LAYERS_OUT",
            "FD2_GOLDEN_R5C_LAYERS_OUT",
            "FD2_GOLDEN_TTF_LAYERS_OUT",
        ] {
            assert!(
                std::env::var(other).unwrap_or_default().is_empty(),
                "FD2_GOLDEN_R5B_LAYERS_OUT 與 {other} 不能同時設（Go 端只有一個 ch01FrameObserver）"
            );
        }
        Some(R5bLayerCapture {
            out: out.into(),
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
        })
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
            g.player_clock.wall = R5B_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = R5B_FAR_FUTURE;
        let step = d * R5B_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str) {
        *self.stats.entry(name.to_string()).or_default() += 1;
    }

    /// Go `r5bDraw`：清空後呼叫整個 Draw，回傳整張 640×400 的雜湊。
    fn draw(&mut self, g: &mut Game) -> String {
        self.screen.clear();
        g.draw(&mut self.screen);
        sha_hex(self.screen.read_pixels())
    }

    /// Go `r5bLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = r5b_layer_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        match branch {
            "ch28" => {
                assert!(
                    g.modern_story_portraits.is_none(),
                    "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）"
                );
                let j = g.native_ch28_post_present.as_ref().unwrap();
                let (frame, frames) = (j.frame, j.vga_frames.len());
                let rgba = self.draw(g);
                self.stat("ch28_frames");
                let state = match g.native_ch28_post_present.as_ref() {
                    None => "job=nil".to_string(),
                    Some(j) => format!("drawn={}|presented={}|hd={}", j.drawn, j.presented_frame, j.presented_hd),
                };
                format!("ch28|frame={frame}/{frames}|{state}|load_err={}|rgba={rgba}", !g.load_err.is_empty())
            }
            "idle" => {
                let j = g.native_ai_idle_recovery.as_ref().unwrap();
                let (frame, frames) = (j.frame, j.frames.len());
                let dac = g.native_map_dac.len();
                let rgba = self.draw(g);
                self.stat("idle_frames");
                let state = match g.native_ai_idle_recovery.as_ref() {
                    None => "job=nil".to_string(),
                    Some(j) => format!("drawn={}", j.drawn),
                };
                format!("idle|frame={frame}/{frames}|dac={dac}|{state}|load_err={}|rgba={rgba}", !g.load_err.is_empty())
            }
            "atk" => {
                assert!(
                    g.modern_story_portraits.is_none(),
                    "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）"
                );
                let before = atk_state(g);
                let rgba = self.draw(g);
                let after = atk_state(g);
                self.stat("atk_frames");
                if let Some(s) = g.atk.as_ref().and_then(|a| a.native_scene.as_ref()) {
                    if s.prelude_frame < s.prelude_images.len() as i64 {
                        self.stat("atk_prelude");
                    } else if s.body.is_some() {
                        self.stat("atk_body");
                    }
                }
                format!("atk|before={before}|after={after}|load_err={}|rgba={rgba}", !g.load_err.is_empty())
            }
            "ramp" | "spawn" => format!("{branch}|rgba={}", self.draw(g)),
            "tail" => {
                let campaign_battle_view =
                    g.camp.is_none() || g.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == "battle");
                if g.native_map_frame_admission(g.story_bg, campaign_battle_view) {
                    self.map_screen.clear();
                    return format!("tail|map_ok={}", g.draw_native_map_frame(&mut self.map_screen));
                }
                "tail".to_string()
            }
            other => other.to_string(),
        }
    }

    /// Go `r5bLayerRecorder.observe`：在一個 `update` 之後呼叫。
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
        self.rows.push(R5bLayerRow { capture: index, count: 1, update_frame: g.frame, node: node_id(g), key });
    }

    /// Go `r5bLayerRecorder.finish`：寫 `r5b-layers.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5b_layers/1",
            "mode": "chapter",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_r5b_layers_test.rs（tools/rust_r5b_layers.sh）",
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
        std::fs::write(self.out.join("r5b-layers.json"), raw).expect("寫 r5b-layers.json");
    }
}
