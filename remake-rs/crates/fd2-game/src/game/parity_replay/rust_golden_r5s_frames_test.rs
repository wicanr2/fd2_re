//! Go `remake/cmd/fd2/rust_golden_r5s_frames_test.go` 的 Rust 端（R5 階段 3 `r5s`：城鎮與設施的整張畫格黃金，預設關閉）。
//!
//! 設 `FD2_GOLDEN_R5S_FRAMES_OUT=<目錄>` 時打開。擷取點與 r5c 的局部黃金（`rust_golden_r5c_layers_test.rs`）相同：
//!
//! - pump：[`Driver::pump`] 每一幀開頭（Go `ch01FrameObserver`），推進與 Go 黃金工具相同的 60 Hz 虛擬時鐘。
//! - gap：戰後 `ensure_postbattle_node` 的迴圈、其中的 `drive_story` 與 `settle_town`（[`R5sFrameCapture::gap_scope`]
//!   打開的期間）每個 `update` 之後，`town_probe`／`town_save` 移動城鎮游標之後，`town_save` 每次酒店輸入之後；
//!   經 [`Driver::observe_r5c_gap`] 轉來，不推進虛擬時鐘。
//!
//! 與 r5c 不同的是每次擷取都呼叫**整個** [`Game::draw`]（`Clear` 的 640×400 畫布），比對整張雜湊。畫之前照 `Draw`
//! 開頭清三個本幀旗標，以 [`golden_map_layers_branch`] 記分支；尾段時另把 TTF 段（[`Game::battle_text_draws`]）畫到
//! 另一張 `Clear` 的畫布，記每次繪字的鍵（同 r5f 的 `ttfLayerDrawKey`）與墨跡外框，供比對程式判斷整張畫格的差異是否
//! 只在 TTF 墨跡內（121 定案的 KNOWN_DIFFERENCES `ttf-raster-coverage`）。
//!
//! 設 `FD2_GOLDEN_R5S_FRAMES_DIFF_REF=<Go 的 r5s-frames.json>` 時，同一擷取序號雜湊與 Go 不同的畫格以雜湊為檔名
//! 寫到 `<目錄>/frames/<sha>.rgba`（[`DiffDump`]）。重播結束寫 `<目錄>/r5s-frames.json`，由
//! `tools/rust_r5_stage3_compare.py` 比對。

use super::*;
use sha2::Digest;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const R5S_FRAME_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const R5S_FRAME_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

/// Go `r5sFrameRow`。
#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct R5sFrameRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) pump_capture: i64,
    pub(crate) update_frame: i64,
    pub(crate) node: String,
    pub(crate) rgba_sha256: String,
    pub(crate) key: String,
}

/// Go `r5sFrameRecorder`。
pub(crate) struct R5sFrameCapture {
    out: std::path::PathBuf,
    screen: Picture,
    ttf: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    pumps: i64,
    seq: sha2::Sha256,
    rows: Vec<R5sFrameRow>,
    branches: BTreeMap<String, i64>,
    stats: BTreeMap<String, i64>,
    diff_dump: Option<DiffDump>,
    /// 同 [`R5cLayerCapture::gap_scope`]：戰後的 `drive_story`、`settle_town` 只在這段期間擷取。
    pub(crate) gap_scope: bool,
}

fn sha_hex(raw: &[u8]) -> String {
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

impl R5sFrameCapture {
    /// 讀 `FD2_GOLDEN_R5S_FRAMES_OUT`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<R5sFrameCapture> {
        let out = std::env::var("FD2_GOLDEN_R5S_FRAMES_OUT").unwrap_or_default();
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
            "FD2_GOLDEN_R5B_LAYERS_OUT",
            "FD2_GOLDEN_R5H_LAYERS_OUT",
        ] {
            assert!(
                std::env::var(other).unwrap_or_default().is_empty(),
                "FD2_GOLDEN_R5S_FRAMES_OUT 與 {other} 不能同時設（Go 端只有一個 ch01FrameObserver）"
            );
        }
        let out: std::path::PathBuf = out.into();
        Some(R5sFrameCapture {
            diff_dump: DiffDump::from_env("FD2_GOLDEN_R5S_FRAMES_DIFF_REF", &out),
            out,
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            ttf: Picture::new_image(LOGICAL_W, LOGICAL_H),
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
            g.player_clock.wall = R5S_FRAME_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = R5S_FRAME_FAR_FUTURE;
        let step = d * R5S_FRAME_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str) {
        *self.stats.entry(name.to_string()).or_default() += 1;
    }

    /// Go `r5sFrameRecorder.ttfKey`：尾段時把 TTF 段畫到 `Clear` 的畫布，回傳繪字鍵與墨跡外框（`ink=` 是非透明像素的
    /// 外框，右下不含；沒有墨跡時 `ink=none`）。不是尾段回 `ttf=-`。
    fn ttf_key(&mut self, g: &mut Game, branch: &str) -> String {
        if branch != "tail" {
            return "ttf=-".to_string();
        }
        self.ttf.clear();
        let listed = if g.font.is_some() { g.battle_text_draws().0 } else { Vec::new() };
        let mut draws = Vec::with_capacity(listed.len());
        for d in listed {
            let f = g.font.as_mut().unwrap();
            let px = (f.base * f.effective_scale(d.scale)).round() as i64;
            let (face, asc) = f.face_for(px);
            let mut bounds = Rectangle::default();
            if let Some(face) = face {
                let b = f.measure.0.clone().expect("face_for 成功時已注入 TextMeasure").bound_string(&face, &d.text);
                let (ox, oy) = (d.x.round() as i64, (d.y.round() + asc) as i64);
                bounds = rect(b.min.x + ox, b.min.y + oy, b.max.x + ox, b.max.y + oy);
            }
            f.draw(&mut self.ttf, &d.text, d.x, d.y, d.scale, d.clr);
            draws.push(TtfLayerDraw { draw: d, px, asc, bounds });
        }
        if draws.is_empty() {
            return "ttf=0".to_string();
        }
        self.stat("ttf_captures");
        let pix = self.ttf.read_pixels();
        let mut ink = [LOGICAL_W, LOGICAL_H, 0, 0];
        for y in 0..LOGICAL_H {
            for x in 0..LOGICAL_W {
                if pix[((y * LOGICAL_W + x) * 4 + 3) as usize] != 0 {
                    ink = [ink[0].min(x), ink[1].min(y), ink[2].max(x + 1), ink[3].max(y + 1)];
                }
            }
        }
        let ink = if ink[2] == 0 { "none".to_string() } else { format!("{},{},{},{}", ink[0], ink[1], ink[2], ink[3]) };
        let parts: Vec<String> = draws.iter().map(ttf_layer_draw_key).collect();
        format!("ttf={}|{}|ink={ink}", draws.len(), parts.join("|"))
    }

    /// Go `r5sFrameRecorder.capture`：清三個本幀旗標、記分支與 TTF 鍵，再 `Clear` 畫布呼叫整個 `Draw`。
    fn capture(&mut self, g: &mut Game) -> (String, String) {
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = golden_map_layers_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        if branch == "tail" {
            assert!(
                g.modern_story_portraits.is_none() && g.modern_hd_text.is_none(),
                "整張畫格黃金只涵蓋忠實模式（modernStoryPortraits、modernHDText 必須是 nil）"
            );
        }
        let ttf = self.ttf_key(g, branch);
        self.screen.clear();
        g.draw(&mut self.screen);
        let pix = self.screen.read_pixels();
        let sha = sha_hex(pix);
        (format!("{branch}|{ttf}|rgba={sha}"), sha)
    }

    fn record(&mut self, g: &Game, key: String, sha: String, pump: i64) {
        self.seq.update(key.as_bytes());
        self.seq.update(b"\n");
        let index = self.captures;
        self.captures += 1;
        if let Some(diff) = self.diff_dump.as_mut() {
            diff.observe(index, &sha, self.screen.read_pixels());
        }
        if let Some(last) = self.rows.last_mut()
            && last.key == key
        {
            last.count += 1;
            return;
        }
        self.rows.push(R5sFrameRow {
            capture: index,
            count: 1,
            pump_capture: pump,
            update_frame: g.frame,
            node: node_id(g),
            rgba_sha256: sha,
            key,
        });
    }

    /// Go `r5sFrameRecorder.observe`：pump 擷取點。
    pub(crate) fn observe(&mut self, g: &mut Game) {
        self.pin_clock(g);
        let (key, sha) = self.capture(g);
        let pump = self.pumps;
        self.pumps += 1;
        self.stat("point_pump");
        self.record(g, format!("pump|{key}"), sha, pump);
    }

    /// Go `r5sFrameRecorder.observeGap`：逐字複本插入的擷取點，不推進虛擬時鐘。
    pub(crate) fn observe_gap(&mut self, g: &mut Game, point: &str) {
        let (key, sha) = self.capture(g);
        self.stat(&format!("point_{point}"));
        self.record(g, format!("gap:{point}|{key}"), sha, -1);
    }

    /// Go `r5sFrameRecorder.finish`：寫 `r5s-frames.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5s_frames/1",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_r5s_frames_test.rs（tools/rust_r5_stage3_verify.sh）",
            "pixel_format": "RGBA8，預乘 alpha（與 Ebiten ReadPixels 相同）",
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
            "rows": self.rows,
        });
        std::fs::create_dir_all(&self.out).expect("建立輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("r5s-frames.json"), raw).expect("寫 r5s-frames.json");
    }
}
