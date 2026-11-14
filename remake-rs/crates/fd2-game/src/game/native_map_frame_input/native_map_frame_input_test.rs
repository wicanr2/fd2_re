//! C 組：原生地圖整幀輸入與素材的測試。Go 來源：`native_map_frame_input_test.go`、`native_map_assets_test.go`。
//! `complete_native_map_frame_fixture` 等夾具是 Go 同名夾具的移植，供其他 C 組測試共用。

use super::*;
use crate::test_support;

/// Go：`native_map_frame_input_test.go:19` `nativeFrameTestSprite`。
pub(crate) fn native_frame_test_sprite(pixel: u8) -> fdicon::Sprite {
    fdicon::Sprite { pixels: vec![pixel; 24 * 24], mask: vec![1; 24 * 24], remap_mask: vec![0; 24 * 24] }
}

/// Go：`native_map_frame_input_test.go:83` `nativeFrameTestBank`。
pub(crate) fn native_frame_test_bank(count: usize, pixel: u8) -> fdicon::Bank {
    fdicon::Bank { sprites: vec![native_frame_test_sprite(pixel); count] }
}

/// Go：`native_map_frame_input_test.go:91` `nativeFrameTestHUDFrame`：width/height 之後逐列寫 run-length 段。
pub(crate) fn native_frame_test_hud_frame(width: i64, height: i64, pixel: u8) -> fdother::Frame {
    let mut raw = Vec::new();
    raw.extend_from_slice(&(width as u16).to_le_bytes());
    raw.extend_from_slice(&(height as u16).to_le_bytes());
    for _ in 0..height {
        let mut remaining = width;
        while remaining > 0 {
            let count = remaining.min(64);
            raw.extend_from_slice(&[(count - 1) as u8, pixel]);
            remaining -= count;
        }
    }
    let mut frame = fdother::Frame::default();
    (frame.width, frame.height, frame.pixels) = (width, height, raw);
    frame
}

/// Go：`native_map_frame_input_test.go:108` `nativeFrameTestHUDFrames`。
pub(crate) fn native_frame_test_hud_frames() -> indexedmap::NativeMapHUDFrames {
    let mut frames = indexedmap::NativeMapHUDFrames {
        panel: native_frame_test_hud_frame(69, 34, 0x5a),
        positive_sign: native_frame_test_hud_frame(6, 7, 0x31),
        negative_sign: native_frame_test_hud_frame(6, 5, 0x42),
        ..Default::default()
    };
    for digit in 0..10 {
        frames.digits[digit] = native_frame_test_hud_frame(6, 8, 0x50 + digit as u8);
        frames.hp_mismatch_digits[digit] = native_frame_test_hud_frame(6, 8, 0x70 + digit as u8);
    }
    frames.digits[1] = native_frame_test_hud_frame(5, 8, 0x51);
    frames.hp_mismatch_digits[1] = native_frame_test_hud_frame(5, 8, 0x71);
    frames.hp_equal_overflow = native_frame_test_hud_frame(18, 8, 0x7a);
    frames.hp_mismatch_overflow = native_frame_test_hud_frame(18, 8, 0x7b);
    frames
}

/// Go：`native_map_frame_input_test.go:122` `completeNativeMapFrameFixture`。
pub(crate) fn complete_native_map_frame_fixture() -> (NativeMapAssets, MapData, battle::State) {
    let controls = vec![0u8; 4];
    let mut assets = NativeMapAssets {
        terrain: Some(native_frame_test_bank(2, 1)),
        range: Some(native_frame_test_bank(20, 2)),
        units: Some(native_frame_test_bank(96, 3)),
        controls: controls.clone(),
        luts: vec![vec![0u8; 256]; 10],
        palette: (0..256).map(|i| ColorRgba { r: i as u8, g: i as u8, b: i as u8, a: 0xff }).collect(),
        palette_dac: vec![0; 256 * 3],
        frames: native_frame_test_hud_frames(),
        ..Default::default()
    };
    for i in 0..(fdother::NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS + 10) as usize {
        assets.command_heal_digits.push(fdother::LMI1Entry { width: 1, height: 1, pixels: vec![i as u8] });
    }
    for i in 0..0x7d {
        assets.fdother6.push(fdother::LMI1Entry { width: 1, height: 1, pixels: vec![i as u8] });
    }
    let field = MapData {
        w: 13,
        h: 8,
        tile_w: 24,
        tile_h: 24,
        tiles: vec![0; 13 * 8],
        native_tile_blit_modes: vec![0xff; 13 * 8],
        native_terrain_control: controls,
        ..Default::default()
    };
    let mut unit = battle::Unit::default();
    (unit.x, unit.y, unit.map_selector_key, unit.has_map_selector_key) = (0, 0, 7, true);
    (unit.native_record_byte5, unit.has_native_record_byte5) = (0, true);
    (unit.battle_fig, unit.has_battle_fig) = (9, true);
    (unit.native_record_race, unit.has_native_record_race) = (3, true);
    (unit.native_record_class, unit.has_native_record_class) = (4, true);
    (unit.hp, unit.native_record_word42, unit.has_native_record_word42) = (5, 10, true);
    (unit.native_record_byte6, unit.has_native_record_byte6) = (0, true);
    let mut state = battle::State::default();
    state.append_native_map_selector_batch(vec![unit]).expect("selector batch");
    assert!(state.materialize_native_map_hud_state(2, 3, 1), "HUD state materialization rejected");
    (state.w, state.h) = (field.w, field.h);
    state.native_tile_blit_modes = field.native_tile_blit_modes.clone();
    state.materialize_native_map_view_state(battle::NativeMapViewState::default()).expect("view state");
    assert!(state.materialize_native_map_range_mode(1), "range mode materialization rejected");
    (assets, field, state)
}

/// Go：`native_map_frame_input_test.go:379` `TestBuildNativeMapFrameInputUsesOnlyRawMaterializedState`
#[test]
fn test_build_native_map_frame_input_uses_only_raw_materialized_state() {
    let (assets, field, mut state) = complete_native_map_frame_fixture();
    assert!(state.advance_native_terrain_phase(3, -1), "terrain phase advance rejected");
    state.native_map_cycle_state = fdicon::NativeMapSpriteCycleState { idle: 2, moving: 3, ..Default::default() };
    assert!(
        state.advance_native_terrain_flip(1) && state.advance_native_unit_pixel_shift(1),
        "binary map timing advance rejected"
    );
    let runtime = NativeMapFrameRuntime::default();
    let got = build_native_map_frame_input(Some(&assets), Some(&field), Some(&state), runtime).expect("frame input");
    let f = &got.frame;
    assert!(
        f.terrain_cycle == 2
            && f.idle_cycle == 2
            && f.moving_cycle == 3
            && f.flip == 1
            && f.pixel_shift == 1
            && f.range_mode == 1
            && std::ptr::eq(f.foreground_bank.unwrap(), assets.terrain.as_ref().unwrap())
            && std::ptr::eq(f.selector_cache.unwrap(), state.native_map_selector_cache.as_ref().unwrap())
            && f.units.len() == 1
            && f.foreground_units.len() == 1
            && std::ptr::eq(got.hud_cache.unwrap(), state.native_map_selector_cache.as_ref().unwrap())
            && got.hud.display_gate_a
            && got.hud.display_gate_b
            && got.hud.anchor_x == 1,
        "frame input cycles={}/{}/{} flip={} shift={} range={} units={} fg={} hud={:?}",
        f.terrain_cycle,
        f.idle_cycle,
        f.moving_cycle,
        f.flip,
        f.pixel_shift,
        f.range_mode,
        f.units.len(),
        f.foreground_units.len(),
        got.hud
    );
}

/// Go：`native_map_frame_input_test.go:404` `TestBuildNativeMapFrameInputRejectsControlDriftAndMissingRawRoster`
#[test]
fn test_build_native_map_frame_input_rejects_control_drift_and_missing_raw_roster() {
    let (assets, mut field, mut state) = complete_native_map_frame_fixture();
    field.native_terrain_control[0] = 1;
    assert!(
        build_native_map_frame_input(Some(&assets), Some(&field), Some(&state), NativeMapFrameRuntime::default())
            .is_err(),
        "accepted editable/native control-table drift"
    );
    field.native_terrain_control[0] = 0;
    state.units[0].has_battle_fig = false;
    // Go 另檢查失敗時回傳的零值輸入沒有單位；Rust 失敗只回錯誤。
    assert!(
        build_native_map_frame_input(Some(&assets), Some(&field), Some(&state), NativeMapFrameRuntime::default())
            .is_err(),
        "partial frame accepted"
    );
}

/// #174 READY：1297D先於1ACF3，HUD使用本次candidate idle，而非預檢舊值。
///
/// Go：`native_map_frame_input_test.go:419` `TestComposeNativeMapHUDUsesAdvancedCycle`
#[test]
#[ignore = "等 B 合併（native_map_hud_input、prepare_native_ch23_steady_stage）"]
fn test_compose_native_map_hud_uses_advanced_cycle() {
    for steady_palette in [true, false] {
        let (mut a, m, mut st) = complete_native_map_frame_fixture();
        st.units[0].on_field = true;
        let units = a.units.as_mut().unwrap();
        units.sprites[7 * 12 + 1] = native_frame_test_sprite(0x33);
        units.sprites[7 * 12 + 2] = native_frame_test_sprite(0x44);
        st.native_map_cycle_state = fdicon::NativeMapSpriteCycleState { idle: 1, moving: 1, last_timer_tick: 0 };
        let mut g = Game { native_map_assets: Some(a), m: Some(m), st: Some(st), ..Default::default() };
        let now = GameInstant(500_000_000_000);
        // Go `g.nativeMapClock.Seed(5, now)`（`r8`）：直接寫入同樣的起點。
        g.native_map_clock = NativeBIOSClock { last: now, remainder: GoDuration(0), elapsed_ticks: 5 };
        let (hud, ok) = g.native_map_hud_input();
        assert!(ok && hud.optional_unit.as_ref().is_some_and(|u| u.raw_state == 1), "HUD前置資料");
        g.compose_native_map_frame_at_with_palette_cycle(now, false, steady_palette).expect("compose");
        let layout = fdicon::native_map_hud_layout_for(1, fdicon::NATIVE_MAP_STRIDE).expect("layout");
        let idle = g.st.as_ref().unwrap().native_map_cycle_state.idle;
        assert!(
            idle == 2 && g.native_map_work[(0x8088 + layout.unit) as usize] == 0x44,
            "steady={steady_palette} idle={idle}，應用本次cycle2／index68"
        );
        assert_eq!(hud.optional_unit.as_ref().unwrap().raw_state, 1, "候選合成污染前置HUD快照");
        let (before_vga, before_cycle, before_clock) =
            (g.native_map_vga.clone(), g.st.as_ref().unwrap().native_map_cycle_state, g.native_map_clock);
        g.st.as_mut().unwrap().units[0].has_native_record_word42 = false;
        let later = now + GoDuration(6 * NATIVE_BIOS_TICK_PERIOD.0);
        assert!(g.compose_native_map_frame_at_with_palette_cycle(later, false, steady_palette).is_err(), "缺raw被接受");
        assert!(
            before_vga == g.native_map_vga
                && g.st.as_ref().unwrap().native_map_cycle_state == before_cycle
                && g.native_map_clock == before_clock,
            "拒收發布了時序或畫面"
        );
    }
}

fn available_assets(map_index: i64) -> NativeMapAssets {
    let mut luts = vec![Vec::new(); 10];
    for lut in luts.iter_mut().skip(1) {
        *lut = vec![0; 256];
    }
    NativeMapAssets {
        map_index,
        terrain: Some(fdicon::Bank::default()),
        range: Some(fdicon::Bank::default()),
        units: Some(fdicon::Bank::default()),
        controls: vec![0],
        luts,
        palette: vec![ColorRgba::default(); 256],
        palette_dac: vec![0; 256 * 3],
        ..Default::default()
    }
}

/// Go：`native_map_assets_test.go:43` `TestNativeMapAssetsRequireRangeOverlayBank`
#[test]
fn test_native_map_assets_require_range_overlay_bank() {
    let mut a = available_assets(0);
    a.range = None;
    assert!(!native_map_assets_available(Some(&a)), "accepted native map bundle without FDOTHER #1 range bank");
    a.range = Some(fdicon::Bank::default());
    assert!(native_map_assets_available(Some(&a)), "complete native map bundle rejected");
    a.luts[8].truncate(255);
    assert!(!native_map_assets_available(Some(&a)), "bundle with malformed later transition LUT accepted");
}

/// Go：`native_map_assets_test.go:65` `TestNativeMapAssetsRequireChapterAuxOnlyForRawMaps28And29`
#[test]
fn test_native_map_assets_require_chapter_aux_only_for_raw_maps28_and29() {
    assert!(
        native_map_assets_available(Some(&available_assets(27))),
        "unrelated map requires chapter auxiliary surface"
    );
    for index in [28, 29] {
        let mut a = available_assets(index);
        assert!(!native_map_assets_available(Some(&a)), "map {index} accepted without FDOTHER #55");
        a.chapter_aux = Some(fdother::NativeChapterAuxSurface { pixels: vec![0; 320 * 200] });
        assert!(native_map_assets_available(Some(&a)), "map {index} rejected complete FDOTHER #55");
    }
}

/// Go：`native_map_assets_test.go:13` `TestLoadNativeMapAssetsUsesOnlySeparatedBanks`
///
/// Go 以 `t.Setenv("FD2_ASSET_PACK", …)` 指向分離素材包，並把 `FD2_FDOTHER` 指到不存在的檔證明沒有讀封存；
/// Rust 的載入端只讀 `separated_asset_path`，測試直接用測試環境的 `FD2_ASSET_PACK`。
#[test]
fn test_load_native_map_assets_uses_only_separated_banks() {
    if test_support::input("FD2_ASSET_PACK", "tilesets/fdshap").is_none() {
        return;
    }
    for (index, sprites, controls) in [(0, 288, 1200), (23, 96, 400), (32, 192, 1200)] {
        let assets = load_native_map_assets(&format!("assets/maps/map{index}"))
            .unwrap_or_else(|e| panic!("map{index}: {e}"))
            .unwrap();
        assert!(
            assets.units.as_ref().is_some_and(|u| u.sprites.len() as i64 == fdicon::SEPARATED_SPRITE_COUNT),
            "map{index} separated FDICON sprites unavailable"
        );
        let terrain = assets.terrain.as_ref().unwrap();
        assert!(
            terrain.sprites.len() == sprites && assets.controls.len() == controls,
            "map{index} separated FDSHAP sprites={} controls={}",
            terrain.sprites.len(),
            assets.controls.len()
        );
    }
    for index in [28, 29] {
        let assets = load_native_map_assets(&format!("assets/maps/map{index}"))
            .unwrap_or_else(|e| panic!("map{index}: {e}"))
            .unwrap();
        assert!(
            assets.chapter_aux.as_ref().is_some_and(|aux| aux.pixels.len() == 320 * 200),
            "map{index} separated chapter auxiliary surface unavailable"
        );
    }
}

/// Rust 新增：`nativeCh12UpWalkFrame` 只在第十二章、地圖 11、玩家自己向上走的那一格成立。
#[test]
fn native_ch12_up_walk_frame_requires_reviewed_owner() {
    let mut g = Game::default();
    assert_eq!(g.native_ch12_up_walk_frame(), (0, 0, false));
    let (assets, _, state) = complete_native_map_frame_fixture();
    g.st = Some(state);
    g.native_map_assets = Some(NativeMapAssets { map_index: 11, ..assets });
    g.walk = Some(WalkAnim {
        u: Some(0),
        path: vec![battle::Cell { x: 3, y: 5 }, battle::Cell { x: 3, y: 4 }],
        tick: 2,
        ..Default::default()
    });
    // 沒有戰役節點 battle_ch12 時不成立。
    assert_eq!(g.native_ch12_up_walk_frame(), (0, 0, false));
}
