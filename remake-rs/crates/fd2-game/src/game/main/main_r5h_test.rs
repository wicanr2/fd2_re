//! R5 階段 2 r5h 組（HUD 與選單）的測試。Go 對本組只有一條不經 HD 的測試
//! （`chapter_parity_ring_open_test.go:35`），其餘 Go 測試走 HD 呈現（`modern_hd_*_test.go`，R6）或要以環境變數指定
//! 的完整存檔（`modern_hd_movement_panel_test.go`，未設時 `t.Skip`）。下面的「Rust 新增」以 Go 程式碼與 `canvas.rs`
//! 的實測公式（121「Ebiten 取整」）算期望值；第一章實際畫到的畫格由局部黃金逐張比對（`PORTING.md`「r5h：局部黃金」）。

use super::*;
use crate::test_support::cjk_font_available;

fn pixel(p: &Picture, x: i64, y: i64) -> [u8; 4] {
    let at = ((y * p.width + x) * 4) as usize;
    [p.pix[at], p.pix[at + 1], p.pix[at + 2], p.pix[at + 3]]
}

/// 256 色測試調色盤：索引 i 是不透明的 (i, 255-i, i/2)。
fn ramp_palette() -> ColorPalette {
    (0..256).map(|i| ColorRgba { r: i as u8, g: 255 - i as u8, b: (i / 2) as u8, a: 0xff }).collect()
}

fn ramp_rgba(index: u8) -> [u8; 4] {
    [index, 255 - index, index / 2, 0xff]
}

/// 320×200 索引畫格：每個像素是 `(x + 3*y + seed) & 0xff`。
fn indexed_frame(seed: u8) -> Vec<u8> {
    (0..200).flat_map(|y| (0..320).map(move |x| ((x + 3 * y + seed as usize) & 0xff) as u8)).collect()
}

/// 畫面是 `frame` 以 `palette` 2 倍最近鄰放大的結果（取樣檢查）。
fn assert_scaled_frame(screen: &Picture, frame: &[u8], palette: impl Fn(u8) -> [u8; 4]) {
    for (x, y) in [(0, 0), (1, 1), (2, 0), (319, 199), (638, 398), (639, 399), (100, 51), (77, 301)] {
        let index = frame[(y / 2 * 320 + x / 2) as usize];
        assert_eq!(pixel(screen, x, y), palette(index), "({x},{y})");
    }
}

/// Go `chapter_parity_ring_open_test.go:35` `TestNativeActionOverlayMapAdmissionRequiresThisDrawAndFaithfulSource`。
#[test]
fn test_native_action_overlay_map_admission_requires_this_draw_and_faithful_source() {
    for name in ["admitted", "not_presented", "no_ring", "no_actor", "missing_cells", "modern"] {
        let mut g = Game {
            ring: true,
            sel: Some(0),
            native_action_cells_raw: vec![fdother::RawCell::default(); NATIVE_ACTION_OVERLAY_CELL_COUNT as usize],
            ..Default::default()
        };
        let mut presented = true;
        match name {
            "not_presented" => {
                presented = false;
                g.native_map_vga = vec![0; 320 * 200];
            }
            "no_ring" => g.ring = false,
            "no_actor" => {
                g.sel = None;
                g.native_system_cursor_overlay = true;
            }
            "missing_cells" => g.native_action_cells_raw.truncate(77),
            "modern" => {
                g.modern_story_portraits =
                    Some(ModernStoryPortraitSet { battle_action_icons: vec![None; 4], ..Default::default() });
            }
            _ => {}
        }
        assert_eq!(g.native_action_overlay_in_map_frame(presented), name == "admitted", "{name}: presented gate");
    }
}

/// `drawCursor`：四條 2 像素寬的不透明白條，各自 `Translate` 到格子四邊；其餘像素不動。
#[test]
fn draw_cursor_paints_four_white_bars() {
    let mut screen = Picture::new_image(640, 400);
    draw_cursor(&mut screen, 10.0, 20.0, 48.0, 48.0);
    let white = [0xff; 4];
    for (x, y) in [(10, 20), (57, 21), (10, 67), (57, 66), (11, 40), (56, 40), (30, 20), (30, 67)] {
        assert_eq!(pixel(&screen, x, y), white, "({x},{y})");
    }
    for (x, y) in [(12, 22), (55, 65), (30, 40), (9, 20), (58, 20), (10, 68), (10, 19)] {
        assert_eq!(pixel(&screen, x, y), [0; 4], "({x},{y})");
    }
    // 小數平移照 Ebiten 頂點對齊（canvas.rs 的 adjust_destination_pixel）：與先平移後畫的結果相同。
    let mut a = Picture::new_image(640, 400);
    draw_cursor(&mut a, 10.25, 20.0, 48.0, 48.0);
    let mut b = Picture::new_image(640, 400);
    let mut bar = Picture::new_image(48, 2);
    bar.fill(ColorRgba { r: 0xff, g: 0xff, b: 0xff, a: 0xff });
    let mut op = DrawImageOptions::default();
    op.geo_m.translate(10.25, 20.0);
    b.draw_image(&bar, &op);
    assert_eq!(a.pix[..640 * 4 * 22], b.pix[..640 * 4 * 22], "上緣白條與單獨畫一條相同");
}

/// `drawNativeEndTurnRecovery`：以 `nativeMapDAC` 轉出的調色盤把目前幀 2 倍貼上並設 `drawn`；幀索引、緩衝長度或
/// DAC 長度不對時回 false、不畫、不設 `drawn`。
#[test]
fn draw_native_end_turn_recovery_scales_current_frame_and_fails_closed() {
    let frames = [indexed_frame(0), indexed_frame(7), indexed_frame(40)];
    // DAC：索引 i 的 6 位元分量 (i&63, 63-(i&63), (i>>2)&63)。
    let dac: Vec<u8> = (0..256).flat_map(|i| [(i & 63) as u8, (63 - (i & 63)) as u8, ((i >> 2) & 63) as u8]).collect();
    let expand = |v: u8| (v << 2) | (v >> 4);
    let palette = |index: u8| {
        let i = index as usize;
        [expand(dac[i * 3]), expand(dac[i * 3 + 1]), expand(dac[i * 3 + 2]), 0xff]
    };
    let mut g = Game {
        native_map_dac: dac.clone(),
        native_end_turn_recovery: Some(NativeEndTurnRecoveryJob {
            frames: frames.clone(),
            frame: 1,
            ..Default::default()
        }),
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    assert!(g.draw_native_end_turn_recovery(&mut screen));
    assert!(g.native_end_turn_recovery.as_ref().unwrap().drawn);
    assert_scaled_frame(&screen, &frames[1], palette);
    for (frame, dac_len) in [(3, 768), (-1, 768), (0, 767)] {
        let mut g = Game {
            native_map_dac: dac[..dac_len].to_vec(),
            native_end_turn_recovery: Some(NativeEndTurnRecoveryJob {
                frames: frames.clone(),
                frame,
                ..Default::default()
            }),
            ..Default::default()
        };
        let mut screen = Picture::new_image(640, 400);
        assert!(!g.draw_native_end_turn_recovery(&mut screen), "frame={frame} dac={dac_len}");
        assert!(!g.native_end_turn_recovery.as_ref().unwrap().drawn);
        assert!(screen.pix.iter().all(|&b| b == 0));
    }
    let mut short = frames.clone();
    short[0].pop();
    let mut g = Game {
        native_map_dac: dac,
        native_end_turn_recovery: Some(NativeEndTurnRecoveryJob { frames: short, frame: 0, ..Default::default() }),
        ..Default::default()
    };
    assert!(!g.draw_native_end_turn_recovery(&mut Picture::new_image(640, 400)), "緩衝長度不是 64000");
}

/// `drawNativeMovementPanel`：每幀先清 `movementPanelDraw`；有選中單位與面板時 `Scale(2,2)` 後平移到
/// `(nativeMovePanelX*2, 18)`，記下面板矩形。
#[test]
fn draw_native_movement_panel_scales_panel_and_records_rect() {
    let mut panel = Picture::new_image(149, 42);
    for y in 0..42 {
        for x in 0..149 {
            panel.set(x, y, ColorRgba { r: x as u8, g: y as u8, b: 9, a: 0xff });
        }
    }
    let mut g = Game {
        sel: Some(0),
        native_move_panel: Some(Arc::new(panel.clone())),
        native_move_panel_x: 11,
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    g.draw_native_movement_panel(&mut screen);
    let draw = g.movement_panel_draw.as_ref().expect("畫成時記下面板");
    assert_eq!(draw.rect, rect(22, 18, 320, 102));
    assert!(!draw.hd && !draw.art_hd);
    for (x, y) in [(22, 18), (23, 19), (24, 18), (319, 101), (100, 60)] {
        assert_eq!(pixel(&screen, x, y), pixel(&panel, (x - 22) / 2, (y - 18) / 2), "({x},{y})");
    }
    assert_eq!(pixel(&screen, 21, 18), [0; 4]);
    assert_eq!(pixel(&screen, 22, 17), [0; 4]);
    assert_eq!(pixel(&screen, 320, 18), [0; 4]);
    g.sel = None;
    g.draw_native_movement_panel(&mut screen);
    assert!(g.movement_panel_draw.is_none(), "沒有選中單位時清掉上一幀的面板");
}

/// `drawNativePlayerStatus`：沒有轉場幀時畫 `steady`，有轉場幀時畫 `frames[frame]`，都用 `nativeUIPalette` 2 倍
/// 貼上並設 `drawn`。
#[test]
fn draw_native_player_status_presents_steady_or_transition_frame() {
    let mut g = Game { native_ui_palette: ramp_palette(), ..Default::default() };
    let mut screen = Picture::new_image(640, 400);
    g.draw_native_player_status(&mut screen);
    assert!(screen.pix.iter().all(|&b| b == 0), "沒有狀態時不畫");
    let (steady, transition) = (indexed_frame(3), indexed_frame(90));
    g.native_player_status = Some(NativePlayerStatusState { steady: steady.clone(), ..Default::default() });
    g.native_map_presented = true;
    g.draw_native_player_status(&mut screen);
    assert!(g.native_player_status.as_ref().unwrap().drawn);
    assert!(!g.native_map_presented, "presentNativeClassFrameWithPalette 清本幀旗標");
    assert_scaled_frame(&screen, &steady, ramp_rgba);
    g.native_player_status = Some(NativePlayerStatusState {
        steady,
        frames: vec![indexed_frame(1), transition.clone()],
        frame: 1,
        ..Default::default()
    });
    g.draw_native_player_status(&mut screen);
    assert!(g.native_player_status.as_ref().unwrap().drawn);
    assert_scaled_frame(&screen, &transition, ramp_rgba);
}

/// `drawNativeSystemInfoUI`：依階段選開框、收框或穩態幀，以 `state.palette` 2 倍貼上並設 `drawn`、清
/// `hdPresented`；幀長度不是 64000 或調色盤不是 256 色時回 false。
#[test]
fn draw_native_system_info_ui_selects_phase_frame_and_fails_closed() {
    let (opening, closing, steady) =
        (vec![indexed_frame(10), indexed_frame(20)], vec![indexed_frame(30)], indexed_frame(50));
    let state = |phase: &str, frame: i64| NativeSystemInfoUIState {
        opening: opening.clone(),
        closing: closing.clone(),
        steady: steady.clone(),
        phase: phase.to_string(),
        frame,
        palette: ramp_palette(),
        hd_presented: true,
        ..Default::default()
    };
    let mut g = Game::default();
    assert!(!g.draw_native_system_info_ui(&mut Picture::new_image(640, 400)), "沒有狀態");
    for (phase, frame, want) in
        [("opening", 1, &opening[1]), ("opening", 2, &steady), ("closing", 0, &closing[0]), ("steady", 0, &steady)]
    {
        g.native_system_info_ui = Some(state(phase, frame));
        let mut screen = Picture::new_image(640, 400);
        assert!(g.draw_native_system_info_ui(&mut screen), "{phase}/{frame}");
        let s = g.native_system_info_ui.as_ref().unwrap();
        assert!(s.drawn && !s.hd_presented, "{phase}/{frame}");
        assert_scaled_frame(&screen, want, ramp_rgba);
    }
    let mut bad = state("steady", 0);
    bad.steady.pop();
    g.native_system_info_ui = Some(bad);
    assert!(!g.draw_native_system_info_ui(&mut Picture::new_image(640, 400)), "幀長度不對");
    let mut bad = state("steady", 0);
    bad.palette.pop();
    g.native_system_info_ui = Some(bad);
    assert!(!g.draw_native_system_info_ui(&mut Picture::new_image(640, 400)), "調色盤不是 256 色");
    assert!(!g.native_system_info_ui.as_ref().unwrap().drawn);
}

/// `drawNativeSystemEndTurn` 的非確認框分支：守門、死亡獎勵訊息等鍵時畫最後一幀、回覆延遲時畫接受或取消的
/// 最後一幀，都經 `presentNativeClassFrame`（`nativeClassUI.palette`）。
#[test]
fn draw_native_system_end_turn_presents_reward_and_outcome_frames() {
    let class_ui = NativeClassUIAssets { palette: ramp_palette(), ..Default::default() };
    let base = || Game {
        native_class_ui: Some(class_ui.clone()),
        native_preparation_ui: Some(NativePreparationUIAssets::default()),
        native_system_end_turn_ui: Some(Box::default()),
        ..Default::default()
    };
    // 守門：缺任一個就回 false。
    for missing in 0..3 {
        let mut g = base();
        match missing {
            0 => g.native_system_end_turn_ui = None,
            1 => g.native_class_ui = None,
            _ => g.native_preparation_ui = None,
        }
        assert!(!g.draw_native_system_end_turn(&mut Picture::new_image(640, 400)), "缺件 {missing}");
    }
    // 沒有任何子狀態：回 false。
    assert!(!base().draw_native_system_end_turn(&mut Picture::new_image(640, 400)));
    // 死亡獎勵訊息：等鍵時畫 final，否則回 false。
    let final_frame = indexed_frame(77);
    for await_ack in [false, true] {
        let mut g = base();
        g.native_system_end_turn_ui.as_mut().unwrap().reward_message =
            Some(NativeDeathRewardMessageState { await_ack, r#final: final_frame.clone(), ..Default::default() });
        let mut screen = Picture::new_image(640, 400);
        assert_eq!(g.draw_native_system_end_turn(&mut screen), await_ack);
        if await_ack {
            assert_scaled_frame(&screen, &final_frame, ramp_rgba);
        } else {
            assert!(screen.pix.iter().all(|&b| b == 0));
        }
    }
    // 回覆延遲：依 acceptedOutcome 取 accepted 或 canceled 的最後一幀；清單是空的回 false。
    for accepted in [false, true] {
        let mut g = base();
        g.native_system_end_turn_delay = 2;
        let s = g.native_system_end_turn_ui.as_mut().unwrap();
        s.accepted_outcome = accepted;
        s.accepted = vec![indexed_frame(1), indexed_frame(2)];
        s.canceled = vec![indexed_frame(3), indexed_frame(4)];
        let want = if accepted { s.accepted[1].clone() } else { s.canceled[1].clone() };
        let mut screen = Picture::new_image(640, 400);
        assert!(g.draw_native_system_end_turn(&mut screen), "accepted={accepted}");
        assert_scaled_frame(&screen, &want, ramp_rgba);
    }
    let mut g = base();
    g.native_system_end_turn_delay = 1;
    assert!(!g.draw_native_system_end_turn(&mut Picture::new_image(640, 400)), "沒有回覆幀");
}

/// `drawPhaseBanner`：守門（沒有橫幅或字型）與原版字樣可用時讓開；只有缺 FDOTHER #5 字模時以重製端字型畫在
/// 中央偏上，顏色 `(0xff,0xc8,0x50)`。
#[test]
fn draw_phase_banner_only_falls_back_without_original_glyphs() {
    let mut g = Game { banner: "PLAYER PHASE".into(), banner_t: 30, ..Default::default() };
    let mut screen = Picture::new_image(640, 400);
    g.draw_phase_banner(&mut screen);
    assert!(screen.pix.iter().all(|&b| b == 0), "沒有字型時不畫");
    if !cjk_font_available() {
        return;
    }
    g.font = Some(load_font().expect("系統 Noto CJK 已在"));
    g.native_map_assets = Some(NativeMapAssets {
        command_heal_digits: vec![Default::default(); PHASE_BANNER_GLYPH_ENEMY as usize + 1],
        ..Default::default()
    });
    g.draw_phase_banner(&mut screen);
    assert!(screen.pix.iter().all(|&b| b == 0), "原版字樣可用時讓開");
    g.native_map_assets = None;
    let (offset, visible) = g.phase_banner_slide_offset();
    if !visible {
        return;
    }
    g.draw_phase_banner(&mut screen);
    assert!(screen.pix.iter().any(|&b| b != 0), "後備字樣沒有畫");
    // 與直接以 Go 的參數呼叫 Font.Draw 逐位元組相同：x = (640-Width)/2 + offset*640/320、y = 400/2-24、倍率 2.2。
    let font = g.font.as_mut().unwrap();
    let w = font.width("PLAYER PHASE", 2.2);
    let mut want = Picture::new_image(640, 400);
    let c = ColorRgba { r: 0xff, g: 0xc8, b: 0x50, a: 0xff };
    font.draw(&mut want, "PLAYER PHASE", (640.0 - w) / 2.0 + (offset * 640 / 320) as f64, 176.0, 2.2, c);
    assert_eq!(screen.pix, want.pix);
}

/// `drawSpellMenu`：底框 `(0x10,0x1c,0x40,0xee)` 以 source-over 疊在 (20,60)，高度 44+30×法術數；每個法術一行。
#[test]
fn draw_spell_menu_box_and_rows() {
    let mut u = battle::Unit::default();
    (u.mp, u.spells) = (5, vec![1, 2].into());
    let mut st = battle::State::default();
    st.units = vec![u];
    let spells = vec![
        battle::Spell { id: 1, mp: 3, name: "火".into(), ..Default::default() },
        battle::Spell { id: 2, mp: 9, name: "冰".into(), ..Default::default() },
    ];
    let mut g = Game { st: Some(st), sel: Some(0), spell_open: true, spells, ..Default::default() };
    let mut screen = Picture::new_image(640, 400);
    g.draw_spell_menu(&mut screen);
    assert!(screen.pix.iter().all(|&b| b == 0), "沒有字型時不畫");
    if !cjk_font_available() {
        return;
    }
    g.font = Some(load_font().expect("系統 Noto CJK 已在"));
    screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
    g.draw_spell_menu(&mut screen);
    // 黑底上的 source-over：rgb 原值，alpha 0xee + mul255(255×0x11) = 0xff。
    let boxed = [0x10, 0x1c, 0x40, 0xff];
    for (x, y) in [(20, 60), (249, 60), (20, 163), (249, 163), (240, 120)] {
        assert_eq!(pixel(&screen, x, y), boxed, "({x},{y})");
    }
    for (x, y) in [(19, 60), (250, 60), (20, 164), (20, 59)] {
        assert_eq!(pixel(&screen, x, y), [0, 0, 0, 0xff], "({x},{y})");
    }
    // 兩行法術文字（96、126 起）都有墨跡。
    for row in [96, 126] {
        let inked = (row..row + 24).any(|y| (32..200).any(|x| pixel(&screen, x, y) != boxed));
        assert!(inked, "第 {row} 列沒有文字");
    }
}
