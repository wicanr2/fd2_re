//! 移植自 `remake/internal/ending/finale_test.go`。

use super::*;
use crate::test_support::{input, remake_asset};

fn phase() -> Option<FinalePhase> {
    let path = remake_asset("endings/native_2c405.json")?;
    Some(load_finale_phase(&path).unwrap())
}

fn shared(c: IndexedCompositor) -> SharedCompositor {
    std::sync::Arc::new(std::sync::Mutex::new(c))
}

#[test]
fn test_native2_c405_phase0_is_editable_but_fail_closed() {
    let Some(phase) = phase() else { return };
    assert!(!phase.ready(), "unrecovered finale text compositor must not be playable");
    let p = &phase.phase;
    assert!(
        p.source_dat == "FDTXT_031"
            && p.string_index == 44
            && p.script == "ch32.json"
            && p.scene_index == 0
            && p.line == 0
            && p.count == 1,
        "editable script reference = {p:?}"
    );
    assert!(
        p.staging_bytes == 0x36b00
            && p.text_offset == 0x12c30
            && p.stride == 320
            && p.viewport_rows == 200
            && p.iterations == 500
            && p.delay_ms == 1
            && p.baseline_palette_initial_delta == 40
            && p.fade_out_through_iteration == 300
            && p.palette_step_cadence == 5,
        "phase-0 native schedule = {p:?}"
    );
}

#[test]
fn test_phase0_composes_exact_raw_glyph_only_string() {
    let Some(text_path) = input("FD2_EXTRACTED_RAW_DIR", "FDTXT/FDTXT_031.bin") else { return };
    let Some(font_path) = input("FD2_EXTRACTED_RAW_DIR", "FDOTHER/FDOTHER_004.bin") else { return };
    let Some(phase) = phase() else { return };
    let text = std::fs::read(text_path).unwrap();
    let font_data = std::fs::read(font_path).unwrap();
    let mut staging = vec![0u8; phase.phase.staging_bytes as usize];
    let end = phase.compose_phase0_text(&mut staging, &text, &font_data).unwrap();
    let p = &phase.phase;
    assert_eq!(end, p.text_offset + 9 * p.line_advance_rows * p.stride + 12 * fdtxt::GLYPH_WIDTH as i64, "cursor");
    let strings = fdtxt::parse(&text).unwrap();
    let words = strings.words(p.string_index).unwrap_or_default();
    let font = fdtxt::parse_font(&font_data).unwrap();
    let mut base = p.text_offset;
    let mut glyph_index = 0;
    for word in &words {
        if *word == 0xfffe {
            base += p.line_advance_rows * p.stride;
            continue;
        }
        for y in 0..fdtxt::GLYPH_HEIGHT as i64 {
            for x in 0..fdtxt::GLYPH_WIDTH as i64 {
                if !font.glyph_bit(i64::from(*word), x, y).unwrap_or(false) {
                    continue;
                }
                let pos = (base + y * p.stride + x) as usize;
                assert_eq!(staging[pos], 0xcd, "glyph {glyph_index} bit({x},{y}) foreground");
                return;
            }
        }
        base += fdtxt::GLYPH_WIDTH as i64;
        glyph_index += 1;
    }
    panic!("phase text contained no set glyph bit");
}

#[test]
fn test_phase0_player_uses_native_scroll_and_palette_cadence_then_stops() {
    let Some(phase) = phase() else { return };
    let mut staging = vec![0u8; phase.phase.staging_bytes as usize];
    for row in 0..(phase.phase.staging_bytes / phase.phase.stride) {
        staging[(row * phase.phase.stride) as usize] = row as u8;
    }
    let mut compositor = new_indexed_compositor();
    compositor.baseline[0] = 50;
    let c = shared(compositor);
    let mut p = new_phase0_player(&phase, &staging, Some(c.clone())).unwrap();
    let state = |c: &SharedCompositor| {
        let c = c.lock().unwrap();
        (c.vga[0], c.palette[0])
    };
    let done = p.advance(0);
    let (pixel, palette) = state(&c);
    assert!(
        matches!(done, Ok(false)) && pixel == 0 && palette == 10 && p.palette_delta == 39,
        "i0 done={done:?} pixel={pixel} palette={palette} nextDelta={}",
        p.palette_delta
    );
    let done = p.advance(1);
    let (pixel, palette) = state(&c);
    assert!(
        matches!(done, Ok(false)) && pixel == 1 && palette == 11,
        "i1 done={done:?} pixel={pixel} palette={palette}"
    );
    (p.iteration, p.palette_delta, p.wait_ms) = (195, 1, 0);
    let done = p.advance(0);
    let (_, palette) = state(&c);
    assert!(
        matches!(done, Ok(false)) && palette == 49 && p.palette_delta == 0,
        "fadeout done={done:?} palette={palette} nextDelta={}",
        p.palette_delta
    );
    (p.iteration, p.palette_delta, p.wait_ms) = (305, 0, 0);
    let done = p.advance(0);
    let (_, palette) = state(&c);
    assert!(
        matches!(done, Ok(false)) && palette == 50 && p.palette_delta == 1,
        "fadein done={done:?} palette={palette} nextDelta={}",
        p.palette_delta
    );
    (p.iteration, p.palette_delta, p.wait_ms) = (499, 0, 0);
    let done = p.advance(1);
    assert!(matches!(done, Ok(true)), "final phase done={done:?}");
}

#[test]
fn test_phase0_assets_require_explicit_baseline() {
    let Some(phase) = phase() else { return };
    let Some(text_root) = input("FD2_ASSET_PACK", "text") else { return };
    let Some(font_root) = input("FD2_ASSET_PACK", "fonts") else { return };
    let strings = fdtxt::load_separated_resource(&text_root, 31).unwrap();
    let font = fdtxt::load_separated_font(&font_root).unwrap();
    let mut baseline = [0u8; 768];
    baseline[0] = 50;
    let c = shared(new_indexed_compositor());
    let mut p = new_phase0_player_from_assets(
        &phase,
        &Phase0Assets { strings: Some(strings), font: Some(font), baseline, ..Default::default() },
        Some(c.clone()),
    )
    .unwrap();
    let done = p.advance(0);
    let palette = c.lock().unwrap().palette[0];
    assert!(matches!(done, Ok(false)) && palette == 10, "done={done:?} palette={palette}");
}
