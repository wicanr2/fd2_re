//! Go `remake/cmd/fd2/rust_golden_r5c_layers_test.go` 的 Rust 端（R5 階段 2 r5c 組「城鎮與設施」的局部黃金，預設關閉）。
//!
//! 設 `FD2_GOLDEN_R5C_LAYERS_OUT=<目錄>` 時打開，兩種擷取點與 Go 相同：
//!
//! - pump：[`Driver::pump`] 每一幀開頭（Go `ch01FrameObserver`），推進與 Go 黃金工具相同的 60 Hz 虛擬時鐘。
//! - gap：Go 端以逐字複本插入的位置，Rust 在同一處呼叫 [`R5cLayerCapture::observe_gap`]，不推進虛擬時鐘：戰後
//!   `ensure_postbattle_node` 的迴圈、其中的 `drive_story` 與 `settle_town`（[`R5cLayerCapture::gap_scope`] 打開的
//!   期間）每個 `update` 之後，`town_probe`／`town_save` 移動城鎮游標之後，`town_save` 每次酒店輸入之後。
//!
//! 每次擷取先照 `Draw` 開頭清三個本幀旗標，以 [`golden_map_layers_branch`] 決定分支；走到尾段時在 `Fill` 黑的
//! 640×400 畫布上依 `Draw` 的順序呼叫 [`Game::draw_campaign_ui`]，`transient_ui` 時再呼叫
//! [`Game::draw_native_class_ui_job`]，讀回取雜湊，組成與 Go 逐字相同的比對鍵。重播結束寫
//! `<目錄>/r5c-layers.json`，由 `tools/rust_r5c_layers_compare.py` 逐擷取點比對。
//!
//! Go 黃金容器裝了 Noto CJK，`g.font` 不是 nil；Rust 測試映像沒有字型時 `load_font` 回 `None`，`drawCampaignUI`
//! 開頭的 `g.font == nil` 守門會讓本組整段不畫。擷取期間沒有字型時暫時代入空的 [`Font`]（`sf` 為 `None`）只為通過
//! 這個守門，畫完立刻換回；本組在第一章走到的分支不呼叫 `Font::draw`，用到時會停在它的 `todo!`。次數記在
//! `rust_font_stub`（只有 Rust 端有）。

use super::*;
use sha2::Digest;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const R5C_LAYER_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const R5C_LAYER_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

/// Go `r5cLayerRow`。
#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct R5cLayerRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) pump_capture: i64,
    pub(crate) update_frame: i64,
    pub(crate) node: String,
    pub(crate) key: String,
}

/// Go `r5cLayerRecorder`。
pub(crate) struct R5cLayerCapture {
    out: std::path::PathBuf,
    screen: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    pumps: i64,
    seq: sha2::Sha256,
    rows: Vec<R5cLayerRow>,
    branches: BTreeMap<String, i64>,
    stats: BTreeMap<String, i64>,
    font_stub: i64,
    /// Go 的戰後逐字複本（`ensurePostbattleNode` 及其中的 `driveStory`、`settleTown`）執行期間為 true；
    /// 只有這段期間 `drive_story`／`settle_town` 的 `update` 之後擷取。
    pub(crate) gap_scope: bool,
}

fn sha_hex(raw: &[u8]) -> String {
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go `r5cJobKey`。
fn r5c_job_key(frame: i64, frames: usize, restore: usize, timeline: usize, drawn: bool) -> String {
    format!("{frame}/{frames}/{restore}/{timeline}/{drawn}")
}

/// Go `r5cLayerRecorder.state`：畫之前的設施狀態（只讀欄位），回傳（節點型別、狀態字串）。
fn r5c_state(g: &Game) -> (String, String) {
    let node_type = g.camp.as_ref().and_then(|c| c.node()).map(|n| n.r#type.clone()).unwrap_or_default();
    let class = g
        .native_class_ui_job
        .as_ref()
        .map_or("-".to_string(), |j| r5c_job_key(j.frame, j.frames.len(), j.restore.len(), j.timeline.len(), j.drawn));
    let shop = g
        .native_shop_ui_job
        .as_ref()
        .map_or("-".to_string(), |j| r5c_job_key(j.frame, j.frames.len(), j.restore.len(), j.timeline.len(), j.drawn));
    let church = g
        .native_church_ui_job
        .as_ref()
        .map_or("-".to_string(), |j| r5c_job_key(j.frame, j.frames.len(), j.restore.len(), 0, j.drawn));
    let state = format!(
        "node={node_type}|shop={}|hotel={}|church={}|prep={}:{}:{}:{}|sel={}:{}:{}|pulse={}:{}:{}|jobs={class},{shop},{church}",
        g.native_shop_mode,
        g.native_hotel_mode,
        g.church_mode,
        g.prep_selecting,
        g.prep_confirm,
        g.prep_record_slots,
        g.prep_confirm_sel,
        g.camp_sel,
        g.hotel_sel,
        g.native_hotel_slot_sel,
        g.native_town_ui_pulse,
        g.native_shop_ui_pulse,
        g.native_class_ui_pulse,
    );
    (node_type, state)
}

impl R5cLayerCapture {
    /// 讀 `FD2_GOLDEN_R5C_LAYERS_OUT`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<R5cLayerCapture> {
        let out = std::env::var("FD2_GOLDEN_R5C_LAYERS_OUT").unwrap_or_default();
        if out.is_empty() {
            return None;
        }
        for other in [
            "FD2_GOLDEN_FRAMES_OUT",
            "FD2_GOLDEN_MAP_LAYERS_OUT",
            "FD2_GOLDEN_DIALOGUE_LAYERS_OUT",
            "FD2_GOLDEN_R5P_LAYERS_OUT",
        ] {
            assert!(
                std::env::var(other).unwrap_or_default().is_empty(),
                "FD2_GOLDEN_R5C_LAYERS_OUT 與 {other} 不能同時設（Go 端只有一個 ch01FrameObserver）"
            );
        }
        Some(R5cLayerCapture {
            out: out.into(),
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            pinned: false,
            clock_mode: "",
            last_frame: 0,
            virtual_frames: 0,
            captures: 0,
            pumps: 0,
            seq: sha2::Sha256::new(),
            rows: Vec::new(),
            branches: BTreeMap::new(),
            stats: BTreeMap::new(),
            font_stub: 0,
            gap_scope: false,
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
            g.player_clock.wall = R5C_LAYER_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = R5C_LAYER_FAR_FUTURE;
        let step = d * R5C_LAYER_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str) {
        *self.stats.entry(name.to_string()).or_default() += 1;
    }

    /// Go `r5cLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = golden_map_layers_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        if branch != "tail" {
            return branch.to_string();
        }
        assert!(
            g.modern_story_portraits.is_none() && g.modern_hd_text.is_none(),
            "局部黃金只涵蓋忠實模式（modernStoryPortraits、modernHDText 必須是 nil）"
        );
        let (node_type, state) = r5c_state(g);
        let err_before = g.load_err.clone();
        self.screen.clear();
        self.screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
        let stub = g.font.is_none();
        if stub {
            g.font = Some(Font::default());
            self.font_stub += 1;
        }
        g.draw_campaign_ui(&mut self.screen);
        let mut job = false;
        if g.transient_ui {
            job = g.draw_native_class_ui_job(&mut self.screen);
        }
        if stub {
            g.font = None;
        }
        let pix = self.screen.read_pixels();
        let drawn = pix.as_chunks::<4>().0.iter().any(|p| *p != [0, 0, 0, 0xff]);
        if !drawn {
            return format!("tail|{state}|transient={}|job={job}|drawn=false", g.transient_ui);
        }
        let sha = sha_hex(pix);
        self.stat("drawn");
        self.stat(&format!("drawn_node_{node_type}"));
        if job {
            self.stat("drawn_transient_class_job");
        }
        let mut load_err = String::new();
        if g.load_err != err_before {
            load_err = g.load_err.clone();
            self.stat("load_err");
        }
        // Go `%q`：正式路徑的 loadErr 是空字串，兩邊都是 `""`；非空時跳脫規則可能不同，比對會失敗而不會誤判相同。
        format!("tail|{state}|transient={}|job={job}|err={load_err:?}|rgba={sha}", g.transient_ui)
    }

    fn record(&mut self, g: &Game, key: String, pump: i64) {
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
        self.rows.push(R5cLayerRow {
            capture: index,
            count: 1,
            pump_capture: pump,
            update_frame: g.frame,
            node: node_id(g),
            key,
        });
    }

    /// Go `r5cLayerRecorder.observe`：pump 擷取點。
    pub(crate) fn observe(&mut self, g: &mut Game) {
        self.pin_clock(g);
        let key = self.layers(g);
        let pump = self.pumps;
        self.pumps += 1;
        self.stat("point_pump");
        self.record(g, format!("pump|{key}"), pump);
    }

    /// Go `r5cLayerRecorder.observeGap`：逐字複本插入的擷取點，不推進虛擬時鐘。
    pub(crate) fn observe_gap(&mut self, g: &mut Game, point: &str) {
        let key = self.layers(g);
        self.stat(&format!("point_{point}"));
        self.record(g, format!("gap:{point}|{key}"), -1);
    }

    /// Go `r5cLayerRecorder.finish`：寫 `r5c-layers.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5c_layers/1",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_r5c_layers_test.rs（tools/rust_r5c_layers.sh）",
            "clock": self.clock_mode,
            "global_rand": GLOBAL_RAND_MODEL,
            "virtual_frames": self.virtual_frames,
            "captures": self.captures,
            "pump_captures": self.pumps,
            "changed_rows": self.rows.len(),
            "key_sequence": "每次擷取一行比對鍵（rows[].key），串接後的 SHA-256",
            "sequence_sha256": seq,
            "branches": self.branches,
            "stats": self.stats,
            "rust_font_stub": self.font_stub,
            "rows": self.rows,
        });
        std::fs::create_dir_all(&self.out).expect("建立輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("r5c-layers.json"), raw).expect("寫 r5c-layers.json");
    }
}

impl Driver {
    /// R5 階段 2 r5c：gap 擷取點（Go 逐字複本裡的 `rec.observeGap(g, point)`）；沒打開時不做事。`scoped` 為 true 的
    /// 擷取點只在 [`R5cLayerCapture::gap_scope`] 打開時擷取（戰後的 `drive_story`、`settle_town`）。
    ///
    /// R5 階段 3 r5s：同一批擷取點也轉給整張畫格黃金 [`R5sFrameCapture`]（Go `rust_golden_r5s_frames_test.go` 的逐字
    /// 複本在同樣的位置插入 `// r5s-insert`）。兩者不會同時打開。
    pub(crate) fn observe_r5c_gap(&mut self, point: &str, scoped: bool) {
        if let Some(capture) = self.r5c_layer_capture.as_mut()
            && (!scoped || capture.gap_scope)
        {
            capture.observe_gap(&mut self.g, point);
        }
        if let Some(capture) = self.r5s_frame_capture.as_mut()
            && (!scoped || capture.gap_scope)
        {
            capture.observe_gap(&mut self.g, point);
        }
    }
}
