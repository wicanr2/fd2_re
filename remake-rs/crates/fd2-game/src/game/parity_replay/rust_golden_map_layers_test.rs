//! Go `remake/cmd/fd2/rust_golden_map_layers_test.go` 的 Rust 端（R5 階段 2 r5m 組的局部黃金，預設關閉）。
//!
//! 設 `FD2_GOLDEN_MAP_LAYERS_OUT=<目錄>` 時，章重播的 [`Driver::pump`] 每一幀在上一個 `update` 之後、`done` 與
//! `ack_presents` 之前呼叫 [`MapLayerCapture::observe`]（與 Go `ch01FrameObserver` 同一位置）：推進與 Go 黃金工具
//! 相同的 60 Hz 虛擬時鐘，照 Go `Draw` 的分支順序只呼叫地圖與單位組的函式，各自畫到空白畫布、讀回取雜湊，
//! 組成與 Go 逐字相同的比對鍵。重播結束寫 `<目錄>/map-layers.json`，`tools/rust_r5_map_layers_compare.py`
//! 逐擷取點比對兩邊。
//!
//! 三種圖層（做法見 Go 檔開頭）：spawn（`draw_native_spawn_intro`）、world（正規化世界層：`tile_at`、
//! `map_sprite_group`、`draw_unit_sprite`，照 Go `main.go:9817..9909` 的迴圈；Draw 本體的移動範圍格、storyZoom
//! 放大、游標與 HUD 不畫）、map（`native_map_frame_admission` 通過時 `draw_native_map_frame`，另以
//! `native_map_frame_image` 再取一次索引畫格與調色盤）。

use super::*;
use sha2::Digest;

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const MAP_LAYER_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const MAP_LAYER_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct MapLayerRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) update_frame: i64,
    pub(crate) node: String,
    pub(crate) key: String,
}

/// Go `mapLayerRecorder`。
pub(crate) struct MapLayerCapture {
    out: std::path::PathBuf,
    screen: Picture,
    view: Picture,
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

/// Go `mapLayerPaletteSHA`：[`ColorRgba`] 已是 `color.RGBA()` 的高 8 位元。
fn palette_sha(p: &[ColorRgba]) -> String {
    let raw: Vec<u8> = p.iter().flat_map(|c| [c.r, c.g, c.b, c.a]).collect();
    sha_hex(&raw)
}

/// Go `goldenMapLayersBranch`：`Draw`（`main.go:9395..9797`）的 early-return 順序，只讀欄位。
pub(crate) fn golden_map_layers_branch(g: &Game) -> &'static str {
    let early = g.startup_blocked
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
        || g.native_palette_pulse.is_some();
    if early {
        return "early";
    }
    if g.spawn_intro_transition.is_some() {
        return "spawn";
    }
    if g.native_turn_staging.as_ref().is_some_and(|j| j.indexed)
        || g.native_field_event61.is_some()
        || g.native_end_turn_recovery.is_some()
        || g.native_ai_idle_recovery.is_some()
        || g.atk.is_some()
    {
        return "early";
    }
    "tail"
}

/// Go `mapLayerRecorder.drawWorld`：照 `main.go:9817..9909` 畫圖塊、單位與 storyActors，回傳（單位數、已行動、
/// 缺 sprite）。
fn draw_world(g: &Game, target: &mut Picture, view_w: i64, view_h: i64) -> (i64, i64, i64) {
    let m = g.m.as_ref().expect("tail 分支有地圖");
    let (tw, th) = (m.tile_w, m.tile_h);
    let x0 = (g.cam_x as i64) / tw;
    let y0 = (g.cam_y as i64) / th;
    let x1 = (g.cam_x as i64 + view_w) / tw + 1;
    let y1 = (g.cam_y as i64 + view_h) / th + 1;
    let mut cy = y0;
    while cy <= y1 && cy < m.h {
        let mut cx = x0;
        while cx <= x1 && cx < m.w {
            if cy >= 0
                && cx >= 0
                && let Some(t) = g.tile_at(m.tiles[(cy * m.w + cx) as usize])
            {
                let mut op = DrawImageOptions::default();
                op.geo_m.translate((cx * tw) as f64 - g.cam_x, (cy * th) as f64 - g.cam_y);
                target.draw_image(&t, &op);
            }
            cx += 1;
        }
        cy += 1;
    }
    let (mut units, mut acted, mut fallback) = (0, 0, 0);
    let mut count = |u: &battle::Unit, group: i64| {
        units += 1;
        if g.sprites.get(&group).is_none_or(|f| f.is_empty()) {
            fallback += 1;
        } else if u.acted {
            acted += 1;
        }
    };
    if let Some(st) = g.st.as_ref() {
        for (i, u) in st.units.iter().enumerate() {
            if !u.on_field || !u.alive() {
                continue;
            }
            let ux = (u.x * tw) as f64 - g.cam_x;
            let uy = (u.y * th) as f64 - g.cam_y;
            if ux < -(tw as f64) || ux > view_w as f64 || uy < -(th as f64) || uy > view_h as f64 {
                continue;
            }
            let group = g.map_sprite_group(i);
            count(u, group);
            g.draw_unit_sprite(target, ux, uy, tw as f64, th as f64, u, group);
        }
    }
    for u in &g.story_actors {
        if !u.on_field || !u.alive() {
            continue;
        }
        let ux = (u.x * tw) as f64 - g.cam_x;
        let uy = (u.y * th) as f64 - g.cam_y;
        count(u, u.fig);
        g.draw_unit_sprite(target, ux, uy, tw as f64, th as f64, u, u.fig);
    }
    (units, acted, fallback)
}

impl MapLayerCapture {
    /// 讀 `FD2_GOLDEN_MAP_LAYERS_OUT`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<MapLayerCapture> {
        let out = std::env::var("FD2_GOLDEN_MAP_LAYERS_OUT").unwrap_or_default();
        if out.is_empty() {
            return None;
        }
        assert!(
            std::env::var("FD2_GOLDEN_FRAMES_OUT").unwrap_or_default().is_empty(),
            "FD2_GOLDEN_MAP_LAYERS_OUT 與 FD2_GOLDEN_FRAMES_OUT 不能同時設（Go 端只有一個 ch01FrameObserver）"
        );
        Some(MapLayerCapture {
            out: out.into(),
            screen: Picture::new_image(LOGICAL_W, LOGICAL_H),
            view: Picture::new_image(LOGICAL_W / STORY_ZOOM, LOGICAL_H / STORY_ZOOM),
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
            g.player_clock.wall = MAP_LAYER_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = MAP_LAYER_FAR_FUTURE;
        let step = d * MAP_LAYER_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str, n: i64) {
        *self.stats.entry(name.to_string()).or_default() += n;
    }

    /// Go `mapLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        // Go：照 Draw 開頭清三個本幀旗標（忠實模式下其餘欄位都是 None）。
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = golden_map_layers_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        match branch {
            "spawn" => {
                self.screen.clear();
                self.screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
                let ok = g.draw_native_spawn_intro(&mut self.screen);
                if ok {
                    self.stat("spawn_drawn", 1);
                }
                format!("spawn|ok={ok}|rgba={}", sha_hex(self.screen.read_pixels()))
            }
            "tail" => {
                assert!(
                    g.modern_story_portraits.is_none(),
                    "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）"
                );
                let campaign_battle_view =
                    g.camp.is_none() || g.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == "battle");
                let legacy_viewport = g.story_bg || g.battle_event.is_some();
                let (target, view_w, view_h) = if legacy_viewport {
                    (&mut self.view, LOGICAL_W / STORY_ZOOM, LOGICAL_H / STORY_ZOOM)
                } else {
                    (&mut self.screen, LOGICAL_W, LOGICAL_H)
                };
                target.clear();
                target.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
                let (units, acted, fallback) = draw_world(g, target, view_w, view_h);
                let world = sha_hex(target.read_pixels());
                self.stat("world_units", units);
                self.stat("world_acted_units", acted);
                self.stat("world_fallback_units", fallback);
                let mut key = format!(
                    "tail|legacy={legacy_viewport}|world={view_w}x{view_h}:{world}|units={units}|acted={acted}|fallback={fallback}"
                );
                if !g.native_map_frame_admission(g.story_bg, campaign_battle_view) {
                    return key + "|admitted=false";
                }
                self.stat("map_admitted", 1);
                self.screen.clear();
                self.screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
                let ok = g.draw_native_map_frame(&mut self.screen);
                key += &format!(
                    "|admitted=true|map_ok={ok}|map={}|hud={}",
                    sha_hex(self.screen.read_pixels()),
                    g.native_interface_hud_visible
                );
                if !ok {
                    return key;
                }
                self.stat("map_drawn", 1);
                if g.phase_banner_step() >= 0 {
                    self.stat("map_banner_mosaic", 1);
                }
                if g.phase_banner_slide_offset().1 {
                    self.stat("map_banner_glyphs", 1);
                }
                let (img, ok) = g.native_map_frame_image();
                assert!(ok, "drawNativeMapFrame 成功後 nativeMapFrameImage 失敗");
                let img = img.unwrap();
                key + &format!("|index={}|palette={}", sha_hex(&img.pix), palette_sha(&img.palette))
            }
            other => other.to_string(),
        }
    }

    /// Go `mapLayerRecorder.observe`：在一個 `update` 之後呼叫。
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

    /// Go `mapLayerRecorder.finish`：寫 `map-layers.json`。
    pub(crate) fn finish(self) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let manifest = serde_json::json!({
            "schema": "fd2_rust_r5_map_layers/1",
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_map_layers_test.rs（tools/rust_r5_map_layers.sh）",
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
        std::fs::write(self.out.join("map-layers.json"), raw).expect("寫 map-layers.json");
    }
}
