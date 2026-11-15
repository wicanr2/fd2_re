//! Go `remake/cmd/fd2/rust_golden_frames_test.go` 的 chapter 模式擷取點（R5 階段 1：只做介面，預設關閉）。
//!
//! 設 `FD2_GOLDEN_FRAMES_MODE=chapter` 與 `FD2_GOLDEN_FRAMES_OUT=<目錄>` 時，章重播的 [`Driver::pump`] 每一幀在
//! 上一個 `update` 之後、`done` 與 `ack_presents` 之前呼叫 [`FrameCapture::observe`]（Go `ch01FrameObserver`，
//! `native_chapter_one_play_test.go:305`）：推進 Go 黃金工具的虛擬時鐘、清空 640×400 畫布、`Game::draw`、
//! 記下讀回的 SHA-256。重播結束寫 `<目錄>/frames.json`，欄位與 Go 的 `frames.json` 同名，序列雜湊算法相同
//! （每次擷取的 SHA-256 串接後再取 SHA-256），可以直接與 Go 黃金比。
//!
//! 沒設這兩個變數時不建立擷取器，重播與 R4 完全相同。R5 階段 1 的繪圖函式還是 `todo!`，所以打開時第一個
//! 擷取點就會停在對應的 `todo!`；比對是階段 3 的工作。時鐘、全域亂數與 gaps 的規格見 `PORTING.md`
//! 「R5 階段 1：畫格擷取介面」；全域亂數（Go `rand.Seed(4)`）還沒有接上，見 [`GLOBAL_RAND_MODEL`]。

use super::*;
use sha2::Digest;

/// Go `goldenFrameTick = time.Second / 60`（整數除法，16666666 奈秒）。
const GOLDEN_FRAME_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`：牆鐘設在遠未來，`PlayerGameClock::sample` 的差值恆為負，只回傳 `game`。
const GOLDEN_FRAME_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

/// Go 黃金在測試開頭以 `rand.Seed(4)` 固定 `math/rand` 全域來源（嘴型動畫）。Rust 的嘴型目前是 R4 的兩個
/// 固定種子來源（`story_mouth_rng`、`native_confirmation_mouth_rng`），還沒有改成共用 Go 全域來源；寫進
/// `frames.json`，階段 3 比對前必須先接上（規格見 `PORTING.md`）。
pub(crate) const GLOBAL_RAND_MODEL: &str = "未接上：嘴型仍用 R4 的固定種子（Go 黃金是 rand.Seed(4) 的全域來源）";

#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct FrameRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) update_frame: i64,
    pub(crate) rgba_sha256: String,
    pub(crate) node: String,
}

#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct FrameGap {
    pub(crate) after_capture: i64,
    pub(crate) uncaptured_updates: i64,
    pub(crate) from_update_frame: i64,
    pub(crate) to_update_frame: i64,
    pub(crate) node: String,
}

/// Go `goldenFrameRecorder` 的 chapter 模式（不寫 PNG、不分類 draw_path）。
pub(crate) struct FrameCapture {
    out: std::path::PathBuf,
    screen: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    seq: sha2::Sha256,
    rows: Vec<FrameRow>,
    gaps: Vec<FrameGap>,
}

impl FrameCapture {
    /// 讀 `FD2_GOLDEN_FRAMES_MODE`／`FD2_GOLDEN_FRAMES_OUT`；兩者都沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<FrameCapture> {
        let out = std::env::var("FD2_GOLDEN_FRAMES_OUT").unwrap_or_default();
        let mode = std::env::var("FD2_GOLDEN_FRAMES_MODE").unwrap_or_default();
        if out.is_empty() && mode.is_empty() {
            return None;
        }
        assert!(
            mode == "chapter" && !out.is_empty(),
            "FD2_GOLDEN_FRAMES_MODE={mode:?}、FD2_GOLDEN_FRAMES_OUT={out:?}：章重播只支援 chapter 模式且要指定輸出目錄"
        );
        Some(FrameCapture {
            out: out.into(),
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            pinned: false,
            clock_mode: "",
            last_frame: 0,
            virtual_frames: 0,
            captures: 0,
            seq: sha2::Sha256::new(),
            rows: Vec::new(),
            gaps: Vec::new(),
        })
    }

    /// Go `goldenFrameRecorder.pinClock`：第一次擷取時把牆鐘設在遠未來（之後 `update` 讀到的遊戲時間凍結），
    /// 之後每次擷取依 `g.frame` 的增量把 `player_clock.game` 推進 `d × 16666666 × speed` 奈秒。
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
            g.player_clock.wall = GOLDEN_FRAME_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = GOLDEN_FRAME_FAR_FUTURE;
        let step = d * GOLDEN_FRAME_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    /// Go `goldenFrameRecorder.observe`：在一個 `update` 之後呼叫。
    pub(crate) fn observe(&mut self, g: &mut Game) {
        let prev = self.last_frame;
        self.pin_clock(g);
        if self.captures > 0 && g.frame - prev > 1 {
            self.gaps.push(FrameGap {
                after_capture: self.captures - 1,
                uncaptured_updates: g.frame - prev - 1,
                from_update_frame: prev,
                to_update_frame: g.frame,
                node: node_id(g),
            });
        }
        self.screen.clear();
        g.draw(&mut self.screen);
        let sum: [u8; 32] = sha2::Sha256::digest(self.screen.read_pixels()).into();
        self.seq.update(sum);
        let hash: String = sum.iter().map(|b| format!("{b:02x}")).collect();
        let index = self.captures;
        self.captures += 1;
        if let Some(last) = self.rows.last_mut()
            && last.rgba_sha256 == hash
        {
            last.count += 1;
            return;
        }
        self.rows.push(FrameRow {
            capture: index,
            count: 1,
            update_frame: g.frame,
            rgba_sha256: hash,
            node: node_id(g),
        });
    }

    /// Go `goldenFrameRecorder.finish`：寫 `frames.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let distinct: BTreeSet<&str> = self.rows.iter().map(|r| r.rgba_sha256.as_str()).collect();
        let gap_updates: i64 = self.gaps.iter().map(|g| g.uncaptured_updates).sum();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5_golden_frames/1",
            "mode": "chapter",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_frames_test.rs",
            "logical_size": [LOGICAL_W, LOGICAL_H],
            "pixel_format": "RGBA8，預乘 alpha（與 Ebiten ReadPixels 相同）",
            "clock": self.clock_mode,
            "global_rand": GLOBAL_RAND_MODEL,
            "virtual_frames": self.virtual_frames,
            "captures": self.captures,
            "changed_frames": self.rows.len(),
            "distinct_frames": distinct.len(),
            "sequence_sha256": seq,
            "uncaptured_gaps": self.gaps.len(),
            "uncaptured_update": gap_updates,
            "rows": self.rows,
            "gaps": self.gaps,
        });
        std::fs::create_dir_all(&self.out).expect("建立畫格輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("frames.json"), raw).expect("寫 frames.json");
    }
}
