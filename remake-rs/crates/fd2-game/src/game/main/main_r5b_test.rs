//! R5 階段 2 r5b 組（戰鬥演出：`drawBattleScene`、`drawNativePhysicalPrelude`、`drawNativePhysicalBody`、
//! `drawNativeAIIdleRecovery`、`drawNativeCh28PostPresent`）的測試。
//!
//! Go 測試移植：`native_physical_scene_test.go:723` `TestNativePhysicalPreludeGPU`（GPU 讀回改成 CPU 畫格；第 8 幀之後的
//! `drawNativePhysicalBase` 段是 r8，沒有移植）。`native_ai_idle_recovery_test.go:74` 移植在
//! `native_end_turn_recovery/native_end_turn_recovery_test.rs`（夾具在那裡）。其餘是「Rust 新增」，期望值照 Go 程式碼與
//! `canvas.rs` 的語意（`NewImageFromImage` 預乘、`Scale(2, 2)` 最近鄰、source-over）逐通道算；整段的逐張比對見
//! `PORTING.md`「r5b：局部黃金」。

use super::*;
use crate::game::native_physical_scene::native_physical_scene_test::{
    physical_scene_test_game, require_physical_scene_pack,
};

fn pixel(p: &Picture, x: i64, y: i64) -> [u8; 4] {
    let at = ((y * p.width + x) * 4) as usize;
    [p.pix[at], p.pix[at + 1], p.pix[at + 2], p.pix[at + 3]]
}

/// Go 測試的逐像素比對：640×400 每一點等於 `palette[pixels[(y/2)*320+x/2]]` 且不透明。回傳第一個不同的點。
fn first_indexed_2x_difference(actual: &Picture, pixels: &[u8], palette: &[ColorRgba]) -> Option<(i64, i64)> {
    assert_eq!((actual.width, actual.height), (640, 400));
    for y in 0..400 {
        for x in 0..640 {
            let c = palette[pixels[((y / 2) * 320 + x / 2) as usize] as usize];
            if pixel(actual, x, y) != [c.r, c.g, c.b, 255] {
                return Some((x, y));
            }
        }
    }
    None
}

/// 256 色、每色不同、不透明的測試調色盤。
fn test_palette() -> ColorPalette {
    (0..256).map(|i| ColorRgba { r: i as u8, g: 255 - i as u8, b: (i / 2) as u8, a: 0xff }).collect()
}

fn pattern(seed: u8) -> Vec<u8> {
    (0..64000).map(|i| ((i * 7 + seed as usize) % 251) as u8).collect()
}

/// Go：`native_physical_scene_test.go:723` `TestNativePhysicalPreludeGPU`。
///
/// Go 以 `ebiten.RunGame` 每一格先 `Update`（`stepAttackPresentationTick`）再 `Draw`（`drawBattleScene`）；這裡照同一順序
/// 迴圈。第 8 幀時 Go 另外驗 `drawNativePhysicalBase`（狀態欄合成與快取），那個函式是 r8（開場與第一章沒有走到），
/// 這一段沒有移植。
#[test]
fn test_native_physical_prelude_gpu() {
    if !require_physical_scene_pack() {
        return;
    }
    let Some(mut g) = physical_scene_test_game() else {
        return;
    };
    let scene = g.prepare_native_physical_scene(Some(0), 1).unwrap().expect("有原生視圖時要建場景");
    g.atk = Some(AtkAnim { native_scene: Some(scene), timer: 100, total: 100, ..Default::default() });
    for _ in 0..10 {
        g.step_attack_presentation_tick().expect("step");
    }
    {
        let a = g.atk.as_ref().unwrap();
        assert!(a.native_scene.as_ref().unwrap().prelude_frame == 0 && a.timer == 100, "undrawn prelude advanced");
    }
    let mut seen = [false; 9];
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..64 {
        if seen.iter().all(|s| *s) {
            break;
        }
        g.step_attack_presentation_tick().expect("step");
        // Draw
        let scene = g.atk.as_ref().unwrap().native_scene.as_ref().unwrap();
        assert!(scene.prelude_frame < 9, "prelude completed without nine GPU acknowledgements");
        let i = scene.prelude_frame as usize;
        let frame = scene.prelude[i].clone();
        g.draw_battle_scene(&mut screen);
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&frame.dac).expect("DAC"));
        if let Some((x, y)) = first_indexed_2x_difference(&screen, &frame.pixels, &palette) {
            panic!("GPU prelude stage{} differs at {x},{y}", frame.stage);
        }
        seen[i] = true;
    }
    for (i, seen) in seen.iter().enumerate() {
        assert!(*seen, "GPU prelude stage{} was not presented", 8 - i);
    }
    assert_eq!(g.atk.as_ref().unwrap().timer, 100, "attack timeline advanced before the last prelude acknowledgement");
    let a = g.atk.as_mut().unwrap();
    a.native_scene.as_mut().unwrap().prelude_frame = 9;
    a.counter = Some(AtkStage { total: 20, body_ticks: 16, ..Default::default() });
    a.begin_counter_stage();
    let scene = a.native_scene.as_ref().unwrap();
    assert!(scene.counter_active && scene.prelude_frame == 9, "counterattack restarted the entry prelude");
}

/// 物理本體的測試場景：三個工作（有像素且 palette0 半透明、cue-only、有像素），沒有 HD 來源。
fn body_scene() -> NativePhysicalScene {
    let mut a = NativePhysicalBodyJob { pixels: pattern(3), ..Default::default() };
    a.palette0 = Some(ColorNrgba { r: 200, g: 100, b: 50, a: 128 });
    let cue = NativePhysicalBodyJob { cue: 2, ..Default::default() };
    let b = NativePhysicalBodyJob { pixels: pattern(9), ..Default::default() };
    NativePhysicalScene {
        body: Some(NativePhysicalBodyPlayback { jobs: vec![a, cue, b], ..Default::default() }),
        ..Default::default()
    }
}

/// Rust 新增：`drawNativePhysicalBody` 的 2 倍貼上、palette0（Go `color.NRGBA` 經 `NewImageFromImage` 預乘）、
/// 快取畫面重用、cue-only 不畫不設 `drawn`、caller 等待時取 `lastPresent`、沒有目前畫面時回錯誤。
#[test]
fn draw_native_physical_body_scales_caches_and_fails_closed() {
    let mut g = Game { native_ui_palette: test_palette(), ..Default::default() };
    let mut scene = body_scene();
    let mut screen = Picture::new_image(640, 400);
    g.draw_native_physical_body(&mut screen, &mut scene).expect("job0");
    let mut palette = test_palette();
    palette[0] = ColorNrgba { r: 200, g: 100, b: 50, a: 128 }.premultiplied();
    assert_eq!(palette[0], ColorRgba { r: 100, g: 50, b: 25, a: 128 });
    let job0 = pattern(3);
    // 索引 0 是半透明：畫到全透明畫布上 source-over 等於來源本身；其餘不透明。
    for y in 0..400 {
        for x in 0..640 {
            let c = palette[job0[((y / 2) * 320 + x / 2) as usize] as usize];
            assert_eq!(pixel(&screen, x, y), [c.r, c.g, c.b, c.a], "({x},{y})");
        }
    }
    let p = scene.body.as_ref().unwrap();
    assert!(p.drawn && p.image.is_some());
    let cached = p.image.clone().unwrap();
    // 同一個工作再畫一次：沿用快取（Go `p.image != nil` 不重建）。
    screen.clear();
    g.draw_native_physical_body(&mut screen, &mut scene).expect("job0 again");
    assert!(Arc::ptr_eq(&cached, scene.body.as_ref().unwrap().image.as_ref().unwrap()));
    assert_eq!(pixel(&screen, 0, 0), {
        let c = palette[job0[0] as usize];
        [c.r, c.g, c.b, c.a]
    });
    // cue-only：沒有像素，不畫、不設 drawn。
    {
        let p = scene.body.as_mut().unwrap();
        (p.index, p.image, p.drawn) = (1, None, false);
    }
    screen.clear();
    g.draw_native_physical_body(&mut screen, &mut scene).expect("cue-only");
    let p = scene.body.as_ref().unwrap();
    assert!(!p.drawn && p.image.is_none() && screen.pix.iter().all(|&v| v == 0), "cue-only 項目不能有畫面");
    // caller 等待：index 已到結尾、還有等待時畫 lastPresent。
    {
        let p = scene.body.as_mut().unwrap();
        (p.index, p.last_present, p.return_wait_millis) = (3, 2, 5.0);
    }
    g.draw_native_physical_body(&mut screen, &mut scene).expect("return wait");
    assert_eq!(first_indexed_2x_difference(&screen, &pattern(9), &test_palette()), None);
    assert!(scene.body.as_ref().unwrap().drawn);
    // 沒有等待時 index == len 是錯誤；沒有 body 也是錯誤。
    {
        let p = scene.body.as_mut().unwrap();
        (p.return_wait_millis, p.image) = (0.0, None);
    }
    let err = g.draw_native_physical_body(&mut screen, &mut scene).unwrap_err();
    assert_eq!(err.to_string(), "native physical body has no current present");
    scene.body = None;
    assert!(g.draw_native_physical_body(&mut screen, &mut scene).is_err());
}

/// Rust 新增：`Game::draw` 的攻擊分支經 `drawBattleScene` → `drawNativePhysicalPrelude`：前奏幀 2 倍貼上並設
/// `preludeDrawn`；前奏結束後畫本體並設 body 的 `drawn`；本體錯誤寫 `loadErr` 仍算畫過（Go 回 true）；
/// 場景在 Draw 之後放回 `g.atk`；沒有原生場景時 `drawNativePhysicalPrelude` 回 false。
#[test]
fn draw_battle_scene_presents_prelude_then_body() {
    let prelude_pix = pattern(5);
    let prelude_palette: ColorPalette =
        (0..256).map(|i| ColorRgba { r: (255 - i) as u8, g: i as u8, b: 7, a: 0xff }).collect();
    let mut scene = body_scene();
    scene.prelude_images =
        vec![IndexedFrame { width: 320, height: 200, pix: prelude_pix.clone(), palette: prelude_palette.clone() }];
    let mut g = Game {
        native_ui_palette: test_palette(),
        m: Some(MapData { w: 1, h: 1, tiles: vec![0], ..Default::default() }),
        atk: Some(AtkAnim { native_scene: Some(scene), ..Default::default() }),
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    g.draw(&mut screen);
    let scene = g.atk.as_ref().unwrap().native_scene.as_ref().expect("Draw 之後場景要放回");
    assert!(scene.prelude_drawn && !scene.body.as_ref().unwrap().drawn);
    assert_eq!(first_indexed_2x_difference(&screen, &prelude_pix, &prelude_palette), None);
    // 前奏結束：畫本體第 2 個工作。
    {
        let scene = g.atk.as_mut().unwrap().native_scene.as_mut().unwrap();
        scene.prelude_frame = 1;
        scene.body.as_mut().unwrap().index = 2;
    }
    g.draw(&mut screen);
    let scene = g.atk.as_ref().unwrap().native_scene.as_ref().unwrap();
    assert!(scene.body.as_ref().unwrap().drawn && g.load_err.is_empty());
    assert_eq!(first_indexed_2x_difference(&screen, &pattern(9), &test_palette()), None);
    // 本體沒有目前畫面：寫 loadErr，Go 仍回 true（不往下畫近似畫面）。
    {
        let p = g.atk.as_mut().unwrap().native_scene.as_mut().unwrap().body.as_mut().unwrap();
        (p.index, p.image) = (3, None);
    }
    g.draw(&mut screen);
    assert_eq!(g.load_err, "native physical body: native physical body has no current present");
    assert!(g.atk.as_ref().unwrap().native_scene.is_some());
    // 沒有原生場景：drawNativePhysicalPrelude 回 false（drawBattleScene 其後是 r8）。
    g.atk.as_mut().unwrap().native_scene = None;
    assert!(!g.draw_native_physical_prelude(&mut screen));
    g.atk = None;
    assert!(!g.draw_native_physical_prelude(&mut screen));
}

/// Rust 新增：`drawNativeCh28PostPresent` 以工作的調色盤貼 `nativeMapVGA`，記 `presentedFrame`／`presentedHD`
/// 與 `drawn`；調色盤不是 256 色、VGA 長度不對或沒有工作時回 false 且不改狀態。
#[test]
fn draw_native_ch28_post_present_uses_job_palette_and_fails_closed() {
    let vga = pattern(11);
    let mut g = Game {
        native_map_vga: vga.clone(),
        native_ch28_post_present: Some(NativeCh28PostPresentJob {
            palette: test_palette(),
            frame: 5,
            presented_frame: -1,
            presented_hd: true,
            ..Default::default()
        }),
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    assert!(g.draw_native_ch28_post_present(&mut screen));
    assert_eq!(first_indexed_2x_difference(&screen, &vga, &test_palette()), None);
    let j = g.native_ch28_post_present.as_ref().unwrap();
    assert!(j.drawn && j.presented_frame == 5 && !j.presented_hd);
    for case in ["palette", "vga", "none"] {
        let mut g = Game {
            native_map_vga: vga.clone(),
            native_ch28_post_present: Some(NativeCh28PostPresentJob { palette: test_palette(), ..Default::default() }),
            ..Default::default()
        };
        match case {
            "palette" => {
                g.native_ch28_post_present.as_mut().unwrap().palette.pop();
            }
            "vga" => g.native_map_vga.truncate(63999),
            _ => g.native_ch28_post_present = None,
        }
        assert!(!g.draw_native_ch28_post_present(&mut screen), "{case}");
        assert!(g.native_ch28_post_present.as_ref().is_none_or(|j| !j.drawn), "{case}");
    }
}

/// Rust 新增：`drawNativeAIIdleRecovery` 用素材調色盤，`nativeMapDAC` 是 768 位元組且可解時改用它；幀索引越界、
/// 素材調色盤不是 256 色或畫格長度不對時回 false 且不設 `drawn`。
#[test]
fn draw_native_ai_idle_recovery_prefers_current_dac_and_fails_closed() {
    let frames = vec![pattern(1), pattern(2), pattern(4)];
    let assets = NativeMapAssets { palette: test_palette(), ..Default::default() };
    let new_game = || Game {
        native_map_assets: Some(assets.clone()),
        native_ai_idle_recovery: Some(NativeAIIdleRecoveryJob {
            frames: frames.clone(),
            frame: 1,
            ..Default::default()
        }),
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    let mut g = new_game();
    assert!(g.draw_native_ai_idle_recovery(&mut screen));
    assert!(g.native_ai_idle_recovery.as_ref().unwrap().drawn);
    assert_eq!(first_indexed_2x_difference(&screen, &frames[1], &test_palette()), None);
    // 目前 DAC：6 位元值 v → (v<<2)|(v>>4)。
    let mut g = new_game();
    g.native_map_dac = (0..768).map(|i| (i % 64) as u8).collect();
    let dac_palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&g.native_map_dac).unwrap());
    assert!(g.draw_native_ai_idle_recovery(&mut screen));
    assert_eq!(first_indexed_2x_difference(&screen, &frames[1], &dac_palette), None);
    // DAC 長度不對時沿用素材調色盤。
    let mut g = new_game();
    g.native_map_dac = vec![9; 767];
    assert!(g.draw_native_ai_idle_recovery(&mut screen));
    assert_eq!(first_indexed_2x_difference(&screen, &frames[1], &test_palette()), None);
    for case in ["frame", "negative", "palette", "assets", "length"] {
        let mut g = new_game();
        let job = g.native_ai_idle_recovery.as_mut().unwrap();
        match case {
            "frame" => job.frame = 3,
            "negative" => job.frame = -1,
            "length" => job.frames[1].truncate(100),
            "palette" => {
                g.native_map_assets.as_mut().unwrap().palette.pop();
            }
            _ => g.native_map_assets = None,
        }
        assert!(!g.draw_native_ai_idle_recovery(&mut screen), "{case}");
        assert!(!g.native_ai_idle_recovery.as_ref().unwrap().drawn, "{case}");
    }
}
