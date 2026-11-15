//! Go `remake/cmd/fd2/rust_golden_dialogue_layers_test.go` 的 Rust 端（R5 階段 2 r5d 組的局部黃金，預設關閉）。
//!
//! 設 `FD2_GOLDEN_DIALOGUE_LAYERS_OUT=<目錄>` 時，章重播的 [`Driver::pump`] 每一幀在上一個 `update` 之後、`done` 與
//! `ack_presents` 之前呼叫 [`DialogueLayerCapture::observe`]（與 Go `ch01FrameObserver` 同一位置）：推進與 Go 黃金
//! 工具相同的 60 Hz 虛擬時鐘，每次記嘴型狀態（全域亂數的消費者）；照 Go `Draw` 的分支順序走到尾段時呼叫
//! [`Game::draw_native_story_dialogue`]，畫到 `Fill` 黑的 640×400 畫布、讀回取雜湊，組成與 Go 逐字相同的比對鍵。
//! 重播結束寫 `<目錄>/dialogue-layers.json`，`tools/rust_r5_map_layers_compare.py` 逐擷取點比對兩邊（同一個
//! rows／key 格式）。打開時章重播以 [`pin_global_rand`] 接上 Go `rand.Seed(4)` 的全域來源。

use super::*;
use sha2::Digest;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const DIALOGUE_LAYER_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const DIALOGUE_LAYER_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

/// Go `dialogueLayerRecorder`（rows 沿用 [`MapLayerRow`]，與 Go 的 `mapLayerRow` 同欄位）。
pub(crate) struct DialogueLayerCapture {
    out: std::path::PathBuf,
    screen: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    seq: sha2::Sha256,
    rows: Vec<MapLayerRow>,
    branches: BTreeMap<String, i64>,
    stats: BTreeMap<String, i64>,
}

fn sha_hex(raw: &[u8]) -> String {
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go `goldenDialogueMouthKey`。
fn golden_dialogue_mouth_key(g: &Game) -> String {
    format!(
        "mouth={}:{}|confirm={}:{}:{}",
        g.mouth_state.open,
        g.mouth_state.countdown,
        g.native_confirmation_mouth_owner.is_some(),
        g.native_confirmation_mouth.open,
        g.native_confirmation_mouth.countdown
    )
}

/// Go `goldenDialogueStage`：回傳（階段、說話嘴型、等鍵張嘴、等鍵箭頭）。
fn golden_dialogue_stage(g: &Game) -> (&'static str, bool, bool, bool) {
    if g.native_dialogue_closing_live {
        return ("closing", false, false, false);
    }
    if g.dlg_phase == 2 {
        return ("opening", false, false, false);
    }
    if g.dlg_phase != 0 {
        return ("phase", false, false, false);
    }
    if g.dlg_page < 0
        || g.dlg_page >= g.native_dialogue_progressive.len() as i64
        || g.native_dialogue_progressive[g.dlg_page as usize].is_empty()
    {
        return ("empty", false, false, false);
    }
    let count = g.native_dialogue_progressive[g.dlg_page as usize].len() as i64;
    let progress = g.native_dialogue_progress.max(0).min(count - 1);
    let speaking = g.locale_id == "zh-Hant"
        && progress > 0
        && progress < count - 1
        && g.native_dialogue_speaking_frame > 0
        && g.native_dialogue_portraits.len() == 4;
    let mouth_open = g.mouth_open && progress == count - 1 && g.dlg_page < g.native_dialogue_mouth_open.len() as i64;
    let last_page_arrow = g.native_level_up_dialogue.as_ref().is_some_and(|d| d.await_last_key && d.close_timer == 0);
    let arrow = g.native_story_dialogue_at_input_wait()
        && (g.dlg_page + 1 < g.native_dialogue_progressive.len() as i64 || last_page_arrow);
    ("steady", speaking, mouth_open, arrow)
}

impl DialogueLayerCapture {
    /// 讀 `FD2_GOLDEN_DIALOGUE_LAYERS_OUT`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<DialogueLayerCapture> {
        let out = std::env::var("FD2_GOLDEN_DIALOGUE_LAYERS_OUT").unwrap_or_default();
        if out.is_empty() {
            return None;
        }
        for other in ["FD2_GOLDEN_FRAMES_OUT", "FD2_GOLDEN_MAP_LAYERS_OUT"] {
            assert!(
                std::env::var(other).unwrap_or_default().is_empty(),
                "FD2_GOLDEN_DIALOGUE_LAYERS_OUT 與 {other} 不能同時設（Go 端只有一個 ch01FrameObserver）"
            );
        }
        Some(DialogueLayerCapture {
            out: out.into(),
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
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
            g.player_clock.wall = DIALOGUE_LAYER_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = DIALOGUE_LAYER_FAR_FUTURE;
        let step = d * DIALOGUE_LAYER_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str) {
        *self.stats.entry(name.to_string()).or_default() += 1;
    }

    /// Go `dialogueLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        // Go：照 Draw 開頭清三個本幀旗標（忠實模式下其餘欄位都是 None）。
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let mouth = golden_dialogue_mouth_key(g);
        let branch = golden_map_layers_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        if branch != "tail" {
            return format!("{branch}|{mouth}");
        }
        assert!(g.modern_story_portraits.is_none(), "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）");
        let (stage, speaking, mouth_open, arrow) = golden_dialogue_stage(g);
        let state = format!(
            "stage={stage}|closing_t={}|dlg_t={}|page={}|progress={}|speaking={}:{speaking}|mouth_open={mouth_open}|arrow={arrow}:{}",
            g.native_dialogue_closing_t,
            g.dlg_t,
            g.dlg_page,
            g.native_dialogue_progress,
            g.native_dialogue_speaking_frame,
            g.native_dialogue_arrow_ticks
        );
        let err_before = g.load_err.clone();
        self.screen.clear();
        self.screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
        let drawn = g.draw_native_story_dialogue(&mut self.screen);
        if !drawn {
            return format!("tail|dialogue=false|{mouth}");
        }
        self.stat("dialogue_drawn");
        self.stat(&format!("dialogue_{stage}"));
        if speaking {
            self.stat("dialogue_speaking_mouth");
        }
        if mouth_open {
            self.stat("dialogue_wait_mouth_open");
        }
        if arrow {
            self.stat("dialogue_wait_arrow");
        }
        let mut load_err = String::new();
        if g.load_err != err_before {
            load_err = g.load_err.clone();
            self.stat("dialogue_load_err");
        }
        // Go `%q`：正式路徑的 loadErr 是空字串，兩邊都是 `""`；非空時 Rust 的 `{:?}` 與 Go 的跳脫規則可能不同，
        // 比對會失敗而不會誤判相同。
        format!("tail|dialogue=true|{state}|err={load_err:?}|rgba={}|{mouth}", sha_hex(self.screen.read_pixels()))
    }

    /// Go `dialogueLayerRecorder.observe`：在一個 `update` 之後呼叫。
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
        self.rows.push(MapLayerRow { capture: index, count: 1, update_frame: g.frame, node: node_id(g), key });
    }

    /// Go `dialogueLayerRecorder.finish`：寫 `dialogue-layers.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5_dialogue_layers/1",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_dialogue_layers_test.rs（tools/rust_r5_dialogue_layers.sh）",
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
        std::fs::write(self.out.join("dialogue-layers.json"), raw).expect("寫 dialogue-layers.json");
    }
}
