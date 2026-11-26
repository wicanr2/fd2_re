//! Go `remake/cmd/fd2/rust_golden_r5p_layers_test.go` 的 Rust 端（R5 階段 2 r5p 組的局部黃金，預設關閉）。
//!
//! 兩種模式，掛點、步驟與 60 Hz 虛擬時鐘同 R5 黃金畫格：
//! - chapter：設 `FD2_GOLDEN_R5P_LAYERS_MODE=chapter`、`FD2_GOLDEN_R5P_LAYERS_OUT=<目錄>` 時，章重播的
//!   [`Driver::pump`] 每一幀開頭呼叫 [`R5pLayerCapture::observe`]（Go `ch01FrameObserver` 同一位置）。
//! - opening：`test_rust_golden_r5p_layers_opening`（`#[ignore]`），步驟同 `rust_golden_frames_opening_test.rs`
//!   （Go `goldenFramesOpening`），每個 `update` 之後擷取。
//!
//! 每次擷取依 `Draw` 的分支順序只畫本組的圖層（做法見 Go 檔開頭）：ramp 分支呼叫整個 [`Game::draw`]；尾段在
//! 640×400 `Fill` 黑後呼叫 [`Game::draw_normalized_world_layer`]，再照 `Draw` 把原生整幀畫到另一張畫布（只為了
//! 留下 `composeNativeMapFrame` 對 `native_map_vga` 的重建，鍵只記成功與否），`fade` 非 `None` 時再呼叫
//! [`Game::draw_story_fade_overlay`]。比對鍵與 Go 逐字同格式，重播結束寫 `<目錄>/r5p-layers.json`，
//! `tools/rust_r5p_layers_compare.py` 逐擷取點比對。

use super::*;
use sha2::Digest;
use std::path::{Path, PathBuf};

/// Go `goldenFrameTick`（同 `rust_golden_frames_test.rs`）。
const R5P_TICK: i64 = 1_000_000_000 / 60;

/// Go `goldenFrameFarFuture`（同 `rust_golden_frames_test.rs`）。
const R5P_FAR_FUTURE: GameInstant = GameInstant(i64::MAX / 2);

#[derive(Debug, Clone, serde::Serialize)]
pub(crate) struct R5pLayerRow {
    pub(crate) capture: i64,
    pub(crate) count: i64,
    pub(crate) update_frame: i64,
    pub(crate) node: String,
    pub(crate) key: String,
}

/// Go `r5pLayerRecorder`。
pub(crate) struct R5pLayerCapture {
    out: PathBuf,
    screen: Picture,
    map_screen: Picture,
    pinned: bool,
    clock_mode: &'static str,
    last_frame: i64,
    virtual_frames: i64,
    captures: i64,
    seq: sha2::Sha256,
    rows: Vec<R5pLayerRow>,
    branches: BTreeMap<String, i64>,
    stats: BTreeMap<String, i64>,
}

fn sha_hex(raw: &[u8]) -> String {
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go `r5pLayerBranch`：`Draw` 的 early-return 順序，只讀欄位。`nativePaletteRamp` 之前的分支先 return 時是
/// early；其後同 [`golden_map_layers_branch`]。
pub(crate) fn r5p_layer_branch(g: &Game) -> &'static str {
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
        || g.native_town_departure.is_some();
    if early {
        return "early";
    }
    if g.native_palette_ramp.is_some() {
        return "ramp";
    }
    golden_map_layers_branch(g)
}

/// Go `r5pFadeAlpha`：`main.go:10175..10181` 的 `uint8(alpha * 0xff)`，只供比對鍵顯示。
fn fade_alpha(fade: &StoryFade) -> u8 {
    let frac = fade.t as f64 / fade.total as f64;
    let mut alpha = frac;
    if !fade.out {
        alpha = 1.0 - frac;
    }
    (alpha * 255.0) as i64 as u8
}

impl R5pLayerCapture {
    pub(crate) fn new(out: PathBuf) -> R5pLayerCapture {
        R5pLayerCapture {
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

    /// chapter 模式：讀 `FD2_GOLDEN_R5P_LAYERS_OUT`、`FD2_GOLDEN_R5P_LAYERS_MODE`；沒設回 `None`（預設關閉）。
    pub(crate) fn from_env() -> Option<R5pLayerCapture> {
        let var = |name: &str| std::env::var(name).unwrap_or_default();
        let out = var("FD2_GOLDEN_R5P_LAYERS_OUT");
        if out.is_empty() {
            return None;
        }
        let mode = var("FD2_GOLDEN_R5P_LAYERS_MODE");
        if mode == "opening" {
            // opening 模式由 test_rust_golden_r5p_layers_opening 自己建立，章重播不掛。
            return None;
        }
        assert_eq!(mode, "chapter", "FD2_GOLDEN_R5P_LAYERS_MODE={mode:?}；只允許 chapter 或 opening");
        assert!(
            var("FD2_GOLDEN_FRAMES_OUT").is_empty() && var("FD2_GOLDEN_MAP_LAYERS_OUT").is_empty(),
            "FD2_GOLDEN_R5P_LAYERS_OUT 不能與其他擷取同時設（Go 端只有一個 ch01FrameObserver）"
        );
        Some(R5pLayerCapture::new(out.into()))
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
            g.player_clock.wall = R5P_FAR_FUTURE;
            return;
        }
        let d = g.frame - self.last_frame;
        self.last_frame = g.frame;
        assert!(d >= 0, "g.frame 倒退：{d}");
        self.virtual_frames += d;
        g.player_clock.wall = R5P_FAR_FUTURE;
        let step = d * R5P_TICK * normalize_player_speed(g.player_clock.speed);
        g.player_clock.game = g.player_clock.game + GoDuration(step);
    }

    fn stat(&mut self, name: &str, n: i64) {
        *self.stats.entry(name.to_string()).or_default() += n;
    }

    /// Go `r5pLayerRecorder.layers`。
    fn layers(&mut self, g: &mut Game) -> String {
        // Go：照 Draw 開頭清三個本幀旗標（同 rust_golden_map_layers_test.rs）。
        g.native_map_presented = false;
        g.movement_panel_draw = None;
        g.native_interface_hud_visible = false;
        let branch = r5p_layer_branch(g);
        *self.branches.entry(branch.to_string()).or_default() += 1;
        match branch {
            "ramp" => {
                let hd = g.native_palette_ramp.as_ref().unwrap().hd_battle_sources.is_some();
                self.screen.clear();
                g.draw(&mut self.screen);
                let rgba = sha_hex(self.screen.read_pixels());
                let j = g.native_palette_ramp.as_ref().expect("Draw 不會清掉調色盤斜坡");
                let (step, drawn) = (j.step, j.drawn);
                self.stat("ramp_frames", 1);
                if drawn {
                    self.stat("ramp_drawn", 1);
                }
                format!(
                    "ramp|step={step}|drawn={drawn}|hd_sources={hd}|dac_white={}|dac_black={}|help={}|rgba={rgba}",
                    g.native_full_dac_white, g.native_full_dac_black, g.help_visible
                )
            }
            "tail" => {
                assert!(
                    g.modern_story_portraits.is_none(),
                    "局部黃金只涵蓋忠實模式（modernStoryPortraits 必須是 nil）"
                );
                assert!(!g.unit_labels, "FD2_UNIT_LABELS 不在局部黃金範圍");
                let legacy_viewport = g.story_bg || g.battle_event.is_some();
                let sel = g.sel.is_some();
                let reach = if sel { g.reach.len() as i64 } else { 0 };
                let cast = sel && g.cast_sp.is_some();
                let attack = sel && g.cast_sp.is_none() && g.moved && !g.ring && !g.spell_open;
                self.screen.clear();
                self.screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
                g.draw_normalized_world_layer(&mut self.screen, legacy_viewport);
                let world = sha_hex(self.screen.read_pixels());
                self.stat("world_frames", 1);
                if legacy_viewport {
                    self.stat("world_legacy", 1);
                }
                if sel {
                    self.stat("world_sel", 1);
                    self.stat("world_reach_cells", reach);
                }
                if cast {
                    self.stat("world_cast", 1);
                }
                if attack {
                    self.stat("world_attack", 1);
                }
                let mut key = format!(
                    "tail|legacy={legacy_viewport}|sel={sel}|reach={reach}|cast={cast}|attack={attack}|world={world}"
                );
                // Go：Draw 的下一步是原生整幀（r5m），照畫到另一張畫布，只為了留下它對 nativeMapVGA 的重建。
                let campaign_battle_view =
                    g.camp.is_none() || g.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == "battle");
                if g.native_map_frame_admission(g.story_bg, campaign_battle_view) {
                    self.map_screen.clear();
                    key += &format!("|map_ok={}", g.draw_native_map_frame(&mut self.map_screen));
                }
                let Some(fade) = g.fade.as_ref() else {
                    return key + "|fade=false";
                };
                let (alpha, out, t, total) = (fade_alpha(fade), fade.out, fade.t, fade.total);
                g.draw_story_fade_overlay(&mut self.screen);
                let rgba = sha_hex(self.screen.read_pixels());
                self.stat("fade_frames", 1);
                key + &format!("|fade=true|fade_out={out}|fade_t={t}/{total}|fade_alpha={alpha}|fade_rgba={rgba}")
            }
            other => other.to_string(),
        }
    }

    /// Go `r5pLayerRecorder.observe`：在一個 `update` 之後呼叫。
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
        self.rows.push(R5pLayerRow { capture: index, count: 1, update_frame: g.frame, node: node_id(g), key });
    }

    /// Go `r5pLayerRecorder.finish`：寫 `r5p-layers.json`。
    pub(crate) fn finish(self, mode: &str, extra: serde_json::Map<String, serde_json::Value>) {
        let seq: String = self.seq.finalize().iter().map(|b| format!("{b:02x}")).collect();
        let mut manifest = serde_json::json!({
            "schema": "fd2_rust_r5p_layers/1",
            "mode": mode,
            "plan": "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md",
            "producer": "remake-rs/crates/fd2-game parity_replay/rust_golden_r5p_layers_test.rs（tools/rust_r5p_layers.sh）",
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
        for (k, v) in extra {
            manifest[k] = v;
        }
        std::fs::create_dir_all(&self.out).expect("建立輸出目錄");
        let mut raw = serde_json::to_vec_pretty(&manifest).unwrap();
        raw.push(b'\n');
        std::fs::write(self.out.join("r5p-layers.json"), raw).expect("寫 r5p-layers.json");
    }
}

/// opening 模式的一步：`parityUpdateGame` 之後擷取（同 `rust_golden_frames_opening_test.rs` 的 `opening_step`）。
fn r5p_opening_step(d: &mut Driver, capture: &mut R5pLayerCapture) {
    if let Err(err) = parity_update_game(d) {
        panic!("Update：{err}");
    }
    if !d.g.load_err.is_empty() {
        panic!("執行期錯誤：{}", d.g.load_err);
    }
    capture.observe(&mut d.g);
}

/// Go `TestRustGoldenR5pLayers` 的 opening 分支。父行程檢查環境變數，再以 Go `t.Setenv` 的那組變數在子行程執行。
#[test]
#[ignore = "R5 r5p 局部黃金（opening）：由 tools/rust_r5p_layers.sh 以 --ignored 執行"]
fn test_rust_golden_r5p_layers_opening() {
    let var = |name: &str| std::env::var(name).unwrap_or_default();
    let (out, mode, slot) =
        (var("FD2_GOLDEN_R5P_LAYERS_OUT"), var("FD2_GOLDEN_R5P_LAYERS_MODE"), var("FD2_PARITY_SLOT"));
    if out.is_empty() {
        eprintln!("略過：需要 FD2_GOLDEN_R5P_LAYERS_OUT（R5 r5p 局部黃金，只由 tools/rust_r5p_layers.sh 執行）");
        return;
    }
    assert_eq!(mode, "opening", "FD2_GOLDEN_R5P_LAYERS_MODE={mode:?}；本測試只跑 opening 模式");
    assert!(!slot.is_empty(), "opening 模式需要 FD2_PARITY_SLOT（與 chapter 模式同一個建構槽）");
    let out = PathBuf::from(out);
    let child = std::env::var_os("FD2_GAME_F_TEST_DIR").is_some();
    let slot_copy = out.join("FD2.SAV");
    if !child {
        std::fs::create_dir_all(&out).expect("建立輸出目錄");
        let raw = std::fs::read(&slot).unwrap_or_else(|e| panic!("{slot}：{e}"));
        std::fs::write(&slot_copy, raw).expect("寫 FD2.SAV 複本");
    }
    let slot_copy_env = slot_copy.clone();
    let Some(_data_home) = f_child_env(module_path!(), "test_rust_golden_r5p_layers_opening", move |dir| {
        vec![
            ("FD2_TITLE", "1".into()),
            ("FD2_SEED", "4".into()),
            ("FD2_MUTE", "1".into()),
            ("FD2_CAMPAIGN", CANONICAL_CAMPAIGN_REFERENCE.into()),
            ("FD2_SHOT_AI", "1".into()),
            ("FD2_NATIVE_SAVE", slot_copy_env.clone().into_os_string()),
            ("XDG_DATA_HOME", dir.as_os_str().to_os_string()),
        ]
    }) else {
        return;
    };
    run_r5p_layers_opening(&out);
}

/// Go `r5pLayersOpening` 在 `t.Setenv` 之後的本體（子行程）；步驟同 `run_golden_frames_opening`。
fn run_r5p_layers_opening(out: &Path) {
    let mut capture = R5pLayerCapture::new(out.to_path_buf());
    let g = load_game().expect("loadGame");
    // R5 r5f：Go 黃金載入了系統的 NotoSansCJK-Regular.ttc。
    require_golden_font(&g);
    if !g.load_err.is_empty() {
        panic!("{}", g.load_err);
    }
    if g.title_phase != "cutscene" {
        panic!("正常START未從開場動畫開始：{:?}", g.title_phase);
    }
    let mut d = Driver {
        g,
        clock: VirtualFrameClock::default(),
        frame_capture: None,
        map_layer_capture: None,
        dialogue_layer_capture: None,
        r5p_layer_capture: None,
        r5c_layer_capture: None,
        ttf_layer_capture: None,
        r5b_layer_capture: None,
        r5h_layer_capture: None,
        r5s_frame_capture: None,
        hp_policy: None,
    };
    // Go 黃金的 rand.Seed(4)（r5d 接上的全域亂數；本組圖層不取用，照 Go 的步驟設定）。
    pin_global_rand(&mut d.g);
    let mut phases = serde_json::Map::new();
    let mut extra = serde_json::Map::new();
    extra.insert(
        "scope".into(),
        "標題開場動畫 → 選單 → START → 序章故事 → 第一次進入 battle_ch01；每個 Update 之後擷取（另含 loadGame 後一張）"
            .into(),
    );
    // loadGame 之後、第一個 Update 之前。
    capture.observe(&mut d.g);
    // 1. 開場動畫到標題選單（同 startFromNormalTitle）。
    let mut frame = 0;
    while d.g.title_phase != "menu" {
        if frame >= JOURNEY_TITLE_FRAMES || d.g.title_phase.is_empty() {
            panic!("正常開場動畫未進標題選單");
        }
        ack_presents(&mut d.g);
        r5p_opening_step(&mut d, &mut capture);
        frame += 1;
    }
    phases.insert("title_menu".into(), capture.captures.into());
    // 2. START（選單第 0 項），等標題結束。
    if d.g.title_sel != 0 || !d.g.apply_title_menu_event(TITLE_MENU_CONFIRM) {
        panic!("正常START未由標題owner消費");
    }
    let mut frame = 0;
    while !d.g.title_phase.is_empty() {
        if frame >= 600 {
            panic!("START未正常離開標題");
        }
        ack_presents(&mut d.g);
        answer_native_treasure_prompt(&mut d.g);
        r5p_opening_step(&mut d, &mut capture);
        frame += 1;
    }
    if d.g.camp.is_none() || node_id(&d.g) != JOURNEY_OPENING_NODE {
        panic!("START未正常進入序章");
    }
    phases.insert("story_ch00".into(), capture.captures.into());
    // 3. 同 parityReplay.settleTown。
    for frame in 0..JOURNEY_TOWN_SETTLE_MAX {
        let g = &d.g;
        if g.native_class_ui_job.is_none()
            && g.fade.is_none()
            && g.transition_reveal.is_none()
            && g.indexed_transition.is_none()
            && frame > 10
        {
            break;
        }
        ack_presents(&mut d.g);
        r5p_opening_step(&mut d, &mut capture);
    }
    // 4. 同 driveStory（停止條件：進入 battle_ch01）。
    let mut j = new_journey_trace();
    let (mut last_sig, mut still) = (String::new(), 0);
    let mut frame = 0;
    while node_id(&d.g) != "battle_ch01" {
        if frame >= JOURNEY_STORY_FRAMES {
            panic!("故事節點 {} 在 {JOURNEY_STORY_FRAMES} 幀內沒有結束", node_id(&d.g));
        }
        j.observe(&d.g);
        if let Some(mut n) = node(&d.g)
            && (n.r#type == "story" || n.r#type == "cutscene")
            && story_enter_ready(&d.g)
        {
            let before = story_signature(&d.g);
            d.g.handle_native_story_input(Some(&mut n), NativeStoryInput { enter: true, escape: false });
            if story_signature(&d.g) != before {
                j.press(&journey_phase(&d.g));
            }
        }
        ack_presents(&mut d.g);
        r5p_opening_step(&mut d, &mut capture);
        let sig = progress_signature(&d.g);
        if sig == last_sig {
            still += 1;
            if still > JOURNEY_STALL_FRAMES {
                panic!("故事節點 {} 卡住 {still} 幀", node_id(&d.g));
            }
        } else {
            (last_sig, still) = (sig, 0);
        }
        frame += 1;
    }
    phases.insert("battle_ch01".into(), capture.captures.into());
    extra.insert("story_nodes".into(), j.nodes.unwrap_or_default().join(" → ").into());
    extra.insert("story_presses".into(), j.presses.into());
    extra.insert("phase_captures".into(), phases.into());
    capture.finish("opening", extra);
}
