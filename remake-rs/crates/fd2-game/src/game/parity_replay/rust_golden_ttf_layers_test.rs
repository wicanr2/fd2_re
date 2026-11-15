//! Go `remake/cmd/fd2/rust_golden_ttf_layers_test.go` 的 Rust 端（R5 階段 2 r5f 組的局部黃金，預設關閉）。
//!
//! - 設 `FD2_GOLDEN_TTF_LAYERS_OUT=<目錄>` 時，章重播的 [`Driver::pump`] 每一幀在上一個 `update` 之後、`done` 與
//!   `ack_presents` 之前呼叫 [`TtfLayerCapture::observe`]（與 Go `ch01FrameObserver` 同一位置）：推進與 Go 黃金工具
//!   相同的 60 Hz 虛擬時鐘；照 Go `Draw` 的分支順序走到尾段時，以 [`Game::battle_text_draws`]（`draw_field` 用的同一份
//!   程式）列出 TTF 戰場短訊與勝負字樣，逐一 `Font::draw` 到 `Clear` 的 640×400 畫布。比對鍵是文字、座標、倍率、
//!   像素尺寸、ascent、顏色與 `BoundString` 外框，與 Go 逐字相同才算通過；畫素統計與裁切 RGBA 另存 `renders`，
//!   由 `tools/rust_r5f_ttf_layers_compare.py` 量化差異（KNOWN_DIFFERENCES `ttf-raster-coverage`）。
//! - [`test_rust_golden_ttf_metrics`]（`#[ignore]`）讀 Go `TestRustGoldenTTFMetrics` 的輸出當輸入清單，以 Rust 的
//!   量測重算同樣的欄位，寫 `FD2_GOLDEN_TTF_METRICS_OUT/ttf-metrics.json`。

use super::*;
use crate::go_x_image_font::{XImageFace, XImageFont};
use crate::test_support::{NOTO_SANS_CJK_REGULAR_PATH, cjk_font_available};
use base64::Engine;
use sha2::Digest;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const TTF_LAYER_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const TTF_LAYER_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

/// Go `ttfLayerRender`。
#[derive(Debug, Clone, serde::Serialize)]
struct TtfLayerRender {
    captures: i64,
    nonzero: i64,
    opaque: i64,
    alpha_sum: i64,
    ink_box: [i64; 4],
    outside_bounds: i64,
    rgba_sha256: String,
    crop: [i64; 4],
    crop_rgba_base64: String,
    conflicts: i64,
}

/// Go `ttfLayerDraw`（R5 階段 3 `r5s` 的整張畫格黃金也用它組 TTF 鍵）。
pub(crate) struct TtfLayerDraw {
    pub(crate) draw: BattleTextDraw,
    pub(crate) px: i64,
    pub(crate) asc: f64,
    pub(crate) bounds: Rectangle,
}

/// Go `ttfLayerRecorder`（rows 沿用 [`MapLayerRow`]）。
pub(crate) struct TtfLayerCapture {
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
    renders: BTreeMap<String, TtfLayerRender>,
}

fn sha_hex(raw: &[u8]) -> String {
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go `strconv.QuoteToASCII`（`%+q`）：可列印 ASCII 原樣，`"`、`\` 與控制字元跳脫，非 ASCII 寫成 `\uXXXX`／`\UXXXXXXXX`。
fn go_quote_to_ascii(s: &str) -> String {
    let mut out = String::from("\"");
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\x07' => out.push_str("\\a"),
            '\x08' => out.push_str("\\b"),
            '\x0c' => out.push_str("\\f"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\x0b' => out.push_str("\\v"),
            c if (c as u32) < 0x20 || c as u32 == 0x7f => out.push_str(&format!("\\x{:02x}", c as u32)),
            c if c.is_ascii() => out.push(c),
            c if (c as u32) < 0x10000 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push_str(&format!("\\U{:08x}", c as u32)),
        }
    }
    out.push('"');
    out
}

/// Go `ttfLayerDrawKey`。
pub(crate) fn ttf_layer_draw_key(d: &TtfLayerDraw) -> String {
    let c = d.draw.clr;
    format!(
        "text={};x={:.3};y={:.3};scale={:.3};px={};asc={};clr={:02x}{:02x}{:02x}{:02x};bound={},{},{},{}",
        go_quote_to_ascii(&d.draw.text),
        d.draw.x,
        d.draw.y,
        d.draw.scale,
        d.px,
        d.asc as i64,
        c.r,
        c.g,
        c.b,
        c.a,
        d.bounds.min.x,
        d.bounds.min.y,
        d.bounds.max.x,
        d.bounds.max.y
    )
}

/// Go `image.Rectangle.Union`（空矩形不參與）。
fn rect_union(r: Rectangle, s: Rectangle) -> Rectangle {
    if rect_empty(&r) {
        return s;
    }
    if rect_empty(&s) {
        return r;
    }
    rect(r.min.x.min(s.min.x), r.min.y.min(s.min.y), r.max.x.max(s.max.x), r.max.y.max(s.max.y))
}

impl TtfLayerCapture {
    /// 讀 `FD2_GOLDEN_TTF_LAYERS_OUT`；沒設回 `None`（預設關閉）。打開時字型必須是 Go 黃金同一檔。
    pub(crate) fn from_env() -> Option<TtfLayerCapture> {
        let out = std::env::var("FD2_GOLDEN_TTF_LAYERS_OUT").unwrap_or_default();
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
                "FD2_GOLDEN_TTF_LAYERS_OUT 與 {other} 不能同時設（Go 端只有一個 ch01FrameObserver）"
            );
        }
        assert!(cjk_font_available(), "FD2_GOLDEN_TTF_LAYERS_OUT 需要 {NOTO_SANS_CJK_REGULAR_PATH}");
        Some(TtfLayerCapture {
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
            renders: BTreeMap::new(),
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
            g.player_clock.wall = TTF_LAYER_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = TTF_LAYER_FAR_FUTURE;
        let step = d * TTF_LAYER_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str) {
        *self.stats.entry(name.to_string()).or_default() += 1;
    }

    /// Go `ttfLayerRecorder.render`。
    fn render(&mut self, key: &str, draws: &[TtfLayerDraw]) {
        let union = draws.iter().fold(Rectangle::default(), |u, d| rect_union(u, d.bounds));
        let grown = rect(union.min.x - 2, union.min.y - 2, union.max.x + 2, union.max.y + 2);
        let crop = rect_intersect(grown, rect(0, 0, LOGICAL_W, LOGICAL_H));
        let pix = self.screen.read_pixels();
        let mut rd = TtfLayerRender {
            captures: 1,
            nonzero: 0,
            opaque: 0,
            alpha_sum: 0,
            ink_box: [LOGICAL_W, LOGICAL_H, 0, 0],
            outside_bounds: 0,
            rgba_sha256: sha_hex(pix),
            crop: [crop.min.x, crop.min.y, crop.max.x, crop.max.y],
            crop_rgba_base64: String::new(),
            conflicts: 0,
        };
        for y in 0..LOGICAL_H {
            for x in 0..LOGICAL_W {
                let a = i64::from(pix[((y * LOGICAL_W + x) * 4 + 3) as usize]);
                if a == 0 {
                    continue;
                }
                rd.nonzero += 1;
                rd.alpha_sum += a;
                if a == 0xff {
                    rd.opaque += 1;
                }
                let b = &mut rd.ink_box;
                *b = [b[0].min(x), b[1].min(y), b[2].max(x + 1), b[3].max(y + 1)];
                let inside = draws
                    .iter()
                    .any(|d| x >= d.bounds.min.x && x < d.bounds.max.x && y >= d.bounds.min.y && y < d.bounds.max.y);
                if !inside {
                    rd.outside_bounds += 1;
                }
            }
        }
        if let Some(prev) = self.renders.get_mut(key) {
            prev.captures += 1;
            if prev.rgba_sha256 != rd.rgba_sha256 {
                prev.conflicts += 1;
            }
            return;
        }
        let mut cropped = Vec::with_capacity((rect_dx(&crop) * rect_dy(&crop) * 4).max(0) as usize);
        for y in crop.min.y..crop.max.y {
            cropped.extend_from_slice(
                &pix[((y * LOGICAL_W + crop.min.x) * 4) as usize..((y * LOGICAL_W + crop.max.x) * 4) as usize],
            );
        }
        rd.crop_rgba_base64 = base64::engine::general_purpose::STANDARD.encode(cropped);
        self.renders.insert(key.to_string(), rd);
    }

    /// Go `ttfLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = golden_map_layers_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        if branch != "tail" {
            return branch.to_string();
        }
        assert!(g.modern_story_portraits.is_none(), "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）");
        self.screen.clear();
        let font = g.font.is_some();
        let (listed, _) = if font { g.battle_text_draws() } else { (Vec::new(), true) };
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
            f.draw(&mut self.screen, &d.text, d.x, d.y, d.scale, d.clr);
            draws.push(TtfLayerDraw { draw: d, px, asc, bounds });
        }
        if draws.is_empty() {
            return "tail|ttf=0".to_string();
        }
        let parts: Vec<String> = draws.iter().map(ttf_layer_draw_key).collect();
        let key = format!("tail|ttf={}|{}", draws.len(), parts.join("|"));
        self.stat("ttf_captures");
        self.stat(&format!("ttf_draws_{}", draws.len()));
        if g.should_draw_transient_battle_message() {
            self.stat("ttf_msg");
        }
        if !g.result.is_empty() {
            self.stat("ttf_result");
        }
        self.render(&key, &draws);
        key
    }

    /// Go `ttfLayerRecorder.observe`：在一個 `update` 之後呼叫。
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

    /// Go `ttfLayerRecorder.finish`：寫 `ttf-layers.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5_ttf_layers/1",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_ttf_layers_test.rs（tools/rust_r5f_ttf_layers.sh）",
            "font": NOTO_SANS_CJK_REGULAR_PATH,
            "rasterizer": "swash 0.2.10 重新匯出的 zeno 0.3.3（Mask，非零規則）；度量照 x/image v0.12.0（go_x_image_font.rs）",
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
            "renders": self.renders,
        });
        std::fs::create_dir_all(&self.out).expect("建立輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("ttf-layers.json"), raw).expect("寫 ttf-layers.json");
    }
}

/// 章重播與 opening 模式的驅動端在 `load_game` 之後呼叫：Go 黃金（`fd2-go-test-local`）的 `loadFont` 讀到系統的
/// `NotoSansCJK-Regular.ttc`，Rust 端必須是同一檔（SHA-256 相符），而且 `load_font` 真的用了它（`fontPaths` 前兩個
/// 相對路徑都不存在）。缺檔時照 `FD2_REQUIRE_ORIGINAL` 略過或失敗。
pub(crate) fn require_golden_font(g: &Game) {
    if !cjk_font_available() {
        return;
    }
    assert!(
        FONT_PATHS[..2].iter().all(|p| !file_exists(&asset_path(p))),
        "fontPaths 前兩個相對路徑有檔案，load_font 不會讀系統的 {NOTO_SANS_CJK_REGULAR_PATH}"
    );
    assert!(g.font.is_some() && g.font_nm.is_some(), "{NOTO_SANS_CJK_REGULAR_PATH} 在，load_font 卻失敗");
}

/// Go `TestRustGoldenTTFMetrics` 的 Rust 端：`FD2_GOLDEN_TTF_METRICS_IN` 是 Go 的 `ttf-metrics.json`（只取輸入清單：
/// 尺寸、字元、字串），輸出同欄位到 `FD2_GOLDEN_TTF_METRICS_OUT/ttf-metrics.json`。kern 的字對照 Go 重建：
/// 每個字串的相鄰字元加上 ASCII 列印字元兩兩成對。
#[test]
#[ignore = "R5 r5f 度量黃金，只由 tools/rust_r5f_ttf_layers.sh 執行"]
fn test_rust_golden_ttf_metrics() {
    let input = std::env::var("FD2_GOLDEN_TTF_METRICS_IN").expect("需要 FD2_GOLDEN_TTF_METRICS_IN");
    let out = std::env::var("FD2_GOLDEN_TTF_METRICS_OUT").expect("需要 FD2_GOLDEN_TTF_METRICS_OUT");
    assert!(cjk_font_available(), "需要 {NOTO_SANS_CJK_REGULAR_PATH}");
    let go: serde_json::Value = serde_json::from_slice(&std::fs::read(&input).expect("讀 Go 輸出")).unwrap();
    let ints =
        |v: &serde_json::Value| -> Vec<i64> { v.as_array().unwrap().iter().map(|x| x.as_i64().unwrap()).collect() };
    let sizes = ints(&go["sizes"]);
    let runes: Vec<char> = ints(&go["runes"]).into_iter().map(|r| char::from_u32(r as u32).expect("Go 字元")).collect();
    let strs: Vec<String> = go["strings"].as_array().unwrap().iter().map(|s| s.as_str().unwrap().to_string()).collect();
    let bound_sizes = ints(&go["bound_sizes"]);
    let kern_px = go["kern_px"].as_i64().unwrap();

    let mut f = load_font().expect("load_font");
    let xf = XImageFont::parse(Arc::new(std::fs::read(NOTO_SANS_CJK_REGULAR_PATH).unwrap())).expect("解析 TTC");
    let mut metrics = Vec::new();
    for &px in &sizes {
        let (_, asc) = f.face_for(px);
        let m = XImageFace::new(&xf, px as f64, 72.0).metrics();
        metrics.push(serde_json::json!({
            "px": px, "height": m.height, "ascent": m.ascent, "descent": m.descent, "ascent_round": asc as i64,
        }));
    }
    let mut glyphs = serde_json::Map::new();
    for &r in &runes {
        let mut row = vec![i64::from(xf.glyph_index(r))];
        for &px in &sizes {
            let (b, a, ok) = XImageFace::new(&xf, px as f64, 72.0).glyph_bounds(r);
            row.extend([b.min_x, b.min_y, b.max_x, b.max_y, a].map(i64::from));
            row.push(i64::from(ok));
        }
        glyphs.insert((r as u32).to_string(), row.into());
    }
    let mut pairs = BTreeSet::new();
    for s in &strs {
        let cs: Vec<char> = s.chars().collect();
        for w in cs.windows(2) {
            pairs.insert((w[0], w[1]));
        }
    }
    for a in 0x20u8..0x7f {
        for b in 0x20u8..0x7f {
            pairs.insert((a as char, b as char));
        }
    }
    let kface = XImageFace::new(&xf, kern_px as f64, 72.0);
    let mut kerns = serde_json::Map::new();
    for &(a, b) in &pairs {
        let k = kface.kern(a, b);
        if k != 0 {
            kerns.insert(format!("{},{}", a as u32, b as u32), k.into());
        }
    }
    let mut bounds = Vec::with_capacity(strs.len());
    for s in &strs {
        let mut row = Vec::new();
        for &px in &bound_sizes {
            let (face, _) = f.face_for(px);
            let b = f.measure.0.clone().unwrap().bound_string(&face.unwrap(), s);
            row.extend([b.min.x, b.min.y, b.max.x, b.max.y]);
        }
        bounds.push(row);
    }
    let manifest = serde_json::json!({
        "schema": "fd2_rust_r5_ttf_metrics/1",
        "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_ttf_layers_test.rs test_rust_golden_ttf_metrics",
        "font": NOTO_SANS_CJK_REGULAR_PATH,
        "fractional_points": xf.fractional_points(),
        "sizes": sizes,
        "metrics": metrics,
        "runes": runes.iter().map(|&r| r as u32).collect::<Vec<_>>(),
        "glyphs": glyphs,
        "kern_px": kern_px,
        "kern_pairs": pairs.len(),
        "kern_nonzero": kerns,
        "bound_sizes": bound_sizes,
        "strings": strs,
        "bounds": bounds,
    });
    std::fs::create_dir_all(&out).expect("建立輸出目錄");
    let mut raw = serde_json::to_vec(&manifest).unwrap();
    raw.push(b'\n');
    std::fs::write(std::path::Path::new(&out).join("ttf-metrics.json"), raw).expect("寫 ttf-metrics.json");
}
