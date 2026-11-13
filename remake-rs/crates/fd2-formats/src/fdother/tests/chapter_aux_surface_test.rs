//! 移植自 `chapter_aux_surface_test.go`。

use super::original;
use crate::fdother::*;

#[test]
fn decode_native_chapter_aux_surface_from_player_archive() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    assert_eq!(decode_native_chapter_aux_surface(&path).unwrap().pixels.len(), 320 * 200);
}

#[test]
fn blit_native_chapter_aux_viewport_uses_raw_row_phase_table() {
    let pixels: Vec<u8> = (0..200).flat_map(|_| (0..320).map(|x| x as u8)).collect();
    let mut dst = vec![0u8; 320 * 192];
    blit_native_chapter_aux_viewport(&mut dst, 320, &NativeChapterAuxSurface { pixels }, 15).unwrap();
    assert_eq!(dst[0], 1, "row0 first pixel");
    assert_eq!(dst[320], 2, "row1 first pixel");
    assert_eq!(dst[312], 0, "copy crossed 312-byte viewport");
}

#[test]
fn blit_native_chapter_aux_viewport_rejects_without_mutation() {
    let mut dst = vec![0u8; 320 * 192];
    dst[0] = 0x7a;
    let surface = NativeChapterAuxSurface { pixels: vec![0; 320 * 200] };
    assert!(blit_native_chapter_aux_viewport(&mut dst, 320, &surface, 16).is_err(), "expected phase rejection");
    assert_eq!(dst[0], 0x7a, "rejected auxiliary viewport mutated destination");
}
