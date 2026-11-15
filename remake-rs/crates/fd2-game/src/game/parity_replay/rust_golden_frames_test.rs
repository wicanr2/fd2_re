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
//! 「R5 階段 1：畫格擷取介面」；全域亂數（Go `rand.Seed(4)`）在 R5 階段 2 r5d 接上，見 [`pin_global_rand`]。
//!
//! R5 階段 2（`r5t`）：同一個擷取器也供 opening 模式（`rust_golden_frames_opening_test.rs`）使用：
//! [`FrameCapture::new`] 指定模式，[`FrameCapture::try_observe`] 在 `Game::draw` 停在未移植的 `todo!` 時記下位置並
//! 回傳 false，`frames.json` 多一個 `stopped` 欄位；`finish` 收 Go `extra` 那幾個欄位（`phase_captures` 等）。

use super::*;
use sha2::Digest;

/// Go `goldenFrameTick = time.Second / 60`（整數除法，16666666 奈秒）。
const GOLDEN_FRAME_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`：牆鐘設在遠未來，`PlayerGameClock::sample` 的差值恆為負，只回傳 `game`。
const GOLDEN_FRAME_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

/// Go 黃金在測試開頭以 `rand.Seed(4)` 固定 `math/rand` 全域來源（嘴型動畫；`goldenFrameGlobalRandSeed`）。
pub(crate) const GOLDEN_GLOBAL_RAND_SEED: i64 = 4;

/// R5 階段 2 r5d：擷取打開時 `Game::go_global_rand` 是 `GoMathRand::new(4)`（[`pin_global_rand`]），故事對白嘴型
/// `rand.Intn(30)` 與確認框嘴型 `rand.New(rand.NewSource(rand.Int63()))` 依 Go 的取用順序共用它；寫進 `frames.json`。
pub(crate) const GLOBAL_RAND_MODEL: &str =
    "Go math/rand 全域來源：Game::go_global_rand = GoMathRand::new(4)（Go rand.Seed(4)），故事對白與確認框嘴型共用";

/// R5 階段 2 r5d：Go 黃金的 `rand.Seed(goldenFrameGlobalRandSeed)`。Go 在建立 `Game` 之前呼叫，`loadGame` 不取用全域
/// 來源，所以在 `load_game` 之後、第一個 `update` 之前設定等價。只在畫格擷取打開時呼叫；關閉時 `go_global_rand` 維持
/// `None`，嘴型用 R4 的固定種子。
pub(crate) fn pin_global_rand(g: &mut Game) {
    assert!(g.go_global_rand.is_none(), "全域亂數只能在重播開始前設定一次");
    assert!(
        g.story_mouth_rng.is_none() && g.native_confirmation_mouth_rng.is_none(),
        "設定全域亂數之前嘴型亂數已被取用"
    );
    g.go_global_rand = Some(battle::GoMathRand::new(GOLDEN_GLOBAL_RAND_SEED));
}

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

/// Go `goldenFrameRecorder` 的 chapter 與 opening 模式（不寫 PNG、不分類 draw_path）。
pub(crate) struct FrameCapture {
    mode: &'static str,
    out: std::path::PathBuf,
    screen: Picture,
    /// R5 階段 2：opening 模式以 [`FrameCapture::try_observe`] 擷取，`Game::draw` 停在未移植的 `todo!` 時記下
    /// 擷取序號與訊息，寫進 `frames.json` 的 `stopped`。
    stopped: Option<serde_json::Value>,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    seq: sha2::Sha256,
    rows: Vec<FrameRow>,
    gaps: Vec<FrameGap>,
    /// `FD2_GOLDEN_FRAMES_DUMP=<擷取序號,…>`：把這些擷取的 640×400 RGBA 原樣寫到 `<out>/frames/<序號>.rgba`，
    /// 供逐像素找出與 Go 黃金 PNG 的差異（Rust 新增的除錯輸出，不影響雜湊）。
    dump: BTreeSet<i64>,
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
        Some(FrameCapture::new("chapter", out.into()))
    }

    /// 指定模式與輸出目錄建立擷取器（`from_env` 與 opening 模式共用）。
    pub(crate) fn new(mode: &'static str, out: std::path::PathBuf) -> FrameCapture {
        FrameCapture {
            mode,
            out,
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            stopped: None,
            pinned: false,
            clock_mode: "",
            last_frame: 0,
            virtual_frames: 0,
            captures: 0,
            seq: sha2::Sha256::new(),
            rows: Vec::new(),
            gaps: Vec::new(),
            dump: std::env::var("FD2_GOLDEN_FRAMES_DUMP")
                .unwrap_or_default()
                .split(',')
                .filter_map(|v| v.trim().parse().ok())
                .collect(),
        }
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

    /// 已擷取的次數（Go `rec.capture`，opening 模式的 `phase_captures` 用）。
    pub(crate) fn captures(&self) -> i64 {
        self.captures
    }

    /// Go `goldenFrameRecorder.observe`：在一個 `update` 之後呼叫。
    pub(crate) fn observe(&mut self, g: &mut Game) {
        self.observe_with(g, |g, screen| {
            g.draw(screen);
            Ok(())
        });
    }

    /// opening 模式的 `observe`：`Game::draw` panic（停在未移植的 `todo!`）時不往外傳，把擷取序號、
    /// `update_frame` 與訊息記進 `stopped`，回傳 false 讓驅動端停下；這次擷取不算進序列。
    pub(crate) fn try_observe(&mut self, g: &mut Game) -> bool {
        self.observe_with(g, |g, screen| {
            std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| g.draw(screen))).map_err(|payload| {
                payload
                    .downcast_ref::<String>()
                    .cloned()
                    .or_else(|| payload.downcast_ref::<&str>().map(|s| s.to_string()))
                    .unwrap_or_default()
            })
        })
    }

    fn observe_with(
        &mut self,
        g: &mut Game,
        draw: impl FnOnce(&mut Game, &mut Picture) -> std::result::Result<(), String>,
    ) -> bool {
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
        if let Err(message) = draw(g, &mut self.screen) {
            eprintln!("畫格擷取停在第 {} 次（update_frame={}）：{message}", self.captures, g.frame);
            self.stopped = Some(serde_json::json!({
                "capture": self.captures,
                "update_frame": g.frame,
                "node": node_id(g),
                "title_phase": g.title_phase,
                "panic": message,
            }));
            return false;
        }
        let pix = self.screen.read_pixels();
        let sum: [u8; 32] = sha2::Sha256::digest(pix).into();
        self.seq.update(sum);
        let hash: String = sum.iter().map(|b| format!("{b:02x}")).collect();
        let index = self.captures;
        self.captures += 1;
        if self.dump.contains(&index) {
            let dir = self.out.join("frames");
            std::fs::create_dir_all(&dir).expect("建立 frames 目錄");
            std::fs::write(dir.join(format!("{index:06}.rgba")), pix).expect("寫 RGBA 傾印");
        }
        if let Some(last) = self.rows.last_mut()
            && last.rgba_sha256 == hash
        {
            last.count += 1;
            return true;
        }
        self.rows.push(FrameRow {
            capture: index,
            count: 1,
            update_frame: g.frame,
            rgba_sha256: hash,
            node: node_id(g),
        });
        true
    }

    /// Go `goldenFrameRecorder.finish`：寫 `frames.json`。
    pub(crate) fn finish(self, extra: serde_json::Map<String, serde_json::Value>) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let distinct: BTreeSet<&str> = self.rows.iter().map(|r| r.rgba_sha256.as_str()).collect();
        let gap_updates: i64 = self.gaps.iter().map(|g| g.uncaptured_updates).sum();
        let mut manifest = serde_json::json!({
            "schema": "fd2_rust_r5_golden_frames/1",
            "mode": self.mode,
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
            "stopped": self.stopped,
            "rows": self.rows,
            "gaps": self.gaps,
        });
        manifest.as_object_mut().unwrap().extend(extra);
        std::fs::create_dir_all(&self.out).expect("建立畫格輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("frames.json"), raw).expect("寫 frames.json");
    }
}
