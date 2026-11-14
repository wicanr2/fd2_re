//! 移植自 Go `indexedmap/frame_test.go`。

use super::*;
use crate::indexedmap::test_util::{bank, hud_frames, identity, solid, transparent_bank};
use crate::indexedmap::{NativeMapHUDOptionalUnit, Result};
use fd2_formats::fdicon::{
    self, NativeForegroundLayerEntry, NativeSelectorCache, NativeTerrainCell, NativeUnitLayerEntry,
};
use fd2_formats::fdother::{self, LMI1Entry, NativeChapterAuxSurface, NativeMapParallaxSurface};

const WS: usize = WORK_STRIDE as usize;
const WB: usize = WORK_BASE as usize;
const VW: usize = VIEW_WIDTH as usize;
const SVO: usize = STEADY_VIEWPORT_OFFSET as usize;
const SVW: usize = STEADY_VIEWPORT_WIDTH as usize;
const WORK_SIZE: usize = NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
const VGA_SIZE: usize = NATIVE_MAP_VGA_SIZE as usize;

fn no_hud(_: &mut [u8]) -> Result<()> {
    Ok(())
}

fn cache0() -> NativeSelectorCache {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    cache
}

fn cells_ff(n: usize) -> Vec<NativeTerrainCell> {
    vec![NativeTerrainCell { tile: 0, blit_mode: 0xff }; n]
}

#[test]
fn test_seed_native_ch23_staging_uses_zero_offset_and456_stride() {
    let mut work = vec![0u8; WORK_SIZE];
    let mut staging = vec![0u8; 312 * 192];
    for row in 0..192 {
        for col in 0..312 {
            staging[row * 312 + col] = (row + col) as u8;
        }
    }
    seed_native_ch23_staging(&mut work, &staging).unwrap();
    for row in [0, 1, 191] {
        for col in [0, 17, 311] {
            let (got, want) = (work[WB + row * WS + col], staging[row * 312 + col]);
            assert_eq!(got, want, "row={row} col={col}");
        }
    }
    assert_eq!(work[WB + 312], 0, "ch23 staging overwrote the 456-stride row padding");
    let mut short = vec![0u8; WORK_SIZE - 1];
    let before = short.clone();
    assert!(
        seed_native_ch23_staging(&mut short, &staging).is_err() && short == before,
        "invalid ch23 work buffer was accepted or partially mutated"
    );
}

#[test]
fn test_native_parallax_survives_transparent_terrain_across_frame_paths() {
    let cache = NativeSelectorCache::default();
    let surface = NativeMapParallaxSurface { pixels: vec![73; 462 * 226] };
    let valid_transparent = |count: usize| {
        let mut b = bank(count, 0);
        for s in &mut b.sprites {
            s.mask.fill(0);
        }
        b
    };
    let (terrain, range, unit, fg) =
        (valid_transparent(12), valid_transparent(20), valid_transparent(12), valid_transparent(12));
    let controls = [0u8, 0, 0, 0];
    let lut = vec![0u8; 256];
    let mut in_ = FrameInput {
        terrain_bank: Some(&terrain),
        range_bank: Some(&range),
        unit_bank: Some(&unit),
        foreground_bank: Some(&fg),
        selector_cache: Some(&cache),
        cells: vec![NativeTerrainCell::default(); 13 * 8],
        controls: &controls,
        lut: &lut,
        map_width: 13,
        parallax: Some(&surface),
        ..Default::default()
    };
    let (mut work, mut vga) = (vec![0u8; WORK_SIZE], vec![0u8; VGA_SIZE]);
    compose_frame(&mut work, &mut vga, &mut in_, Some(&mut no_hud)).unwrap();
    assert_eq!(vga[SVO], 73, "正常幀透明地形未保留山景");
    compose_native_step_frame(&mut work, &mut vga, &in_).unwrap();
    assert_eq!(vga[SVO], 73, "走行幀未保留山景");
    let transition = NativeTransitionFrameInput {
        terrain_bank: in_.terrain_bank,
        unit_bank: in_.unit_bank,
        foreground_bank: in_.foreground_bank,
        selector_cache: Some(&cache),
        cells: in_.cells.clone(),
        controls: in_.controls,
        terrain_lut: in_.lut,
        map_width: 13,
        parallax: Some(&surface),
        ..Default::default()
    };
    let ident = identity();
    let pass = fdother::build_native_indexed_transition_pass(6, 6, 10, 0, 192).unwrap();
    compose_native_transition_frame(&mut work, &mut vga, &transition, pass, &ident).unwrap();
    assert_eq!(vga[0], 73, "轉場幀未保留山景");
    assert!(compose_frame(&mut vec![0u8; WS], &mut vga, &mut in_, Some(&mut no_hud)).is_err(), "缺viewport緩衝卻成功");
    let (before_work, before_vga) = (work.clone(), vga.clone());
    // Go 改寫共用的 `in.Parallax.Pixels`；Rust 以截短的新表面代替。
    let short = NativeMapParallaxSurface { pixels: surface.pixels[..1].to_vec() };
    in_.parallax = Some(&short);
    assert!(
        compose_native_step_frame(&mut work, &mut vga, &in_).is_err() && work == before_work && vga == before_vga,
        "壞山景輸入未原子拒收"
    );
}

#[test]
fn test_compose_frame_seeds_chapter_aux_before_terrain_atomically() {
    let cache = cache0();
    let mut pixels = vec![0u8; 320 * 200];
    for y in 0..200 {
        for x in 0..320 {
            pixels[y * 320 + x] = x as u8;
        }
    }
    let (mut work, mut vga) = (vec![0u8; WS * 300], vec![0u8; VGA_SIZE]);
    let before_work = work.clone();
    let (t, r, u, f) = (transparent_bank(12), transparent_bank(20), transparent_bank(12), transparent_bank(12));
    let aux = NativeChapterAuxSurface { pixels };
    let controls = [0u8; 4];
    let lut = vec![0u8; 256];
    let mut in_ = FrameInput {
        terrain_bank: Some(&t),
        range_bank: Some(&r),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        cells: vec![NativeTerrainCell::default(); 13 * 8],
        controls: &controls,
        lut: &lut,
        map_width: 13,
        chapter_aux: Some(&aux),
        chapter_aux_phase: 15,
        ..Default::default()
    };
    let mut observer = |stage: FrameStage, observed_work: &mut [u8], _: &mut [u8]| -> Result<()> {
        if stage != FRAME_STAGE_TERRAIN {
            return Ok(());
        }
        assert_eq!(observed_work[WB], 1, "auxiliary seeded pixel");
        assert_eq!(observed_work[WB + WS], 2, "next-row seeded pixel");
        Err(crate::indexedmap::Error::new("stop after terrain evidence"))
    };
    let result = compose_frame_observed(&mut work, &mut vga, &mut in_, Some(&mut no_hud), Some(&mut observer));
    assert!(
        result.is_err() && work == before_work && vga == vec![0u8; vga.len()],
        "observer rejection did not preserve compositor atomicity"
    );
}

#[test]
fn test_compose_frame_preserves_native_layer_order() {
    let cache = cache0();
    let mut foreground = bank(12, 0);
    foreground.sprites[1] = solid(4); // tile 0 + foreground index-one rule
    let (mut work, mut vga) = (vec![0u8; 456 * 300], vec![0u8; VGA_SIZE]);
    let (t, r, u) = (bank(12, 1), bank(20, 2), bank(12, 3));
    let controls = [0x80u8, 0, 0, 0];
    let lut = vec![0u8; 256];
    let mut in_ = FrameInput {
        terrain_bank: Some(&t),
        range_bank: Some(&r),
        unit_bank: Some(&u),
        foreground_bank: Some(&foreground),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        lut: &lut,
        map_width: 13,
        range_mode: 1,
        units: vec![NativeUnitLayerEntry { x: 0, y: 0, slot: 0, ..Default::default() }],
        foreground_units: vec![NativeForegroundLayerEntry { x: 0, y: 0, ..Default::default() }],
        ..Default::default()
    };
    let mut hud = |frame: &mut [u8]| -> Result<()> {
        let off = WB;
        assert_eq!(frame[off], 4, "pre-HUD pixel, want foreground 4"); // terrain -> range -> unit -> foreground
        frame[off] = 5;
        Ok(())
    };
    compose_frame(&mut work, &mut vga, &mut in_, Some(&mut hud)).unwrap();
    assert_eq!(vga[SVO], 5, "viewport pixel, want HUD 5");
    assert!(vga[SVO - 1] == 0 && vga[SVO + SVW] == 0, "steady viewport overwrote the native four-pixel border");
}

#[test]
fn test_compose_frame_observer_reports_detached_ordered_stages() {
    let cache = cache0();
    let mut foreground = bank(12, 0);
    foreground.sprites[1] = solid(4);
    let (mut work, mut vga) = (vec![0u8; WS * 300], vec![0u8; VGA_SIZE]);
    let (t, r, u) = (bank(12, 1), bank(20, 2), bank(12, 3));
    let controls = [0x80u8, 0, 0, 0];
    let lut = vec![0u8; 256];
    let mut in_ = FrameInput {
        terrain_bank: Some(&t),
        range_bank: Some(&r),
        unit_bank: Some(&u),
        foreground_bank: Some(&foreground),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        lut: &lut,
        map_width: 13,
        range_mode: 1,
        units: vec![NativeUnitLayerEntry { x: 0, y: 0, slot: 0, ..Default::default() }],
        foreground_units: vec![NativeForegroundLayerEntry { x: 0, y: 0, ..Default::default() }],
        ..Default::default()
    };
    let mut stages = Vec::new();
    let mut hud = |frame: &mut [u8]| -> Result<()> {
        frame[WB] = 5;
        Ok(())
    };
    let mut observer = |stage: FrameStage, observed_work: &mut [u8], observed_vga: &mut [u8]| -> Result<()> {
        stages.push(stage);
        // Observer slices are detached. Deliberately corrupt both snapshots;
        // neither write may reach the pending transaction.
        observed_work[WB] = 0xee;
        observed_vga[SVO] = 0xee;
        Ok(())
    };
    compose_frame_observed(&mut work, &mut vga, &mut in_, Some(&mut hud), Some(&mut observer)).unwrap();
    let want = [
        FRAME_STAGE_TERRAIN,
        FRAME_STAGE_RANGE,
        FRAME_STAGE_UNITS,
        FRAME_STAGE_FOREGROUND,
        FRAME_STAGE_HUD,
        FRAME_STAGE_VIEWPORT,
    ];
    assert_eq!(stages, want, "observer stages");
    assert!(work[WB] == 5 && vga[SVO] == 5, "observer mutated transaction: work={:#x} vga={:#x}", work[WB], vga[SVO]);
}

#[test]
fn test_compose_frame_observer_failure_is_atomic() {
    let cache = cache0();
    let (mut work, mut vga) = (vec![0u8; WS * 300], vec![0u8; VGA_SIZE]);
    (work[WB], vga[SVO]) = (0x31, 0x32);
    let (before_work, before_vga) = (work.clone(), vga.clone());
    let (t, r, u, f) = (bank(12, 1), bank(20, 2), bank(12, 3), bank(12, 4));
    let controls = [0u8; 4];
    let lut = vec![0u8; 256];
    let mut in_ = FrameInput {
        terrain_bank: Some(&t),
        range_bank: Some(&r),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        lut: &lut,
        map_width: 13,
        range_mode: 1,
        ..Default::default()
    };
    let mut observer = |stage: FrameStage, _: &mut [u8], _: &mut [u8]| -> Result<()> {
        if stage == FRAME_STAGE_UNITS {
            return Err(crate::indexedmap::Error::new("stop after units"));
        }
        Ok(())
    };
    let result = compose_frame_observed(&mut work, &mut vga, &mut in_, Some(&mut no_hud), Some(&mut observer));
    assert!(
        result.is_err() && work == before_work && vga == before_vga,
        "observer failure leaked transaction: err={result:?}"
    );
}

#[test]
fn test_compose_frame_rejects_missing_hud_before_mutation() {
    let (mut work, mut vga) = (vec![0u8; 456 * 300], vec![0u8; VGA_SIZE]);
    let (before_work, before_vga) = (work.clone(), vga.clone());
    assert!(compose_frame(&mut work, &mut vga, &mut FrameInput::default(), None).is_err(), "incomplete input accepted");
    assert_eq!(work, before_work, "rejected input mutated work buffer");
    assert_eq!(vga, before_vga, "rejected input mutated VGA buffer");
}

#[test]
fn test_compose_frame_mode6_mutates_between_terrain_and_foreground_atomically() {
    let cache = cache0();
    let (t, r, u, f) = (bank(12, 1), bank(20, 2), bank(12, 3), bank(12, 4));
    let controls = [0u8; 4];
    let lut = vec![0u8; 256];
    let make_input = |cells: Vec<NativeTerrainCell>| FrameInput {
        terrain_bank: Some(&t),
        range_bank: Some(&r),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        cells,
        controls: &controls,
        lut: &lut,
        map_width: 13,
        range_mode: 6,
        cursor_x: 1,
        cursor_y: 0,
        ..Default::default()
    };
    let (mut work, mut vga) = (vec![0u8; WS * 320], vec![0u8; VGA_SIZE]);
    let mut in_ = make_input(cells_ff(13 * 8));
    compose_frame(&mut work, &mut vga, &mut in_, Some(&mut no_hud)).unwrap();
    let cells = &in_.cells;
    assert!(cells[1].blit_mode == 0 && cells[0].blit_mode == 0xff, "mode6 cells={:?}/{:?}", cells[0], cells[1]);

    let mut failed = make_input(cells_ff(13 * 8));
    let (before_work, before_vga) = (work.clone(), vga.clone());
    let mut failing_hud = |_: &mut [u8]| -> Result<()> { Err(crate::indexedmap::Error::new("HUD fail")) };
    assert!(
        compose_frame(&mut work, &mut vga, &mut failed, Some(&mut failing_hud)).is_err(),
        "mode6 frame accepted failed HUD"
    );
    assert!(
        failed.cells[1].blit_mode == 0xff && work == before_work && vga == before_vga,
        "failed mode6 frame leaked field or framebuffer mutation"
    );
}

#[test]
fn test_compose_native_frame_binds_recovered_hud_instead_of_callback() {
    // 0x11cfa passes work+0x8088 to 0x1acf3. The HUD offsets are relative to
    // that pointer and the following viewport copy uses the same base.
    let mut work = vec![0u8; WS * 320];
    let mut vga = vec![0u8; VGA_SIZE];
    let mut terrain = bank(2, 0);
    terrain.sprites[0] = solid(1);
    terrain.sprites[1] = solid(0x66);
    let range_bank = bank(20, 0);
    let mut units = bank(12, 0);
    units.sprites[0] = solid(2);
    units.sprites[1] = solid(0x77);
    let cache = cache0();
    let controls = [0u8, 2, 0, 0, 0, 2, 0, 0];
    let mut in_ = NativeFrameInput {
        frame: FrameInput {
            terrain_bank: Some(&terrain),
            range_bank: Some(&range_bank),
            unit_bank: Some(&units),
            foreground_bank: Some(&terrain),
            selector_cache: Some(&cache),
            cells: cells_ff(13 * 8),
            controls: &controls,
            map_width: 13,
            range_mode: 1,
            units: vec![NativeUnitLayerEntry { x: 0, y: 0, slot: 0, ..Default::default() }],
            ..Default::default()
        },
        hud: NativeMapHUDInput {
            display_gate_a: true,
            display_gate_b: true,
            anchor_x: 136,
            terrain_descriptor: 1,
            terrain_control: 2,
            optional_unit: Some(NativeMapHUDOptionalUnit { selector_slot: 0, raw_state: 3, current: 7, maximum: 8 }),
        },
        frames: hud_frames(),
        hud_terrain: Some(&terrain),
        hud_units: Some(&units),
        hud_cache: Some(&cache),
    };
    compose_native_frame(&mut work, &mut vga, &mut in_).unwrap();
    let layout = fdicon::native_map_hud_layout_for(136, WORK_STRIDE).unwrap();
    let px = |off: i64| work[WB + off as usize];
    assert!(
        layout.terrain == layout.unit
            && px(layout.frame) == 0x5a
            && px(layout.terrain) == 0x77
            && px(layout.unit) == 0x77
            && px(layout.hp) == 0x70,
        "native HUD missing from viewport-relative work: {:#x}/{:#x}/{:#x}/{:#x}",
        px(layout.frame),
        px(layout.terrain),
        px(layout.unit),
        px(layout.hp)
    );
    // 0x11eb0 places the same viewport-relative HUD position at VGA (4,4).
    let got = vga[(157 + 4) * VW + 136 + 4];
    assert_eq!(got, 0x5a, "native HUD did not reach viewport: {got:#x}");
}

#[test]
fn test_compose_native_transition_frame_preserves_native_layer_order() {
    let mut work = vec![0u8; WS * 320];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    let cache = cache0();
    let ident = identity();
    let pass = fdother::build_native_indexed_transition_pass(6, 6, 10, 0, 192).unwrap();
    let (t, u, f) = (bank(12, 7), bank(12, 0), bank(12, 0));
    let controls = [0u8; 4];
    let in_ = NativeTransitionFrameInput {
        terrain_bank: Some(&t),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        terrain_lut: &ident,
        map_width: 13,
        ..Default::default()
    };
    compose_native_transition_frame(&mut work, &mut vga, &in_, pass, &ident).unwrap();
    assert_eq!(vga[0], 7, "transition viewport first pixel want terrain 7");
}

#[test]
fn test_compose_native_transition_frame_rejects_missing_raw_input_atomically() {
    let mut work = vec![0u8; WS * 320];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    (work[0], vga[0]) = (9, 8);
    let pass = fdother::build_native_indexed_transition_pass(6, 6, 10, 0, 192).unwrap();
    assert!(
        compose_native_transition_frame(&mut work, &mut vga, &NativeTransitionFrameInput::default(), pass, &[])
            .is_err(),
        "missing transition input accepted"
    );
    assert!(work[0] == 9 && vga[0] == 8, "rejected transition mutated caller buffers");
}

#[test]
fn test_native_unit_present_snapshot_redraw_and_viewport_stay_separated() {
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    let cache = cache0();
    let ident = identity();
    let mut foreground = bank(12, 0);
    foreground.sprites[1] = solid(4);
    let (t, u) = (bank(12, 1), bank(12, 3));
    let controls = [0x80u8, 0, 0, 0];
    let in_ = NativeTransitionFrameInput {
        terrain_bank: Some(&t),
        unit_bank: Some(&u),
        foreground_bank: Some(&foreground),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        terrain_lut: &ident,
        map_width: 13,
        units: vec![NativeUnitLayerEntry { x: 0, y: 0, slot: 0, ..Default::default() }],
        foreground_units: vec![NativeForegroundLayerEntry { x: 0, y: 0, ..Default::default() }],
        ..Default::default()
    };
    compose_native_unit_present_terrain_snapshot(&mut work, &in_).unwrap();
    assert_eq!(work[WB], 1, "terrain snapshot pixel");
    redraw_native_unit_present_objects(&mut work, &in_).unwrap();
    assert_eq!(work[WB], 4, "object redraw pixel");
    for y in 0..VIEW_HEIGHT as usize {
        for x in 312..VW {
            vga[y * VW + x] = 0xee;
        }
    }
    copy_native_unit_present_viewport(&mut vga, &work).unwrap();
    assert!(vga[0] == 4 && vga[312] == 0xee, "unit-present viewport={} tail={:#x}", vga[0], vga[312]);
}

#[test]
fn test_native_unit_present_snapshot_rejects_wrong_allocation_atomically() {
    let mut work = vec![0u8; WORK_SIZE - 1];
    work[0] = 9;
    let r = compose_native_unit_present_terrain_snapshot(&mut work, &NativeTransitionFrameInput::default());
    assert!(r.is_err() && work[0] == 9, "short snapshot mutation/error={r:?} first={}", work[0]);
    let r = redraw_native_unit_present_objects(&mut work, &NativeTransitionFrameInput::default());
    assert!(r.is_err() && work[0] == 9, "short redraw mutation/error={r:?} first={}", work[0]);
}

#[test]
fn test_compose_native_unit_present_intro_and_lut_frames_produce_viewport() {
    let mut work = vec![0u8; WORK_SIZE];
    let mut snapshot = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    let cache = cache0();
    let ident = identity();
    let (t, u, f) = (bank(12, 1), bank(12, 3), bank(12, 0));
    let controls = [0u8; 4];
    let in_ = NativeTransitionFrameInput {
        terrain_bank: Some(&t),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        terrain_lut: &ident,
        map_width: 13,
        ..Default::default()
    };
    compose_native_unit_present_terrain_snapshot(&mut snapshot, &in_).unwrap();
    let entry = LMI1Entry { width: 1, height: 1, pixels: vec![7] };
    // Native origin for map(0,0), camera(0,0) is viewport x=0,y=1.
    compose_native_unit_present_intro_frame(&mut work, &mut vga, &snapshot, &in_, &entry, 0, 0).unwrap();
    assert_eq!(vga[VW], 7, "intro viewport LMI pixel");
    let mut lut_snapshot = vec![0u8; WORK_SIZE];
    compose_native_unit_present_lut_snapshot(&mut lut_snapshot, &snapshot, &in_, &entry, 0, 0).unwrap();
    let lmi_origin = fdother::native_unit_present_byte_origin(0, 0, 0, 0) as usize;
    assert!(
        lut_snapshot[lmi_origin] == 7 && snapshot[lmi_origin] != 7,
        "LUT snapshot did not preserve separate terrain+final-LMI phase: lut={} terrain={}",
        lut_snapshot[lmi_origin],
        snapshot[lmi_origin]
    );
    let frames = fdother::native_unit_present_lut_frames(3, 2).unwrap();
    let lut = identity();
    compose_native_unit_present_lut_frame(&mut work, &mut vga, &lut_snapshot, &lut, &in_, &frames[5]).unwrap();
    assert_eq!(vga[0], 1, "LUT viewport terrain pixel");
}

#[test]
fn test_compose_native_unit_present_lut_snapshot_rejects_invalid_input_atomically() {
    let mut dst = vec![0u8; WORK_SIZE];
    dst[0] = 9;
    let entry = LMI1Entry { width: 1, height: 1, pixels: vec![7] };
    let r = compose_native_unit_present_lut_snapshot(
        &mut dst,
        &vec![0u8; WORK_SIZE - 1],
        &NativeTransitionFrameInput::default(),
        &entry,
        0,
        0,
    );
    assert!(r.is_err() && dst[0] == 9, "invalid LUT snapshot input mutated destination: err={r:?} first={}", dst[0]);
}

#[test]
fn test_native_unit_present_strip_layout_matches22390() {
    let top = native_unit_present_strip_layout_for(7, 3, 5, 3);
    assert_eq!(
        top,
        NativeUnitPresentStripLayout { work_offset: WORK_BASE + 2 * 24, vga_offset: 2 * 24, rows: 18 },
        "top-row layout"
    );
    let lower = native_unit_present_strip_layout_for(7, 6, 5, 3);
    assert_eq!(
        lower,
        NativeUnitPresentStripLayout {
            work_offset: WORK_BASE + 2 * 24 + (3 * 24 - 6) * WORK_STRIDE,
            vga_offset: 2 * 24 + (3 * 24 - 6) * VIEW_WIDTH,
            rows: 24,
        },
        "lower layout"
    );
}

#[test]
fn test_run_native_unit_present_strip_bridge_copies_rows_progressively() {
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    let layout = native_unit_present_strip_layout_for(2, 2, 0, 0);
    for row in 0..layout.rows {
        for x in 0..24 {
            work[(layout.work_offset + row * WORK_STRIDE + x) as usize] = (row + 1) as u8;
        }
    }
    let mut delayed = Vec::new();
    let mut delay = |row: i64, vga: &[u8]| -> Result<()> {
        delayed.push(row);
        let got = vga[(layout.vga_offset + row * VIEW_WIDTH) as usize];
        assert_eq!(got, (row + 1) as u8, "row {row} was not visible before delay");
        Ok(())
    };
    run_native_unit_present_strip_bridge(&work, &mut vga, 2, 2, 0, 0, Some(&mut delay)).unwrap();
    assert_eq!(delayed.len(), 24, "delay calls");
    assert_eq!(vga[(layout.vga_offset + 23 * VIEW_WIDTH + 23) as usize], 24, "last 24-byte row was not copied");
}

#[test]
fn test_run_native_unit_present_strip_bridge_preflights_bounds() {
    let work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    vga[0] = 9;
    let mut calls = 0;
    let mut delay = |_: i64, _: &[u8]| -> Result<()> {
        calls += 1;
        Ok(())
    };
    let r = run_native_unit_present_strip_bridge(&work, &mut vga, 0, 0, 5, 5, Some(&mut delay));
    assert!(
        r.is_err() && calls == 0 && vga[0] == 9,
        "out-of-range bridge was not atomic: err={r:?} calls={calls} first={}",
        vga[0]
    );
}

#[test]
fn test_compose_native_unit_present_strip_bridge_uses_no_full_viewport_copy() {
    let mut work = vec![0u8; WORK_SIZE];
    let mut snapshot = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; (VIEW_WIDTH * VIEW_HEIGHT) as usize];
    let cache = cache0();
    let ident = identity();
    let (u, f) = (bank(12, 3), bank(12, 0));
    let controls = [0u8; 4];
    let in_ = NativeTransitionFrameInput {
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        terrain_lut: &ident,
        map_width: 13,
        ..Default::default()
    };
    let layout = native_unit_present_strip_layout_for(2, 2, 0, 0);
    for row in 0..layout.rows {
        for x in 0..24 {
            snapshot[(layout.work_offset + row * WORK_STRIDE + x) as usize] = (row + 1) as u8;
        }
    }
    vga[0] = 0xa5; // outside the revealed strip; a full viewport copy would overwrite it.
    let mut delays = 0;
    let mut delay = |_: i64, _: &[u8]| -> Result<()> {
        delays += 1;
        Ok(())
    };
    compose_native_unit_present_strip_bridge(
        &mut work,
        &mut vga,
        &snapshot,
        &ident,
        &in_,
        2,
        2,
        2,
        2,
        Some(&mut delay),
    )
    .unwrap();
    assert!(delays == 24 && vga[0] == 0xa5, "bridge delays={delays} untouched VGA={:#x}", vga[0]);
    assert_eq!(vga[layout.vga_offset as usize], 1, "first revealed row");
}

#[test]
fn test_build_native_terrain_cells_requires_exporter_arrays() {
    let cells = build_native_terrain_cells(&[1, 2], &[0xff, 0x00]);
    let ok = matches!(&cells, Ok(c) if c.len() == 2 && c[0].tile == 1 && c[1].blit_mode == 0);
    assert!(ok, "cells={cells:?}");
    for (tiles, modes) in [(vec![1i64], vec![]), (vec![0x400], vec![0u8])] {
        assert!(
            build_native_terrain_cells(&tiles, &modes).is_err(),
            "accepted incomplete/invalid arrays: {tiles:?} {modes:?}"
        );
    }
}

/// TestComposeNativeStepFrameOmitsRangeAndHUD 釘住走行重繪的層集合。
///
/// 原版走行期間每一幀只進地形、單位與前景（`0x11EEE`／`0x127E0`／`0x127A9`），
/// `0x122DC` 與 `0x1AD72` 一次都沒有，即使 overlay selector 仍是 1。所以同一份
/// 輸入交給走行重繪，結果必須等於「range mode 0 ＋ 空 HUD」的整幀排程，而且
/// 與輸入的 RangeMode 無關。
#[test]
fn test_compose_native_step_frame_omits_range_and_hud() {
    let cache = cache0();
    let mut foreground = bank(12, 0);
    foreground.sprites[1] = solid(4);
    let (t, r, u) = (bank(12, 1), bank(20, 2), bank(12, 3));
    let controls = [0x80u8, 0, 0, 0];
    let lut = vec![0u8; 256];
    let new_input = |range_mode: i64| FrameInput {
        terrain_bank: Some(&t),
        range_bank: Some(&r),
        unit_bank: Some(&u),
        foreground_bank: Some(&foreground),
        selector_cache: Some(&cache),
        cells: cells_ff(13 * 8),
        controls: &controls,
        lut: &lut,
        map_width: 13,
        range_mode,
        // 單位與前景放在別格，游標格才看得到範圍圖示有沒有被畫上去；
        // 疊在同一格會被後面的層蓋掉，比較就失去鑑別力。
        units: vec![NativeUnitLayerEntry { x: 2, y: 2, slot: 0, ..Default::default() }],
        foreground_units: vec![NativeForegroundLayerEntry { x: 2, y: 2, ..Default::default() }],
        ..Default::default()
    };
    let (mut base_work, mut base_vga) = (vec![0u8; 456 * 300], vec![0u8; VGA_SIZE]);
    compose_frame(&mut base_work, &mut base_vga, &mut new_input(0), Some(&mut no_hud)).unwrap();
    for mode in [0, 1, 3, 5] {
        let (mut step_work, mut step_vga) = (vec![0u8; 456 * 300], vec![0u8; VGA_SIZE]);
        compose_native_step_frame(&mut step_work, &mut step_vga, &new_input(mode))
            .unwrap_or_else(|e| panic!("range mode {mode}: {e}"));
        assert_eq!(step_vga, base_vga, "range mode {mode} 的走行重繪畫出了範圍／游標圖示或 HUD");
    }
    let (mut with_range, mut range_vga) = (vec![0u8; 456 * 300], vec![0u8; VGA_SIZE]);
    compose_frame(&mut with_range, &mut range_vga, &mut new_input(1), Some(&mut no_hud)).unwrap();
    assert_ne!(range_vga, base_vga, "測試素材畫不出範圍圖示，這個比較沒有鑑別力");
}

/// Invalid recovered step tuples must not partially publish terrain or VGA.
#[test]
fn test_native_up_step_rejects_invalid_input_atomically() {
    let (t, u, f) = (bank(12, 31), bank(12, 42), bank(12, 53));
    let cache = NativeSelectorCache::default();
    let controls = [0u8; 4];
    let lut = vec![0u8; 256];
    let aux = NativeChapterAuxSurface::default();
    let parallax = NativeMapParallaxSurface::default();
    let valid = FrameInput {
        terrain_bank: Some(&t),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        map_width: 13,
        cells: vec![NativeTerrainCell::default(); 13 * 12],
        controls: &controls,
        lut: &lut,
        camera_y: 1,
        ..Default::default()
    };
    for (name, motion) in
        [("motion0", 0), ("motion7", 7), ("missingbank", 6), ("negativecamera", 6), ("aux", 6), ("parallax", 6)]
    {
        let mut in_ = valid.clone();
        match name {
            "missingbank" => in_.terrain_bank = None,
            "negativecamera" => in_.camera_y = -1,
            "aux" => in_.chapter_aux = Some(&aux),
            "parallax" => in_.parallax = Some(&parallax),
            _ => {}
        }
        let mut work = vec![69u8; WORK_SIZE];
        let mut vga = vec![87u8; VGA_SIZE];
        let (old_work, old_vga) = (work.clone(), vga.clone());
        assert!(
            compose_native_up_step_frame(&mut work, &mut vga, &in_, 2, motion).is_err(),
            "{name}: invalid input accepted"
        );
        assert!(work == old_work && vga == old_vga, "{name}: invalid input partly published");
    }
}
