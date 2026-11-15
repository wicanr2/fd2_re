//! R5 階段 2 r5m 組（地圖與單位）的測試。Go 來源：`phase_banner_test.go`（逐幀步驟部分；長度部分在
//! `main_d_test.rs`）、`phase_banner_glyph_test.go`、`native_action_overlay_test.go:119`、
//! `modern_story_portrait_test.go:350`、`native_player_movement_test.go:57`、`native_map_frame_input_test.go:179`。
//! 「Rust 新增」的幾項補 Go 沒有直接測的繪圖細節（以 Go 程式碼為準）。

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;
use crate::test_support;

/// Go `TestPhaseBannerScheduleFollowsTheBiosTickContract`（`phase_banner_test.go:13`）的逐幀部分：馬賽克 33 步依序
/// 各出現一次、期間字樣停住，前後各有滑入與滑出，結束後回 -1。
#[test]
fn test_phase_banner_schedule_follows_the_bios_tick_contract_steps() {
    for banner in [PHASE_BANNER_ENEMY_TEXT, PHASE_BANNER_PLAYER_TEXT] {
        let mut g = Game { banner: banner.to_string(), banner_t: phase_banner_frames(banner), ..Default::default() };
        let mut seen = Vec::new();
        let mut last = -1;
        let (mut slide_in_frames, mut slide_out_frames) = (0, 0);
        let mut mosaic_seen = false;
        while g.banner_t > 0 {
            let step = g.phase_banner_step();
            let (offset, visible) = g.phase_banner_slide_offset();
            assert!(visible, "{banner} 橫幅期間字樣不可見");
            if step < 0 {
                // 馬賽克之外的兩段是字樣滑入與滑出，那時畫的是正常地圖。
                if mosaic_seen {
                    slide_out_frames += 1;
                } else {
                    slide_in_frames += 1;
                }
                g.banner_t -= 1;
                continue;
            }
            mosaic_seen = true;
            assert_eq!(offset, 0, "{banner} 馬賽克期間字樣偏移 {offset}，應該停住");
            assert!(step >= last, "{banner} 步驟由 {last} 倒退到 {step}");
            if step != last {
                seen.push(step);
                last = step;
            }
            g.banner_t -= 1;
        }
        assert!(
            slide_in_frames > 0 && slide_out_frames > 0,
            "{banner} 沒有走到滑入（{slide_in_frames}）或滑出（{slide_out_frames}）"
        );
        assert_eq!(seen.len() as i64, indexedmap::PHASE_BANNER_STEPS, "{banner} 走過的步數");
        for (i, &step) in seen.iter().enumerate() {
            assert_eq!(step, i as i64, "{banner} 第 {i} 個出現的步驟");
        }
        assert_eq!(g.phase_banner_step(), -1, "{banner} 橫幅結束後仍回報步驟");
    }
}

/// Go `TestPhaseBannerHoldKeepsTheLastEnterStep`（`phase_banner_test.go:86`）：中段停留期間停在進場最後一步。
#[test]
fn test_phase_banner_hold_keeps_the_last_enter_step() {
    let mut g = Game { banner: PHASE_BANNER_PLAYER_TEXT.to_string(), ..Default::default() };
    let total = phase_banner_frames(PHASE_BANNER_PLAYER_TEXT);
    let enter_millis = indexedmap::PHASE_BANNER_ENTER_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS;
    let hold = phase_banner_hold_millis(PHASE_BANNER_PLAYER_TEXT);
    let slide_in = indexedmap::PHASE_BANNER_SLIDE_IN_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS;
    for at in [enter_millis, enter_millis + hold / 2.0] {
        g.banner_t = total - ((slide_in + at) * 60.0 / 1000.0) as i64;
        assert_eq!(g.phase_banner_step(), indexedmap::PHASE_BANNER_ENTER_STEPS - 1, "停留 {at:.1} 毫秒時的步驟");
    }
    // 幀是離散的（60 Hz ＝ 16.7 毫秒），所以要往後推一整幀才算離開停留段。
    g.banner_t = total - (((slide_in + enter_millis + hold) * 60.0 / 1000.0).ceil() as i64 + 1);
    assert!(g.phase_banner_step() >= indexedmap::PHASE_BANNER_ENTER_STEPS, "停留結束後應進退場段");
}

/// Go `TestPhaseBannerGlyphIndexesPointAtTheOriginalCells`（`phase_banner_glyph_test.go:14`）。
#[test]
fn test_phase_banner_glyph_indexes_point_at_the_original_cells() {
    let Some(root) = test_support::input("FD2_ASSET_PACK", "ui/fdother_005_lmi1_opaque") else {
        return;
    };
    let bank = fdother::load_separated_lmi1_bank(root, fdother::NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE).unwrap();
    for (name, index, width, height) in [
        ("PLAYER", PHASE_BANNER_GLYPH_PLAYER, 75, 14),
        ("PHASE", PHASE_BANNER_GLYPH_PHASE, 63, 14),
        ("ENEMY", PHASE_BANNER_GLYPH_ENEMY, 72, 14),
    ] {
        assert!((index as usize) < bank.len(), "{name} 的 index 0x{index:02X} 超出 bank（{} 格）", bank.len());
        let cell = &bank[index as usize];
        assert_eq!((cell.width, cell.height), (width, height), "{name}（0x{index:02X}）的尺寸");
        // 索引 0 是透明，字身落在 DAC 暗化不動的 0x00..0x0F。整格都不透明或整格都透明都代表抓錯格。
        let clear = cell.pixels.iter().filter(|&&v| v == 0).count();
        let opaque = cell.pixels.len() - clear;
        assert!(clear > 0 && opaque > 0, "{name} 的透明像素 {clear}、不透明 {opaque}，不像字樣");
    }
}

/// Go `phaseBannerGlyphFixture`（`phase_banner_glyph_test.go:56`）：整格填同一個非零值的假字樣。
fn phase_banner_glyph_fixture(banner: &str) -> (Game, Vec<u8>) {
    let mut bank =
        vec![fdother::LMI1Entry { width: 1, height: 1, pixels: vec![1] }; PHASE_BANNER_GLYPH_ENEMY as usize + 1];
    let mut fill = |index: i64, w: i64, h: i64, v: u8| {
        bank[index as usize] = fdother::LMI1Entry { width: w, height: h, pixels: vec![v; (w * h) as usize] };
    };
    fill(PHASE_BANNER_GLYPH_PLAYER, 75, 14, 0x21);
    fill(PHASE_BANNER_GLYPH_ENEMY, 72, 14, 0x21);
    fill(PHASE_BANNER_GLYPH_PHASE, 63, 14, 0x22);
    let g = Game {
        banner: banner.to_string(),
        native_map_assets: Some(NativeMapAssets { command_heal_digits: bank, ..Default::default() }),
        ..Default::default()
    };
    (g, vec![0; 320 * 200])
}

/// Go `spanOf`（`phase_banner_glyph_test.go:77`）。
fn span_of(dst: &[u8], y: i64, v: u8) -> (i64, i64) {
    let (mut first, mut last) = (-1, -1);
    for x in 0..320 {
        if dst[(y * 320 + x) as usize] == v {
            if first < 0 {
                first = x;
            }
            last = x;
        }
    }
    (first, last)
}

/// Go `TestPhaseBannerGlyphsLandOnTheOriginalCoordinates`（`phase_banner_glyph_test.go:93`）。
#[test]
fn test_phase_banner_glyphs_land_on_the_original_coordinates() {
    for banner in [PHASE_BANNER_ENEMY_TEXT, PHASE_BANNER_PLAYER_TEXT] {
        let (g, mut dst) = phase_banner_glyph_fixture(banner);
        assert!(g.blit_phase_banner_glyphs(&mut dst, 0), "{banner} 沒有畫成");
        let want_last = PHASE_BANNER_GLYPH_FIRST_X + if banner == PHASE_BANNER_PLAYER_TEXT { 74 } else { 71 };
        assert_eq!(
            span_of(&dst, PHASE_BANNER_GLYPH_Y, 0x21),
            (PHASE_BANNER_GLYPH_FIRST_X, want_last),
            "{banner} 第一塊"
        );
        assert_eq!(
            span_of(&dst, PHASE_BANNER_GLYPH_Y, 0x22),
            (PHASE_BANNER_GLYPH_SECOND_X, PHASE_BANNER_GLYPH_SECOND_X + 62),
            "{banner} 第二塊"
        );
        // 上下界：14 列高，超出的列必須乾淨。
        assert!(span_of(&dst, PHASE_BANNER_GLYPH_Y - 1, 0x21).0 < 0, "{banner} 畫到了字樣上緣之上");
        assert!(span_of(&dst, PHASE_BANNER_GLYPH_Y + 14, 0x21).0 < 0, "{banner} 畫到了字樣下緣之下");
    }
}

/// Go `TestPhaseBannerGlyphsSlideTowardEachOther`（`phase_banner_glyph_test.go:126`）。
#[test]
fn test_phase_banner_glyphs_slide_toward_each_other() {
    let (mut previous_first, mut previous_second) = (-1, 320);
    for step in 0..5 {
        // 參數 100、75、50、25、0 這五步是單調的
        let (g, mut dst) = phase_banner_glyph_fixture(PHASE_BANNER_ENEMY_TEXT);
        let offset = indexedmap::phase_banner_slide_in_offset(step);
        assert!(g.blit_phase_banner_glyphs(&mut dst, offset), "第 {step} 步沒有畫成");
        let (first, _) = span_of(&dst, PHASE_BANNER_GLYPH_Y, 0x21);
        let (_, second_last) = span_of(&dst, PHASE_BANNER_GLYPH_Y, 0x22);
        assert!(
            first >= 0 && second_last >= 0,
            "第 {step} 步有一塊完全在畫面外：first={first} secondLast={second_last}"
        );
        assert!(first > previous_first, "第 {step} 步第一塊左緣 {first} 沒有往右（前一步 {previous_first}）");
        assert!(
            second_last < previous_second,
            "第 {step} 步第二塊右緣 {second_last} 沒有往左（前一步 {previous_second}）"
        );
        (previous_first, previous_second) = (first, second_last);
    }
    assert_eq!(previous_first, PHASE_BANNER_GLYPH_FIRST_X, "滑入結束時第一塊的位置");
    assert_eq!(previous_second, PHASE_BANNER_GLYPH_SECOND_X + 62, "滑入結束時第二塊右緣的位置");
}

/// Go `TestPhaseBannerFallsBackWithoutTheOriginalBank`（`phase_banner_glyph_test.go:161`）。
#[test]
fn test_phase_banner_falls_back_without_the_original_bank() {
    let (mut g, mut dst) = phase_banner_glyph_fixture(PHASE_BANNER_ENEMY_TEXT);
    g.native_map_assets.as_mut().unwrap().command_heal_digits.truncate(PHASE_BANNER_GLYPH_ENEMY as usize);
    assert!(!g.phase_banner_glyphs_available(), "bank 少一格時仍宣稱有原版字樣");
    assert!(!g.blit_phase_banner_glyphs(&mut dst, 0), "bank 少一格時仍宣稱畫成了");
    assert!(dst.iter().all(|&v| v == 0), "失敗即關閉之後畫面仍被動過");
    g.native_map_assets = None;
    assert!(
        !g.phase_banner_glyphs_available() && !g.blit_phase_banner_glyphs(&mut dst, 0),
        "沒有資產時仍宣稱畫得出原版字樣"
    );
}

/// Go `TestPhaseBannerGlyphsStayInsideTheCopiedViewport`（`phase_banner_glyph_test.go:186`）。
#[test]
fn test_phase_banner_glyphs_stay_inside_the_copied_viewport() {
    let (g, mut dst) = phase_banner_glyph_fixture(PHASE_BANNER_ENEMY_TEXT);
    assert!(g.blit_phase_banner_glyphs(&mut dst, indexedmap::phase_banner_slide_in_offset(0)), "滑入第一步沒有畫成");
    for y in 0..200 {
        for x in 0..320 {
            let inside = (indexedmap::NATIVE_MAP_VIEWPORT_X0..=indexedmap::NATIVE_MAP_VIEWPORT_X1).contains(&x)
                && (indexedmap::NATIVE_MAP_VIEWPORT_Y0..=indexedmap::NATIVE_MAP_VIEWPORT_Y1).contains(&y);
            assert!(inside || dst[(y * 320 + x) as usize] == 0, "窗格外的 ({x},{y}) 被畫到了");
        }
    }
    // 兩塊都要真的碰到窗格邊界，否則上面那圈檢查是空話。
    let row = PHASE_BANNER_GLYPH_Y * 320;
    assert_ne!(dst[(row + indexedmap::NATIVE_MAP_VIEWPORT_X0) as usize], 0, "第一塊沒有畫到窗格左緣");
    assert_ne!(dst[(row + indexedmap::NATIVE_MAP_VIEWPORT_X1) as usize], 0, "第二塊沒有畫到窗格右緣");
}

/// Go `TestNativeMapFrameAdmissionKeepsActionOverlayOnCompleteFrame`（`native_action_overlay_test.go:119`）。
/// Go 的 `sel: &battle.Unit{}` 在 Rust 是任一 `Some(索引)`（守門只看有沒有選中）。
#[test]
fn test_native_map_frame_admission_keeps_action_overlay_on_complete_frame() {
    assert!(
        Game { ring: true, ..Default::default() }.native_map_frame_admission(false, true),
        "action overlay was excluded"
    );
    assert!(
        Game { native_command_open: true, ..Default::default() }.native_map_frame_admission(false, true),
        "native command grid was excluded"
    );
    assert!(
        Game { sel: Some(0), native_item_targeting: true, ..Default::default() }
            .native_map_frame_admission(false, true),
        "item target modal was excluded"
    );
    assert!(
        Game { sel: Some(0), native_item_relocating: true, ..Default::default() }
            .native_map_frame_admission(false, true),
        "item relocation modal was excluded"
    );
    assert!(
        !Game { sel: Some(0), ..Default::default() }.native_map_frame_admission(false, true),
        "ordinary selected-unit movement was admitted without a complete modal state"
    );
    for state in [
        Game { ring: true, spell_open: true, ..Default::default() },
        Game { native_command_open: true, item_open: true, ..Default::default() },
        Game { native_command0_targeting: true, cast_sp: Some(battle::Spell::default()), ..Default::default() },
    ] {
        assert!(!state.native_map_frame_admission(false, true), "incompatible modal state was admitted");
    }
    assert!(
        !Game { ring: true, ..Default::default() }.native_map_frame_admission(true, true),
        "legacy viewport was admitted"
    );
}

/// Go `TestModernThemeRejectsNativeFullFrameCover`（`modern_story_portrait_test.go:350`）。
#[test]
fn test_modern_theme_rejects_native_full_frame_cover() {
    let g = Game { modern_story_portraits: Some(ModernStoryPortraitSet::default()), ..Default::default() };
    assert!(!g.native_map_frame_admission(false, true), "modern theme admitted an original indexed full frame");
}

/// Go `TestNativePlayerMovementAdmitsOnlyPreparedNativePreview`（`native_player_movement_test.go:57`）。
#[test]
fn test_native_player_movement_admits_only_prepared_native_preview() {
    let mut g = Game { sel: Some(0), ..Default::default() };
    assert!(!g.native_map_frame_admission(false, true), "unprepared selection was admitted");
    g.native_move_plan = Some(battle::NativePlayerMovement::default());
    assert!(g.native_map_frame_admission(false, true), "prepared movement lost native viewport");
}

/// Go `TestDrawNativeMapFramePresentsCorrectedVGABorder`（`native_map_frame_input_test.go:179`）。
#[test]
fn test_draw_native_map_frame_presents_corrected_vga_border() {
    let (assets, field, state) = complete_native_map_frame_fixture();
    // Go 的 gameNow 讀 time.Now()（永遠不是零值）；Rust 讀虛擬時鐘，起點照 A 組慣例設成非零。
    let mut g = Game {
        native_map_assets: Some(assets),
        m: Some(field),
        st: Some(state),
        virtual_now: VIRTUAL_CLOCK_ORIGIN,
        ..Default::default()
    };
    let mut screen = Picture::new_image(640, 400);
    assert!(g.draw_native_map_frame(&mut screen), "complete materialized native map frame was not presented");
    assert_eq!(g.native_map_vga.len() as i64, indexedmap::NATIVE_MAP_VGA_SIZE, "VGA bytes");
    assert!(g.native_map_vga[0] == 0 && g.native_map_vga[4 * 320 + 3] == 0, "native four-pixel border was overwritten");
    assert_eq!(g.native_map_vga[4 * 320 + 4], 3, "first viewport pixel, want unit layer 3");
    // Rust 新增：畫面是索引畫格的 2 倍最近鄰放大，旗標照 Go 設定。
    assert!(g.native_map_presented && !g.native_interface_hud_visible);
    // 調色盤是合成後的 nativeMapDAC（nativeMapFrameImage 的第二次呼叫與 Draw 同一狀態，結果相同）。
    let (img, ok) = g.native_map_frame_image();
    let img = img.unwrap();
    assert!(ok && img.pix == g.native_map_vga);
    for (x, y) in [(0, 0), (8, 8), (9, 9), (639, 399)] {
        let c = img.palette[img.pix[(y / 2) * 320 + x / 2] as usize];
        let at = (y * 640 + x) * 4;
        assert_eq!(&screen.pix[at..at + 4], &[c.r, c.g, c.b, c.a], "({x},{y})");
    }
}

/// Rust 新增：`drawNativeSpawnIntro`（`native_spawn_intro.go:360`）的 2 倍貼上、`drawn` 與兩個失敗條件。
#[test]
fn draw_native_spawn_intro_scales_and_marks_drawn() {
    let palette: ColorPalette = (0..256).map(|i| ColorRgba { r: i as u8, g: 1, b: 2, a: 0xff }).collect();
    let vga: Vec<u8> = (0..320 * 200).map(|i| (i % 251) as u8).collect();
    let job = NativeSpawnIntroJob { vga: vga.clone(), palette: palette.clone(), ..Default::default() };
    let mut g = Game { spawn_intro_transition: Some(job), ..Default::default() };
    let mut screen = Picture::new_image(640, 400);
    assert!(g.draw_native_spawn_intro(&mut screen));
    assert!(g.spawn_intro_transition.as_ref().unwrap().drawn);
    for (x, y) in [(0usize, 0usize), (1, 1), (2, 0), (639, 399), (321, 17)] {
        let c = palette[vga[(y / 2) * 320 + x / 2] as usize];
        let at = (y * 640 + x) * 4;
        assert_eq!(&screen.pix[at..at + 4], &[c.r, c.g, c.b, c.a], "({x},{y})");
    }
    let mut short = Game {
        spawn_intro_transition: Some(NativeSpawnIntroJob { vga: vec![0; 10], palette, ..Default::default() }),
        ..Default::default()
    };
    assert!(!short.draw_native_spawn_intro(&mut screen) && !short.spawn_intro_transition.as_ref().unwrap().drawn);
    let mut no_palette =
        Game { spawn_intro_transition: Some(NativeSpawnIntroJob { vga, ..Default::default() }), ..Default::default() };
    assert!(!no_palette.draw_native_spawn_intro(&mut screen));
    assert!(!Game::default().draw_native_spawn_intro(&mut screen));
}

/// Rust 新增：`tileAt`（`main.go:8363`）的邊界與 nil 圖塊。
#[test]
fn tile_at_bounds_and_missing_tiles() {
    let tile = Arc::new(Picture::new_image(1, 1));
    let g = Game { tiles: vec![Some(tile.clone()), None], ..Default::default() };
    assert!(g.tile_at(0).is_some_and(|t| Arc::ptr_eq(&t, &tile)));
    assert!(g.tile_at(1).is_none() && g.tile_at(2).is_none() && g.tile_at(-1).is_none());
}

/// Rust 新增：`drawUnitSprite`（`main.go:11528`）的分鏡選擇、位移與已行動變暗。期望值照畫布的 `ColorScale`
/// 實測公式（`canvas.rs` 的 `u2f`／`f2u`，121「Ebiten 取整」）逐通道算。
#[test]
fn draw_unit_sprite_selects_frame_offsets_and_darkens_acted() {
    let color = |i: usize| ColorRgba { r: 100 + i as u8, g: 30 + i as u8, b: 200, a: 0xff };
    let frames = |n: usize| {
        (0..n)
            .map(|i| {
                Some(Arc::new(Picture {
                    width: 1,
                    height: 1,
                    pix: {
                        let c = color(i);
                        vec![c.r, c.g, c.b, c.a]
                    },
                }))
            })
            .collect::<Vec<_>>()
    };
    let g = Game { sprites: BTreeMap::from([(7, frames(12)), (8, frames(3))]), frame: 16, ..Default::default() };
    let pixel = |screen: &Picture, x: usize, y: usize| screen.pix[(y * 4 + x) * 4..(y * 4 + x) * 4 + 4].to_vec();
    // frame 16 → 走動相位 (16/8)%3 = 2；dir 1 → 1×3+2 = 5。位置 (x+OffX, y+OffY-4)。
    let mut u = battle::Unit::default();
    (u.dir, u.off_x, u.off_y) = (1, 1.0, 2.0);
    let mut screen = Picture::new_image(4, 4);
    g.draw_unit_sprite(&mut screen, 1.0, 3.0, 24.0, 24.0, &u, 7);
    let c = color(5);
    assert_eq!(pixel(&screen, 2, 1), vec![c.r, c.g, c.b, c.a]);
    // 不足 12 幀：idx 超出時退回 f % len（2 % 3 = 2）。
    let mut screen = Picture::new_image(4, 4);
    g.draw_unit_sprite(&mut screen, 1.0, 3.0, 24.0, 24.0, &u, 8);
    let c = color(2);
    assert_eq!(pixel(&screen, 2, 1), vec![c.r, c.g, c.b, c.a]);
    // 已行動：ColorScale.Scale(0.55, 0.55, 0.6, 1)。
    u.acted = true;
    let mut screen = Picture::new_image(4, 4);
    screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
    g.draw_unit_sprite(&mut screen, 1.0, 3.0, 24.0, 24.0, &u, 7);
    let c = color(5);
    let k = |v: u8, s: f32| f2u(u2f(v) * s);
    assert_eq!(pixel(&screen, 2, 1), vec![k(c.r, 0.55), k(c.g, 0.55), k(c.b, 0.6), 0xff]);
    assert_eq!(pixel(&screen, 1, 1), vec![0, 0, 0, 0xff], "只畫 1×1");
}

/// Rust 新增：`nativeMapFrameImage`（`main.go:12469`）在橫幅期間第一次畫時凍結整幀，之後的馬賽克只取那張快照。
#[test]
fn native_map_frame_image_freezes_banner_snapshot() {
    let palette: ColorPalette = (0..256).map(|i| ColorRgba { r: i as u8, g: 0, b: 0, a: 0xff }).collect();
    let mut g = Game {
        native_map_assets: Some(NativeMapAssets { palette, ..Default::default() }),
        native_map_vga: vec![5; 320 * 200],
        banner: PHASE_BANNER_ENEMY_TEXT.to_string(),
        banner_t: phase_banner_frames(PHASE_BANNER_ENEMY_TEXT),
        ..Default::default()
    };
    assert_eq!(g.phase_banner_step(), -1, "第一幀還在滑入");
    let (img, ok) = g.native_map_frame_image();
    assert!(ok && img.unwrap().pix.iter().all(|&v| v == 5));
    assert_eq!(g.banner_frame, vec![5; 320 * 200], "第一次繪製時凍結");
    // 地圖之後改了，馬賽克期間整幀都從快照取（`MosaicNativeMapViewport` 先整幀複製 src，再把窗格內馬賽克化）。
    g.native_map_vga = vec![7; 320 * 200];
    while g.phase_banner_step() < 0 {
        g.banner_t -= 1;
    }
    let (img, ok) = g.native_map_frame_image();
    let img = img.unwrap();
    assert!(ok);
    assert!(img.pix.iter().all(|&v| v == 5), "馬賽克只取凍結的快照");
    // 滑出段（step = -1、字樣仍可見）畫的是當前畫面。
    while g.phase_banner_step() >= 0 {
        g.banner_t -= 1;
    }
    let (img, ok) = g.native_map_frame_image();
    assert!(ok && g.banner_t > 0 && img.unwrap().pix.iter().all(|&v| v == 7), "滑出段不取快照");
    // 沒有資產或畫面太短時失敗。
    assert!(!Game { native_map_vga: vec![0; 320 * 200], ..Default::default() }.native_map_frame_image().1);
    let mut short =
        Game { native_map_assets: g.native_map_assets.clone(), native_map_vga: vec![0; 10], ..Default::default() };
    assert!(!short.native_map_frame_image().1);
}
